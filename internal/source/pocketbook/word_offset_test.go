package pocketbook

import "testing"

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
