package download

import (
	"context"
	"fmt"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/MingxuanGame/OsuBeatmapSync/onedrive"
	"github.com/rs/zerolog/log"
	"strings"
	"sync"
	"time"
)

func sanitizeFileName(fileName string) string {
	invalidChars := []struct {
		old, new string
	}{
		{"<", "_"}, {">", "_"}, {":", "_"}, {"\"", "_"}, {"/", "_"},
		{"\\", "_"}, {"|", "_"}, {"?", "_"}, {"*", "_"},
	}

	for _, char := range invalidChars {
		fileName = strings.ReplaceAll(fileName, char.old, char.new)
	}
	return fileName
}

func makePath(root string, gameMode GameMode, modeMap map[GameMode]string, beatmapStatus BeatmapStatus, statusMap map[BeatmapStatus]string, typ string) string {
	return root + "/" + modeMap[gameMode] + "/" + statusMap[beatmapStatus] + "/" + typ
}

func MakeFilename(beatmapsetId int, artist, name string) string {
	return sanitizeFileName(fmt.Sprintf("%d %s - %s.osz", beatmapsetId, artist, name))
}

func downloadBeatmap(downloader BeatmapDownloader, beatmap *BeatmapsetMetadata) ([]byte, error) {
	var data []byte
	var err error
	log.Info().Str("downloader", downloader.Name()).Int("sid", beatmap.BeatmapsetId).Msg("Downloading beatmapset...")
	data, err = downloader.DownloadBeatmapset(beatmap.BeatmapsetId)
	if err != nil {
		return nil, fmt.Errorf("[%s] failed to download beatmap set: %w", downloader.Name(), err)
	}
	return data, nil
}

func uploadBeatmap(graph *onedrive.GraphClient, path, filename string, data []byte) error {
	err := graph.UploadLargeFile(path, filename, data)
	if err != nil {
		return fmt.Errorf("[onedrive][%s] failed to upload file: %w", filename, err)
	}
	return nil
}

func upload(root, typ string, graph *onedrive.GraphClient, beatmapset BeatmapsetMetadata, beatmap BeatmapMetadata, data []byte, modeMap map[GameMode]string, statusMap map[BeatmapStatus]string) (string, string) {
	path := makePath(root, beatmap.GameMode, modeMap, beatmap.Status, statusMap, typ)
	filename := MakeFilename(beatmapset.BeatmapsetId, beatmap.Artist, beatmap.Title)
	item, err := graph.GetItem(path, filename)
	if err != nil {
		log.Warn().Err(err).Int("sid", beatmapset.BeatmapsetId).Str("type", typ).Msg("Failed to get item")
	}
	if item != nil {
		log.Info().Int("sid", beatmapset.BeatmapsetId).Str("type", typ).Msg("File already exists")
		if item.VerifyQuickXorHash(data) {
			log.Info().Int("sid", beatmapset.BeatmapsetId).Str("type", typ).Msg("File is the same")
			goto skipUpload
		}
	}
	log.Info().Int("sid", beatmapset.BeatmapsetId).Str("type", typ).Msg("Uploading...")
	err = uploadBeatmap(graph, path, filename, data)
	if err != nil {
		panic("[onedrive] failed to upload file: " + err.Error())
	}

skipUpload:
	if item == nil {
		item, err = graph.GetItem(path, filename)
		if err != nil {
			panic(fmt.Errorf("[onedrive] failed to get item: %w", err))
		}
	}
	link, err := graph.MakeShareLink(item.Id)
	if err != nil {
		panic(fmt.Errorf("[onedrive] failed to make share link: %w", err))
	}
	return link, path + "/" + filename

}

