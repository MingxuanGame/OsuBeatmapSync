package application

import (
	"context"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/MingxuanGame/OsuBeatmapSync/onedrive"
	"github.com/pelletier/go-toml/v2"
	"log"
	"os"
)

func Login(config *Config, ctx context.Context) (*onedrive.GraphClient, error) {
	var client *onedrive.GraphClient
	if config.OneDrive.Token.AccessToken == "" || config.OneDrive.Token.RefreshToken == "" {
		log.Println("No existed token found, login...")
		var err error
		client, err = onedrive.NewGraphClient(config.OneDrive.ClientId, config.OneDrive.ClientSecret, config.OneDrive.Tenant, ctx)
		if err != nil {
			return nil, err
		}
	} else {
		log.Println("Existed token found, login...")
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
		log.Printf("Drive info:\n")
		log.Printf("  Drive: %s\n", drive.Id)
		log.Printf("  DriveType: %s\n", drive.DriveType)
		log.Printf("  Total: %d\n", drive.Quota.Total)
		log.Printf("  Used: %d\n", drive.Quota.Used)
		log.Printf("  Remaining: %d\n", drive.Quota.Remaining)
	}
	log.Println("Login successful...")
	return client, nil
}
