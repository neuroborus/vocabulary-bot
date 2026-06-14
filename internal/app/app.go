package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/config"
	"github.com/neuroborus/vocabulary-bot/internal/logging"
	"github.com/neuroborus/vocabulary-bot/internal/source"
	"github.com/neuroborus/vocabulary-bot/internal/source/pocketbook"
	"github.com/neuroborus/vocabulary-bot/internal/source/spreadsheet"
	"github.com/neuroborus/vocabulary-bot/internal/storage/memory"
	syncer "github.com/neuroborus/vocabulary-bot/internal/sync"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func Run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger, closeLogger, err := logging.NewFileLogger(cfg.LogPath)
	if err != nil {
		return err
	}
	defer closeLogger()

	repository := memory.NewVocabularyRepository()
	vocabularyService := vocabulary.NewService(repository, time.Now)

	sources := buildSources(cfg)
	syncService := syncer.NewService(sources, vocabularyService, logger)

	logger.Info(
		"application initialized",
		slog.String("env", cfg.AppEnv),
		slog.Bool("sync_enabled", cfg.SyncEnabled),
		slog.Bool("notifications_enabled", cfg.NotificationsEnabled),
		slog.Int("sources", len(sources)),
	)

	if !cfg.SyncEnabled {
		logger.Info("sync skipped because global sync is disabled")
		return nil
	}

	summary, err := syncService.Run(ctx)
	if err != nil {
		return err
	}

	logger.Info(
		"sync completed",
		slog.Int("drafts_processed", summary.DraftsProcessed),
		slog.Int("created", summary.Created),
		slog.Int("updated", summary.Updated),
		slog.Int("ambiguous", summary.Ambiguous),
		slog.Int("source_errors", summary.SourceErrors),
	)

	return nil
}

func buildSources(cfg config.Config) []source.Adapter {
	adapters := make([]source.Adapter, 0, 2)

	if cfg.PocketBook.Enabled {
		adapters = append(adapters, pocketbook.NewAdapter())
	}

	if cfg.GoogleSheet.Enabled {
		adapters = append(adapters, spreadsheet.NewAdapter(cfg.GoogleSheet.SheetName))
	}

	return adapters
}
