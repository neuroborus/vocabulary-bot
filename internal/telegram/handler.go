package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/neuroborus/vocabulary-bot/internal/logging"
	syncer "github.com/neuroborus/vocabulary-bot/internal/sync"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type SyncRunner interface {
	Run(ctx context.Context) (syncer.Summary, error)
}

type CommandHandler struct {
	notifier             Notifier
	syncRunner           SyncRunner
	repository           vocabulary.Repository
	logger               *slog.Logger
	allowedUserID        int64
	targetChatID         int64
	logPath              string
	syncEnabled          bool
	notificationsEnabled bool
}

type CommandHandlerOptions struct {
	Notifier             Notifier
	SyncRunner           SyncRunner
	Repository           vocabulary.Repository
	Logger               *slog.Logger
	AllowedUserID        int64
	TargetChatID         int64
	LogPath              string
	SyncEnabled          bool
	NotificationsEnabled bool
}

func NewCommandHandler(options CommandHandlerOptions) *CommandHandler {
	if options.Logger == nil {
		options.Logger = slog.Default()
	}

	return &CommandHandler{
		notifier:             options.Notifier,
		syncRunner:           options.SyncRunner,
		repository:           options.Repository,
		logger:               options.Logger,
		allowedUserID:        options.AllowedUserID,
		targetChatID:         options.TargetChatID,
		logPath:              options.LogPath,
		syncEnabled:          options.SyncEnabled,
		notificationsEnabled: options.NotificationsEnabled,
	}
}

func (h *CommandHandler) HandleMessage(ctx context.Context, message Message) error {
	if strings.TrimSpace(message.Text) == "" {
		return nil
	}
	if h.allowedUserID != 0 && message.From.ID != h.allowedUserID {
		h.logger.Warn(
			"telegram message rejected",
			slog.Int64("from_user_id", message.From.ID),
			slog.Int64("chat_id", message.Chat.ID),
		)
		return nil
	}

	command := parseCommand(message.Text)
	if command == "" {
		return nil
	}

	chatID := h.replyChatID(message)
	switch command {
	case CommandHealth:
		return h.sendMessage(ctx, chatID, h.healthText(ctx))
	case CommandSync:
		return h.handleSync(ctx, chatID)
	case CommandListWords:
		return h.handleListWords(ctx, chatID)
	case CommandLogs:
		return h.handleLogs(ctx, chatID)
	case CommandTurnOff:
		h.syncEnabled = false
		h.notificationsEnabled = false
		return h.sendMessage(ctx, chatID, "Sync and notifications disabled.")
	case CommandTurnOn:
		h.syncEnabled = true
		h.notificationsEnabled = true
		return h.sendMessage(ctx, chatID, "Sync and notifications enabled.")
	case CommandPush:
		return h.sendMessage(ctx, chatID, "Review push is not implemented yet.")
	default:
		return h.sendMessage(ctx, chatID, "Unknown command.\n\n"+knownCommandsText())
	}
}

func (h *CommandHandler) healthText(ctx context.Context) string {
	wordCount := 0
	status := "ok"

	if h.repository != nil {
		items, err := h.repository.List(ctx)
		if err != nil {
			status = "degraded"
			h.logger.Error("telegram health word count failed", slog.String("error", logging.SanitizeError(err)))
		} else {
			wordCount = len(items)
		}
	}

	return fmt.Sprintf(
		"Health: %s\nWords: %d\nSync: %t\nNotifications: %t",
		status,
		wordCount,
		h.syncEnabled,
		h.notificationsEnabled,
	)
}

func (h *CommandHandler) handleSync(ctx context.Context, chatID int64) error {
	if !h.syncEnabled {
		return h.sendMessage(ctx, chatID, "Sync is disabled. Use /turn-on first.")
	}
	if h.syncRunner == nil {
		return h.sendMessage(ctx, chatID, "Sync service is not configured.")
	}

	summary, err := h.syncRunner.Run(ctx)
	if err != nil {
		return h.sendMessage(ctx, chatID, "Sync failed: "+logging.SanitizeError(err))
	}

	return h.sendMessage(ctx, chatID, formatSyncSummary(summary))
}

func (h *CommandHandler) handleListWords(ctx context.Context, chatID int64) error {
	if h.repository == nil {
		return h.sendMessage(ctx, chatID, "Vocabulary repository is not configured.")
	}

	items, err := h.repository.List(ctx)
	if err != nil {
		return h.sendMessage(ctx, chatID, "List words failed: "+logging.SanitizeError(err))
	}

	file, err := os.CreateTemp("", "vocabulary-*.json")
	if err != nil {
		return err
	}
	path := file.Name()
	removeFile := true
	defer func() {
		if removeFile {
			_ = os.Remove(path)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(items); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	if err := h.notifier.SendDocument(ctx, chatID, path, "Vocabulary export"); err != nil {
		return err
	}

	return nil
}

func (h *CommandHandler) handleLogs(ctx context.Context, chatID int64) error {
	if h.logPath == "" {
		return h.sendMessage(ctx, chatID, "Log file path is not configured.")
	}
	if _, err := os.Stat(h.logPath); err != nil {
		return h.sendMessage(ctx, chatID, "Log file is not available yet.")
	}

	return h.notifier.SendDocument(ctx, chatID, h.logPath, "Current log file")
}

func (h *CommandHandler) replyChatID(message Message) int64 {
	if h.targetChatID != 0 {
		return h.targetChatID
	}

	return message.Chat.ID
}

func (h *CommandHandler) sendMessage(ctx context.Context, chatID int64, text string) error {
	return h.notifier.SendMessage(ctx, chatID, text)
}

func parseCommand(text string) string {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
		return ""
	}

	command := fields[0]
	if index := strings.Index(command, "@"); index >= 0 {
		command = command[:index]
	}

	return command
}

func formatSyncSummary(summary syncer.Summary) string {
	var builder strings.Builder
	builder.WriteString("Sync completed.\n")
	builder.WriteString(fmt.Sprintf("Drafts processed: %d\n", summary.DraftsProcessed))
	builder.WriteString(fmt.Sprintf("Created: %d\n", summary.Created))
	builder.WriteString(fmt.Sprintf("Updated: %d\n", summary.Updated))
	builder.WriteString(fmt.Sprintf("Ambiguous: %d\n", summary.Ambiguous))
	builder.WriteString(fmt.Sprintf("Source errors: %d", summary.SourceErrors))

	for _, source := range summary.Sources {
		builder.WriteString(fmt.Sprintf("\n%s: %d drafts", source.Name, source.Drafts))
		if source.Error != "" {
			builder.WriteString(" error: " + source.Error)
		}
	}

	return builder.String()
}

func knownCommandsText() string {
	return "Known commands:\n" + strings.Join(KnownCommands(), "\n")
}
