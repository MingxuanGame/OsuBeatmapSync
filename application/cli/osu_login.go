package cli

import (
	"context"
	"github.com/MingxuanGame/OsuBeatmapSync/application"
	downloader "github.com/MingxuanGame/OsuBeatmapSync/osu/download"
)

func writeTokenToConfig(accessToken string, refreshToken string) error {
	config, err := application.LoadConfig()
	if err != nil {
		return err
	}
	config.Osu.OfficialDownloader.AccessToken = accessToken
	config.Osu.OfficialDownloader.RefreshToken = refreshToken
	return application.SaveConfig(&config)
}

func LoginToOsuUseLocal() error {
	accessToken, refreshToken, err := downloader.GetAccessTokenFromLocal()
	if err != nil {
		return err
	}
	return writeTokenToConfig(accessToken, refreshToken)
}

func LoginToOsu(ctx context.Context, username string, password string) error {
	client, err := downloader.NewOfficialDownloaderLogin(ctx, username, password)
	if err != nil {
		return err
	}
	return writeTokenToConfig(client.AccessToken, client.RefreshToken)
}
