package download

import (
	"context"
	"fmt"
	"github.com/MingxuanGame/OsuBeatmapSync/utils"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
	"time"
)

const nerinyanApi = "https://api.nerinyan.moe"

var nerinyanLogger = log.With().Str("module", "osu.download.nerinyan").Logger()

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
	nerinyanLogger.Trace().Msgf("Requesting %s %s", req.Method, req.URL.String())
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
		nerinyanLogger.Warn().Msgf("Rate limited, sleeping for %s.", retryAfter)
		time.Sleep(retryAfter)
		data, err = d.do(req)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	if resp.StatusCode != 200 {
		return nil, &url.Error{
			Op:  "",
			URL: req.URL.String(),
			Err: fmt.Errorf("status: %s", resp.Status),
		}
	}
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (d *NerinyanDownloader) DownloadBeatmapset(beatmapsetId int) ([]byte, error) {
	req, err := http.NewRequestWithContext(d.ctx, "GET", fmt.Sprintf("%s/d/%d", nerinyanApi, beatmapsetId), nil)
	if err != nil {
		return nil, err
	}
	return d.do(req)
}

func (d *NerinyanDownloader) Name() string {
	return "nerinyan"
}
