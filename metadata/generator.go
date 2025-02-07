package metadata

import (
	"context"
	"fmt"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	. "github.com/MingxuanGame/OsuBeatmapSync/model/onedrive"
	"github.com/MingxuanGame/OsuBeatmapSync/onedrive"
	"github.com/MingxuanGame/OsuBeatmapSync/osu"
	"github.com/MingxuanGame/OsuBeatmapSync/utils"
	"github.com/rs/zerolog/log"
	"strings"
	"sync"
	"time"
)

type Generator struct {
	client *osu.LegacyOfficialClient
	ctx    context.Context
	graph  *onedrive.GraphClient

	Failed   []DriveItem
	Metadata *Metadata

	sem chan struct{}
	mux sync.RWMutex
}

var logger = log.With().Str("module", "metadata").Logger()

func NewGenerator(client *osu.LegacyOfficialClient, graph *onedrive.GraphClient, ctx context.Context, maxConcurrency int, metadata *Metadata) *Generator {
	return &Generator{
		client:   client,
		ctx:      ctx,
		graph:    graph,
		sem:      make(chan struct{}, maxConcurrency),
		Metadata: metadata,
	}
}

func (g *Generator) generateSingle(item DriveItem) (typ string, beatmaps []BeatmapMetadata, err error) {
	filename := item.Name
	var result []BeatmapMetadata
	_, _, beatmapsetId := utils.ParseFilename(filename)
	apiData, err := g.client.GetBeatmapBySetId(beatmapsetId)
	if err != nil {
		return "", nil, err
	}
	path := item.ParentReference.Path + "/" + item.Name
	fileStruct, err := ParseFilenameStruct(path)
	if err != nil {
		return "", nil, err
	}
	beatmapType := fileStruct.Type
	link, err := g.graph.MakeShareLink(item.Id)
	if err != nil {
		return "", nil, err
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

func (g *Generator) GenerateExistedFileMetadata(files []DriveItem) {
	g.Failed = make([]DriveItem, 0)
	var wg sync.WaitGroup
	for _, file := range files {
		select {
		case <-g.ctx.Done():
			logger.Info().Msg("Context canceled, stopping task creation.")
			return
		default:
		}

		g.sem <- struct{}{}
		wg.Add(1)

		go func(file DriveItem, wg *sync.WaitGroup) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					g.Failed = append(g.Failed, file)
					if strings.Contains(fmt.Sprint(r), "context canceled") {
						return
					}
					logger.Warn().Err(r.(error)).Msgf("Failed to make metadata: %s", file.Name)
				}
			}()
			defer func() { <-g.sem }()

			typ, beatmapMetadata, err := g.generateSingle(file)
			if err != nil {
				panic(err)
			}

			g.mux.Lock()
			beatmaps := make(map[int]BeatmapMetadata)
			lastUpdate := int64(0)
			beatmapsetId := beatmapMetadata[0].BeatmapsetId
			beatmapset, ok := g.Metadata.Beatmapsets[beatmapsetId]
			if !ok {
				beatmapset = BeatmapsetMetadata{
					Beatmaps:      make(map[int]BeatmapMetadata),
					BeatmapsetId:  beatmapsetId,
					Link:          beatmapMetadata[0].Link,
					Path:          beatmapMetadata[0].Path,
					HasStoryboard: beatmapMetadata[0].HasStoryboard,
					HasVideo:      beatmapMetadata[0].HasVideo,
				}
			}

			for _, b := range beatmapMetadata {
				// in beatmaps
				origin, ok := g.Metadata.Beatmaps[b.BeatmapId]
				if !ok {
					origin = b
				}
				origin.Link[typ] = b.Link[typ]
				origin.Path[typ] = b.Path[typ]
				beatmaps[b.BeatmapId] = origin

				// in game mode
				gameModeUpdateTime, ok := g.Metadata.GameMode[b.GameMode]
				if !ok {
					g.Metadata.GameMode[b.GameMode] = MetadataGameMode{
						UpdateTime: b.LastUpdate,
					}
				} else {
					if gameModeUpdateTime.UpdateTime < b.LastUpdate {
						gameModeUpdateTime.UpdateTime = b.LastUpdate
						g.Metadata.GameMode[b.GameMode] = gameModeUpdateTime
					}
				}
				if b.LastUpdate > lastUpdate {
					lastUpdate = b.LastUpdate
				}

				// in beatmapset
				origin, ok = beatmapset.Beatmaps[b.BeatmapId]
				if !ok {
					origin = b
				}
				origin.Link[typ] = b.Link[typ]
				origin.Path[typ] = b.Path[typ]
				if b.LastUpdate > beatmapset.LastUpdate {
					beatmapset.LastUpdate = b.LastUpdate
				}
				if b.HasStoryboard {
					beatmapset.HasStoryboard = true
				}
				if b.HasVideo {
					beatmapset.HasVideo = true
				}
				beatmapset.Beatmaps[b.BeatmapId] = origin
			}
			g.Metadata.Beatmapsets[beatmapsetId] = beatmapset
			logger.Info().Msgf("Generated: %s", file.Name)
			g.mux.Unlock()
		}(file, &wg)
	}

	wg.Wait()
	return
}
