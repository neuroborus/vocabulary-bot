package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/save"
	"github.com/neuroborus/vocabulary-bot/internal/storage/memory"
	syncer "github.com/neuroborus/vocabulary-bot/internal/sync"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func TestCommandHandlerRejectsUnauthorizedUser(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:    notifier,
		AdminID:     42,
		SyncEnabled: true,
	})

	err := handler.HandleMessage(context.Background(), Message{
		From: User{ID: 100},
		Chat: Chat{ID: 200},
		Text: CommandHealth,
	})
	if err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}
	if len(notifier.messages) != 0 {
		t.Fatalf("messages = %#v, want none", notifier.messages)
	}
}

func TestCommandHandlerHealthReportsWordCount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := memory.NewVocabularyRepository()
	service := vocabulary.NewService(repository, func() time.Time {
		return time.Date(2026, 6, 14, 8, 0, 0, 0, time.UTC)
	})
	if _, err := service.MergeDraft(ctx, vocabulary.Draft{
		Source:  vocabulary.SourceGoogleSheet,
		RawWord: "to decelerate",
	}); err != nil {
		t.Fatalf("seed vocabulary: %v", err)
	}

	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:             notifier,
		Repository:           repository,
		AdminID:              42,
		SyncEnabled:          true,
		NotificationsEnabled: true,
	})

	if err := handler.HandleMessage(ctx, Message{
		From: User{ID: 42},
		Chat: Chat{ID: 200},
		Text: CommandHealth + "@VocabularyBot",
	}); err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}

	if len(notifier.messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(notifier.messages))
	}
	if !strings.Contains(notifier.messages[0].text, "Words: <b>1</b>") {
		t.Fatalf("health text = %q", notifier.messages[0].text)
	}
	if !strings.Contains(notifier.messages[0].text, "Chat ID: <code>200</code>") {
		t.Fatalf("health text = %q, want current chat id", notifier.messages[0].text)
	}
	if !strings.Contains(notifier.messages[0].text, "Caller ID: <code>42</code>") {
		t.Fatalf("health text = %q, want caller id", notifier.messages[0].text)
	}
	if notifier.messages[0].chatID != 200 {
		t.Fatalf("chatID = %d, want 200", notifier.messages[0].chatID)
	}
}

func TestCommandHandlerStartAndInfoUseCommandDescriptions(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:    notifier,
		AdminID:     42,
		SyncEnabled: true,
		Schedule: ScheduleInfo{
			Timezone: "Europe/Kyiv",
			SyncCron: "0 9 * * *",
			PushCron: "0 12-21/2 * * *",
			LogsCron: "0 21 * * 5",
		},
	})

	if err := handler.HandleMessage(context.Background(), Message{
		From: User{ID: 42},
		Chat: Chat{ID: 200},
		Text: CommandStart,
	}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := handler.HandleMessage(context.Background(), Message{
		From: User{ID: 42},
		Chat: Chat{ID: 200},
		Text: CommandInfo,
	}); err != nil {
		t.Fatalf("info: %v", err)
	}

	if len(notifier.messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(notifier.messages))
	}
	for _, message := range notifier.messages {
		if !strings.Contains(message.text, "<code>"+CommandInfo+"</code>") {
			t.Fatalf("message does not include info command: %q", message.text)
		}
		if !strings.Contains(message.text, "<code>"+CommandListWords+"</code>") {
			t.Fatalf("message does not include list_words command: %q", message.text)
		}
	}
	info := notifier.messages[1].text
	if !strings.Contains(info, "<b>Schedule</b>") {
		t.Fatalf("info missing schedule: %q", info)
	}
	if !strings.Contains(info, "Auto sync: <code>0 9 * * *</code>") {
		t.Fatalf("info missing sync cron: %q", info)
	}
}

