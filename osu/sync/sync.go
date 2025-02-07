package sync

import (
	"context"
	"fmt"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/MingxuanGame/OsuBeatmapSync/onedrive"
	"github.com/MingxuanGame/OsuBeatmapSync/osu/download"
	"github.com/MingxuanGame/OsuBeatmapSync/utils"
	"github.com/rs/zerolog/log"
	"path"
	"strings"
	"sync"
	"time"
)

var logger = log.With().Str("module", "osu.sync").Logger()

type Syncer struct {
	ctx            context.Context
	graph          *onedrive.GraphClient
	root           string
	uploadMultiple int
	maxConcurrency int
	modeMap        map[GameMode]string
	statusMap      map[BeatmapStatus]string

	Metadata *Metadata
	Failed   []BeatmapsetMetadata

	downloadSem chan struct{}
	uploadSem   chan struct{}
	mux         sync.RWMutex
	created     map[int]struct{}
	result      map[int]BeatmapsetMetadata
}

func NewSyncer(ctx context.Context, metadata *Metadata, graph *onedrive.GraphClient, root string, maxConcurrency, uploadMultiple int, modeMap map[GameMode]string, statusMap map[BeatmapStatus]string) *Syncer {
	return &Syncer{
		ctx:            ctx,
		Metadata:       metadata,
		graph:          graph,
		root:           root,
		maxConcurrency: maxConcurrency,
		uploadMultiple: uploadMultiple,
		modeMap:        modeMap,
		statusMap:      statusMap,
		downloadSem:    make(chan struct{}, maxConcurrency),
		uploadSem:      make(chan struct{}, maxConcurrency),
		created:        make(map[int]struct{}),
		result:         make(map[int]BeatmapsetMetadata),
	}
}

func (s *Syncer) makePath(gameMode GameMode, beatmapStatus BeatmapStatus, typ string) string {
	return path.Join(s.root, s.modeMap[gameMode], s.statusMap[beatmapStatus], typ)
}

func downloadBeatmap(downloader download.BeatmapDownloader, beatmapset *BeatmapsetMetadata) ([]byte, error) {
	var beatmap BeatmapMetadata
	for _, v := range beatmapset.Beatmaps {
		beatmap = v
		break
	}

	var data []byte
	var err error
	logger.Info().Str("downloader", downloader.Name()).Int("sid", beatmap.BeatmapsetId).Msgf("Downloading beatmapset %s", beatmapset.String())
	data, err = downloader.DownloadBeatmapset(beatmap.BeatmapsetId)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *Syncer) uploadBeatmap(beatmapset BeatmapsetMetadata, typ string, data []byte) (link, cloudPath string, err error) {
	var beatmap BeatmapMetadata
	for _, v := range beatmapset.Beatmaps {
		beatmap = v
		break
	}

	uploadPath := s.makePath(beatmap.GameMode, beatmap.Status, typ)
	filename := utils.MakeFilename(beatmapset.BeatmapsetId, beatmap.Artist, beatmap.Title)

	item, err := s.graph.GetItem(uploadPath, filename)
	if err != nil {
		logger.Warn().Err(err).Int("sid", beatmapset.BeatmapsetId).Str("type", typ).Msgf("Failed to get item %s/%s", uploadPath, filename)
	}
	if item != nil {
		logger.Info().Int("sid", beatmapset.BeatmapsetId).Str("type", typ).Msgf("File %s/%s already exists", uploadPath, filename)
		if item.VerifyQuickXorHash(data) {
			logger.Info().Int("sid", beatmapset.BeatmapsetId).Str("type", typ).Msgf("File %s/%s is the same, skip", uploadPath, filename)
			goto skipUpload
		}
	}

	err = s.graph.UploadLargeFile(uploadPath, filename, data)
	if err != nil {
		return
	}

skipUpload:
	if item == nil {
		item, err = s.graph.GetItem(uploadPath, filename)
		if err != nil {
			return
		}
	}
	if item == nil {
		return "", "", fmt.Errorf("item %s/%s is not found", uploadPath, filename)
	}
	link, err = s.graph.MakeShareLink(item.Id)
	if err != nil {
		return
	}
	cloudPath = path.Join(uploadPath, filename)
	return

}

