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

	t.Run("rejects invalid chat id list", func(t *testing.T) {
		t.Setenv("TELEGRAM_ALLOWED_CHAT_IDS", "42,oops")

		_, err := config.Load()
		if err == nil {
			t.Fatal("Load() error = nil, want invalid chat id list")
		}
		if !strings.Contains(err.Error(), "TELEGRAM_ALLOWED_CHAT_IDS") {
			t.Fatalf("error = %v, want TELEGRAM_ALLOWED_CHAT_IDS", err)
		}
	})

	t.Run("parses comma-separated chat ids", func(t *testing.T) {
		t.Setenv("TELEGRAM_ALLOWED_CHAT_IDS", "42, -100123")

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if len(cfg.Telegram.AllowedChatIDs) != 2 {
			t.Fatalf("AllowedChatIDs = %#v, want 2 ids", cfg.Telegram.AllowedChatIDs)
		}
		if cfg.Telegram.AllowedChatIDs[0] != 42 || cfg.Telegram.AllowedChatIDs[1] != -100123 {
			t.Fatalf("AllowedChatIDs = %#v", cfg.Telegram.AllowedChatIDs)
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

	t.Run("accepts quoted and plain env values", func(t *testing.T) {
		const (
			mongoURI   = "mongodb+srv://example.test/db?retryWrites=true&w=majority"
			botToken   = "1234567890:FAKE_TELEGRAM_BOT_TOKEN_FOR_TEST_ONLY"
			password   = "p@ss&word"
			sheetRange = "Vocabulary!A:F"
		)

		cases := []struct {
			name string
			env  map[string]string
			want func(t *testing.T, cfg config.Config)
		}{
			{
				name: "string secrets and paths",
				env: map[string]string{
					"MONGODB_URI":         mongoURI,
					"TELEGRAM_BOT_TOKEN":  botToken,
					"POCKETBOOK_PASSWORD": password,
					"GOOGLE_SHEET_RANGE":  sheetRange,
				},
				want: func(t *testing.T, cfg config.Config) {
					t.Helper()
					if cfg.MongoDB.URI != mongoURI {
						t.Fatalf("MongoDB.URI = %q", cfg.MongoDB.URI)
					}
					if cfg.Telegram.BotToken != botToken {
						t.Fatalf("BotToken = %q", cfg.Telegram.BotToken)
					}
					if cfg.PocketBook.Password != password {
						t.Fatalf("Password = %q", cfg.PocketBook.Password)
					}
					if cfg.GoogleSheet.Range != sheetRange {
						t.Fatalf("Range = %q", cfg.GoogleSheet.Range)
					}
				},
			},
			{
				name: "single-quoted values",
				env: map[string]string{
					"MONGODB_URI":                 `'` + mongoURI + `'`,
					"TELEGRAM_BOT_TOKEN":          `'` + botToken + `'`,
					"POCKETBOOK_PASSWORD":         `'` + password + `'`,
					"SYNC_ENABLED":                `'false'`,
					"POCKETBOOK_BOOK_CACHE_MAX":   `'5'`,
					"REVIEW_DOCUMENT_PUSH_FACTOR": `'0.8'`,
					"TELEGRAM_ADMIN_ID":           `'42'`,
					"AUTO_SYNC_CRON":              `'0 9 * * *'`,
					"GOOGLE_SHEET_RANGE":          `'` + sheetRange + `'`,
				},
				want: func(t *testing.T, cfg config.Config) {
					t.Helper()
					if cfg.MongoDB.URI != mongoURI {
						t.Fatalf("MongoDB.URI = %q", cfg.MongoDB.URI)
					}
					if cfg.Telegram.BotToken != botToken {
						t.Fatalf("BotToken = %q", cfg.Telegram.BotToken)
					}
					if cfg.PocketBook.Password != password {
						t.Fatalf("Password = %q", cfg.PocketBook.Password)
					}
					if cfg.SyncEnabled {
						t.Fatal("SyncEnabled = true, want false from quoted 'false'")
					}
					if cfg.PocketBook.BookCacheMax != 5 {
						t.Fatalf("BookCacheMax = %d, want 5", cfg.PocketBook.BookCacheMax)
					}
					if cfg.Review.DocumentPushFactor != 0.8 {
						t.Fatalf("DocumentPushFactor = %v, want 0.8", cfg.Review.DocumentPushFactor)
					}
					if cfg.Telegram.AdminID != 42 {
						t.Fatalf("AdminID = %d, want 42", cfg.Telegram.AdminID)
					}
					if cfg.Schedule.AutoSyncCron != "0 9 * * *" {
						t.Fatalf("AutoSyncCron = %q", cfg.Schedule.AutoSyncCron)
					}
					if cfg.GoogleSheet.Range != sheetRange {
						t.Fatalf("Range = %q", cfg.GoogleSheet.Range)
					}
				},
			},
			{
				name: "double-quoted values",
				env: map[string]string{
					"MONGODB_URI":       `"` + mongoURI + `"`,
					"AUTO_PUSH_CRON":    `"0 12-21/2 * * *"`,
					"AUTO_LOGS_CRON":    `"0 21 * * 5"`,
					"SCHEDULE_TIMEZONE": `"Europe/Kyiv"`,
				},
				want: func(t *testing.T, cfg config.Config) {
					t.Helper()
					if cfg.MongoDB.URI != mongoURI {
						t.Fatalf("MongoDB.URI = %q", cfg.MongoDB.URI)
					}
					if cfg.Schedule.AutoPushCron != "0 12-21/2 * * *" {
						t.Fatalf("AutoPushCron = %q", cfg.Schedule.AutoPushCron)
					}
					if cfg.Schedule.AutoLogsCron != "0 21 * * 5" {
						t.Fatalf("AutoLogsCron = %q", cfg.Schedule.AutoLogsCron)
					}
					if cfg.Schedule.Timezone != "Europe/Kyiv" {
						t.Fatalf("Timezone = %q", cfg.Schedule.Timezone)
					}
				},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				for key, value := range tc.env {
					t.Setenv(key, value)
				}

				cfg, err := config.Load()
				if err != nil {
					t.Fatalf("Load() error = %v", err)
				}

				tc.want(t, cfg)
			})
		}
	})
}