func TestCommandHandlerSyncUsesRunner(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	runner := &fakeSyncRunner{
		summary: syncer.Summary{
			DraftsProcessed: 3,
			Created:         1,
			Updated:         2,
			Sources: []syncer.SourceSummary{
				{Name: "pocketbook", Drafts: 3},
			},
		},
	}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:    notifier,
		SyncRunner:  runner,
		AdminID:     42,
		SyncEnabled: true,
	})

	if err := handler.HandleMessage(context.Background(), Message{
		From: User{ID: 42},
		Chat: Chat{ID: 42, Type: "private"},
		Text: CommandSync,
	}); err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}

	if runner.calls != 1 {
		t.Fatalf("sync calls = %d, want 1", runner.calls)
	}
	if len(notifier.messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(notifier.messages))
	}
	if notifier.messages[0].chatID != 42 {
		t.Fatalf("chatID = %d, want admin private chat 42", notifier.messages[0].chatID)
	}
	if !strings.Contains(notifier.messages[0].text, "Drafts processed: <b>3</b>") {
		t.Fatalf("sync text = %q", notifier.messages[0].text)
	}
}

func TestCommandHandlerSyncRejectedOutsideAdminPrivateChat(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	runner := &fakeSyncRunner{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:    notifier,
		SyncRunner:  runner,
		AdminID:     42,
		SyncEnabled: true,
	})

	err := handler.HandleMessage(context.Background(), Message{
		From: User{ID: 42},
		Chat: Chat{ID: -100999, Type: "supergroup"},
		Text: CommandSync,
	})
	if err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}
	if runner.calls != 0 {
		t.Fatalf("sync calls = %d, want 0", runner.calls)
	}
	if len(notifier.messages) != 1 {
		t.Fatalf("messages = %d, want 1 rejection notice", len(notifier.messages))
	}
	if !strings.Contains(notifier.messages[0].text, "Admin private chat only") {
		t.Fatalf("message = %q", notifier.messages[0].text)
	}
}

func TestCommandHandlerLogsRejectedOutsideAdminPrivateChat(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier: notifier,
		AdminID:  42,
		LogPath:  t.TempDir() + "/vocabulary.log",
	})

	err := handler.HandleMessage(context.Background(), Message{
		From: User{ID: 42},
		Chat: Chat{ID: 42, Type: "group"},
		Text: CommandLogs,
	})
	if err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}
	if len(notifier.documents) != 0 {
		t.Fatalf("documents = %d, want none", len(notifier.documents))
	}
	if len(notifier.messages) != 1 || !strings.Contains(notifier.messages[0].text, "Admin private chat only") {
		t.Fatalf("messages = %#v, want rejection notice", notifier.messages)
	}
}

func TestCommandHandlerSaveRepliesWithRowSpoiler(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	saver := &fakeVocabularySaver{
		result: save.Result{
			Word:        "teasel",
			Translation: "чесало",
			Context:     "Pat Teasely walked in.",
			RowNumber:   42,
		},
	}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:        notifier,
		AdminID:         42,
		VocabularySaver: saver,
	})

	err := handler.HandleMessage(context.Background(), Message{
		From: User{ID: 42},
		Chat: Chat{ID: 200},
		Text: CommandSave + " teasel",
	})
	if err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}
	if saver.input != "teasel" {
		t.Fatalf("input = %q, want teasel", saver.input)
	}
	if len(notifier.messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(notifier.messages))
	}
	if !strings.Contains(notifier.messages[0].text, "<tg-spoiler>Row: 42</tg-spoiler>") {
		t.Fatalf("message = %q, want row spoiler", notifier.messages[0].text)
	}
	if strings.Contains(notifier.messages[0].text, "Saved to Google Sheets") {
		t.Fatalf("message = %q, want no sheet status", notifier.messages[0].text)
	}
	if !strings.Contains(notifier.messages[0].text, "teasel") {
		t.Fatalf("message = %q", notifier.messages[0].text)
	}
	if !strings.Contains(notifier.messages[0].text, "чесало") {
		t.Fatalf("message = %q, want translation", notifier.messages[0].text)
	}
	if strings.Contains(notifier.messages[0].text, "/sync") {
		t.Fatalf("message = %q, want no sync hint", notifier.messages[0].text)
	}
}

func TestCommandHandlerSaveAcceptsTextBeforeCommand(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	saver := &fakeVocabularySaver{
		result: save.Result{
			Word:      "teasel",
			RowNumber: 42,
		},
	}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:        notifier,
		AdminID:         42,
		VocabularySaver: saver,
	})

	err := handler.HandleMessage(context.Background(), Message{
		From: User{ID: 42},
		Chat: Chat{ID: 200},
		Text: "teasel " + CommandSave,
	})
	if err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}
	if saver.input != "teasel" {
		t.Fatalf("input = %q, want teasel", saver.input)
	}
}

