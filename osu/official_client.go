package osu

import (
	"encoding/json"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var logger = log.With().Str("module", "osu.legacy").Logger()

// LegacyOfficialClient osu! v1 API
type LegacyOfficialClient struct {
	ApiKey string
	client *http.Client
}

func (client *LegacyOfficialClient) GetBeatmapBySetId(s int) (*[]Beatmap, error) {
	return client.GetBeatmap(map[string]interface{}{"s": strconv.Itoa(s)})
}

func (client *LegacyOfficialClient) GetBeatmapByBeatmapId(b int) (*[]Beatmap, error) {
	return client.GetBeatmap(map[string]interface{}{"b": strconv.Itoa(b)})
}

func (client *LegacyOfficialClient) GetBeatmapByTime(since time.Time) (*[]Beatmap, error) {
	return client.GetBeatmap(map[string]interface{}{"since": since.Format(time.DateTime)})
}

func (client *LegacyOfficialClient) GetBeatmap(searchParam map[string]interface{}) (*[]Beatmap, error) {
	query := url.Values{}
	query.Set("k", client.ApiKey)
	for k, v := range searchParam {
		query.Set(k, v.(string))
	}
	logger.Trace().Msgf("Getting beatmaps: %v", query)
	reqUrl := "https://osu.ppy.sh/api/get_beatmaps?" + query.Encode()
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return nil, &url.Error{
			Op:  "",
			URL: reqUrl,
			Err: err,
		}
	}

	resp, err := client.client.Do(req)
	if err != nil {
		return nil, &url.Error{
			Op:  "",
			URL: reqUrl,
			Err: err,
		}
	}
	if resp.StatusCode >= 400 {
		return nil, &url.Error{
			Op:  "",
			URL: reqUrl,
			Err: err,
		}
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &url.Error{
			Op:  "",
			URL: reqUrl,
			Err: err,
		}
	}
	var response []Beatmap
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, &url.Error{
			Op:  "",
			URL: reqUrl,
			Err: err,
		}
	}
	return &response, nil
}

func NewLegacyOfficialClient(apiKey string) *LegacyOfficialClient {
	return &LegacyOfficialClient{
		ApiKey: apiKey,
		client: &http.Client{},
	}
}
