package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/neuroborus/vocabulary-bot/internal/app"
	"github.com/neuroborus/vocabulary-bot/internal/logging"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx); err != nil {
		slog.Error("application failed", slog.String("error", logging.SanitizeError(err)))
		os.Exit(1)
	}
}
