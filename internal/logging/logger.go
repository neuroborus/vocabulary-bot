package logging

import (
	"log/slog"
	"os"
	"path/filepath"
)

func NewFileLogger(path string) (*slog.Logger, func() error, error) {
	if path == "" {
		path = "logs/vocabulary.log"
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
