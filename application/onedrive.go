package application

import (
	"context"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/MingxuanGame/OsuBeatmapSync/onedrive"
	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog/log"
	"os"
)

func Login(config *Config, ctx context.Context) (*onedrive.GraphClient, error) {
	var client *onedrive.GraphClient
	if config.OneDrive.Token.AccessToken == "" || config.OneDrive.Token.RefreshToken == "" {
		log.Info().Msg("No existed token found, login...")
		var err error
		client, err = onedrive.NewGraphClient(config.OneDrive.ClientId, config.OneDrive.ClientSecret, config.OneDrive.Tenant, ctx)
		if err != nil {
			return nil, err
		}
	} else {
		log.Info().Msg("Existed token found, login...")
		client = onedrive.NewExistedGraphClient(&config.OneDrive, ctx)
	}
	config.OneDrive.Token.AccessToken = client.Config.Token.AccessToken
	config.OneDrive.Token.RefreshToken = client.Config.Token.RefreshToken
	content, err := toml.Marshal(config)
	if err != nil {
		panic("Failed to marshal config.toml")
	}
	err = os.WriteFile("./config.toml", content, 0644)
	if err != nil {
		return nil, err
	}
	drive, err := client.GetDrive()
	if err != nil {
		return nil, err
	}
	for _, drive := range *drive {
		log.Info().Msg("Drive info:")
		log.Info().Msgf("  Drive: %s", drive.Id)
		log.Info().Msgf("  DriveType: %s", drive.DriveType)
		log.Info().Msgf("  Total: %d", drive.Quota.Total)
		log.Info().Msgf("  Used: %d", drive.Quota.Used)
		log.Info().Msgf("  Remaining: %d", drive.Quota.Remaining)
	}
	log.Info().Msg("Login successful...")
	return client, nil
}