func TestCommandHandlerSaveAllowsNonAdminUser(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	saver := &fakeVocabularySaver{
		result: save.Result{
			Word:      "teasel",
			RowNumber: 42,
		},
	}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:        notifier,
		AdminID:         42,
		VocabularySaver: saver,
	})

	err := handler.HandleMessage(context.Background(), Message{
		From: User{ID: 100},
		Chat: Chat{ID: 200},
		Text: CommandSave + " teasel",
	})
	if err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}
	if saver.input != "teasel" {
		t.Fatalf("input = %q, want teasel", saver.input)
	}
	if len(notifier.messages) != 1 {
		t.Fatalf("messages = %d, want save confirmation", len(notifier.messages))
	}
}

func TestCommandHandlerSaveEmptyInputAsksForReplyOrText(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:        notifier,
		AdminID:         42,
		VocabularySaver: &fakeVocabularySaver{},
	})

	err := handler.HandleMessage(context.Background(), Message{
		From: User{ID: 42},
		Chat: Chat{ID: 200},
		Text: CommandSave,
	})
	if err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}
	if len(notifier.messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(notifier.messages))
	}
	if !strings.Contains(notifier.messages[0].text, "Reply to the message you want to save") {
		t.Fatalf("message = %q, want reply instruction", notifier.messages[0].text)
	}
	if !strings.Contains(notifier.messages[0].text, "before or after") {
		t.Fatalf("message = %q, want before/after instruction", notifier.messages[0].text)
	}
}

func TestBotCommandsAreTelegramMenuCompatible(t *testing.T) {
	t.Parallel()

	commands := BotCommands()
	if len(commands) != len(KnownCommands()) {
		t.Fatalf("BotCommands length = %d, want %d", len(commands), len(KnownCommands()))
	}

	hasSave := false
	for _, command := range commands {
		if strings.HasPrefix(command.Command, "/") {
			t.Fatalf("bot command %q must not include slash", command.Command)
		}
		if strings.Contains(command.Command, "-") {
			t.Fatalf("bot command %q must use snake_case, not kebab-case", command.Command)
		}
		if command.Description == "" {
			t.Fatalf("bot command %q has empty description", command.Command)
		}
		if command.Command == "save" {
			hasSave = true
		}
	}
	if !hasSave {
		t.Fatal("BotCommands() does not include save")
	}
}

func TestClientSetMyCommands(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/botfake-token/setMyCommands" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}

		var payload struct {
			Commands []BotCommand `json:"commands"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if len(payload.Commands) == 0 {
			t.Fatalf("commands payload is empty")
		}
		if payload.Commands[0].Command != "start" {
			t.Fatalf("first command = %q, want start", payload.Commands[0].Command)
		}
		if payload.Commands[0].Description == "" {
			t.Fatalf("first command description is empty")
		}
		if !botCommandsContain(payload.Commands, "save") {
			t.Fatalf("commands payload = %#v, want save", payload.Commands)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		BotToken: "fake-token",
		BaseURL:  server.URL,
	})
	if err := client.SetMyCommands(context.Background(), BotCommands()); err != nil {
		t.Fatalf("SetMyCommands() error = %v", err)
	}
}

func TestClientSetMyCommandsForChat(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/botfake-token/setMyCommands" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}

		var payload struct {
			Commands []BotCommand     `json:"commands"`
			Scope    *BotCommandScope `json:"scope"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if !botCommandsContain(payload.Commands, "save") {
			t.Fatalf("commands payload = %#v, want save", payload.Commands)
		}
		if payload.Scope == nil {
			t.Fatal("scope = nil, want chat scope")
		}
		if payload.Scope.Type != "chat" || payload.Scope.ChatID != 42 {
			t.Fatalf("scope = %#v, want chat 42", payload.Scope)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		BotToken: "fake-token",
		BaseURL:  server.URL,
	})
	if err := client.SetMyCommandsForChat(context.Background(), 42, BotCommands()); err != nil {
		t.Fatalf("SetMyCommandsForChat() error = %v", err)
	}
}

func botCommandsContain(commands []BotCommand, commandName string) bool {
	for _, command := range commands {
		if command.Command == commandName {
			return true
		}
	}
	return false
}

