package utils

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"regexp"
	"strconv"
)

const OsuFilenameRegex = `^(?P<beatmapsetId>\d+)\s+(?P<artist>[\x00-\xff]+)\s+-\s+(?P<name>.+)\.osz$`

func ParseFilename(filename string) (artist, name string, beatmapsetId int) {
	re := regexp.MustCompile(OsuFilenameRegex)
	match := re.FindStringSubmatch(filename)
	result := make(map[string]string)
	if match == nil {
		log.Info().Msgf("Filename not match regex: %s", filename)
		return "", "", 0
	}
	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}
	beatmapsetId, _ = strconv.Atoi(result["beatmapsetId"])
	artist = result["artist"]
	name = result["name"]
	return
}

func MakeFilename(beatmapsetId int, artist, name string) string {
	return SanitizeFileName(fmt.Sprintf("%d %s - %s.osz", beatmapsetId, artist, name))
}
