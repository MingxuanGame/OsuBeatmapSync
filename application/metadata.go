package application

import (
	"encoding/json"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	. "github.com/MingxuanGame/OsuBeatmapSync/model/onedrive"
	"github.com/MingxuanGame/OsuBeatmapSync/onedrive"
	"github.com/MingxuanGame/OsuBeatmapSync/sql"
	"log"
	"os"
)

func getMetadataFromRemote(graph *onedrive.GraphClient, root string) (Metadata, error) {
	metadata := Metadata{
		GameMode:    make(map[GameMode]MetadataGameMode),
		Beatmaps:    make(map[int]BeatmapMetadata),
		Beatmapsets: make(map[int]BeatmapsetMetadata),
	}
	files, err := graph.ListFiles(root, 200, "")
	if err != nil {
		return Metadata{}, err
	}
	var metadataFile DriveItem
	if files == nil {
		log.Println("No existed metadata file found")
		return metadata, nil
	}
	for _, file := range *files {
		if file.Name == "metadata.db" {
			metadataFile = file
			break
		}
	}
	if metadataFile.Name != "" {
		log.Println("Found existed metadata file, downloading...")
		data, err := graph.DownloadFile(metadataFile.Id)
		if err != nil {
			return Metadata{}, err
		}
		err = os.WriteFile("metadata.db", data, 0644)
		if err != nil {
			return Metadata{}, err
		}
		db, err := sql.OpenDatabase()
		if err != nil {
			return Metadata{}, err
		}
		metadata, err = db.ReadMetadata()
		if err != nil {
			return Metadata{}, err
		}
		err = db.Close()
		if err != nil {
			return Metadata{}, err
		}
		_ = os.Remove("metadata.db")
	} else {
		metadata = Metadata{
			GameMode:    make(map[GameMode]MetadataGameMode),
			Beatmaps:    make(map[int]BeatmapMetadata),
			Beatmapsets: make(map[int]BeatmapsetMetadata),
		}
	}
	return metadata, nil
}

func ReadLocalMetadata(filename string) (Metadata, error, bool) {
	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		return Metadata{}, nil, false
	}
	if err != nil {
		return Metadata{}, err, false
	}
	var metadata Metadata
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return Metadata{}, err, false
	}
	return metadata, nil, true
}

func GetMetadata(graph *onedrive.GraphClient, root string) (Metadata, error) {
	metadata, err, ok := ReadLocalMetadata("metadata.json")
	if err != nil {
		return Metadata{}, err
	}
	if !ok {
		metadata, err = getMetadataFromRemote(graph, root)
		if err != nil {
			return Metadata{}, err
		}
	}
	return metadata, nil
}

func SaveMetadataToLocal(metadata *Metadata) error {
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	err = os.WriteFile("metadata.json", jsonData, 0644)
	return err
}

func SaveMetadataToLocalDB(metadata *Metadata) error {
	db, err := sql.OpenDatabase()
	if err != nil {
		return err
	}
	err = db.DropAllMetadata()
	if err != nil {
		return err
	}
	err = db.WriteMetadata(metadata)
	if err != nil {
		return err
	}
	defer func(db *sql.Database) {
		err := db.Close()
		if err != nil {
			log.Println(err)
		}
	}(db)
	return nil
}

func UploadMetadata(graph *onedrive.GraphClient, root string, metadata *Metadata) error {
	err := SaveMetadataToLocalDB(metadata)
	if err != nil {
		return err
	}
	log.Println("Uploading metadata...")
	data, err := os.ReadFile("metadata.db")
	if err != nil {
		return err
	}
	err = graph.UploadLargeFile(root, "metadata.db", data)
	if err != nil {
		return err
	}
	_ = os.Remove("metadata.db")
	_ = os.Remove("metadata.json")
	return nil
}
