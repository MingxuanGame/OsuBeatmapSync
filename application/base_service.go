package application

import (
	"context"
	"fmt"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const ConfigPath = "./config.toml"

func CreateLog() (*os.File, error) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	file, err := os.OpenFile(fmt.Sprintf("log-%s.txt", time.Now().Format(time.DateOnly)), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.DateTime, NoColor: false}
	fileWriter := zerolog.ConsoleWriter{Out: file, TimeFormat: time.DateTime, NoColor: true}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}
	log.Logger = zerolog.New(zerolog.MultiLevelWriter(output, fileWriter)).With().Timestamp().Logger()
	return file, nil
}

func LoadConfig() (Config, error) {
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

func CreateSignalCancelContext() context.Context {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-signalChan
		log.Info().Msg("\nReceived interrupt signal. Canceling tasks...")
		cancel()
	}()
	return ctx
}
