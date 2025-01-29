package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

const sayobotApi = `https://txy1.sayobot.cn/beatmaps/download`

type SayobotDownloader struct {
	client *http.Client
	server string
	ctx    context.Context
}

func NewSayobotDownloader(server string, ctx context.Context) *SayobotDownloader {
	return &SayobotDownloader{
		client: &http.Client{
			//CheckRedirect: func(req *http.Request, via []*http.Request) error {
			//	return http.ErrUseLastResponse
			//},
		},
		server: server, ctx: ctx,
	}
}

func (d *SayobotDownloader) download(beatmapsetId int, typ string) ([]byte, error) {
	req, err := http.NewRequestWithContext(d.ctx, "GET", fmt.Sprintf("%s/%s/%d?server=%s", sayobotApi, typ, beatmapsetId, d.server), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/132.0.0.0`)
	req.Header.Set("Referer", "https://osu.sayobot.cn/")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	//location, err := url.Parse(resp.Header.Get("Location"))
	//filename, err := url.QueryUnescape(location.Query().Get("filename"))
	//if err != nil {
	//	return nil, err
	//}
	//newUrl := fmt.Sprintf("%s://%s%s?filename=%s", location.Scheme, location.Host, location.Path, url.QueryEscape(filename))
	//req, err = http.NewRequest("GET", newUrl, nil)
	//if err != nil {
	//	return nil, err
	//}
	//resp, err = d.client.Do(req)
	//if err != nil {
	//	return nil, err
	//}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("[osu! sayobot] status code: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (d *SayobotDownloader) DownloadBeatmapset(beatmapsetId int) ([]byte, error) {
	return d.download(beatmapsetId, "full")
}

func (d *SayobotDownloader) DownloadBeatmapsetNoVideo(beatmapsetId int) ([]byte, error) {
	return d.download(beatmapsetId, "novideo")
}

func (d *SayobotDownloader) DownloadBeatmapsetMini(beatmapsetId int) ([]byte, error) {
	return d.download(beatmapsetId, "mini")
}

func (d *SayobotDownloader) Name() string {
	return "Sayobot"
}
