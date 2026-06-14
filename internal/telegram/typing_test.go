package telegram

import (
	"context"
	"testing"

	"github.com/neuroborus/vocabulary-bot/internal/storage/memory"
)

func TestCommandHandlerShowsTypingWhileExecuting(t *testing.T) {
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
		Text: CommandHealth,
	}); err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}

	if len(notifier.chatActions) == 0 {
		t.Fatal("expected typing chat action before response")
	}
	if notifier.chatActions[0].action != chatActionTyping {
		t.Fatalf("first chat action = %q, want %q", notifier.chatActions[0].action, chatActionTyping)
	}
	if notifier.chatActions[0].chatID != 200 {
		t.Fatalf("chat action chatID = %d, want 200", notifier.chatActions[0].chatID)
	}
}

func TestCommandHandlerUsesUploadActionForDocumentCommands(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:      notifier,
		Repository:    memory.NewVocabularyRepository(),
		AllowedUserID: 42,
		LogPath:       t.TempDir() + "/missing.log",
	})

	if err := handler.HandleMessage(context.Background(), Message{
		From: User{ID: 42},
		Chat: Chat{ID: 200},
		Text: CommandLogs,
	}); err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}

	if len(notifier.chatActions) == 0 {
		t.Fatal("expected upload chat action before response")
	}
	if notifier.chatActions[0].action != chatActionUploadDocument {
		t.Fatalf("first chat action = %q, want %q", notifier.chatActions[0].action, chatActionUploadDocument)
	}
}
