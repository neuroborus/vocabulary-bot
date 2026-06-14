package logging

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTruncateFileClearsActiveLog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "vocabulary.log")
	if err := os.WriteFile(logPath, []byte("old log\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := TruncateFile(logPath); err != nil {
		t.Fatalf("TruncateFile() error = %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("truncated log size = %d, want 0", len(data))
	}
}
