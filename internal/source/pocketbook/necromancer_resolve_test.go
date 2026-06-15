package pocketbook

import (
	"os"
	"testing"
)

func TestNecromancerResolveOffsets(t *testing.T) {
	path := os.Getenv("NECROMANCER_FB2_PATH")
	if path == "" {
		t.Skip("set NECROMANCER_FB2_PATH")
	}
	text, err := FlattenFB2(path)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		word string
		off  int
	}{
		{"carve", 463},
		{"gauge", 854},
		{"teasel", 1168},
	}
	for _, tc := range cases {
		resolved, ok := resolveWordOffset(text, tc.word, tc.off)
		if !ok {
			t.Fatalf("resolveWordOffset(%q, %d) = false", tc.word, tc.off)
		}
		sentence, ok := ExtractSentenceAtOffset(text, resolved)
		if !ok {
			t.Fatalf("ExtractSentenceAtOffset(%q, %d) = false", tc.word, resolved)
		}
		t.Logf("%s@%d -> %d sentence=%q", tc.word, tc.off, resolved, sentence)
	}
}
