package download

import (
	"context"
	"fmt"
	"github.com/MingxuanGame/OsuBeatmapSync/utils"
	"io"
	"log"
	"net/http"
	"time"
)

const nerinyanApi = "https://api.nerinyan.moe"

type NerinyanDownloader struct {
	*http.Client
	ctx context.Context
}

func NewNerinyanDownloader(ctx context.Context) *NerinyanDownloader {
	return &NerinyanDownloader{
		Client: &http.Client{},
		ctx:    ctx,
	}
}

func (d *NerinyanDownloader) do(req *http.Request) ([]byte, error) {
	resp, err := d.Client.Do(req)
	var data []byte
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 429 {
		retryAfter, ctx, err := utils.GetLimitSecond(resp.Header.Get("X-Retry-After"), d.ctx)
		if err != nil {
			return nil, err
		}
		d.ctx = ctx
		log.Printf("[osu! nerinyan] Rate limited, sleeping for %s.\n", retryAfter)
		time.Sleep(retryAfter)
		data, err = d.do(req)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("[osu! nerinyan] status code: %d", resp.StatusCode)
	}
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (d *NerinyanDownloader) download(beatmapsetId int, noStoryBoard bool, noVideo bool) ([]byte, error) {
	req, err := http.NewRequestWithContext(d.ctx, "GET", fmt.Sprintf("%s/d/%d?NoStoryboard=%t&noVideo=%t", nerinyanApi, beatmapsetId, noStoryBoard, noVideo), nil)
	if err != nil {
		return nil, err
	}
	return d.do(req)
}

func (d *NerinyanDownloader) DownloadBeatmapset(setId int) ([]byte, error) {
	return d.download(setId, false, false)
}

func (d *NerinyanDownloader) DownloadBeatmapsetNoVideo(setId int) ([]byte, error) {
	return d.download(setId, false, true)
}

func (d *NerinyanDownloader) DownloadBeatmapsetMini(setId int) ([]byte, error) {
	return d.download(setId, true, true)
}

func (d *NerinyanDownloader) Name() string {
	return "nerinyan"
}
