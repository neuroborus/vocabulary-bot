package logging

import (
	"log/slog"
	"os"
	"path/filepath"
)

const logAppDirName = "vocabulary-bot"

// DefaultLogPath returns the writable default log file under the system temp dir.
func DefaultLogPath() string {
	return filepath.Join(os.TempDir(), logAppDirName, "logs", "vocabulary.log")
}

func NewFileLogger(path string) (*slog.Logger, func() error, error) {
	if path == "" {
		path = DefaultLogPath()
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, nil, err
	}

	logger := slog.New(NewSanitizingHandler(slog.NewJSONHandler(file, &slog.HandlerOptions{})))

	return logger, file.Close, nil
}
