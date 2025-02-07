package base_service

import (
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/pelletier/go-toml/v2"
	"os"
)

const ConfigPath = "./config.toml"

var GlobalConfig *Config

func LoadConfig() (Config, error) {
	if GlobalConfig == nil {
		return LoadConfigFromFile()
	}
	return *GlobalConfig, nil
}

func LoadConfigFromFile() (Config, error) {
	content, err := os.ReadFile(ConfigPath)
	if err != nil {
		return Config{}, err
	}

	config := Config{}
	err = toml.Unmarshal(content, &config)
	if err != nil {
		return Config{}, err
	}
	return config, nil
}

func SaveConfig(config *Config) error {
	content, err := toml.Marshal(config)
	if err != nil {
		return err
	}
	err = os.WriteFile(ConfigPath, content, 0644)
	if err != nil {
		return err
	}
	return nil
}
