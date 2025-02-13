package sync

import (
	"context"
	"fmt"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/MingxuanGame/OsuBeatmapSync/onedrive"
	"github.com/MingxuanGame/OsuBeatmapSync/osu/download"
	"github.com/MingxuanGame/OsuBeatmapSync/utils"
	"github.com/MingxuanGame/OsuBeatmapSync/utils/beatmap_processing"
	"github.com/rs/zerolog/log"
	"path"
	"strings"
	"sync"
	"time"
)

var logger = log.With().Str("module", "osu.sync").Logger()

type Syncer struct {
	ctx       context.Context
	graph     *onedrive.GraphClient
	config    *Config
	modeMap   map[GameMode]string
	statusMap map[BeatmapStatus]string

	Metadata *Metadata
	Failed   []BeatmapsetMetadata

	downloadSem chan struct{}
	uploadSem   chan struct{}
	mux         sync.RWMutex
	created     map[int]struct{}
	result      map[int]BeatmapsetMetadata
}

func NewSyncer(ctx context.Context, metadata *Metadata, graph *onedrive.GraphClient, config *Config) *Syncer {
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
	return &Syncer{
		ctx:         ctx,
		Metadata:    metadata,
		graph:       graph,
		config:      config,
		modeMap:     modeMap,
		statusMap:   statusMap,
		downloadSem: make(chan struct{}, config.General.MaxConcurrent),
		uploadSem:   make(chan struct{}, config.General.MaxConcurrent),
		created:     make(map[int]struct{}),
		result:      make(map[int]BeatmapsetMetadata),
	}
}

func (s *Syncer) makePath(gameMode GameMode, beatmapStatus BeatmapStatus, typ string) string {
	return path.Join(s.config.Path.Root, s.modeMap[gameMode], s.statusMap[beatmapStatus], typ)
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

	var processors []beatmap_processing.Processor
	if (beatmapset.HasVideo || beatmapset.HasStoryboard) && utils.In(s.config.Osu.ProcessTypes, "mini") {
		processors = append(processors, beatmap_processing.NewMiniProcessor())
	}
	if beatmapset.HasVideo && utils.In(s.config.Osu.ProcessTypes, "no_video") {
		processors = append(processors, beatmap_processing.NewNoVideoProcessor())

	}
	if beatmapset.HasStoryboard && utils.In(s.config.Osu.ProcessTypes, "no_storyboard") {
		processors = append(processors, beatmap_processing.NewNoStoryboardProcessor())
	}
	if utils.In(s.config.Osu.ProcessTypes, "no_hit_sound") {
		processors = append(processors, beatmap_processing.NewNoHitSoundProcessor())
	}
	if utils.In(s.config.Osu.ProcessTypes, "no_bg") {
		processors = append(processors, beatmap_processing.NewNoBackgroundProcessor())
	}

	linkMap := make(map[string]string)
	pathMap := make(map[string]string)
	link, cloudPath, err := s.uploadBeatmap(beatmapset, "full", data)
	if err != nil {
		panic(err)
	}
	linkMap["full"] = link
	pathMap["full"] = cloudPath
	for _, p := range processors {
		logger.Debug().Msgf("Processing %s with mode %s", beatmapset.String(), p.String())
		processed, err := beatmap_processing.Process(p, data)
		if err != nil {
			panic(err)
		}
		typ := p.String()
		link, cloudPath, err = s.uploadBeatmap(beatmapset, typ, processed)
		if err != nil {
			panic(err)
		}
		linkMap[typ] = link
		pathMap[typ] = cloudPath
	}

	for bid, v := range beatmapset.Beatmaps {
		v.Link = linkMap
		v.Path = pathMap
		beatmapset.Beatmaps[bid] = v
	}
	beatmapset.Link = linkMap
	beatmapset.Path = pathMap
	s.mux.Lock()
	s.result[beatmapset.BeatmapsetId] = beatmapset
	s.mux.Unlock()
	logger.Info().Int("sid", beatmapset.BeatmapsetId).Msgf("Upload %s successfully", beatmapset.String())
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
	if beatmapset.CannotDownload || beatmapset.NoAudio {
		logger.Warn().Int("sid", beatmapset.BeatmapsetId).Msgf("Beatmapset %s is missing download or audio, skip", beatmapset.String())
		s.mux.Lock()
		delete(s.created, beatmapset.BeatmapsetId)
		s.mux.Unlock()
		return
	}
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
		if len(s.created) < s.config.General.MaxConcurrent*s.config.General.UploadMultiple {
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
		for bid, beatmap := range v.Beatmaps {
			s.Metadata.Beatmaps[bid] = beatmap
			if v.LastUpdate < beatmap.LastUpdate {
				v.LastUpdate = beatmap.LastUpdate
			}
			gamemode, ok := s.Metadata.GameMode[beatmap.GameMode]
			if !ok || gamemode.UpdateTime < beatmap.LastUpdate {
				s.Metadata.GameMode[beatmap.GameMode] = MetadataGameMode{
					UpdateTime: beatmap.LastUpdate,
				}
			}
		}
		s.Metadata.Beatmapsets[k] = v
	}
}

func (s *Syncer) ReSync(downloaders []download.BeatmapDownloader) {
	failed := s.Failed
	s.Failed = nil
	s.SyncNewBeatmap(downloaders, failed)
}
