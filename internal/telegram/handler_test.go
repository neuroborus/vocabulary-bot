package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/storage/memory"
	syncer "github.com/neuroborus/vocabulary-bot/internal/sync"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func TestCommandHandlerRejectsUnauthorizedUser(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:      notifier,
		AllowedUserID: 42,
		SyncEnabled:   true,
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
		AllowedUserID:        42,
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
	if !strings.Contains(notifier.messages[0].text, "Words: 1") {
		t.Fatalf("health text = %q", notifier.messages[0].text)
	}
	if notifier.messages[0].chatID != 200 {
		t.Fatalf("chatID = %d, want 200", notifier.messages[0].chatID)
	}
}

func TestCommandHandlerStartAndInfoUseCommandDescriptions(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:      notifier,
		AllowedUserID: 42,
		SyncEnabled:   true,
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
		if !strings.Contains(message.text, CommandInfo+" - ") {
			t.Fatalf("message does not include info description: %q", message.text)
		}
		if !strings.Contains(message.text, CommandListWords+" - ") {
			t.Fatalf("message does not include list_words description: %q", message.text)
		}
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
		Notifier:      notifier,
		SyncRunner:    runner,
		AllowedUserID: 42,
		TargetChatID:  300,
		SyncEnabled:   true,
	})

	if err := handler.HandleMessage(context.Background(), Message{
		From: User{ID: 42},
		Chat: Chat{ID: 200},
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
	if notifier.messages[0].chatID != 300 {
		t.Fatalf("chatID = %d, want target chat 300", notifier.messages[0].chatID)
	}
	if !strings.Contains(notifier.messages[0].text, "Drafts processed: 3") {
		t.Fatalf("sync text = %q", notifier.messages[0].text)
	}
}

func TestBotCommandsAreTelegramMenuCompatible(t *testing.T) {
	t.Parallel()

	commands := BotCommands()
	if len(commands) != len(KnownCommands()) {
		t.Fatalf("BotCommands length = %d, want %d", len(commands), len(KnownCommands()))
	}

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

func TestCommandHandlerTurnOffBlocksSync(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	runner := &fakeSyncRunner{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:      notifier,
		SyncRunner:    runner,
		AllowedUserID: 42,
		SyncEnabled:   true,
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
		Chat: Chat{ID: 200},
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

type fakeSyncRunner struct {
	calls   int
	summary syncer.Summary
}

func (r *fakeSyncRunner) Run(ctx context.Context) (syncer.Summary, error) {
	if err := ctx.Err(); err != nil {
		return syncer.Summary{}, err
	}

	r.calls++
	return r.summary, nil
}

type fakeNotifier struct {
	messages  []fakeMessage
	documents []fakeDocument
}

type fakeMessage struct {
	chatID int64
	text   string
}

type fakeDocument struct {
	chatID  int64
	path    string
	caption string
}

func (n *fakeNotifier) SendMessage(ctx context.Context, chatID int64, text string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	n.messages = append(n.messages, fakeMessage{chatID: chatID, text: text})
	return nil
}

func (n *fakeNotifier) SendDocument(ctx context.Context, chatID int64, path string, caption string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	n.documents = append(n.documents, fakeDocument{chatID: chatID, path: path, caption: caption})
	return nil
}
