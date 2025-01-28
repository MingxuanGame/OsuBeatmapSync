package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MingxuanGame/OsuBeatmapSync/application"
	. "github.com/MingxuanGame/OsuBeatmapSync/metadata"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	. "github.com/MingxuanGame/OsuBeatmapSync/model/onedrive"
	"github.com/MingxuanGame/OsuBeatmapSync/onedrive"
	"github.com/MingxuanGame/OsuBeatmapSync/osu"
	"github.com/MingxuanGame/OsuBeatmapSync/utils"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const needMakeListFilename = "needMakeList.json"

func getAllFile(graph *onedrive.GraphClient, root string) ([]DriveItem, error) {
	allFiles, err := graph.ListAllFiles(root, 200)
	if err != nil {
		return nil, err
	}
	return allFiles, nil
}

func readLocalNeedMakeList(filename string) ([]DriveItem, error, bool) {
	var needMakeList []DriveItem
	savedData, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		return nil, nil, false
	}
	if err != nil {
		return nil, err, false
	}
	if len(savedData) != 0 {
		err = json.Unmarshal(savedData, &needMakeList)
		if err != nil {
			return nil, err, false
		}
	}
	return needMakeList, nil, true
}

func getNeedMakeList(graph *onedrive.GraphClient, root string, metadata *Metadata) ([]DriveItem, error) {
	needMakeList, err, ok := readLocalNeedMakeList(needMakeListFilename)
	if err != nil {
		return nil, err
	}
	if !ok {
		allFiles, err := getAllFile(graph, root)
		if err != nil {
			return nil, err
		}
		log.Println("All files count: ", len(allFiles))
		for _, file := range allFiles {
			if file.IsFolder() || !strings.HasSuffix(file.Name, ".osz") {
				continue
			}
			_, _, beatmapsetId := utils.ParseFilename(file.Name)
			if _, ok := metadata.Beatmapsets[beatmapsetId]; !ok {
				needMakeList = append(needMakeList, file)
			}
		}
		needMakeListData, err := json.Marshal(needMakeList)
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(needMakeListFilename, needMakeListData, 0644)
		if err != nil {
			return nil, err
		}
	}
	return needMakeList, nil
}

func makeMetadata(g *Generator, needMakeList []DriveItem, ctx context.Context) error {
	log.Println("Need to made metadata count: ", len(needMakeList))
	if len(needMakeList) == 0 {
		return nil
	}
	log.Printf("Generating Beatmapset: %d\n", len(needMakeList))
	g.GenerateExistedFileMetadata(needMakeList)
	metadata := g.Metadata
	err := application.SaveMetadataToLocal(metadata)
	if err != nil {
		return err
	}
	log.Printf("Generated Beatmapset: %d\n", len(metadata.Beatmapsets))
	select {
	case <-ctx.Done():
		os.Exit(0)
	default:
	}
	for {
		if len(g.Failed) == 0 {
			break
		}
		log.Println("Failed: ", len(g.Failed))
		time.Sleep(time.Minute)
		g.GenerateExistedFileMetadata(g.Failed)
		err := application.SaveMetadataToLocal(metadata)
		if err != nil {
			return err
		}
	}
	return nil
}

func MakeMetadata(ctx context.Context, tasks, worker int, start bool) error {
	config, err := application.LoadConfig()
	if err != nil {
		return err
	}

	client, err := application.Login(&config, ctx)
	if err != nil {
		return err
	}

	osuClient := osu.NewLegacyOfficialClient(config.Osu.V1ApiKey)
	log.Println("Start making metadata...")
	root := config.Path.Root
	metadata, err := application.GetMetadata(client, root)
	if err != nil {
		return err
	}
	needMakeList, err := getNeedMakeList(client, root, &metadata)
	if err != nil {
		return err
	}
	if tasks > 1 {
		taskList := utils.SplitSlice(needMakeList, tasks)
		for i, task := range taskList {
			taskJson, err := json.Marshal(task)
			if err != nil {
				return err
			}
			err = os.WriteFile("needMakeList"+strconv.Itoa(i+1)+".json", taskJson, 0644)
			if err != nil {
				return err
			}
		}
		if worker == 0 {
			worker = 1
		}
	}
	generator := NewGenerator(osuClient, client, ctx, config.General.MaxConcurrent, &metadata)
	if worker == 0 {
		if len(needMakeList) > 0 {
			err := makeMetadata(generator, needMakeList, ctx)
			if err != nil {
				return err
			}
			err = application.UploadMetadata(client, root, &metadata)
			if err != nil {
				return err
			}
		}
		_ = os.Remove(application.MetadataTempFilename)
		_ = os.Remove(needMakeListFilename)
	} else if start {
		log.Println("Current worker: ", worker)
		needMakeList, err, ok := readLocalNeedMakeList("needMakeList" + strconv.Itoa(worker) + ".json")
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("no needMakeList file")
		}
		err = makeMetadata(generator, needMakeList, ctx)
		if err != nil {
			return err
		}
	}
	_ = os.Remove("allFiles.json")
	return nil
}

func MergeMetadata(isUpload bool, files []string) error {
	var mergedMetadata Metadata
	for _, file := range files {
		currMetadata, err, ok := application.ReadLocalMetadata(file)
		if err != nil {
			return err
		}
		if !ok {
			log.Printf("file %s not found\n", file)
			continue
		}
		if mergedMetadata.GameMode == nil {
			mergedMetadata = currMetadata
			continue
		}

		for k, v := range currMetadata.Beatmapsets {
			if beatmapset2, ok := mergedMetadata.Beatmapsets[k]; ok {
				if v.LastUpdate > beatmapset2.LastUpdate {
					mergedMetadata.Beatmapsets[k] = v
				} else {
					mergedMetadata.Beatmapsets[k] = beatmapset2
				}
			}
		}
		for k, v := range currMetadata.Beatmaps {
			if beatmap2, ok := mergedMetadata.Beatmaps[k]; ok {
				if v.LastUpdate > beatmap2.LastUpdate {
					mergedMetadata.Beatmaps[k] = v
				} else {
					mergedMetadata.Beatmaps[k] = beatmap2
				}
			}
		}
		for k, v := range currMetadata.GameMode {
			if mode2, ok := mergedMetadata.GameMode[k]; ok {
				if v.UpdateTime > mode2.UpdateTime {
					mergedMetadata.GameMode[k] = v
				} else {
					mergedMetadata.GameMode[k] = mode2
				}
			}
		}
	}

	if isUpload {
		config, err := application.LoadConfig()
		if err != nil {
			return err
		}

		ctx := application.CreateSignalCancelContext()
		client, err := application.Login(&config, ctx)
		err = application.UploadMetadata(client, config.Path.Root, &mergedMetadata)
		if err != nil {
			return err
		}
	} else {
		err := application.SaveMetadataToLocal(&mergedMetadata)
		if err != nil {
			return err
		}
	}
	return nil
}
