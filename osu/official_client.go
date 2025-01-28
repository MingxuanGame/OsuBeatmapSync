package osu

import (
	"encoding/json"
	"fmt"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

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
	req, err := http.NewRequest("GET", "https://osu.ppy.sh/api/get_beatmaps?"+query.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("[osu! legacy api] failed to create request: %w", err)
	}

	resp, err := client.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[osu! legacy api] failed to send request: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("[osu! legacy api] failed to get response: %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("[osu! legacy api] failed to read response: %w", err)
	}
	var response []Beatmap
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, fmt.Errorf("[osu! legacy api] failed to unmarshal response: %w", err)
	}
	return &response, nil
}

func NewLegacyOfficialClient(apiKey string) *LegacyOfficialClient {
	return &LegacyOfficialClient{
		ApiKey: apiKey,
		client: &http.Client{},
	}
}
