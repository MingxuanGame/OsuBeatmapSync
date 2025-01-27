package onedrive

import (
	"encoding/json"
	. "github.com/MingxuanGame/OsuBeatmapSync/model/onedrive"
	"io"
)

//type IdentitySet

func (client *GraphClient) GetDrive() (*[]Drive, error) {
	req, err := client.NewRequest("GET", "/me/drives", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var response struct {
		Value []Drive `json:"value"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}
	drive := response.Value
	return &drive, nil
}