func uploadGoroutinefunc(wg *sync.WaitGroup, root string, graph *onedrive.GraphClient, beatmapset BeatmapsetMetadata, beatmap BeatmapMetadata, data []byte, modeMap map[GameMode]string, statusMap map[BeatmapStatus]string, result map[int]BeatmapMetadata, retryBeatmaps []BeatmapsetMetadata, mux *sync.RWMutex, uploadSem chan struct{}, created *map[int]struct{}) {
	defer wg.Done()
	defer func() {
		mux.Lock()
		delete(*created, beatmapset.BeatmapsetId)
		mux.Unlock()
	}()
	uploadSem <- struct{}{}
	defer func() {
		if r := recover(); r != nil {
			log.Warn().Int("sid", beatmapset.BeatmapsetId).Msgf("Failed upload: %v", r)
			retryBeatmaps = append(retryBeatmaps, beatmapset)
		}
	}()
	defer func() { <-uploadSem }()

	var noVideoData, miniData []byte
	var err error
	if beatmapset.HasVideo || beatmapset.HasStoryboard {
		noVideoData, miniData, err = ProcessBeatmapset(data)
		if err != nil {
			panic(fmt.Errorf("failed to process beatmapset: %w", err))
		}
	}
	linkMap := make(map[string]string)
	pathMap := make(map[string]string)
	var link, path string
	link, path = upload(root, "full", graph, beatmapset, beatmap, data, modeMap, statusMap)
	linkMap["full"] = link
	pathMap["full"] = path

	if beatmapset.HasVideo {
		link, path = upload(root, "no_video", graph, beatmapset, beatmap, noVideoData, modeMap, statusMap)
		linkMap["no_video"] = link
		pathMap["no_video"] = path
	}
	if beatmapset.HasStoryboard {
		link, path = upload(root, "mini", graph, beatmapset, beatmap, miniData, modeMap, statusMap)
		linkMap["mini"] = link
		pathMap["mini"] = path
	}

	for _, v := range beatmapset.Beatmaps {
		v.Link = linkMap
		v.Path = pathMap
		mux.Lock()
		result[v.BeatmapId] = v
		delete(*created, beatmapset.BeatmapsetId)
		mux.Unlock()
	}
}

func SyncNewBeatmap(metadata *Metadata, graph *onedrive.GraphClient, root string, downloaders []BeatmapDownloader, needSyncBeatmaps []BeatmapsetMetadata, maxConcurrency int, modeMap map[GameMode]string, statusMap map[BeatmapStatus]string, ctx context.Context) []BeatmapsetMetadata {
	created := make(map[int]struct{})
	downloadSem := make(chan struct{}, maxConcurrency)
	mux := sync.RWMutex{}
	uploadSem := make(chan struct{}, maxConcurrency)
	//uploadChannel := make(chan model.BeatmapMetadata, maxConcurrency)
	var wg sync.WaitGroup
	result := make(map[int]BeatmapMetadata)
	var retryBeatmaps []BeatmapsetMetadata

	for i, beatmap := range needSyncBeatmaps {
		select {
		case <-ctx.Done():
			log.Info().Msg("Context canceled, stopping task creation.")
			goto jumpLoop
		default:
		}

		downloader := downloaders[i%len(downloaders)]
		wg.Add(1)
		downloadSem <- struct{}{}
		mux.Lock()
		created[beatmap.BeatmapsetId] = struct{}{}
		mux.Unlock()
		go func(beatmapset BeatmapsetMetadata, wg *sync.WaitGroup) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					retryBeatmaps = append(retryBeatmaps, beatmapset)
					mux.Lock()
					delete(created, beatmapset.BeatmapsetId)
					mux.Unlock()
					if strings.Contains(fmt.Sprint(r), "context canceled") {
						return
					}
					log.Warn().Int("sid", beatmapset.BeatmapsetId).Msgf("Failed download: %v", r)
				}
			}()
			defer func() { <-downloadSem }()

			data, err := downloadBeatmap(downloader, &beatmapset)
			if err != nil {
				panic(fmt.Errorf("failed to download beatmapset: %w", err))
			}

			// get 1st beatmap
			var beatmap BeatmapMetadata
			for _, v := range beatmapset.Beatmaps {
				beatmap = v
				break
			}
			for {
				select {
				case <-ctx.Done():
					return
				default:

				}
				if len(created) < (maxConcurrency+1)*2 {
					break
				}
				time.Sleep(time.Second)
			}
			wg.Add(1)
			go uploadGoroutinefunc(wg, root, graph, beatmapset, beatmap, data, modeMap, statusMap, result, retryBeatmaps, &mux, uploadSem, &created)

		}(beatmap, &wg)
	}

jumpLoop:
	wg.Wait()

	for k, v := range result {
		metadata.Beatmaps[k] = v
		metadata.GameMode[v.GameMode] = MetadataGameMode{
			UpdateTime: time.Now().Unix(),
		}
		beatmapset, ok := metadata.Beatmapsets[v.BeatmapsetId]
		if !ok {
			beatmapset = BeatmapsetMetadata{
				BeatmapsetId:  v.BeatmapsetId,
				HasVideo:      v.HasVideo,
				HasStoryboard: v.HasStoryboard,
				Beatmaps:      make(map[int]BeatmapMetadata),
				Link:          v.Link,
				Path:          v.Path,
			}
		}
		beatmapset.Beatmaps[v.BeatmapId] = v
		metadata.Beatmapsets[v.BeatmapsetId] = beatmapset
	}

	return retryBeatmaps

}
