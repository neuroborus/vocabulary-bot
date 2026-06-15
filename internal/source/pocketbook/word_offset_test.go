package pocketbook

import (
	"strings"
	"testing"
)

func TestResolveWordOffsetMatchesInflectedForms(t *testing.T) {
	t.Parallel()

	text := normalizeBookText("He leaned on the carved handle while the gauges flickered.")

	offset, ok := resolveWordOffset(text, "carve", stringsIndex(text, "carved"))
	if !ok {
		t.Fatal("resolveWordOffset(carve) = false")
	}
	if !strings.HasPrefix(text[offset:], "carved") {
		t.Fatalf("carve offset = %q", text[offset:offset+10])
	}

	offset, ok = resolveWordOffset(text, "gauge", stringsIndex(text, "gauges"))
	if !ok {
		t.Fatal("resolveWordOffset(gauge) = false")
	}
	if !strings.HasPrefix(text[offset:], "gauges") {
		t.Fatalf("gauge offset = %q", text[offset:offset+10])
	}
}

func TestResolveWordOffsetMatchesNameStem(t *testing.T) {
	t.Parallel()

	text := normalizeBookText(`"Right. Pat Teasely." He held out a small, square hand.`)

	offset, ok := resolveWordOffset(text, "teasel", stringsIndex(text, "Teasely"))
	if !ok {
		t.Fatal("resolveWordOffset(teasel) = false")
	}
	if !strings.HasPrefix(text[offset:], "Teasely") {
		t.Fatalf("teasel offset = %q", text[offset:offset+10])
	}
}

func TestResolveWordOffsetPrefersWindowBeforeGlobal(t *testing.T) {
	t.Parallel()

	gap := repeatSpaces(1000)
	text := "alpha frowning beta." + gap + "gamma frowning delta."
	first := stringsIndex(text, "frowning")
	second := stringsIndex(text[first+len("frowning"):], "frowning")
	if second < 0 {
		t.Fatal("test text missing second frowning")
	}
	second += first + len("frowning")
	preferred := second - 40

	offset, ok := resolveWordOffset(text, "frowning", preferred)
	if !ok {
		t.Fatal("resolveWordOffset() = false")
	}
	if offset != second {
		t.Fatalf("offset = %d, want second frowning at %d", offset, second)
	}
}

func TestResolveWordOffsetFallsBackToGlobalNearest(t *testing.T) {
	t.Parallel()

	gap := repeatSpaces(2000)
	text := "frowning once." + gap + "frowning twice."
	first := stringsIndex(text, "frowning")
	preferred := first + 5

	offset, ok := resolveWordOffset(text, "frowning", preferred)
	if !ok {
		t.Fatal("resolveWordOffset() = false")
	}
	if offset != first {
		t.Fatalf("offset = %d, want first frowning at %d", offset, first)
	}
}

func TestResolveWordOffsetFallsBackToNearestOccurrence(t *testing.T) {
	t.Parallel()

	text := "alpha beta gamma frowning delta. Much later Paul watched her, frowning a bit."
	preferred := 10

	offset, ok := resolveWordOffset(text, "frowning", preferred)
	if !ok {
		t.Fatal("resolveWordOffset() = false")
	}
	if offset != stringsIndex(text, "frowning") {
		t.Fatalf("offset = %d, want first frowning at %d", offset, stringsIndex(text, "frowning"))
	}

	sentence, ok := ExtractSentenceAtOffset(text, offset)
	if !ok || sentence != "alpha beta gamma frowning delta." {
		t.Fatalf("sentence = %q, ok=%v", sentence, ok)
	}
}

func TestResolveWordOffsetMatchesPossessiveForm(t *testing.T) {
	t.Parallel()

	text := "He picked up a teasel's head from the bench."
	preferred := 5

	offset, ok := resolveWordOffset(text, "teasel", preferred)
	if !ok {
		t.Fatal("resolveWordOffset() = false")
	}
	if text[offset:offset+len("teasel")] != "teasel" {
		t.Fatalf("offset = %d, text = %q", offset, text[offset:])
	}
}

func TestResolveWordOffsetMatchesHyphenatedLineBreak(t *testing.T) {
	t.Parallel()

	text := normalizeBookText("They used a tea-\nsel to raise the nap.")
	preferred := stringsIndex(text, "teasel")

	offset, ok := resolveWordOffset(text, "teasel", preferred)
	if !ok {
		t.Fatal("resolveWordOffset() = false")
	}

	sentence, ok := ExtractSentenceAtOffset(text, offset)
	if !ok {
		t.Fatal("ExtractSentenceAtOffset() = false")
	}
	if sentence != "They used a teasel to raise the nap." {
		t.Fatalf("sentence = %q", sentence)
	}
}

func TestResolveWordOffsetFindsWordNearPreferredOffset(t *testing.T) {
	t.Parallel()

	text := "Intro padding. " + repeatSpaces(400) + "The gauge read zero."
	preferred := 420

	offset, ok := resolveWordOffset(text, "gauge", preferred)
	if !ok {
		t.Fatal("resolveWordOffset() = false")
	}
	if text[offset:offset+len("gauge")] != "gauge" {
		t.Fatalf("offset = %d, text = %q", offset, text[offset:])
	}
}

func repeatSpaces(count int) string {
	buf := make([]byte, count)
	for i := range buf {
		buf[i] = ' '
	}
	return string(buf)
}

func stringsIndex(text, needle string) int {
	for index := 0; index+len(needle) <= len(text); index++ {
		if text[index:index+len(needle)] == needle {
			return index
		}
	}
	return -1
}
