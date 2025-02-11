package base_service

import "github.com/rs/zerolog"

func init() {
	config, err := LoadConfig()
	if err != nil {
		LogLevel = zerolog.InfoLevel
	} else {
		LogLevel = zerolog.Level(config.General.LogLevel)
	}
	GlobalConfig = &config
}
