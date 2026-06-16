package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/logging"
	"github.com/neuroborus/vocabulary-bot/internal/review"
	"github.com/neuroborus/vocabulary-bot/internal/save"
	syncer "github.com/neuroborus/vocabulary-bot/internal/sync"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type SyncRunner interface {
	Run(ctx context.Context) (syncer.Summary, error)
}

type VocabularySaver interface {
	SaveFromInput(ctx context.Context, input string) (save.Result, error)
}

type CommandHandler struct {
	notifier                  Notifier
	syncRunner                SyncRunner
	repository                vocabulary.Repository
	logger                    *slog.Logger
	adminID                   int64
	targetChannelID           int64
	reviewSpoilerTranslations bool
	logPath                   string
	stateMu                   sync.RWMutex
	syncMu                    sync.Mutex
	syncEnabled               bool
	notificationsEnabled      bool
	reviewSelection           review.SelectionOptions
	vocabularySaver           VocabularySaver
	now                       func() time.Time
}

type CommandHandlerOptions struct {
	Notifier                  Notifier
	SyncRunner                SyncRunner
	Repository                vocabulary.Repository
	Logger                    *slog.Logger
	AdminID                   int64
	TargetChannelID           int64
	ReviewSpoilerTranslations bool
	LogPath                   string
	SyncEnabled               bool
	NotificationsEnabled      bool
	ReviewSelection           review.SelectionOptions
	VocabularySaver           VocabularySaver
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
		adminID:                   options.AdminID,
		targetChannelID:           options.TargetChannelID,
		reviewSpoilerTranslations: options.ReviewSpoilerTranslations,
		logPath:                   options.LogPath,
		syncEnabled:               options.SyncEnabled,
		notificationsEnabled:      options.NotificationsEnabled,
		reviewSelection:           options.ReviewSelection,
		vocabularySaver:           options.VocabularySaver,
		now:                       options.Now,
	}
}

func (h *CommandHandler) HandleMessage(ctx context.Context, message Message) error {
	if strings.TrimSpace(message.Text) == "" {
		return nil
	}

	command := parseCommand(message.Text)
	if command == "" {
		return nil
	}

	if h.adminID != 0 && message.From.ID != h.adminID && commandRequiresAdminUser(command) {
		h.logger.Warn(
			"telegram command rejected",
			slog.String("command", command),
			slog.Int64("from_user_id", message.From.ID),
			slog.Int64("chat_id", message.Chat.ID),
		)
		return nil
	}

	chatID := message.Chat.ID
	if commandRequiresAdminPrivateChat(command) && !h.isAdminPrivateChat(message.Chat) {
		h.logger.Warn(
			"telegram command rejected outside admin private chat",
			slog.String("command", command),
			slog.Int64("chat_id", chatID),
			slog.String("chat_type", message.Chat.Type),
		)
		return h.sendHTMLMessage(
			ctx,
			chatID,
			formatNotice(
				"Admin private chat only",
				"<code>"+escapeHTML(command)+"</code> is available only in a private chat with the bot.",
			),
		)
	}

	return runWithChatAction(ctx, h.notifier, chatID, commandChatAction(command), func(ctx context.Context) error {
		return h.dispatchCommand(ctx, chatID, message.From.ID, command, message)
	})
}

func commandRequiresAdminUser(command string) bool {
	return command != CommandSave
}

func commandRequiresAdminPrivateChat(command string) bool {
	switch command {
	case CommandSync, CommandLogs:
		return true
	default:
		return false
	}
}

func (h *CommandHandler) isAdminPrivateChat(chat Chat) bool {
	return h.adminID != 0 && chat.ID == h.adminID && chat.Type == "private"
}

func commandChatAction(command string) string {
	switch command {
	case CommandListWords, CommandLogs:
		return chatActionUploadDocument
	default:
		return chatActionTyping
	}
}

