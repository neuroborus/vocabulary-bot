package telegram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neuroborus/vocabulary-bot/internal/save"
)

func TestBotLeavesDisallowedSupergroupMessage(t *testing.T) {
	t.Parallel()

	var leaveCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/botfake-token/leaveChat":
			leaveCalls++
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if r.Form.Get("chat_id") != "-100999" {
				t.Fatalf("chat_id = %q, want -100999", r.Form.Get("chat_id"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		BotToken: "fake-token",
		BaseURL:  server.URL,
	})
	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier: notifier,
		AdminID:  42,
	})
	bot := NewBot(client, handler, NewChatAllowlist([]int64{42, 200}), true, nil)

	err := bot.handleUpdate(context.Background(), Update{
		Message: &Message{
			From: User{ID: 42},
			Chat: Chat{ID: -100999, Type: "supergroup"},
			Text: CommandHealth,
		},
	})
	if err != nil {
		t.Fatalf("handleUpdate() error = %v", err)
	}
	if leaveCalls != 1 {
		t.Fatalf("leaveCalls = %d, want 1", leaveCalls)
	}
	if len(notifier.messages) != 0 {
		t.Fatalf("messages = %#v, want none", notifier.messages)
	}
}

func TestBotLeavesDisallowedChatWhenAdded(t *testing.T) {
	t.Parallel()

	var leaveCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/botfake-token/leaveChat" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		leaveCalls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		BotToken: "fake-token",
		BaseURL:  server.URL,
	})
	bot := NewBot(client, NewCommandHandler(CommandHandlerOptions{}), NewChatAllowlist([]int64{42}), true, nil)

	err := bot.handleUpdate(context.Background(), Update{
		MyChatMember: &ChatMemberUpdated{
			Chat: Chat{ID: -100555, Type: "supergroup"},
			NewChatMember: ChatMember{
				Status: "member",
			},
		},
	})
	if err != nil {
		t.Fatalf("handleUpdate() error = %v", err)
	}
	if leaveCalls != 1 {
		t.Fatalf("leaveCalls = %d, want 1", leaveCalls)
	}
}

func TestBotDoesNotLeaveDisallowedChatWhenDisabled(t *testing.T) {
	t.Parallel()

	var leaveCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/botfake-token/leaveChat" {
			leaveCalls++
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		BotToken: "fake-token",
		BaseURL:  server.URL,
	})
	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier: notifier,
		AdminID:  42,
	})
	bot := NewBot(client, handler, NewChatAllowlist([]int64{42, 200}), false, nil)

	err := bot.handleUpdate(context.Background(), Update{
		Message: &Message{
			From: User{ID: 42},
			Chat: Chat{ID: -100999, Type: "supergroup"},
			Text: CommandHealth,
		},
	})
	if err != nil {
		t.Fatalf("handleUpdate() error = %v", err)
	}
	if leaveCalls != 0 {
		t.Fatalf("leaveCalls = %d, want 0", leaveCalls)
	}
	if len(notifier.messages) != 0 {
		t.Fatalf("messages = %#v, want none", notifier.messages)
	}
}

func TestBotRejectsSaveFromDisallowedChat(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	saver := &fakeVocabularySaver{
		result: save.Result{Word: "teasel"},
	}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier:        notifier,
		AdminID:         42,
		VocabularySaver: saver,
	})
	bot := NewBot(nil, handler, NewChatAllowlist([]int64{42}), true, nil)

	err := bot.handleUpdate(context.Background(), Update{
		Message: &Message{
			From: User{ID: 100},
			Chat: Chat{ID: 200, Type: "private"},
			Text: CommandSave + " teasel",
		},
	})
	if err != nil {
		t.Fatalf("handleUpdate() error = %v", err)
	}
	if saver.input != "" {
		t.Fatalf("input = %q, want rejected before handler", saver.input)
	}
	if len(notifier.messages) != 0 {
		t.Fatalf("messages = %#v, want none", notifier.messages)
	}
}

func TestBotAllowsSaveFromNonAdminInAllowedChat(t *testing.T) {
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
	bot := NewBot(nil, handler, NewChatAllowlist([]int64{200}), true, nil)

	err := bot.handleUpdate(context.Background(), Update{
		Message: &Message{
			From: User{ID: 100},
			Chat: Chat{ID: 200, Type: "private"},
			Text: CommandSave + " teasel",
		},
	})
	if err != nil {
		t.Fatalf("handleUpdate() error = %v", err)
	}
	if saver.input != "teasel" {
		t.Fatalf("input = %q, want teasel", saver.input)
	}
	if len(notifier.messages) != 1 {
		t.Fatalf("messages = %d, want save confirmation", len(notifier.messages))
	}
}

func TestBotAllowsListedChat(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	handler := NewCommandHandler(CommandHandlerOptions{
		Notifier: notifier,
		AdminID:  42,
	})
	bot := NewBot(nil, handler, NewChatAllowlist([]int64{200}), true, nil)

	err := bot.handleUpdate(context.Background(), Update{
		Message: &Message{
			From: User{ID: 42},
			Chat: Chat{ID: 200, Type: "private"},
			Text: CommandStart,
		},
	})
	if err != nil {
		t.Fatalf("handleUpdate() error = %v", err)
	}
	if len(notifier.messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(notifier.messages))
	}
}
