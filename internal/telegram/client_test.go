package telegram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientSendChatAction(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/botfake-token/sendChatAction" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("chat_id") != "42" {
			t.Fatalf("chat_id = %q, want 42", r.Form.Get("chat_id"))
		}
		if r.Form.Get("action") != chatActionTyping {
			t.Fatalf("action = %q, want %q", r.Form.Get("action"), chatActionTyping)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		BotToken: "fake-token",
		BaseURL:  server.URL,
	})
	if err := client.SendChatAction(context.Background(), 42, chatActionTyping); err != nil {
		t.Fatalf("SendChatAction() error = %v", err)
	}
}

func TestClientSendHTMLMessageSetsParseMode(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/botfake-token/sendMessage" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("parse_mode") != parseModeHTML {
			t.Fatalf("parse_mode = %q, want %q", r.Form.Get("parse_mode"), parseModeHTML)
		}
		if !strings.Contains(r.Form.Get("text"), "<b>Health</b>") {
			t.Fatalf("text = %q", r.Form.Get("text"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		BotToken: "fake-token",
		BaseURL:  server.URL,
	})
	if err := client.SendHTMLMessage(context.Background(), 42, "<b>Health</b>"); err != nil {
		t.Fatalf("SendHTMLMessage() error = %v", err)
	}
}

func TestClientHTTPErrorIncludesAPIDescription(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":400,"description":"Bad Request: BUTTON_DATA_INVALID"}`))
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		BotToken: "fake-token",
		BaseURL:  server.URL,
	})
	err := client.SendMessage(context.Background(), 42, "plain text")
	if err == nil {
		t.Fatal("SendMessage() error = nil, want HTTP error")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Fatalf("error = %v, want HTTP status", err)
	}
	if !strings.Contains(err.Error(), "BUTTON_DATA_INVALID") {
		t.Fatalf("error = %v, want telegram description", err)
	}
}

func TestClientHTTPErrorWithoutJSONBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream unavailable"))
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		BotToken: "fake-token",
		BaseURL:  server.URL,
	})
	err := client.SendMessage(context.Background(), 42, "plain text")
	if err == nil {
		t.Fatal("SendMessage() error = nil, want HTTP error")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Fatalf("error = %v, want HTTP status", err)
	}
	if strings.Contains(err.Error(), "upstream unavailable") {
		t.Fatalf("error = %v, want status only without raw body", err)
	}
}

func TestClientSendMessageWithoutParseMode(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("parse_mode") != "" {
			t.Fatalf("parse_mode = %q, want empty", r.Form.Get("parse_mode"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		BotToken: "fake-token",
		BaseURL:  server.URL,
	})
	if err := client.SendMessage(context.Background(), 42, "plain text"); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
}
