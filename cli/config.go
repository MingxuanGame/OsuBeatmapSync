package cli

import (
	"fmt"
	"github.com/MingxuanGame/OsuBeatmapSync/base_service"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog"
	"os"
)

func GenerateConfig() error {
	_, err := base_service.LoadConfig()
	if err == nil {
		return fmt.Errorf("config file already exists")
	}
	config := Config{
		General: GeneralConfig{MaxConcurrent: 20, UploadMultiple: 2, LogLevel: int8(zerolog.InfoLevel)},
		OneDrive: OneDrive{
			ClientId:     "your_client_id",
			ClientSecret: "your_client_secret",
			Tenant:       "your_tenant_id",
		},
		Osu: Osu{
			V1ApiKey: "your_v1_api_key",
			Sayobot: struct {
				Server string `toml:"server"`
			}{Server: "auto"},
			EnableSayobot:  true,
			EnableNerinyan: true,
			EnableCatboy:   true,
			EnableOfficial: true,
			ProcessTypes:   []string{},
		},
		Path: OneDrivePath{
			Root:          "your_root",
			StdPath:       "std",
			TaikoPath:     "taiko",
			CatchPath:     "catch",
			ManiaPath:     "mania",
			RankedPath:    "ranked",
			LovedPath:     "loved",
			QualifiedPath: "qualified",
		}}
	content, err := toml.Marshal(config)
	if err != nil {
		return err
	}
	err = os.WriteFile(base_service.ConfigPath, content, 0644)
	if err != nil {
		return err
	}
	return nil
}
