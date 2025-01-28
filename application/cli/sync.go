package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MingxuanGame/OsuBeatmapSync/application"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/MingxuanGame/OsuBeatmapSync/onedrive"
	"github.com/MingxuanGame/OsuBeatmapSync/osu"
	downloader "github.com/MingxuanGame/OsuBeatmapSync/osu/download"
	"github.com/MingxuanGame/OsuBeatmapSync/utils"
	"github.com/rs/zerolog/log"
	"os"
	"strconv"
	"time"
)

func sync(config *Config, metadata *Metadata, graph *onedrive.GraphClient, downloaders []downloader.BeatmapDownloader, needSyncBeatmaps []BeatmapsetMetadata, ctx context.Context) bool {
	modeMap := map[GameMode]string{
		GameModeOsu:   config.Path.StdPath,
		GameModeTaiko: config.Path.TaikoPath,
		GameModeCtb:   config.Path.CatchPath,
		GameModeMania: config.Path.ManiaPath,
	}
	statusMap := map[BeatmapStatus]string{
		StatusRanked:    config.Path.RankedPath,
		StatusLoved:     config.Path.LovedPath,
		StatusApproved:  config.Path.RankedPath,
		StatusQualified: config.Path.QualifiedPath,
	}

	retry := downloader.SyncNewBeatmap(metadata, graph, config.Path.Root, downloaders, needSyncBeatmaps, config.General.MaxConcurrent, modeMap, statusMap, ctx)

	err := application.SaveMetadataToLocal(metadata)
	if err != nil {
		return false
	}
	select {
	case <-ctx.Done():
		return false
	default:
	}
	for {
		if len(retry) == 0 {
			break
		}
		log.Info().Msgf("Failed: %d", len(retry))
		time.Sleep(time.Minute)
		retry = downloader.SyncNewBeatmap(metadata, graph, config.Path.Root, downloaders, retry, config.General.MaxConcurrent, modeMap, statusMap, ctx)
		err := application.SaveMetadataToLocal(metadata)
		if err != nil {
			log.Error().Err(err).Msg("Failed to save metadata")
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
	log.Info().Msgf("Getting all beatmaps since %s", lastTime.Format(time.RFC3339))
	allSyncBeatmapset, err := application.GetAllNewBeatmapset(ctx, osuClient, lastTime)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("Total: %d, end: %s", len(allSyncBeatmapset), lastTime.Format(time.RFC3339))

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
	config, err := application.LoadConfig()
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
		needSyncBeatmaps, err := getNeedSyncBeatmaps(&metadata, osuClient, ctx, since)
		if err != nil {
			return err
		}
		log.Info().Msgf("Need sync: %d", len(needSyncBeatmaps))
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
	log.Info().Msg("Start Syncing...")
	if worker == 0 {
		sync(&config, &metadata, client, downloaders, needSyncBeatmaps, ctx)
		err := application.UploadMetadata(client, config.Path.Root, &metadata)
		if err != nil {
			return err
		}
	} else if start {
		log.Info().Msgf("Current worker: %d", worker)
		needSyncBeatmaps, err, ok := getNeedSyncBeatmapsLocal("needSync"+strconv.Itoa(worker)+".json", &metadata)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("no needSync file")
		}
		finished := sync(&config, &metadata, client, downloaders, needSyncBeatmaps, ctx)
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

	log.Info().Msg("Sync finished...")
	return nil
}