func (s *Syncer) uploadTask(wg *sync.WaitGroup, beatmapset BeatmapsetMetadata, data []byte) {
	var noVideoData, miniData []byte
	var err error

	defer wg.Done()
	defer func() {
		s.mux.Lock()
		delete(s.created, beatmapset.BeatmapsetId)
		s.mux.Unlock()
	}()
	s.uploadSem <- struct{}{}
	defer func() { <-s.uploadSem }()
	defer func() {
		if r := recover(); r != nil {
			s.mux.Lock()
			s.Failed = append(s.Failed, beatmapset)
			s.mux.Unlock()
			if strings.Contains(fmt.Sprint(r), "context canceled") {
				return
			}
			logger.Warn().Err(r.(error)).Int("sid", beatmapset.BeatmapsetId).Msgf("Failed upload %s", beatmapset.String())
		}
	}()

	if beatmapset.HasVideo || beatmapset.HasStoryboard {
		noVideoData, miniData, err = utils.ProcessBeatmapset(data)
		if err != nil {
			panic(fmt.Errorf("failed to process beatmapset: %w", err))
		}
	}
	linkMap := make(map[string]string)
	pathMap := make(map[string]string)
	var link, cloudPath string
	link, cloudPath, err = s.uploadBeatmap(beatmapset, "full", data)
	if err != nil {
		panic(err)
	}
	linkMap["full"] = link
	pathMap["full"] = cloudPath
	if beatmapset.HasVideo {
		link, cloudPath, err = s.uploadBeatmap(beatmapset, "no_video", noVideoData)
		if err != nil {
			panic(err)
		}
		linkMap["no_video"] = link
		pathMap["no_video"] = cloudPath
	}
	if beatmapset.HasStoryboard {
		link, cloudPath, err = s.uploadBeatmap(beatmapset, "mini", miniData)
		if err != nil {
			panic(err)
		}
		linkMap["mini"] = link
		pathMap["mini"] = cloudPath
	}

	for _, v := range beatmapset.Beatmaps {
		v.Link = linkMap
		v.Path = pathMap
	}
	beatmapset.Link = linkMap
	beatmapset.Path = pathMap
	s.mux.Lock()
	s.result[beatmapset.BeatmapsetId] = beatmapset
	s.mux.Unlock()
}

func (s *Syncer) syncSingleBeatmapset(wg *sync.WaitGroup, downloader download.BeatmapDownloader, beatmapset BeatmapsetMetadata) {
	defer func() {
		if r := recover(); r != nil {
			s.Failed = append(s.Failed, beatmapset)
			s.mux.Lock()
			delete(s.created, beatmapset.BeatmapsetId)
			s.mux.Unlock()
			if strings.Contains(fmt.Sprint(r), "context canceled") {
				return
			}
			logger.Warn().Err(r.(error)).Int("sid", beatmapset.BeatmapsetId).Msgf("Failed download %s", beatmapset.String())
		}
	}()
	defer wg.Done()
	defer func() { <-s.downloadSem }()
	s.mux.Lock()
	s.created[beatmapset.BeatmapsetId] = struct{}{}
	s.mux.Unlock()
	data, err := downloadBeatmap(downloader, &beatmapset)
	if err != nil {
		panic(err)
	}

	// Wait
	for {
		select {
		case <-s.ctx.Done():
			return
		default:

		}
		if len(s.created) < s.maxConcurrency*s.uploadMultiple {
			break
		}
		time.Sleep(time.Second)
	}

	wg.Add(1)
	go s.uploadTask(wg, beatmapset, data)
}

func (s *Syncer) SyncNewBeatmap(downloaders []download.BeatmapDownloader, needSyncBeatmapset []BeatmapsetMetadata) {
	var wg sync.WaitGroup
	for i, beatmapset := range needSyncBeatmapset {
		select {
		case <-s.ctx.Done():
			goto jumpLoop
		default:
		}
		wg.Add(1)
		s.downloadSem <- struct{}{}
		downloader := downloaders[i%len(downloaders)]
		go s.syncSingleBeatmapset(&wg, downloader, beatmapset)
	}

jumpLoop:
	logger.Info().Msg("Waiting for all tasks to finish, it may need some time...")
	wg.Wait()
	clear(s.created)
	for k, v := range s.result {
		s.Metadata.Beatmapsets[k] = v
		for _, beatmap := range v.Beatmaps {
			s.Metadata.GameMode[beatmap.GameMode] = MetadataGameMode{
				UpdateTime: time.Now().Unix(),
			}
		}
	}
}

func (s *Syncer) ReSync(downloaders []download.BeatmapDownloader) {
	failed := s.Failed
	s.Failed = nil
	s.SyncNewBeatmap(downloaders, failed)
}
