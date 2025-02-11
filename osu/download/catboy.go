package download

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	netUrl "net/url"
)

type CatboyDownloader struct {
	ctx context.Context
}

var catboyLogger = log.With().Str("module", "osu.download.catboy").Logger()

func NewCatboyDownloader(ctx context.Context) *CatboyDownloader {
	return &CatboyDownloader{ctx: ctx}
}

const catboyApi = "https://catboy.best"

func (d *CatboyDownloader) DownloadBeatmapset(beatmapsetId int) ([]byte, error) {
	req, err := http.NewRequestWithContext(d.ctx, "GET", fmt.Sprintf("%s/d/%d", catboyApi, beatmapsetId), nil)
	if err != nil {
		return nil, err
	}
	catboyLogger.Trace().Msgf("Requesting %s %s", req.Method, req.URL.String())

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
			return nil, &netUrl.Error{
				Op:  "",
				URL: req.URL.String(),
				Err: fmt.Errorf("status: %s", resp.Status),
			}
		}
		return nil, &netUrl.Error{
			Op:  "",
			URL: req.URL.String(),
			Err: fmt.Errorf("status: %s, error: %s", resp.Status, responseBody.Error),
		}
	}
	return io.ReadAll(resp.Body)
}

func (d *CatboyDownloader) Name() string {
	return "catboy"
}
