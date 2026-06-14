package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv               string
	LogPath              string
	SyncEnabled          bool
	NotificationsEnabled bool
	MongoDB              MongoDBConfig
	Telegram             TelegramConfig
	Review               ReviewConfig
	Schedule             ScheduleConfig
	PocketBook           PocketBookConfig
	GoogleSheet          GoogleSheetConfig
}

type ReviewConfig struct {
	DocumentPushFactor float64
	BookPushFactor     float64
}

type ScheduleConfig struct {
	Timezone     string
	AutoSyncCron string
	AutoPushCron string
	AutoLogsCron string
}

type MongoDBConfig struct {
	URI    string
	DBName string
}

type TelegramConfig struct {
	BotToken                  string
	AllowedUserID             int64
	TargetChatID              int64
	APIBaseURL                string
	PollingEnabled            bool
	ReviewSpoilerTranslations bool
}

type PocketBookConfig struct {
	Enabled            bool
	Email              string
	Password           string
	RefreshToken       string
	ShopName           string
	BaseURL            string
	TokenPath          string
	BookContextEnabled bool
	BookCacheDir       string
	BookCacheMax       int
}

type GoogleSheetConfig struct {
	Enabled            bool
	ServiceAccountJSON string
	SpreadsheetID      string
	SheetName          string
	Range              string
}

func Load() (Config, error) {
	allowedUserID, err := optionalInt64("TELEGRAM_ALLOWED_USER_ID")
	if err != nil {
		return Config{}, err
	}

	targetChatID, err := optionalInt64("TELEGRAM_TARGET_CHAT_ID")
	if err != nil {
		return Config{}, err
	}

	syncEnabled, err := getenvBool("SYNC_ENABLED", true)
	if err != nil {
		return Config{}, err
	}

	notificationsEnabled, err := getenvBool("NOTIFICATIONS_ENABLED", true)
	if err != nil {
		return Config{}, err
	}

	pollingEnabled, err := getenvBool("TELEGRAM_POLLING_ENABLED", true)
	if err != nil {
		return Config{}, err
	}

	reviewSpoilerTranslations, err := getenvBool("TELEGRAM_REVIEW_SPOILER_TRANSLATIONS", true)
	if err != nil {
		return Config{}, err
	}

	documentPushFactor, err := getenvFloat("REVIEW_DOCUMENT_PUSH_FACTOR", 0.7)
	if err != nil {
		return Config{}, err
	}

	bookPushFactor, err := getenvFloat("REVIEW_BOOK_PUSH_FACTOR", 1)
	if err != nil {
		return Config{}, err
	}

	pocketbookSyncEnabled, err := getenvBool("POCKETBOOK_SYNC_ENABLED", true)
	if err != nil {
		return Config{}, err
	}

	bookContextEnabled, err := getenvBool("POCKETBOOK_BOOK_CONTEXT_ENABLED", true)
	if err != nil {
		return Config{}, err
	}

	bookCacheMax, err := getenvInt("POCKETBOOK_BOOK_CACHE_MAX", 2)
	if err != nil {
		return Config{}, err
	}

	googleSheetSyncEnabled, err := getenvBool("GOOGLE_SHEET_SYNC_ENABLED", true)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppEnv:               getenv("APP_ENV", "local"),
		LogPath:              getenv("LOG_PATH", DefaultLogPath()),
		SyncEnabled:          syncEnabled,
		NotificationsEnabled: notificationsEnabled,
		MongoDB: MongoDBConfig{
			URI:    getenv("MONGODB_URI", ""),
			DBName: getenv("MONGODB_DB_NAME", "vocabulary_bot"),
		},
		Telegram: TelegramConfig{
			BotToken:                  getenv("TELEGRAM_BOT_TOKEN", ""),
			AllowedUserID:             allowedUserID,
			TargetChatID:              targetChatID,
			APIBaseURL:                getenv("TELEGRAM_API_BASE_URL", ""),
			PollingEnabled:            pollingEnabled,
			ReviewSpoilerTranslations: reviewSpoilerTranslations,
		},
		Review: ReviewConfig{
			DocumentPushFactor: documentPushFactor,
			BookPushFactor:     bookPushFactor,
		},
		Schedule: ScheduleConfig{
			Timezone:     getenv("SCHEDULE_TIMEZONE", ""),
			AutoSyncCron: lookupEnvOrDefault("AUTO_SYNC_CRON", "0 9 * * *"),
			AutoPushCron: lookupEnvOrDefault("AUTO_PUSH_CRON", "0 12-21/2 * * *"),
			AutoLogsCron: lookupEnvOrDefault("AUTO_LOGS_CRON", "0 21 * * 5"),
		},
		PocketBook: PocketBookConfig{
			Enabled:            pocketbookSyncEnabled,
			Email:              firstEnv([]string{"POCKETBOOK_EMAIL", "POCKETBOOK_LOGIN"}, ""),
			Password:           getenv("POCKETBOOK_PASSWORD", ""),
			RefreshToken:       getenv("POCKETBOOK_REFRESH_TOKEN", ""),
			ShopName:           getenv("POCKETBOOK_SHOP_NAME", ""),
			BaseURL:            getenv("POCKETBOOK_API_BASE_URL", ""),
			TokenPath:          getenv("POCKETBOOK_TOKEN_PATH", ""),
			BookContextEnabled: bookContextEnabled,
			BookCacheDir:       getenv("POCKETBOOK_BOOK_CACHE_DIR", ""),
			BookCacheMax:       bookCacheMax,
		},
		GoogleSheet: GoogleSheetConfig{
			Enabled:            googleSheetSyncEnabled,
			ServiceAccountJSON: getenv("GOOGLE_SERVICE_ACCOUNT_JSON", ""),
			SpreadsheetID:      firstEnv([]string{"GOOGLE_SPREADSHEET_ID", "GOOGLE_SHEET_ID"}, ""),
			SheetName:          getenv("GOOGLE_SHEET_NAME", "Vocabulary"),
			Range:              getenv("GOOGLE_SHEET_RANGE", "Vocabulary!A:F"),
		},
	}

	return cfg, nil
}

func getenv(key string, fallback string) string {
	value := unquoteEnvValue(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func unquoteEnvValue(raw string) string {
	value := strings.TrimSpace(raw)
	if len(value) < 2 {
		return value
	}

	if value[0] == '"' && value[len(value)-1] == '"' {
		return strings.NewReplacer(
			`\\`, `\`,
			`\"`, `"`,
			`\n`, "\n",
		).Replace(value[1 : len(value)-1])
	}

	if value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}

	return value
}

func lookupEnvOrDefault(key string, fallback string) string {
	raw, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	return strings.TrimSpace(raw)
}

func firstEnv(keys []string, fallback string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}

	return fallback
}

func getenvBool(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}

func getenvInt(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}

func getenvFloat(key string, fallback float64) (float64, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("parse %s: value must be positive", key)
	}

	return parsed, nil
}

func optionalInt64(key string) (int64, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}

const logAppDirName = "vocabulary-bot"

// DefaultLogPath returns the writable default log file under the system temp dir.
func DefaultLogPath() string {
	return filepath.Join(os.TempDir(), logAppDirName, "logs", "vocabulary.log")
}
