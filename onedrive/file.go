package onedrive

import (
	"encoding/json"
	"fmt"
	. "github.com/MingxuanGame/OsuBeatmapSync/model/onedrive"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const shareLinkRegex = `https:\/\/(\S+).sharepoint.com\/:\S:\/g\/personal\/(\S+)\/([a-zA-Z_\-0-9]+)`

func (client *GraphClient) ListFiles(path string, length int, nextUrl string) (*[]DriveItem, error) {
	u := "/me/drive/root:/" + path + ":/children?select=id,name,size,file,folder,shared,parentReference&top=" + strconv.Itoa(length)
	if nextUrl != "" {
		u = u + "&$skiptoken=" + nextUrl
	}
	req, err := client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := client.ReadData(resp)
	if err != nil {
		return nil, err
	}
	var response struct {
		Value    []DriveItem `json:"value"`
		NextItem string      `json:"@odata.nextLink"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}
	files := response.Value
	if len(files) == 0 {
		return nil, nil
	}

	if len(files) > length {
		return &files, nil
	}
	for {
		logger.Info().Str("path", path).Msgf("Got %d, remaining %d", len(files), length-len(files))
		if response.NextItem == "" || len(files) >= length {
			break
		}
		m, err := url.ParseQuery(strings.Split(response.NextItem, "?")[1])
		if err != nil {
			return nil, err
		}
		nextUrl := m.Get("$skiptoken")
		nextFiles, err := client.ListFiles(path, length-len(files), nextUrl)
		if err != nil {
			return nil, err
		}
		files = append(files, *nextFiles...)
		break
	}
	return &files, nil
}

func (client *GraphClient) ListAllFiles(root string, length int) ([]DriveItem, error) {
	var wg sync.WaitGroup

	rootFiles, err := client.ListFiles(root, length, "")
	if err != nil {
		return nil, err
	}
	var dirs []DriveItem
	var allFiles []DriveItem
	if rootFiles == nil {
		return nil, nil
	}
	for _, file := range *rootFiles {
		if file.Folder != nil {
			dirs = append(dirs, file)
		} else if strings.HasSuffix(file.Name, ".osz") {
			allFiles = append(allFiles, file)
		}
	}

	for _, dir := range dirs {
		wg.Add(1)
		go func(dir *DriveItem, wg *sync.WaitGroup) {
			defer wg.Done()
			files, err := client.ListAllFiles(root+"/"+dir.Name, dir.Folder.ChildCount)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to list files")
				return
			}
			allFiles = append(allFiles, files...)
		}(&dir, &wg)
	}
	wg.Wait()
	return allFiles, nil
}

func (client *GraphClient) MakeShareLink(item string) (string, error) {
	//req, err := client.NewRequestWithBuffer("POST", "/me/drive/items/"+item+"/createLink", bytes.NewBufferString(`{"type": "view", "scope": "anonymous"}`))
	req, err := client.NewRequestJson("POST", "/me/drive/items/"+item+"/createLink", map[string]interface{}{
		"type": "view", "scope": "anonymous",
	})
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	data, err := client.ReadData(resp)
	if err != nil {
		return "", err
	}
	var response struct {
		Link ShareLink `json:"link"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return "", err
	}

	r := regexp.MustCompile(shareLinkRegex)
	matches := r.FindStringSubmatch(response.Link.WebUrl)
	if len(matches) == 0 {
		return "", fmt.Errorf("failed to extract share link: %s", response.Link.WebUrl)
	}
	//logger.Printf("[%s] Created Share link\n", item)
	return fmt.Sprintf("https://%s.sharepoint.com/personal/%s/_layouts/15/download.aspx?share=%s", matches[1], matches[2], matches[3]), nil
}

func (client *GraphClient) DownloadFile(itemId string) ([]byte, error) {
	req, err := client.NewRequest("GET", fmt.Sprintf("/me/drive/items/%s/content", itemId), nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := client.ReadData(resp)
	if err != nil {
		return nil, err
	}
	return data, nil

}

func (client *GraphClient) GetItem(path, filename string) (*DriveItem, error) {
	if filename != "" {
		path = path + "/" + url.PathEscape(filename)
	}
	req, err := client.NewRequest("GET", "/me/drive/root:/"+path+":/?select=id,name,size,file,folder,shared,parentReference", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 404 {
		return nil, nil
	}
	data, err := client.ReadData(resp)
	if err != nil {
		return nil, err
	}
	var item DriveItem
	err = json.Unmarshal(data, &item)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (client *GraphClient) DeleteItem(itemId string) error {
	req, err := client.NewRequest("DELETE", "/me/drive/items/"+itemId, nil)
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

func (client *GraphClient) MoveItem(itemId, targetId string) error {
	req, err := client.NewRequestJson("PATCH", "/me/drive/items/"+itemId, map[string]interface{}{
		"parentReference": map[string]string{"id": targetId},
	})
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

func (client *GraphClient) CreateFolder(parentId, name string) (*DriveItem, error) {
	req, err := client.NewRequestJson("POST", "/me/drive/items/"+parentId+"/children", map[string]interface{}{
		"name": name, "folder": map[string]interface{}{},
	})
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	data, err := client.ReadData(resp)
	if err != nil {
		return nil, err
	}
	var item DriveItem
	err = json.Unmarshal(data, &item)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (client *GraphClient) CreateFolderRecursive(path string) (*DriveItem, error) {
	paths := strings.Split(path, "/")
	var parentId string
	for i, p := range paths {
		if i == 0 {
			parentId = "root"
		}
		item, err := client.GetItem(parentId, p)
		if err != nil {
			return nil, err
		}
		if item == nil {
			item, err = client.CreateFolder(parentId, p)
			if err != nil {
				return nil, err
			}
		}
		parentId = item.Id
	}
	return client.GetItem(parentId, "")
}
