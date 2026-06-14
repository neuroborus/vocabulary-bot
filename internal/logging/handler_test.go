package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestSanitizingHandlerWithAttrsRedactsSensitiveValues(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger := slog.New(NewSanitizingHandler(slog.NewTextHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelInfo})))
	logger = logger.With(
		slog.String("refresh_token", "FAKE_REFRESH_TOKEN_FOR_TEST_ONLY"),
		slog.String("password", "FAKE_PASSWORD_FOR_TEST_ONLY"),
	)

	logger.Info("request failed")

	output := buffer.String()
	if strings.Contains(output, "FAKE_REFRESH_TOKEN_FOR_TEST_ONLY") {
		t.Fatalf("output still contains refresh token: %s", output)
	}
	if strings.Contains(output, "FAKE_PASSWORD_FOR_TEST_ONLY") {
		t.Fatalf("output still contains password: %s", output)
	}
	if !strings.Contains(output, "refresh_token=***") {
		t.Fatalf("output = %q, want sanitized refresh_token", output)
	}
	if !strings.Contains(output, "password=***") {
		t.Fatalf("output = %q, want sanitized password", output)
	}
}

func TestSanitizingHandlerHandleRedactsRecordAttrs(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := NewSanitizingHandler(slog.NewTextHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelInfo}))
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "sync failed", 0)
	record.AddAttrs(slog.String("access_token", "FAKE_ACCESS_TOKEN_FOR_TEST_ONLY"))

	if err := handler.Handle(context.Background(), record); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	output := buffer.String()
	if strings.Contains(output, "FAKE_ACCESS_TOKEN_FOR_TEST_ONLY") {
		t.Fatalf("output still contains access token: %s", output)
	}
	if !strings.Contains(output, "access_token=***") {
		t.Fatalf("output = %q, want sanitized access_token", output)
	}
}
