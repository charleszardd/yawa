//go:build windows

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/charleszardd/yawa/internal/agentapp"
	"github.com/charleszardd/yawa/internal/logging"
)

func main() {
	logger, logFile, err := logging.Open()
	if err != nil {
		os.Exit(1)
	}
	defer logFile.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := agentapp.New(logger).Run(ctx); err != nil {
		logger.Error("background agent failed", "error", err)
		os.Exit(1)
	}
}
