package file

import (
	"context"
	"fmt"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	. "github.com/MingxuanGame/OsuBeatmapSync/model/onedrive"
	"github.com/MingxuanGame/OsuBeatmapSync/onedrive"
	"github.com/MingxuanGame/OsuBeatmapSync/osu"
	"github.com/MingxuanGame/OsuBeatmapSync/utils"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

func generateSingleFileMetadata(item DriveItem, client *osu.LegacyOfficialClient, graph *onedrive.GraphClient) (string, []BeatmapMetadata, error) {
	filename := item.Name
	var result []BeatmapMetadata
	_, _, beatmapsetId := utils.ParseFilename(filename)
	apiData, err := client.GetBeatmapBySetId(strconv.Itoa(beatmapsetId))
	if err != nil {
		return "", nil, err
	}
	path := item.ParentReference.Path + "/" + item.Name
	fileStruct, err := ParseFilenameStruct(path)
	if err != nil {
		return "", nil, err
	}
	beatmapType := fileStruct.Type
	link, err := onedrive.MakeShareLink(graph, item.Id)
	if err != nil {
		return "", nil, fmt.Errorf("[onedrive] failed to make share link: %w", err)
	}
	for _, data := range *apiData {
		metadata := BeatmapMetadata{
			Artist:        data.Artist,
			Title:         data.Title,
			ArtistUnicode: data.ArtistUnicode,
			TitleUnicode:  data.TitleUnicode,
			BeatmapId:     data.BeatmapId,
			GameMode:      data.Mode,
			Creator:       data.Creator,
			Status:        data.Status,
			Link: map[string]string{
				beatmapType: link,
			},
			Path:          map[string]string{beatmapType: path},
			BeatmapsetId:  beatmapsetId,
			HasStoryboard: data.HasStoryBoard == 1,
			HasVideo:      data.HasVideo == 1,
		}
		lastUpdate, err := time.Parse(time.DateTime, data.LastUpdate)
		if err != nil {
			return "", nil, err
		}
		metadata.LastUpdate = lastUpdate.Unix()
		result = append(result, metadata)
	}
	return beatmapType, result, nil
}

func GenerateExistedFileMetadata(files []DriveItem, client *osu.LegacyOfficialClient, graph *onedrive.GraphClient, metadata *Metadata, maxConcurrency int, ctx context.Context) (retryFiles []DriveItem) {
	sem := make(chan struct{}, maxConcurrency)
	mux := sync.RWMutex{}
	var wg sync.WaitGroup
	for _, file := range files {
		select {
		case <-ctx.Done():
			log.Println("Context canceled, stopping task creation.")
			return retryFiles
		default:
		}

		sem <- struct{}{}
		wg.Add(1)
		go func(file DriveItem, wg *sync.WaitGroup) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					retryFiles = append(retryFiles, file)
					if strings.Contains(fmt.Sprint(r), "context canceled") {
						return
					}
					log.Printf("[%s] Failed: %s", file.Name, r)
				}
			}()

			defer func() { <-sem }()
			typ, singleBeatmapsetMetadata, err := generateSingleFileMetadata(file, client, graph)
			if err != nil {
				panic(err)
			}
			mux.Lock()
			beatmaps := make(map[int]BeatmapMetadata)
			lastUpdate := int64(0)
			for _, beatmapMetadata := range singleBeatmapsetMetadata {
				metadata.Beatmaps[beatmapMetadata.BeatmapId] = beatmapMetadata
				beatmaps[beatmapMetadata.BeatmapId] = beatmapMetadata

				gameModeUpdateTime, ok := metadata.GameMode[beatmapMetadata.GameMode]
				if !ok {
					metadata.GameMode[beatmapMetadata.GameMode] = MetadataGameMode{
						UpdateTime: beatmapMetadata.LastUpdate,
					}
				} else {
					if gameModeUpdateTime.UpdateTime < beatmapMetadata.LastUpdate {
						gameModeUpdateTime.UpdateTime = beatmapMetadata.LastUpdate
						metadata.GameMode[beatmapMetadata.GameMode] = gameModeUpdateTime
					}
				}
			}
			for _, metadata := range singleBeatmapsetMetadata {
				if metadata.LastUpdate > lastUpdate {
					lastUpdate = metadata.LastUpdate
				}
			}
			var beatmapsetData BeatmapsetMetadata
			sourceBeatmapset, ok := metadata.Beatmapsets[singleBeatmapsetMetadata[0].BeatmapsetId]
			if ok {
				for k, v := range sourceBeatmapset.Beatmaps {
					nowBeatmap, ok := beatmaps[k]
					if !ok {
						beatmaps[k] = v
					}
					nowBeatmap.Link[typ] = v.Link[typ]
					nowBeatmap.Path[typ] = v.Path[typ]
					beatmaps[k] = nowBeatmap
				}
				sourceBeatmapset.Beatmaps = beatmaps
				sourceBeatmapset.LastUpdate = lastUpdate
				sourceBeatmapset.Link[typ] = singleBeatmapsetMetadata[0].Link[typ]
				sourceBeatmapset.Path[typ] = singleBeatmapsetMetadata[0].Path[typ]
				beatmapsetData = sourceBeatmapset
			} else {
				beatmapsetData = BeatmapsetMetadata{
					Beatmaps:      beatmaps,
					BeatmapsetId:  singleBeatmapsetMetadata[0].BeatmapsetId,
					LastUpdate:    lastUpdate,
					Link:          singleBeatmapsetMetadata[0].Link,
					Path:          singleBeatmapsetMetadata[0].Path,
					HasStoryboard: singleBeatmapsetMetadata[0].HasStoryboard,
					HasVideo:      singleBeatmapsetMetadata[0].HasVideo,
				}
			}
			metadata.Beatmapsets[singleBeatmapsetMetadata[0].BeatmapsetId] = beatmapsetData
			log.Printf("Generated: %s\n", file.Name)
			mux.Unlock()
		}(file, &wg)
	}

	wg.Wait()
	return
}
