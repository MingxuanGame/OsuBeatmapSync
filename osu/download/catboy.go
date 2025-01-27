package download

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type CatboyDownloader struct {
	ctx context.Context
}

func NewCatboyDownloader(ctx context.Context) *CatboyDownloader {
	return &CatboyDownloader{ctx: ctx}
}

const catboyApi = "https://catboy.best"

func (d *CatboyDownloader) download(beatmapsetId int, noVideo bool) ([]byte, error) {
	var url string
	if noVideo {
		url = fmt.Sprintf("%s/d/%dn", catboyApi, beatmapsetId)
	} else {
		url = fmt.Sprintf("%s/d/%d", catboyApi, beatmapsetId)
	}
	req, err := http.NewRequestWithContext(d.ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		var responseBody struct {
			Error string `json:"error"`
		}
		err := json.NewDecoder(resp.Body).Decode(&responseBody)
		if err != nil {
			return nil, fmt.Errorf("[osu! catboy] status code: %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("[osu! catboy] status code: %d, error: %s", resp.StatusCode, responseBody.Error)
	}
	return io.ReadAll(resp.Body)
}

func (d *CatboyDownloader) DownloadBeatmapset(beatmapsetId int) ([]byte, error) {
	return d.download(beatmapsetId, false)
}

func (d *CatboyDownloader) DownloadBeatmapsetNoVideo(beatmapsetId int) ([]byte, error) {
	return d.download(beatmapsetId, true)
}

func (d *CatboyDownloader) DownloadBeatmapsetMini(beatmapsetId int) ([]byte, error) {
	return d.download(beatmapsetId, true)
}

func (d *CatboyDownloader) Name() string {
	return "catboy"
}
