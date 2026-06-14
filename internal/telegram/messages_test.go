package telegram

import (
	"strings"
	"testing"

	syncer "github.com/neuroborus/vocabulary-bot/internal/sync"
)

func TestEscapeHTML(t *testing.T) {
	t.Parallel()

	got := escapeHTML(`a & b <tag> "quoted"`)
	want := `a &amp; b &lt;tag&gt; "quoted"`
	if got != want {
		t.Fatalf("escapeHTML() = %q, want %q", got, want)
	}
}

func TestFormatCommandsBlockUsesHTML(t *testing.T) {
	t.Parallel()

	text := formatCommandsBlock()
	if !strings.Contains(text, "<b>Commands</b>") {
		t.Fatalf("commands block missing title: %q", text)
	}
	if !strings.Contains(text, "<code>/sync</code>") {
		t.Fatalf("commands block missing formatted command: %q", text)
	}
	if strings.Contains(text, "/sync - ") {
		t.Fatalf("commands block still uses plain dash format: %q", text)
	}
}

func TestFormatSyncSummaryIncludesSections(t *testing.T) {
	t.Parallel()

	text := formatSyncSummary(syncer.Summary{
		DraftsProcessed: 3,
		Created:         1,
		Updated:         2,
		SourceErrors:    1,
		Sources: []syncer.SourceSummary{
			{Name: "pocketbook", Drafts: 2},
			{Name: "spreadsheet", Drafts: 1, Error: "timeout & retry"},
		},
	})

	for _, want := range []string{
		"<b>Sync complete</b>",
		"<b>Totals</b>",
		"Drafts processed: <b>3</b>",
		"<b>Sources</b>",
		"pocketbook",
		"timeout &amp; retry",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("sync summary missing %q: %q", want, text)
		}
	}
}
