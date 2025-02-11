package application

import (
	"context"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/MingxuanGame/OsuBeatmapSync/osu"
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
			Status:        info.Status,
			Artist:        info.Artist,
			ArtistUnicode: info.ArtistUnicode,
			Title:         info.Title,
			TitleUnicode:  info.TitleUnicode,
			BeatmapsetId:  info.BeatmapsetId,
			BeatmapId:     info.BeatmapId,
			GameMode:      info.Mode,
			Creator:       info.Creator,
			HasStoryboard: info.HasStoryBoard == 1,
			HasVideo:      info.HasVideo == 1,
		}
		if beatmap.HasVideo {
			beatmapset.HasVideo = true
		}
		if beatmap.HasStoryboard {
			beatmapset.HasStoryboard = true
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
		if err != nil {
			return nil, err
		}
		if isEnd {
			break
		}
	}
	return allSyncBeatmapset, nil
}
