package utils

import (
	"log"
	"regexp"
	"strconv"
)

const OsuFilenameRegex = `^(?P<beatmapsetId>\d+)\s+(?P<artist>[\x00-\xff]+)\s+-\s+(?P<name>.+)\.osz$`

func ParseFilename(filename string) (artist, name string, beatmapsetId int) {
	re := regexp.MustCompile(OsuFilenameRegex)
	match := re.FindStringSubmatch(filename)
	result := make(map[string]string)
	if match == nil {
		log.Println("Filename not match regex:", filename)
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
