package onedrive

import (
	"encoding/base64"
	"encoding/hex"
	"github.com/MingxuanGame/OsuBeatmapSync/onedrive/quickxorhash"
)

type DriveItem struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size"`
	File *struct {
		MIMEType string `json:"mimeType"`
		Hashes   struct {
			SHA1Hash     string `json:"sha1Hash,omitempty"`
			CRC32Hash    string `json:"crc32Hash,omitempty"`
			QuickXorHash string `json:"quickXorHash,omitempty"`
		} `json:"hashes,omitempty"`
	} `json:"file"`
	Folder *struct {
		ChildCount int `json:"childCount"`
	} `json:"folder"`
	Shared struct {
		Scope string `json:"scope"`
	} `json:"shared"`
	ParentReference struct {
		Name string `json:"name"`
		Path string `json:"path"`
	} `json:"parentReference"`
}

type ShareLink struct {
	Type   string `json:"type"`
	WebUrl string `json:"webUrl"`
}

func (item DriveItem) IsFile() bool {
	return item.File != nil
}

func (item DriveItem) IsFolder() bool {
	return item.Folder != nil
}

func (item DriveItem) VerifyQuickXorHash(target []byte) bool {
	if item.File == nil {
		return false
	}
	if item.File.Hashes.QuickXorHash == "" {
		return false
	}
	sourceChecksum, err := base64.StdEncoding.DecodeString(item.File.Hashes.QuickXorHash)
	if err != nil {
		return false
	}
	targetChecksum := quickxorhash.Sum(target)
	return hex.EncodeToString(sourceChecksum) == targetChecksum
}
