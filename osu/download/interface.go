package download

type BeatmapDownloader interface {
	Name() string
	DownloadBeatmapset(beatmapId int) ([]byte, error)
	DownloadBeatmapsetNoVideo(beatmapId int) ([]byte, error)
	DownloadBeatmapsetMini(beatmapId int) ([]byte, error) //  no video, no storyboard
}
