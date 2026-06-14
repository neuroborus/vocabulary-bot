package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultLogPathUsesTempDir(t *testing.T) {
	t.Parallel()

	got := DefaultLogPath()
	if !strings.HasPrefix(got, os.TempDir()) {
		t.Fatalf("DefaultLogPath() = %q, want prefix %q", got, os.TempDir())
	}

	wantSuffix := filepath.Join("vocabulary-bot", "logs", "vocabulary.log")
	if !strings.HasSuffix(got, wantSuffix) {
		t.Fatalf("DefaultLogPath() = %q, want suffix %q", got, wantSuffix)
	}
}
