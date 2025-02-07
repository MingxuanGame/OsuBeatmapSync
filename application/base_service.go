package application

import (
	"context"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"syscall"
)

var logger = log.With().Str("module", "application").Logger()

func CreateSignalCancelContext() context.Context {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-signalChan
		logger.Info().Msg("Received interrupt signal. Canceling tasks...")
		cancel()
	}()
	return ctx
}