func TestCommandHandlerPushSendsReviewWordToReviewChat(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := memory.NewVocabularyRepository()
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	service := vocabulary.NewService(repository, func() time.Time { return now })
	if _, err := service.MergeDraft(ctx, vocabulary.Draft{
		Source:       vocabulary.SourcePocketBook,
		RawWord:      "to decelerate",
		Translations: []string{"замедляться"},
		Contexts:     []string{"The car began to decelerate."},
	}); err != nil {
		t.Fatalf("seed vocabulary: %v", err)
	}

	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:        notifier,
		Repository:      repository,
		AdminID:         42,
		TargetChannelID: 900,
		Now:             func() time.Time { return now },
	})

	if err := handler.HandleMessage(ctx, Message{
		From: User{ID: 42},
		Chat: Chat{ID: 200},
		Text: CommandPush,
	}); err != nil {
		t.Fatalf("push: %v", err)
	}

	if len(notifier.messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(notifier.messages))
	}
	if notifier.messages[0].chatID != 900 {
		t.Fatalf("review chatID = %d, want 900", notifier.messages[0].chatID)
	}
	if notifier.messages[0].keyboard == nil {
		t.Fatal("push message has no keyboard")
	}
	if notifier.messages[1].chatID != 200 || !strings.Contains(notifier.messages[1].text, "Review word sent to channel") {
		t.Fatalf("confirmation = %#v", notifier.messages[1])
	}

	items, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if items[0].Review.LastPushedAt == nil {
		t.Fatal("LastPushedAt was not updated")
	}
}

func TestCommandHandlerPushDoesNotMarkPushedWhenDeliveryFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := memory.NewVocabularyRepository()
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	service := vocabulary.NewService(repository, func() time.Time { return now })
	if _, err := service.MergeDraft(ctx, vocabulary.Draft{
		Source:       vocabulary.SourcePocketBook,
		RawWord:      "to decelerate",
		Translations: []string{"замедляться"},
	}); err != nil {
		t.Fatalf("seed vocabulary: %v", err)
	}

	notifier := &fakeNotifier{keyboardMessageErr: fmt.Errorf("telegram unavailable")}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:   notifier,
		Repository: repository,
		AdminID:    42,
		Now:        func() time.Time { return now },
	})

	if err := handler.HandleMessage(ctx, Message{
		From: User{ID: 42},
		Chat: Chat{ID: 200},
		Text: CommandPush,
	}); err != nil {
		t.Fatalf("push: %v", err)
	}

	if len(notifier.messages) != 1 {
		t.Fatalf("messages = %d, want 1 error reply", len(notifier.messages))
	}
	if !strings.Contains(notifier.messages[0].text, "Push failed") {
		t.Fatalf("error reply = %q", notifier.messages[0].text)
	}

	items, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if items[0].Review.LastPushedAt != nil {
		t.Fatal("LastPushedAt must stay unset when Telegram delivery fails")
	}
	if items[0].Review.PushCount != 0 {
		t.Fatalf("PushCount = %d, want 0", items[0].Review.PushCount)
	}
}

func TestCommandHandlerPushSendsReviewWord(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := memory.NewVocabularyRepository()
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	service := vocabulary.NewService(repository, func() time.Time { return now })
	if _, err := service.MergeDraft(ctx, vocabulary.Draft{
		Source:       vocabulary.SourcePocketBook,
		RawWord:      "to decelerate",
		Translations: []string{"замедляться"},
		Contexts:     []string{"The car began to decelerate."},
	}); err != nil {
		t.Fatalf("seed vocabulary: %v", err)
	}

	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:   notifier,
		Repository: repository,
		AdminID:    42,
		Now:        func() time.Time { return now },
	})

	if err := handler.HandleMessage(ctx, Message{
		From: User{ID: 42},
		Chat: Chat{ID: 200},
		Text: CommandPush,
	}); err != nil {
		t.Fatalf("push: %v", err)
	}

	if len(notifier.messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(notifier.messages))
	}
	if notifier.messages[0].chatID != 200 {
		t.Fatalf("chatID = %d, want 200", notifier.messages[0].chatID)
	}
	if notifier.messages[0].keyboard == nil {
		t.Fatal("push message has no keyboard")
	}
	if !strings.Contains(notifier.messages[0].text, "to decelerate") {
		t.Fatalf("push text = %q", notifier.messages[0].text)
	}
}

