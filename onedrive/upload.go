package onedrive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"time"
)

type uploadSession struct {
	uploadUrl string
	*http.Client
	ctx        context.Context
	currSize   int64
	singleSize int64
	expireTime time.Time
	totalSize  int64
}

const chunkSize = 10_485_760 // 10MB

// UploadFile limit: 4MB
func (client *GraphClient) UploadFile(path string, filename string, data []byte) error {
	req, err := client.NewRequest("PUT", fmt.Sprintf("/me/drive/root:/%s/%s:/content", path, filename), data)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	_, err = client.ReadData(resp)
	if err != nil {
		return err
	}
	return nil
}

func (client *GraphClient) createUploadSession(path, filename string, totalSize int64) (*uploadSession, error) {
	req, err := client.NewRequest("POST", fmt.Sprintf("/me/drive/root:/%s:/createUploadSession", path+"/"+url.PathEscape(filename)), nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to create upload session: %d", resp.StatusCode)
	}
	data, err := client.ReadData(resp)
	if err != nil {
		return nil, err
	}
	var response struct {
		UploadUrl          string `json:"uploadUrl"`
		ExpirationDateTime string `json:"expirationDateTime"`
	}

	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}
	expireTime, err := time.Parse(time.RFC3339, response.ExpirationDateTime)
	if err != nil {
		expireTime = time.Now().Add(time.Hour)
	}
	return &uploadSession{
		uploadUrl:  response.UploadUrl,
		Client:     client.Client,
		ctx:        client.ctx,
		totalSize:  totalSize,
		expireTime: expireTime,
		currSize:   0,
	}, nil
}

func (session *uploadSession) uploadChunkWithRetry(data []byte, retry int, serverFailRetry int) (bool, error) {
	req, err := http.NewRequestWithContext(session.ctx, "PUT", session.uploadUrl, bytes.NewReader(data))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))
	req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", session.currSize, session.currSize+int64(len(data))-1, session.totalSize))
	resp, err := session.Do(req)
	if err != nil {
		return false, err
	}

	switch {
	case resp.StatusCode == 202:
		session.currSize += int64(len(data))
		return false, nil
	case resp.StatusCode == 201 || resp.StatusCode == 200:
		return true, nil
	case resp.StatusCode == 416:
		return false, &url.Error{
			Op:  "",
			URL: req.URL.String(),
			Err: fmt.Errorf("chunk %d-%d is already uploaded", session.currSize, session.currSize+int64(len(data))-1),
		}
	case resp.StatusCode == 409:
		return false, &url.Error{
			Op:  "",
			URL: req.URL.String(),
			Err: fmt.Errorf("conflict"),
		}
	case resp.StatusCode == 404:
		return false,
			&url.Error{
				Op:  "",
				URL: req.URL.String(),
				Err: fmt.Errorf("upload session not found"),
			}
	case resp.StatusCode >= 500:
		logger.Warn().Msgf("Server error, retrying in %d seconds", int(math.Pow(2, float64(serverFailRetry))))
		time.Sleep(time.Duration(1000 * math.Pow(2, float64(serverFailRetry))))
		return session.uploadChunkWithRetry(data, retry, serverFailRetry+1)
	case resp.StatusCode >= 400:
		logger.Error().Msgf("Client error, retrying in 10 second (%d remaining)", retry-1)
		if retry == 0 {
			return false, &url.Error{
				Op:  "",
				URL: req.URL.String(),
				Err: fmt.Errorf("retry limit exceeded"),
			}
		}
		time.Sleep(time.Second * 10)
		return session.uploadChunkWithRetry(data, retry-1, serverFailRetry)
	}
	return false, nil
}

func (session *uploadSession) upload(data []byte) error {
	slices := make([][]byte, 0)
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		slices = append(slices, data[i:end])
	}
	for _, slice := range slices {
		done, err := session.uploadChunkWithRetry(slice, 3, 0)
		if err != nil {
			return err
		}
		if done {
			break
		}
	}
	return nil
}

func (client *GraphClient) UploadLargeFile(path, filename string, data []byte) error {
	session, err := client.createUploadSession(path, filename, int64(len(data)))
	if err != nil {
		return err
	}
	return session.upload(data)
}
