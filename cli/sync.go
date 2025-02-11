package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MingxuanGame/OsuBeatmapSync/application"
	"github.com/MingxuanGame/OsuBeatmapSync/base_service"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/MingxuanGame/OsuBeatmapSync/onedrive"
	"github.com/MingxuanGame/OsuBeatmapSync/osu"
	downloader "github.com/MingxuanGame/OsuBeatmapSync/osu/download"
	"github.com/MingxuanGame/OsuBeatmapSync/osu/sync"
	"github.com/MingxuanGame/OsuBeatmapSync/utils"
	"os"
	"strconv"
	"time"
)

func syncAllBeatmapset(config *Config, metadata *Metadata, graph *onedrive.GraphClient, downloaders []downloader.BeatmapDownloader, needSyncBeatmaps []BeatmapsetMetadata, ctx context.Context) bool {

	s := sync.NewSyncer(ctx, metadata, graph, config)
	s.SyncNewBeatmap(downloaders, needSyncBeatmaps)

	err := application.SaveMetadataToLocal(s.Metadata)
	if err != nil {
		return false
	}
	for {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		if len(s.Failed) == 0 {
			break
		}
		logger.Info().Msgf("Failed: %d", len(s.Failed))
		logger.Info().Msg("Retry in 1 minute...")
		time.Sleep(time.Minute)
		s.ReSync(downloaders)
		err := application.SaveMetadataToLocal(s.Metadata)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to save metadata")
			return false
		}
	}
	return true
}

func getNeedSyncBeatmapsLocal(filename string, metadata *Metadata) ([]BeatmapsetMetadata, error, bool) {
	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		return nil, nil, false
	}
	if err != nil {
		return nil, err, false
	}
	var need []BeatmapsetMetadata
	var result []BeatmapsetMetadata
	err = json.Unmarshal(data, &need)
	if err != nil {
		return nil, err, false
	}
	for _, info := range need {
		if _, ok := metadata.Beatmapsets[info.BeatmapsetId]; !ok {
			result = append(result, info)
		} else {
			if !info.Equal(metadata.Beatmapsets[info.BeatmapsetId]) {
				result = append(result, info)
			}
		}
	}
	return result, nil, true
}

func getNeedSyncBeatmaps(metadata *Metadata, osuClient *osu.LegacyOfficialClient, ctx context.Context, since time.Time) ([]BeatmapsetMetadata, error) {
	needSyncBeatmaps, err, ok := getNeedSyncBeatmapsLocal("needSync.json", metadata)
	if err != nil {
		return nil, err
	}
	if ok {
		return needSyncBeatmaps, nil
	}

	lastTime := since
	for _, mode := range metadata.GameMode {
		modeTime := time.Unix(mode.UpdateTime, 0)
		if modeTime.Before(lastTime) {
			lastTime = modeTime
		}
	}
	logger.Info().Msgf("Getting all beatmaps since %s", lastTime.Format(time.RFC3339))
	allSyncBeatmapset, err := application.GetAllNewBeatmapset(ctx, osuClient, &lastTime)
	if err != nil {
		return nil, err
	}

	logger.Info().Msgf("Total: %d, end: %s", len(allSyncBeatmapset), lastTime.Format(time.RFC3339))

	for beatmapsetId, info := range allSyncBeatmapset {
		if _, ok := metadata.Beatmapsets[beatmapsetId]; !ok {
			needSyncBeatmaps = append(needSyncBeatmaps, info)
		} else {
			if !info.Equal(metadata.Beatmapsets[beatmapsetId]) {
				needSyncBeatmaps = append(needSyncBeatmaps, info)
			}
		}
	}
	return needSyncBeatmaps, nil
}

func SyncBeatmaps(ctx context.Context, tasks, worker int, start bool, since time.Time) error {
	config, err := base_service.LoadConfig()
	if err != nil {
		return err
	}

	osuClient := osu.NewLegacyOfficialClient(config.Osu.V1ApiKey)
	client, err := application.Login(&config, ctx)
	if err != nil {
		return err
	}
	metadata, err := application.GetMetadata(client, config.Path.Root)
	if err != nil {
		return err
	}

	var needSyncBeatmaps []BeatmapsetMetadata
	if worker == 0 {
		var ok bool
		needSyncBeatmaps, err, ok = getNeedSyncBeatmapsLocal("needSync.json", &metadata)
		if err != nil {
			return err
		}
		if !ok {
			needSyncBeatmaps, err = getNeedSyncBeatmaps(&metadata, osuClient, ctx, since)
			if err != nil {
				return err
			}
		}
		logger.Info().Msgf("Need sync: %d", len(needSyncBeatmaps))
		if tasks > 1 {
			taskList := utils.SplitSlice(needSyncBeatmaps, tasks)
			for i, task := range taskList {
				taskJson, err := json.Marshal(task)
				if err != nil {
					return err
				}
				err = os.WriteFile("needSync"+strconv.Itoa(i+1)+".json", taskJson, 0644)
				if err != nil {
					return err
				}
			}
			worker = 1
		} else {
			saveNeedSync, err := json.Marshal(needSyncBeatmaps)
			if err != nil {
				return err
			}
			err = os.WriteFile("needSync.json", saveNeedSync, 0644)
			if err != nil {
				return err
			}
		}
	}

	var downloaders []downloader.BeatmapDownloader
	if config.Osu.EnableSayobot {
		downloaders = append(downloaders, downloader.NewSayobotDownloader(config.Osu.Sayobot.Server, ctx))
	}
	if config.Osu.EnableNerinyan {
		downloaders = append(downloaders, downloader.NewNerinyanDownloader(ctx))
	}
	if config.Osu.EnableCatboy {
		downloaders = append(downloaders, downloader.NewCatboyDownloader(ctx))
	}
	if config.Osu.EnableOfficial {
		d, err := downloader.NewOfficialDownloader(ctx, config.Osu.OfficialDownloader.AccessToken, config.Osu.OfficialDownloader.RefreshToken)
		if err != nil {
			return err
		}
		downloaders = append(downloaders, d)

	}
	if len(downloaders) == 0 {
		logger.Fatal().Msg("No downloader enabled, exiting...")
	} else {
		logger.Info().Msgf("Enabled downloader:")
		for _, _downloader := range downloaders {
			logger.Info().Msgf("  %s", _downloader.Name())
		}
	}
	logger.Info().Msg("Start Syncing...")
	if worker == 0 {
		finished := syncAllBeatmapset(&config, &metadata, client, downloaders, needSyncBeatmaps, ctx)
		if finished {
			err := application.UploadMetadata(client, config.Path.Root, &metadata)
			if err != nil {
				return err
			}
		}
	} else if start {
		logger.Info().Msgf("Current worker: %d", worker)
		needSyncBeatmaps, err, ok := getNeedSyncBeatmapsLocal("needSync"+strconv.Itoa(worker)+".json", &metadata)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("no needSync file")
		}
		finished := syncAllBeatmapset(&config, &metadata, client, downloaders, needSyncBeatmaps, ctx)
		if !finished {
			saveNeedSync, err := json.Marshal(needSyncBeatmaps)
			if err != nil {
				return err
			}
			err = os.WriteFile("needSync"+strconv.Itoa(worker)+".json", saveNeedSync, 0644)
			if err != nil {
				return err
			}
			return nil
		}
	}

	logger.Info().Msg("Sync finished...")
	return nil
}