func TestCommandHandlerReviewCallbackEasy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := memory.NewVocabularyRepository()
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	service := vocabulary.NewService(repository, func() time.Time { return now })
	if _, err := service.MergeDraft(ctx, vocabulary.Draft{
		Source:  vocabulary.SourcePocketBook,
		RawWord: "decelerate",
	}); err != nil {
		t.Fatalf("seed vocabulary: %v", err)
	}

	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:   notifier,
		Repository: repository,
		AdminID:    42,
		Now:        func() time.Time { return now },
	})

	if err := handler.HandleCallbackQuery(ctx, CallbackQuery{
		ID:   "cb-1",
		From: User{ID: 42},
		Message: &Message{
			MessageID: 77,
			Chat:      Chat{ID: 900},
		},
		Data: reviewCallbackData(reviewActionEasy, "decelerate"),
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}

	if len(notifier.editedMessages) != 1 {
		t.Fatalf("edited messages = %d, want 1", len(notifier.editedMessages))
	}
	edited := notifier.editedMessages[0]
	if edited.chatID != 900 || edited.messageID != 77 {
		t.Fatalf("edited target = chat %d message %d, want 900/77", edited.chatID, edited.messageID)
	}
	if !strings.Contains(edited.text, "✓ Easy") {
		t.Fatalf("edited text = %q", edited.text)
	}
	if edited.keyboard == nil || len(edited.keyboard.InlineKeyboard) != 0 {
		t.Fatal("edited message should have empty keyboard")
	}

	if len(notifier.callbackResponses) != 1 {
		t.Fatalf("callback responses = %d, want 1", len(notifier.callbackResponses))
	}
	if notifier.callbackResponses[0].text != "" {
		t.Fatalf("callback answer = %q, want empty toast", notifier.callbackResponses[0].text)
	}
	if notifier.callbackResponses[0].showAlert {
		t.Fatal("successful callback should not show an alert")
	}

	items, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if items[0].Review.IntervalDays != 2 {
		t.Fatalf("IntervalDays = %d, want 2", items[0].Review.IntervalDays)
	}
	if items[0].Review.DueAt == nil {
		t.Fatal("DueAt was not set")
	}
}

func TestCommandHandlerReviewCallbackRejectsNonAdmin(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := memory.NewVocabularyRepository()
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	service := vocabulary.NewService(repository, func() time.Time { return now })
	if _, err := service.MergeDraft(ctx, vocabulary.Draft{
		Source:  vocabulary.SourcePocketBook,
		RawWord: "decelerate",
	}); err != nil {
		t.Fatalf("seed vocabulary: %v", err)
	}

	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:   notifier,
		Repository: repository,
		AdminID:    42,
	})

	if err := handler.HandleCallbackQuery(ctx, CallbackQuery{
		ID:   "cb-unauthorized",
		From: User{ID: 99},
		Message: &Message{
			MessageID: 77,
			Chat:      Chat{ID: 900},
		},
		Data: reviewCallbackData(reviewActionEasy, "decelerate"),
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}

	if len(notifier.editedMessages) != 0 {
		t.Fatalf("edited messages = %d, want 0", len(notifier.editedMessages))
	}
	if len(notifier.callbackResponses) != 1 {
		t.Fatalf("callback responses = %d, want 1", len(notifier.callbackResponses))
	}
	response := notifier.callbackResponses[0]
	if response.id != "cb-unauthorized" {
		t.Fatalf("callback id = %q, want cb-unauthorized", response.id)
	}
	if response.text != "You are not authorized to vote." {
		t.Fatalf("callback answer = %q, want unauthorized alert", response.text)
	}
	if !response.showAlert {
		t.Fatal("unauthorized callback should show an alert")
	}

	items, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if items[0].Review.EasyCount != 0 {
		t.Fatalf("EasyCount = %d, want 0", items[0].Review.EasyCount)
	}
	if items[0].Review.DueAt != nil {
		t.Fatalf("DueAt = %v, want nil", items[0].Review.DueAt)
	}
}