func (h *CommandHandler) dispatchCommand(ctx context.Context, chatID, callerID int64, command string, message Message) error {
	switch command {
	case CommandStart:
		return h.sendHTMLMessage(ctx, chatID, formatStartMessage())
	case CommandInfo:
		return h.sendHTMLMessage(ctx, chatID, formatInfoMessage(h.healthText(ctx, 0, 0)))
	case CommandHealth:
		return h.sendHTMLMessage(ctx, chatID, h.healthText(ctx, chatID, callerID))
	case CommandSync:
		return h.handleSync(ctx, chatID)
	case CommandListWords:
		return h.handleListWords(ctx, chatID)
	case CommandLogs:
		return h.handleLogs(ctx, chatID)
	case CommandTurnOff:
		h.setRuntimeFlags(false, false)
		return h.sendHTMLMessage(ctx, chatID, formatNotice("Sync and notifications disabled", ""))
	case CommandTurnOn:
		h.setRuntimeFlags(true, true)
		return h.sendHTMLMessage(ctx, chatID, formatNotice("Sync and notifications enabled", ""))
	case CommandPush:
		return h.handlePush(ctx, chatID)
	case CommandSave:
		return h.handleSave(ctx, chatID, message)
	default:
		return h.sendHTMLMessage(ctx, chatID, formatError("Unknown command", "")+"\n\n"+formatCommandsBlock())
	}
}

func (h *CommandHandler) handleSave(ctx context.Context, chatID int64, message Message) error {
	input, err := extractSaveInput(message)
	if err != nil {
		return h.sendHTMLMessage(ctx, chatID, formatSaveInputRequired(message))
	}

	if h.vocabularySaver == nil {
		return h.sendHTMLMessage(ctx, chatID, formatError("Save is not configured", "Set OPENAI_API_KEY and Google Sheets credentials."))
	}

	result, err := h.vocabularySaver.SaveFromInput(ctx, input)
	if err != nil {
		return h.sendHTMLMessage(ctx, chatID, formatError("Save failed", logging.SanitizeError(err)))
	}

	h.logger.Info(
		"vocabulary appended to spreadsheet from telegram",
		slog.String("word", result.Word),
		slog.Int("row_number", result.RowNumber),
		slog.Int64("chat_id", chatID),
	)

	return h.sendHTMLMessage(ctx, chatID, formatSaveConfirmation(result))
}

