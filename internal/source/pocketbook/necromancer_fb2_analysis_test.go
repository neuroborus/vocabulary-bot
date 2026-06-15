package pocketbook

import (
	"os"
	"strings"
	"testing"
)

func TestAnalyzeNecromancerFB2Fixture(t *testing.T) {
	path := os.Getenv("NECROMANCER_FB2_PATH")
	if path == "" {
		t.Skip("set NECROMANCER_FB2_PATH to run")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	rawText := string(raw)
	for _, word := range []string{"carve", "gauge", "teasel"} {
		t.Logf("raw count %q: %d", word, strings.Count(strings.ToLower(rawText), word))
	}

	text, err := FlattenFB2(path)
	if err != nil {
		t.Fatalf("FlattenFB2() error = %v", err)
	}
	t.Logf("flattened length: %d", len(text))

	for _, word := range []string{"carve", "gauge", "teasel"} {
		count := countWordOccurrences(text, word)
		t.Logf("flat count %q: %d", word, count)
		if count == 0 {
			t.Errorf("flattened text missing %q", word)
		}
	}

	for _, off := range []int{463, 854, 1168} {
		if off >= len(text) {
			t.Logf("offset %d beyond flattened length", off)
			continue
		}
		start, end := off-40, off+60
		if start < 0 {
			start = 0
		}
		if end > len(text) {
			end = len(text)
		}
		t.Logf("flat@%d: %q", off, text[start:end])
	}
}