func TestSyncSourcesRunsOneAtATime(t *testing.T) {
	t.Parallel()

	runner := &fakeSyncRunner{runDelay: 20 * time.Millisecond}
	handler := NewCommandHandler(CommandHandlerOptions{
		SyncRunner:  runner,
		SyncEnabled: true,
	})

	const workers = 4
	errCh := make(chan error, workers)
	for range workers {
		go func() {
			_, err := handler.syncSources(context.Background())
			errCh <- err
		}()
	}

	for range workers {
		if err := <-errCh; err != nil {
			t.Fatalf("syncSources() error = %v", err)
		}
	}

	if runner.calls != workers {
		t.Fatalf("calls = %d, want %d", runner.calls, workers)
	}
	if runner.peakActive != 1 {
		t.Fatalf("peakActive = %d, want 1", runner.peakActive)
	}
}

func TestCommandHandlerTurnOffBlocksSync(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	runner := &fakeSyncRunner{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:    notifier,
		SyncRunner:  runner,
		AdminID:     42,
		SyncEnabled: true,
	})

	if err := handler.HandleMessage(context.Background(), Message{
		From: User{ID: 42},
		Chat: Chat{ID: 200},
		Text: CommandTurnOff,
	}); err != nil {
		t.Fatalf("turn off: %v", err)
	}
	if err := handler.HandleMessage(context.Background(), Message{
		From: User{ID: 42},
		Chat: Chat{ID: 42, Type: "private"},
		Text: CommandSync,
	}); err != nil {
		t.Fatalf("sync: %v", err)
	}

	if runner.calls != 0 {
		t.Fatalf("sync calls = %d, want 0", runner.calls)
	}
	last := notifier.messages[len(notifier.messages)-1].text
	if !strings.Contains(last, "Sync is disabled") {
		t.Fatalf("last message = %q", last)
	}
}

func TestRunAutoLogsSendsActiveLogAndTruncatesIt(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "vocabulary.log")
	if err := os.WriteFile(logPath, []byte("weekly log line\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:             notifier,
		AdminID:              42,
		LogPath:              logPath,
		NotificationsEnabled: true,
	})

	if err := handler.RunAutoLogs(context.Background()); err != nil {
		t.Fatalf("RunAutoLogs() error = %v", err)
	}

	if len(notifier.documents) != 1 {
		t.Fatalf("documents = %d, want 1", len(notifier.documents))
	}
	if notifier.documents[0].chatID != 42 {
		t.Fatalf("delivery chatID = %d, want 42", notifier.documents[0].chatID)
	}
	if notifier.documents[0].path != logPath {
		t.Fatalf("document path = %q, want %q", notifier.documents[0].path, logPath)
	}

	activeData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(active) error = %v", err)
	}
	if len(activeData) != 0 {
		t.Fatalf("active log after delivery = %q, want empty", string(activeData))
	}
}

func TestRunAutoLogsKeepsActiveLogWhenDeliveryFails(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "vocabulary.log")
	if err := os.WriteFile(logPath, []byte("keep me\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	notifier := &fakeNotifier{documentErr: os.ErrPermission}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:             notifier,
		AdminID:              42,
		LogPath:              logPath,
		NotificationsEnabled: true,
	})

	err := handler.RunAutoLogs(context.Background())
	if err == nil {
		t.Fatal("RunAutoLogs() error = nil, want delivery failure")
	}

	activeData, readErr := os.ReadFile(logPath)
	if readErr != nil {
		t.Fatalf("ReadFile(active) error = %v", readErr)
	}
	if string(activeData) != "keep me\n" {
		t.Fatalf("active log after failure = %q, want original content", string(activeData))
	}
}

func TestRunAutoLogsSkipsWhenNotificationsDisabled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "vocabulary.log")
	if err := os.WriteFile(logPath, []byte("still here\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:             notifier,
		AdminID:              42,
		LogPath:              logPath,
		NotificationsEnabled: false,
	})

	if err := handler.RunAutoLogs(context.Background()); err != nil {
		t.Fatalf("RunAutoLogs() error = %v", err)
	}
	if len(notifier.documents) != 0 {
		t.Fatalf("documents = %d, want none", len(notifier.documents))
	}
}

type fakeVocabularySaver struct {
	input  string
	result save.Result
	err    error
}

func (s *fakeVocabularySaver) SaveFromInput(ctx context.Context, input string) (save.Result, error) {
	s.input = input
	if s.err != nil {
		return save.Result{}, s.err
	}
	return s.result, nil
}