func (h *CommandHandler) HandleCallbackQuery(ctx context.Context, query CallbackQuery) error {
	if h.adminID != 0 && query.From.ID != h.adminID {
		h.logger.Warn(
			"telegram callback rejected",
			slog.Int64("from_user_id", query.From.ID),
		)
		return nil
	}

	action, token, ok := parseReviewCallback(query.Data)
	if !ok {
		return h.notifier.AnswerCallbackQuery(ctx, query.ID, "Unknown action")
	}
	if h.repository == nil {
		return h.notifier.AnswerCallbackQuery(ctx, query.ID, "Vocabulary repository is not configured")
	}

	item, found, err := h.findItemByReviewToken(ctx, token)
	if err != nil {
		return h.notifier.AnswerCallbackQuery(ctx, query.ID, "Lookup failed")
	}
	if !found {
		return h.notifier.AnswerCallbackQuery(ctx, query.ID, "Word not found")
	}

	normalizedKey := item.NormalizedKey

	now := h.now().UTC()

	switch action {
	case reviewActionEasy:
		review.MarkEasy(&item, now)
	case reviewActionHard:
		review.MarkHard(&item, now)
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

	if query.Message != nil {
		if err := h.notifier.EditHTMLMessage(
			ctx,
			query.Message.Chat.ID,
			query.Message.MessageID,
			formatReviewAnswered(item, action, h.reviewSpoilerTranslations),
			emptyInlineKeyboard(),
		); err != nil {
			h.logger.Error(
				"review callback message edit failed",
				slog.Int64("chat_id", query.Message.Chat.ID),
				slog.Int("message_id", query.Message.MessageID),
				slog.String("normalized_key", normalizedKey),
				slog.String("action", action),
				slog.String("error", logging.SanitizeError(err)),
			)
			return h.notifier.AnswerCallbackQuery(ctx, query.ID, "Update failed")
		}
	}

	h.logger.Info(
		"review callback handled",
		slog.String("normalized_key", normalizedKey),
		slog.String("action", action),
	)

	return h.notifier.AnswerCallbackQuery(ctx, query.ID, "")
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
	if !h.notificationsEnabledState() {
		h.logger.Info("scheduled push skipped because notifications are disabled")
		return nil
	}

	item, _, err := h.pushReviewWord(ctx, h.adminID)
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

	item, ok := review.SelectNext(items, h.now(), h.reviewSelection)
	if !ok {
		return nil, 0, nil
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

	now := h.now().UTC()
	review.MarkPushed(&item, now)
	item.UpdatedAt = now
	if err := h.repository.Update(ctx, item); err != nil {
		return nil, 0, err
	}

	h.logger.Info(
		"review word pushed",
		slog.String("normalized_key", item.NormalizedKey),
		slog.String("display_word", item.DisplayWord),
		slog.Int64("delivery_chat_id", deliveryChatID),
		slog.Bool("scheduled", commandChatID == h.adminID),
	)

	return &item, deliveryChatID, nil
}

func (h *CommandHandler) reviewDeliveryChatID(commandChatID int64) int64 {
	if h.targetChannelID != 0 {
		return h.targetChannelID
	}

	return commandChatID
}

func (h *CommandHandler) findItemByReviewToken(ctx context.Context, token string) (vocabulary.Item, bool, error) {
	items, err := h.repository.List(ctx)
	if err != nil {
		return vocabulary.Item{}, false, err
	}

	var match vocabulary.Item
	found := false

	for _, item := range items {
		if reviewCallbackToken(item.NormalizedKey) != token {
			continue
		}
		if found {
			return vocabulary.Item{}, false, fmt.Errorf("ambiguous review callback token %q", token)
		}
		match = item
		found = true
	}

	return match, found, nil
}

func (h *CommandHandler) healthText(ctx context.Context, chatID, callerID int64) string {
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

	return formatHealthMessage(status, wordCount, h.syncEnabledState(), h.notificationsEnabledState(), chatID, callerID)
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

func (h *CommandHandler) setRuntimeFlags(syncEnabled, notificationsEnabled bool) {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	h.syncEnabled = syncEnabled
	h.notificationsEnabled = notificationsEnabled
}

func (h *CommandHandler) syncEnabledState() bool {
	h.stateMu.RLock()
	defer h.stateMu.RUnlock()
	return h.syncEnabled
}

func (h *CommandHandler) notificationsEnabledState() bool {
	h.stateMu.RLock()
	defer h.stateMu.RUnlock()
	return h.notificationsEnabled
}

func (h *CommandHandler) syncSources(ctx context.Context) (*syncer.Summary, error) {
	if !h.syncEnabledState() {
		return nil, nil
	}
	if h.syncRunner == nil {
		return nil, fmt.Errorf("sync service is not configured")
	}

	h.syncMu.Lock()
	defer h.syncMu.Unlock()

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

// RunAutoLogs sends the active log file to the allowed admin user and truncates
// it only after Telegram delivery succeeds.
func (h *CommandHandler) RunAutoLogs(ctx context.Context) error {
	if !h.notificationsEnabledState() {
		h.logger.Info("scheduled log delivery skipped because notifications are disabled")
		return nil
	}
	if h.logPath == "" {
		return fmt.Errorf("log file path is not configured")
	}
	if h.adminID == 0 {
		return fmt.Errorf("telegram admin is not configured")
	}

	info, err := os.Stat(h.logPath)
	if err != nil {
		if os.IsNotExist(err) {
			h.logger.Info("scheduled log delivery skipped because log file does not exist")
			return nil
		}
		return err
	}
	if info.Size() == 0 {
		h.logger.Info("scheduled log delivery skipped because log file is empty")
		return nil
	}

	if err := h.notifier.SendDocument(ctx, h.adminID, h.logPath, "📋 Weekly log export"); err != nil {
		h.logger.Error(
			"scheduled log delivery failed",
			slog.String("error", logging.SanitizeError(err)),
			slog.String("log_path", h.logPath),
		)
		return err
	}

	if err := logging.TruncateFile(h.logPath); err != nil {
		return err
	}

	h.logger.Info(
		"scheduled log delivery completed",
		slog.String("log_path", h.logPath),
		slog.Int64("delivery_chat_id", h.adminID),
	)

	return nil
}

func (h *CommandHandler) sendHTMLMessage(ctx context.Context, chatID int64, text string) error {
	return h.notifier.SendHTMLMessage(ctx, chatID, text)
}

func parseCommand(text string) string {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return ""
	}

	if strings.HasPrefix(fields[0], "/") {
		command := fields[0]
		if index := strings.Index(command, "@"); index >= 0 {
			command = command[:index]
		}

		return command
	}

	for _, field := range fields[1:] {
		if commandTokenMatches(field, CommandSave) {
			return CommandSave
		}
	}

	return ""
}
