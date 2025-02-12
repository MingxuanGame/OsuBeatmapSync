package application

import (
	"context"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/MingxuanGame/OsuBeatmapSync/osu"
	"github.com/MingxuanGame/OsuBeatmapSync/utils"
	"github.com/rs/zerolog/log"
	"time"
)

func GetNewBeatmapset(client *osu.LegacyOfficialClient, since time.Time, lastBeatmapsetInfo map[int]BeatmapsetMetadata) (time.Time, bool, error) {
	beatmaps, err := client.GetBeatmapByTime(since)
	if err != nil {
		return since, false, err
	}
	if len(*beatmaps) == 0 {
		return since, true, nil
	}
	lastTime := since
	for _, info := range *beatmaps {
		beatmapset, ok := lastBeatmapsetInfo[info.BeatmapsetId]
		if !ok {
			beatmapset = BeatmapsetMetadata{
				BeatmapsetId: info.BeatmapsetId,
				Beatmaps:     make(map[int]BeatmapMetadata),
				LastUpdate:   0,
			}
		}
		beatmap := BeatmapMetadata{
			Beatmap:        info,
			ApprovedDate:   utils.MustParseTime(info.ApprovedDate, time.DateTime).Unix(),
			SubmitDate:     utils.MustParseTime(info.SubmitDate, time.DateTime).Unix(),
			LastUpdate:     utils.MustParseTime(info.LastUpdate, time.DateTime).Unix(),
			NoAudio:        utils.Itob(info.NoAudio),
			CannotDownload: utils.Itob(info.CannotDownload),
			HasStoryboard:  utils.Itob(info.HasStoryboard),
			HasVideo:       utils.Itob(info.HasVideo),
			Link:           make(map[string]string),
			Path:           make(map[string]string),
		}
		if beatmap.HasVideo {
			beatmapset.HasVideo = true
		}
		if beatmap.HasStoryboard {
			beatmapset.HasStoryboard = true
		}
		if beatmap.CannotDownload {
			beatmapset.CannotDownload = true
		}
		if beatmap.NoAudio {
			beatmapset.NoAudio = true
		}
		beatmapset.Beatmaps[info.BeatmapId] = beatmap
		lastBeatmapsetInfo[info.BeatmapsetId] = beatmapset
		currentTime, err := time.Parse(time.DateTime, info.ApprovedDate)
		if err != nil {
			continue
		}
		if currentTime.After(lastTime) {
			lastTime = currentTime
		}
	}
	return lastTime, false, nil
}

func GetAllNewBeatmapset(ctx context.Context, osuClient *osu.LegacyOfficialClient, lastTime *time.Time) (map[int]BeatmapsetMetadata, error) {
	allSyncBeatmapset := make(map[int]BeatmapsetMetadata)
	var err error
	for {
		select {
		case <-ctx.Done():
			return allSyncBeatmapset, nil
		default:
		}
		var isEnd bool
		*lastTime, isEnd, err = GetNewBeatmapset(osuClient, *lastTime, allSyncBeatmapset)
		log.Info().Msgf("  To: %s, total: %d", lastTime.Format(time.DateTime), len(allSyncBeatmapset))
		if err != nil {
			return nil, err
		}
		if isEnd {
			break
		}
	}
	return allSyncBeatmapset, nil
}
