package download

type BeatmapDownloader interface {
	Name() string
	DownloadBeatmapset(beatmapId int) ([]byte, error)
}
