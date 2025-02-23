package application

import (
	"encoding/json"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	. "github.com/MingxuanGame/OsuBeatmapSync/model/onedrive"
	"github.com/MingxuanGame/OsuBeatmapSync/onedrive"
	"github.com/MingxuanGame/OsuBeatmapSync/sql"
	"github.com/rs/zerolog/log"
	"os"
)

const MetadataTempFilename = "metadata.json"

func getMetadataFromRemote(graph *onedrive.GraphClient, root string) (Metadata, error) {
	metadata := Metadata{
		GameMode:    make(map[GameMode]MetadataGameMode),
		Beatmaps:    make(map[int]BeatmapMetadata),
		Beatmapsets: make(map[int]BeatmapsetMetadata),
	}
	item, err := graph.GetItem(root, "")
	if err != nil {
		return Metadata{}, err
	}
	if item == nil {
		log.Info().Msg("Root folder not found, creating...")
		_, err = graph.CreateFolderRecursive(root)
		if err != nil {
			return Metadata{}, err
		}
	}

	files, err := graph.ListFiles(root, 200, "")
	if err != nil {
		return Metadata{}, err
	}
	var metadataFile DriveItem
	if files == nil {
		logger.Info().Msg("No existed metadata file found")
		return metadata, nil
	}
	for _, file := range *files {
		if file.Name == sql.MetadataDBFilename {
			metadataFile = file
			break
		}
	}
	if metadataFile.Name != "" {
		logger.Info().Msg("Found existed metadata file, downloading...")
		data, err := graph.DownloadFile(metadataFile.Id)
		if err != nil {
			return Metadata{}, err
		}
		f, err := os.CreateTemp("", "beatmap-sync-")
		if err != nil {
			return Metadata{}, err
		}
		logger.Trace().Str("filename", f.Name()).Msg("Saving metadata to temp local database...")
		_, err = f.Write(data)
		if err != nil {
			return Metadata{}, err
		}
		db, err := sql.OpenDatabase(f.Name())
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
		_ = f.Close()
		_ = os.Remove(f.Name())
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
		logger.Trace().Msgf("File %s not found", filename)
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
	metadata, err, ok := ReadLocalMetadata(MetadataTempFilename)
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
	logger.Trace().Msg("Saving metadata to local...")
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	err = os.WriteFile(MetadataTempFilename, jsonData, 0644)
	return err
}

func SaveMetadataToLocalDB(metadata *Metadata) (string, error) {
	f, err := os.CreateTemp("", "beatmap-sync-")
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			logger.Error().Err(err).Str("filename", f.Name()).Msg("Failed to close temp database")
		}
	}(f)
	if err != nil {
		return "", err
	}
	logger.Trace().Str("filename", f.Name()).Msg("Saving metadata to temp local database...")
	db, err := sql.OpenDatabase(f.Name())
	if err != nil {
		return "", err
	}
	err = db.DropAllMetadata()
	if err != nil {
		return "", err
	}
	err = db.WriteMetadata(metadata)
	if err != nil {
		return "", err
	}
	defer func(db *sql.Database) {
		err := db.Close()
		if err != nil {
			logger.Error().Err(err).Msg("Failed to close database")
		}
	}(db)
	return f.Name(), nil
}

func UploadMetadata(graph *onedrive.GraphClient, root string, metadata *Metadata) error {
	filename, err := SaveMetadataToLocalDB(metadata)
	if err != nil {
		return err
	}
	logger.Info().Msg("Uploading metadata...")
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	err = graph.UploadLargeFile(root, sql.MetadataDBFilename, data)
	if err != nil {
		return err
	}
	_ = os.Remove(filename)
	_ = os.Remove(MetadataTempFilename)
	return nil
}
