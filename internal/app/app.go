package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/config"
	"github.com/neuroborus/vocabulary-bot/internal/logging"
	"github.com/neuroborus/vocabulary-bot/internal/schedule"
	"github.com/neuroborus/vocabulary-bot/internal/source"
	"github.com/neuroborus/vocabulary-bot/internal/source/pocketbook"
	"github.com/neuroborus/vocabulary-bot/internal/source/spreadsheet"
	"github.com/neuroborus/vocabulary-bot/internal/storage/memory"
	mongostorage "github.com/neuroborus/vocabulary-bot/internal/storage/mongo"
	syncer "github.com/neuroborus/vocabulary-bot/internal/sync"
	"github.com/neuroborus/vocabulary-bot/internal/telegram"
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

	repository, sessionStore, closeStorage, err := buildStorage(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer closeStorage()

	vocabularyService := vocabulary.NewService(repository, time.Now)

	sources := buildSources(cfg, logger, sessionStore)
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
	} else {
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
	}

	if shouldRunTelegram(cfg) {
		if err := runTelegram(ctx, cfg, logger, repository, syncService); err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
	}

	return nil
}

func buildStorage(ctx context.Context, cfg config.Config, logger *slog.Logger) (vocabulary.Repository, pocketbook.SessionStore, func(), error) {
	sessionStore := pocketbook.SessionStore(pocketbook.NewFileSessionStore(cfg.PocketBook.TokenPath))

	if cfg.MongoDB.URI == "" {
		return memory.NewVocabularyRepository(), sessionStore, func() {}, nil
	}

	client, err := mongostorage.Connect(ctx, cfg.MongoDB.URI)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("connect mongodb: %w", err)
	}

	database := client.Database(cfg.MongoDB.DBName)
	repository := mongostorage.NewVocabularyRepository(database)
	if err := repository.EnsureIndexes(ctx); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, nil, nil, fmt.Errorf("ensure mongodb indexes: %w", err)
	}

	logger.Info(
		"mongodb storage initialized",
		slog.String("database", cfg.MongoDB.DBName),
		slog.String("vocabulary_collection", mongostorage.VocabularyCollectionName),
		slog.String("pocketbook_sessions_collection", mongostorage.PocketBookSessionsCollectionName),
	)

	closeStorage := func() {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.Disconnect(disconnectCtx); err != nil {
			logger.Error("disconnect mongodb failed", slog.String("error", logging.SanitizeError(err)))
		}
	}

	return repository, mongostorage.NewPocketBookSessionStore(database), closeStorage, nil
}

func buildSources(cfg config.Config, logger *slog.Logger, sessionStore pocketbook.SessionStore) []source.Adapter {
	adapters := make([]source.Adapter, 0, 2)

	if cfg.PocketBook.Enabled {
		adapters = append(adapters, pocketbook.NewAdapter(pocketbook.AdapterOptions{
			BaseURL:            cfg.PocketBook.BaseURL,
			Email:              cfg.PocketBook.Email,
			Password:           cfg.PocketBook.Password,
			RefreshToken:       cfg.PocketBook.RefreshToken,
			ShopName:           cfg.PocketBook.ShopName,
			SessionStore:       sessionStore,
			Logger:             logger,
			BookContextEnabled: cfg.PocketBook.BookContextEnabled,
		}))
	}

	if cfg.GoogleSheet.Enabled {
		adapters = append(adapters, spreadsheet.NewAdapter(cfg.GoogleSheet.SheetName))
	}

	return adapters
}

func shouldRunTelegram(cfg config.Config) bool {
	return cfg.Telegram.BotToken != "" && cfg.Telegram.PollingEnabled
}

func runTelegram(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
	repository vocabulary.Repository,
	syncService *syncer.Service,
) error {
	if cfg.Telegram.AllowedUserID == 0 {
		return errors.New("TELEGRAM_ALLOWED_USER_ID is required when Telegram polling is enabled")
	}

	client := telegram.NewClient(telegram.ClientOptions{
		BotToken: cfg.Telegram.BotToken,
		BaseURL:  cfg.Telegram.APIBaseURL,
	})
	handler := telegram.NewCommandHandler(telegram.CommandHandlerOptions{
		Notifier:                  client,
		SyncRunner:                syncService,
		Repository:                repository,
		Logger:                    logger,
		AllowedUserID:             cfg.Telegram.AllowedUserID,
		ReviewChatID:              cfg.Telegram.TargetChatID,
		ReviewSpoilerTranslations: cfg.Telegram.ReviewSpoilerTranslations,
		LogPath:                   cfg.LogPath,
		SyncEnabled:               cfg.SyncEnabled,
		NotificationsEnabled:      cfg.NotificationsEnabled,
	})
	bot := telegram.NewBot(client, handler, logger)

	if err := client.SetMyCommands(ctx, telegram.BotCommands()); err != nil {
		logger.Error("telegram command menu setup failed", slog.String("error", logging.SanitizeError(err)))
	}

	serviceNotifier := telegram.NewServiceNotifier(client, cfg.Telegram.AllowedUserID)
	if err := serviceNotifier.Notify(ctx, telegram.StartupMessage()); err != nil {
		logger.Error(
			"telegram startup notification failed",
			slog.Int64("chat_id", cfg.Telegram.AllowedUserID),
			slog.String("error", logging.SanitizeError(err)),
		)
	}

	schedCtx, schedCancel := context.WithCancel(ctx)
	defer schedCancel()
	if err := startScheduler(schedCtx, cfg, handler, logger); err != nil {
		return err
	}

	logger.Info("telegram polling started")
	return bot.Poll(ctx)
}

func startScheduler(ctx context.Context, cfg config.Config, handler *telegram.CommandHandler, logger *slog.Logger) error {
	jobs, err := schedule.JobsFromConfig(cfg.Schedule, handler, handler)
	if err != nil {
		return fmt.Errorf("build schedule jobs: %w", err)
	}
	if len(jobs) == 0 {
		logger.Info("scheduler disabled because no cron jobs are configured")
		return nil
	}

	runner, err := schedule.NewRunner(schedule.RunnerOptions{
		Timezone: cfg.Schedule.Timezone,
		Jobs:     jobs,
		Logger:   logger,
	})
	if err != nil {
		return err
	}

	logger.Info(
		"scheduler started",
		slog.String("timezone", cfg.Schedule.Timezone),
		slog.String("auto_sync_cron", cfg.Schedule.AutoSyncCron),
		slog.String("auto_push_cron", cfg.Schedule.AutoPushCron),
		slog.Int("jobs", len(jobs)),
	)

	go runner.Run(ctx)
	return nil
}
