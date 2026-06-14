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

func stringsIndex(text, needle string) int {
	for index := 0; index+len(needle) <= len(text); index++ {
		if text[index:index+len(needle)] == needle {
			return index
		}
	}
	return -1
}
