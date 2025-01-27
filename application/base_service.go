package application

import (
	"context"
	"fmt"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/pelletier/go-toml/v2"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const ConfigPath = "./config.toml"

func CreateLog() (*os.File, error) {
	file, err := os.OpenFile(fmt.Sprintf("log-%s.txt", time.Now().Format(time.DateOnly)), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	multiWriter := io.MultiWriter(os.Stderr, file)
	log.SetOutput(multiWriter)
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

func CreateSignalCancelContext() context.Context {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-signalChan
		log.Println("\nReceived interrupt signal. Canceling tasks...")
		cancel()
	}()
	return ctx
}
