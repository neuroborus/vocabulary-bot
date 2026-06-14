package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/logging"
	"github.com/neuroborus/vocabulary-bot/internal/review"
	syncer "github.com/neuroborus/vocabulary-bot/internal/sync"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type SyncRunner interface {
	Run(ctx context.Context) (syncer.Summary, error)
}

type CommandHandler struct {
	notifier                  Notifier
	syncRunner                SyncRunner
	repository                vocabulary.Repository
	logger                    *slog.Logger
	allowedUserID             int64
	reviewChatID              int64
	reviewSpoilerTranslations bool
	logPath                   string
	syncEnabled               bool
	notificationsEnabled      bool
	now                       func() time.Time
}

type CommandHandlerOptions struct {
	Notifier                  Notifier
	SyncRunner                SyncRunner
	Repository                vocabulary.Repository
	Logger                    *slog.Logger
	AllowedUserID             int64
	ReviewChatID              int64
	ReviewSpoilerTranslations bool
	LogPath                   string
	SyncEnabled               bool
	NotificationsEnabled      bool
	Now                       func() time.Time
}

func NewCommandHandler(options CommandHandlerOptions) *CommandHandler {
	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	if options.Now == nil {
		options.Now = time.Now
	}

	return &CommandHandler{
		notifier:                  options.Notifier,
		syncRunner:                options.SyncRunner,
		repository:                options.Repository,
		logger:                    options.Logger,
		allowedUserID:             options.AllowedUserID,
		reviewChatID:              options.ReviewChatID,
		reviewSpoilerTranslations: options.ReviewSpoilerTranslations,
		logPath:                   options.LogPath,
		syncEnabled:               options.SyncEnabled,
		notificationsEnabled:      options.NotificationsEnabled,
		now:                       options.Now,
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

	chatID := message.Chat.ID
	return runWithChatAction(ctx, h.notifier, chatID, commandChatAction(command), func(ctx context.Context) error {
		return h.dispatchCommand(ctx, chatID, command)
	})
}

func commandChatAction(command string) string {
	switch command {
	case CommandListWords, CommandLogs:
		return chatActionUploadDocument
	default:
		return chatActionTyping
	}
}

func (h *CommandHandler) dispatchCommand(ctx context.Context, chatID int64, command string) error {
	switch command {
	case CommandStart:
		return h.sendHTMLMessage(ctx, chatID, formatStartMessage())
	case CommandInfo:
		return h.sendHTMLMessage(ctx, chatID, formatInfoMessage(h.healthText(ctx)))
	case CommandHealth:
		return h.sendHTMLMessage(ctx, chatID, h.healthText(ctx))
	case CommandSync:
		return h.handleSync(ctx, chatID)
	case CommandListWords:
		return h.handleListWords(ctx, chatID)
	case CommandLogs:
		return h.handleLogs(ctx, chatID)
	case CommandTurnOff:
		h.syncEnabled = false
		h.notificationsEnabled = false
		return h.sendHTMLMessage(ctx, chatID, formatNotice("Sync and notifications disabled", ""))
	case CommandTurnOn:
		h.syncEnabled = true
		h.notificationsEnabled = true
		return h.sendHTMLMessage(ctx, chatID, formatNotice("Sync and notifications enabled", ""))
	case CommandPush:
		return h.handlePush(ctx, chatID)
	default:
		return h.sendHTMLMessage(ctx, chatID, formatError("Unknown command", "")+"\n\n"+formatCommandsBlock())
	}
}

func (h *CommandHandler) HandleCallbackQuery(ctx context.Context, query CallbackQuery) error {
	if h.allowedUserID != 0 && query.From.ID != h.allowedUserID {
		h.logger.Warn(
			"telegram callback rejected",
			slog.Int64("from_user_id", query.From.ID),
		)
		return nil
	}

	action, normalizedKey, ok := parseReviewCallback(query.Data)
	if !ok {
		return h.notifier.AnswerCallbackQuery(ctx, query.ID, "Unknown action")
	}
	if h.repository == nil {
		return h.notifier.AnswerCallbackQuery(ctx, query.ID, "Vocabulary repository is not configured")
	}

	item, found, err := h.findItemByNormalizedKey(ctx, normalizedKey)
	if err != nil {
		return h.notifier.AnswerCallbackQuery(ctx, query.ID, "Lookup failed")
	}
	if !found {
		return h.notifier.AnswerCallbackQuery(ctx, query.ID, "Word not found")
	}

	now := h.now().UTC()
	var answer string

	switch action {
	case reviewActionEasy:
		review.MarkEasy(&item, now)
		answer = fmt.Sprintf("Easy. Next review in %d day(s).", item.Review.IntervalDays)
	case reviewActionHard:
		review.MarkHard(&item, now)
		answer = "Hard. Next review tomorrow."
	case reviewActionRemove, "delete":
		review.MarkDisabled(&item, now)
		answer = "Removed from review queue."
	default:
		return h.notifier.AnswerCallbackQuery(ctx, query.ID, "Unknown action")
	}

	item.UpdatedAt = now
	if err := h.repository.Update(ctx, item); err != nil {
		h.logger.Error(
			"review callback update failed",
			slog.String("normalized_key", normalizedKey),
			slog.String("action", action),
			slog.String("error", logging.SanitizeError(err)),
		)
		return h.notifier.AnswerCallbackQuery(ctx, query.ID, "Save failed")
	}

	h.logger.Info(
		"review callback handled",
		slog.String("normalized_key", normalizedKey),
		slog.String("action", action),
	)

	return h.notifier.AnswerCallbackQuery(ctx, query.ID, answer)
}

func (h *CommandHandler) handlePush(ctx context.Context, commandChatID int64) error {
	item, deliveryChatID, err := h.pushReviewWord(ctx, commandChatID)
	if err != nil {
		return h.sendHTMLMessage(ctx, commandChatID, formatError("Push failed", logging.SanitizeError(err)))
	}
	if item == nil {
		return h.sendHTMLMessage(ctx, commandChatID, formatNotice("No review words available", ""))
	}

	if deliveryChatID != commandChatID {
		return h.sendHTMLMessage(
			ctx,
			commandChatID,
			formatNotice("Review word sent to channel", "<code>"+escapeHTML(item.DisplayWord)+"</code>"),
		)
	}

	return nil
}

// RunAutoPush sends one review word when notifications are enabled.
func (h *CommandHandler) RunAutoPush(ctx context.Context) error {
	if !h.notificationsEnabled {
		h.logger.Info("scheduled push skipped because notifications are disabled")
		return nil
	}

	item, _, err := h.pushReviewWord(ctx, h.allowedUserID)
	if err != nil {
		return err
	}
	if item == nil {
		h.logger.Info("scheduled push skipped because no review words are available")
		return nil
	}

	return nil
}

func (h *CommandHandler) pushReviewWord(ctx context.Context, commandChatID int64) (*vocabulary.Item, int64, error) {
	if h.repository == nil {
		return nil, 0, fmt.Errorf("vocabulary repository is not configured")
	}

	items, err := h.repository.List(ctx)
	if err != nil {
		return nil, 0, err
	}

	item, ok := review.SelectNext(items, h.now())
	if !ok {
		return nil, 0, nil
	}

	now := h.now().UTC()
	review.MarkPushed(&item, now)
	item.UpdatedAt = now
	if err := h.repository.Update(ctx, item); err != nil {
		return nil, 0, err
	}

	deliveryChatID := h.reviewDeliveryChatID(commandChatID)
	if err := h.notifier.SendHTMLMessageWithKeyboard(
		ctx,
		deliveryChatID,
		formatReviewReminder(item, h.reviewSpoilerTranslations),
		reviewKeyboard(item.NormalizedKey),
	); err != nil {
		return nil, 0, err
	}

	h.logger.Info(
		"review word pushed",
		slog.String("normalized_key", item.NormalizedKey),
		slog.String("display_word", item.DisplayWord),
		slog.Int64("delivery_chat_id", deliveryChatID),
		slog.Bool("scheduled", commandChatID == h.allowedUserID),
	)

	return &item, deliveryChatID, nil
}

func (h *CommandHandler) reviewDeliveryChatID(commandChatID int64) int64 {
	if h.reviewChatID != 0 {
		return h.reviewChatID
	}

	return commandChatID
}

func (h *CommandHandler) findItemByNormalizedKey(ctx context.Context, normalizedKey string) (vocabulary.Item, bool, error) {
	matches, err := h.repository.FindByLookupKeys(ctx, []string{normalizedKey})
	if err != nil {
		return vocabulary.Item{}, false, err
	}
	for _, item := range matches {
		if item.NormalizedKey == normalizedKey {
			return item, true, nil
		}
	}

	return vocabulary.Item{}, false, nil
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

	return formatHealthMessage(status, wordCount, h.syncEnabled, h.notificationsEnabled)
}

func (h *CommandHandler) handleSync(ctx context.Context, chatID int64) error {
	summary, err := h.syncSources(ctx)
	if err != nil {
		return h.sendHTMLMessage(ctx, chatID, formatError("Sync failed", logging.SanitizeError(err)))
	}
	if summary == nil {
		return h.sendHTMLMessage(ctx, chatID, formatNotice("Sync is disabled", "Use <code>/turn_on</code> first."))
	}

	return h.sendHTMLMessage(ctx, chatID, formatSyncSummary(*summary))
}

// RunAutoSync runs source synchronization when sync is enabled.
func (h *CommandHandler) RunAutoSync(ctx context.Context) error {
	summary, err := h.syncSources(ctx)
	if err != nil {
		return err
	}
	if summary == nil {
		h.logger.Info("scheduled sync skipped because sync is disabled")
		return nil
	}

	h.logger.Info(
		"scheduled sync completed",
		slog.Int("drafts_processed", summary.DraftsProcessed),
		slog.Int("created", summary.Created),
		slog.Int("updated", summary.Updated),
		slog.Int("ambiguous", summary.Ambiguous),
		slog.Int("source_errors", summary.SourceErrors),
	)

	return nil
}

func (h *CommandHandler) syncSources(ctx context.Context) (*syncer.Summary, error) {
	if !h.syncEnabled {
		return nil, nil
	}
	if h.syncRunner == nil {
		return nil, fmt.Errorf("sync service is not configured")
	}

	summary, err := h.syncRunner.Run(ctx)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

func (h *CommandHandler) handleListWords(ctx context.Context, chatID int64) error {
	if h.repository == nil {
		return h.sendHTMLMessage(ctx, chatID, formatError("Vocabulary repository is not configured", ""))
	}

	items, err := h.repository.List(ctx)
	if err != nil {
		return h.sendHTMLMessage(ctx, chatID, formatError("List words failed", logging.SanitizeError(err)))
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

	caption := fmt.Sprintf("📄 Vocabulary export — <b>%d</b> item", len(items))
	if len(items) != 1 {
		caption += "s"
	}
	if err := h.notifier.SendDocument(ctx, chatID, path, caption); err != nil {
		return err
	}

	return nil
}

func (h *CommandHandler) handleLogs(ctx context.Context, chatID int64) error {
	if h.logPath == "" {
		return h.sendHTMLMessage(ctx, chatID, formatError("Log file path is not configured", ""))
	}
	if _, err := os.Stat(h.logPath); err != nil {
		return h.sendHTMLMessage(ctx, chatID, formatNotice("Log file is not available yet", ""))
	}

	return h.notifier.SendDocument(ctx, chatID, h.logPath, "📋 Current log file")
}

func (h *CommandHandler) sendHTMLMessage(ctx context.Context, chatID int64, text string) error {
	return h.notifier.SendHTMLMessage(ctx, chatID, text)
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
