package application

import (
	"context"
	"github.com/MingxuanGame/OsuBeatmapSync/base_service"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/MingxuanGame/OsuBeatmapSync/onedrive"
)

func Login(config *Config, ctx context.Context) (*onedrive.GraphClient, error) {
	var client *onedrive.GraphClient
	if config.OneDrive.Token.AccessToken == "" || config.OneDrive.Token.RefreshToken == "" {
		logger.Info().Msg("No existed token found, login...")
		var err error
		client, err = onedrive.NewGraphClient(config.OneDrive.ClientId, config.OneDrive.ClientSecret, config.OneDrive.Tenant, ctx)
		if err != nil {
			return nil, err
		}
	} else {
		logger.Info().Msg("Existed token found, login...")
		client = onedrive.NewExistedGraphClient(&config.OneDrive, ctx)
	}
	config.OneDrive.Token.AccessToken = client.Config.Token.AccessToken
	config.OneDrive.Token.RefreshToken = client.Config.Token.RefreshToken
	err := base_service.SaveConfig(config)
	if err != nil {
		return nil, err
	}
	drive, err := client.GetDrive()
	if err != nil {
		return nil, err
	}
	for _, drive := range *drive {
		logger.Info().Msg("Drive info:")
		logger.Info().Msgf("  Drive: %s", drive.Id)
		logger.Info().Msgf("  DriveType: %s", drive.DriveType)
		logger.Info().Msgf("  Total: %d", drive.Quota.Total)
		logger.Info().Msgf("  Used: %d", drive.Quota.Used)
		logger.Info().Msgf("  Remaining: %d", drive.Quota.Remaining)
	}
	logger.Info().Msg("Login successful...")
	return client, nil
}
