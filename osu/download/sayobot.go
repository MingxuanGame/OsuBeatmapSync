package download

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	netUrl "net/url"
	"strings"
)

const sayobotApi = `https://txy1.sayobot.cn/beatmaps/download`

var sayobotLogger = log.With().Str("module", "osu.download.sayobot").Logger()

type SayobotDownloader struct {
	client *http.Client
	server string
	ctx    context.Context
}

func NewSayobotDownloader(server string, ctx context.Context) *SayobotDownloader {
	return &SayobotDownloader{
		client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		server: server, ctx: ctx,
	}
}

func (d *SayobotDownloader) DownloadBeatmapset(beatmapsetId int) ([]byte, error) {
	req, err := http.NewRequestWithContext(d.ctx, "GET", fmt.Sprintf("%s/full/%d?server=%s", sayobotApi, beatmapsetId, d.server), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/132.0.0.0`)
	req.Header.Set("Referer", "https://osu.sayobot.cn/")
	sayobotLogger.Trace().Msgf("Requesting %s %s", req.Method, req.URL.String())
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	location, err := netUrl.Parse(resp.Header.Get("Location"))
	filename, err := netUrl.PathUnescape(location.Query().Get("filename"))
	if err != nil {
		return nil, err
	}
	newUrl := fmt.Sprintf("%s://%s%s?filename=%s", location.Scheme, location.Host, location.Path, netUrl.PathEscape(filename))
	req, err = http.NewRequest("GET", newUrl, nil)
	if err != nil {
		return nil, err
	}
	resp, err = d.client.Do(req)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, &netUrl.Error{
			Op:  resp.Request.Method,
			URL: resp.Request.URL.String(),
			//-1 = 服务器资源不足
			//-2 = 读取失败
			//-3 = ppy不给下载
			//-4 = 不知道发生了什么
			Err: fmt.Errorf("status: %s, %s", resp.Status, strings.Split(string(data), "\n")[9]),
		}
	}
	return data, nil
}

func (d *SayobotDownloader) Name() string {
	return "Sayobot"
}
