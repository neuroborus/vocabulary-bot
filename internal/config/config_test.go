package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neuroborus/vocabulary-bot/internal/config"
)

func TestDefaultLogPathUsesTempDir(t *testing.T) {
	t.Parallel()

	got := config.DefaultLogPath()
	if !strings.HasPrefix(got, os.TempDir()) {
		t.Fatalf("DefaultLogPath() = %q, want prefix %q", got, os.TempDir())
	}

	wantSuffix := filepath.Join("vocabulary-bot", "logs", "vocabulary.log")
	if !strings.HasSuffix(got, wantSuffix) {
		t.Fatalf("DefaultLogPath() = %q, want suffix %q", got, wantSuffix)
	}
}

func TestLoadEnvValidation(t *testing.T) {
	t.Run("rejects invalid bool", func(t *testing.T) {
		t.Setenv("TELEGRAM_POLLING_ENABLED", "flase")

		_, err := config.Load()
		if err == nil {
			t.Fatal("Load() error = nil, want invalid bool")
		}
		if !strings.Contains(err.Error(), "TELEGRAM_POLLING_ENABLED") {
			t.Fatalf("error = %v, want TELEGRAM_POLLING_ENABLED", err)
		}
	})

	t.Run("rejects invalid int", func(t *testing.T) {
		t.Setenv("POCKETBOOK_BOOK_CACHE_MAX", "oops")

		_, err := config.Load()
		if err == nil {
			t.Fatal("Load() error = nil, want invalid int")
		}
		if !strings.Contains(err.Error(), "POCKETBOOK_BOOK_CACHE_MAX") {
			t.Fatalf("error = %v, want POCKETBOOK_BOOK_CACHE_MAX", err)
		}
	})

	t.Run("rejects invalid float", func(t *testing.T) {
		t.Setenv("REVIEW_BOOK_PUSH_FACTOR", "nope")

		_, err := config.Load()
		if err == nil {
			t.Fatal("Load() error = nil, want invalid float")
		}
		if !strings.Contains(err.Error(), "REVIEW_BOOK_PUSH_FACTOR") {
			t.Fatalf("error = %v, want REVIEW_BOOK_PUSH_FACTOR", err)
		}
	})

	t.Run("rejects non-positive float", func(t *testing.T) {
		t.Setenv("REVIEW_DOCUMENT_PUSH_FACTOR", "0")

		_, err := config.Load()
		if err == nil {
			t.Fatal("Load() error = nil, want non-positive float")
		}
		if !strings.Contains(err.Error(), "REVIEW_DOCUMENT_PUSH_FACTOR") {
			t.Fatalf("error = %v, want REVIEW_DOCUMENT_PUSH_FACTOR", err)
		}
	})

	t.Run("uses defaults when typed env unset", func(t *testing.T) {
		t.Setenv("SYNC_ENABLED", "")
		t.Setenv("POCKETBOOK_BOOK_CACHE_MAX", "")
		t.Setenv("REVIEW_DOCUMENT_PUSH_FACTOR", "")

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if !cfg.SyncEnabled {
			t.Fatal("SyncEnabled = false, want default true")
		}
		if cfg.PocketBook.BookCacheMax != 2 {
			t.Fatalf("BookCacheMax = %d, want 2", cfg.PocketBook.BookCacheMax)
		}
		if cfg.Review.DocumentPushFactor != 0.7 {
			t.Fatalf("DocumentPushFactor = %v, want 0.7", cfg.Review.DocumentPushFactor)
		}
	})

	t.Run("strips single quotes from env values", func(t *testing.T) {
		t.Setenv("MONGODB_URI", `'mongodb+srv://example.test/db?retryWrites=true&w=majority'`)

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.MongoDB.URI != "mongodb+srv://example.test/db?retryWrites=true&w=majority" {
			t.Fatalf("MongoDB.URI = %q, want quoted value unwrapped", cfg.MongoDB.URI)
		}
	})

	t.Run("strips double quotes from env values", func(t *testing.T) {
		t.Setenv("MONGODB_URI", `"mongodb+srv://example.test/db?retryWrites=true&w=majority"`)

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.MongoDB.URI != "mongodb+srv://example.test/db?retryWrites=true&w=majority" {
			t.Fatalf("MongoDB.URI = %q, want quoted value unwrapped", cfg.MongoDB.URI)
		}
	})
}
