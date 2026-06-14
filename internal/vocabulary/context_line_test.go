package vocabulary

import "testing"

func TestCleanContextLineStripsDanglingDialogueQuotes(t *testing.T) {
	t.Parallel()

	got := CleanContextLine(`" The voice from the lips was deep and pleasantly sardonic.`)
	want := "The voice from the lips was deep and pleasantly sardonic."
	if got != want {
		t.Fatalf("CleanContextLine() = %q, want %q", got, want)
	}
}

func TestCleanContextLinePreservesBalancedQuotes(t *testing.T) {
	t.Parallel()

	got := CleanContextLine(`"Hello," he said.`)
	want := `Hello," he said.`
	if got != want {
		t.Fatalf("CleanContextLine() = %q, want %q", got, want)
	}
}
