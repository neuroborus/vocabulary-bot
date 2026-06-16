package telegram

import (
	"context"
	"testing"
)

func TestServiceNotifierUsesAdminID(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	serviceNotifier := NewServiceNotifier(notifier, 490734700)

	if err := serviceNotifier.Notify(context.Background(), StartupMessage()); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
	if len(notifier.messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(notifier.messages))
	}
	if notifier.messages[0].chatID != 490734700 {
		t.Fatalf("chatID = %d, want allowed user chat 490734700", notifier.messages[0].chatID)
	}
}