type fakeSyncRunner struct {
	calls      int
	summary    syncer.Summary
	runDelay   time.Duration
	active     int
	peakActive int
	mu         sync.Mutex
}

func (r *fakeSyncRunner) Run(ctx context.Context) (syncer.Summary, error) {
	if err := ctx.Err(); err != nil {
		return syncer.Summary{}, err
	}

	r.mu.Lock()
	r.active++
	if r.active > r.peakActive {
		r.peakActive = r.active
	}
	r.mu.Unlock()

	if r.runDelay > 0 {
		time.Sleep(r.runDelay)
	}

	r.mu.Lock()
	r.calls++
	r.active--
	r.mu.Unlock()

	return r.summary, nil
}

type fakeNotifier struct {
	messages           []fakeMessage
	editedMessages     []fakeEditedMessage
	documents          []fakeDocument
	callbackResponses  []fakeCallbackResponse
	chatActions        []fakeChatAction
	leftChats          []int64
	documentErr        error
	keyboardMessageErr error
}

type fakeChatAction struct {
	chatID int64
	action string
}

type fakeMessage struct {
	chatID   int64
	text     string
	keyboard *InlineKeyboardMarkup
}

type fakeEditedMessage struct {
	chatID    int64
	messageID int
	text      string
	keyboard  *InlineKeyboardMarkup
}

type fakeCallbackResponse struct {
	id        string
	text      string
	showAlert bool
}

type fakeDocument struct {
	chatID  int64
	path    string
	caption string
}

func (n *fakeNotifier) SendMessage(ctx context.Context, chatID int64, text string) error {
	return n.recordMessage(ctx, chatID, text, nil)
}

func (n *fakeNotifier) SendHTMLMessage(ctx context.Context, chatID int64, text string) error {
	return n.recordMessage(ctx, chatID, text, nil)
}

func (n *fakeNotifier) SendHTMLMessageWithKeyboard(ctx context.Context, chatID int64, text string, keyboard InlineKeyboardMarkup) error {
	if n.keyboardMessageErr != nil {
		return n.keyboardMessageErr
	}

	keyboardCopy := keyboard
	return n.recordMessage(ctx, chatID, text, &keyboardCopy)
}

func (n *fakeNotifier) EditHTMLMessage(ctx context.Context, chatID int64, messageID int, text string, keyboard InlineKeyboardMarkup) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	keyboardCopy := keyboard
	n.editedMessages = append(n.editedMessages, fakeEditedMessage{
		chatID:    chatID,
		messageID: messageID,
		text:      text,
		keyboard:  &keyboardCopy,
	})
	return nil
}

func (n *fakeNotifier) AnswerCallbackQuery(ctx context.Context, callbackQueryID string, text string) error {
	return n.recordCallback(ctx, callbackQueryID, text, false)
}

func (n *fakeNotifier) AnswerCallbackAlert(ctx context.Context, callbackQueryID string, text string) error {
	return n.recordCallback(ctx, callbackQueryID, text, true)
}

func (n *fakeNotifier) recordCallback(ctx context.Context, callbackQueryID string, text string, showAlert bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	n.callbackResponses = append(n.callbackResponses, fakeCallbackResponse{
		id:        callbackQueryID,
		text:      text,
		showAlert: showAlert,
	})
	return nil
}

func (n *fakeNotifier) recordMessage(ctx context.Context, chatID int64, text string, keyboard *InlineKeyboardMarkup) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	n.messages = append(n.messages, fakeMessage{chatID: chatID, text: text, keyboard: keyboard})
	return nil
}

func (n *fakeNotifier) SendChatAction(ctx context.Context, chatID int64, action string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	n.chatActions = append(n.chatActions, fakeChatAction{chatID: chatID, action: action})
	return nil
}

func (n *fakeNotifier) SendDocument(ctx context.Context, chatID int64, path string, caption string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n.documentErr != nil {
		return n.documentErr
	}

	n.documents = append(n.documents, fakeDocument{chatID: chatID, path: path, caption: caption})
	return nil
}

func (n *fakeNotifier) LeaveChat(ctx context.Context, chatID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	n.leftChats = append(n.leftChats, chatID)
	return nil
}
