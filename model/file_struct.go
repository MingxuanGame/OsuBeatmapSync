package model

import (
	"github.com/MingxuanGame/OsuBeatmapSync/utils"
	"log"
	"strings"
)

type FilenameStruct struct {
	Root       string
	GameMode   string
	Status     string
	Type       string
	Beatmapset string
}

func MakeFilenameStruct(root, gameMode, status, typ, beatmapset string) FilenameStruct {
	return FilenameStruct{
		Root:       root,
		GameMode:   gameMode,
		Status:     status,
		Type:       typ,
		Beatmapset: beatmapset,
	}
}

func ParseFilenameStruct(path string) (*FilenameStruct, error) {
	// Root / Status / Type / "sid Artist - Title.osz"
	path = utils.Reverse(path)
	node := strings.SplitN(path, "/", 5)
	if len(node) != 5 {
		log.Println("Filename not match regex:", utils.Reverse(path))
		return nil, nil
	}
	return &FilenameStruct{
		Root:       utils.Reverse(node[4]),
		GameMode:   utils.Reverse(node[3]),
		Status:     utils.Reverse(node[2]),
		Type:       utils.Reverse(node[1]),
		Beatmapset: utils.Reverse(node[0]),
	}, nil
}
