package telegram

import (
	"strings"
	"testing"

	"github.com/neuroborus/vocabulary-bot/internal/save"
	"github.com/neuroborus/vocabulary-bot/internal/source"
	syncer "github.com/neuroborus/vocabulary-bot/internal/sync"
)

func TestFormatSaveConfirmationHidesMetaInSpoiler(t *testing.T) {
	t.Parallel()

	text := formatSaveConfirmation(save.Result{
		Word:        "presence",
		Context:     "Her presence in the room made everyone feel more comfortable.",
		Translation: "присутствие",
		RowNumber:   285,
	})

	for _, want := range []string{
		"<b>Word</b>",
		"presence",
		"<b>Context</b>",
		"feel more comfortable.",
		"<b>Translation</b>",
		"присутствие",
		"<tg-spoiler>",
		"<b>Saved to Google Sheets</b>",
		"Row: <code>285</code>",
		"/sync",
		"</tg-spoiler>",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("save confirmation missing %q: %q", want, text)
		}
	}

	if strings.Index(text, "<b>Word</b>") > strings.Index(text, "<tg-spoiler>") {
		t.Fatalf("word should appear before spoiler: %q", text)
	}
	if strings.Index(text, "<b>Translation</b>") > strings.Index(text, "<tg-spoiler>") {
		t.Fatalf("translation should appear before spoiler: %q", text)
	}
	if strings.Index(text, "Saved to Google Sheets") < strings.Index(text, "<tg-spoiler>") {
		t.Fatalf("sheet status should be inside spoiler: %q", text)
	}
}

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

func TestFormatHealthMessageIncludesChatID(t *testing.T) {
	t.Parallel()

	text := formatHealthMessage("ok", 3, true, false, -1004299028040, 490734700)
	if !strings.Contains(text, "Chat ID: <code>-1004299028040</code>") {
		t.Fatalf("health message = %q, want chat id", text)
	}
	if !strings.Contains(text, "Caller ID: <code>490734700</code>") {
		t.Fatalf("health message = %q, want caller id", text)
	}

	text = formatHealthMessage("ok", 3, true, false, 0, 0)
	if strings.Contains(text, "Chat ID:") || strings.Contains(text, "Caller ID:") {
		t.Fatalf("health message = %q, want no chat or caller id when unset", text)
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

func TestFormatSyncSummaryIncludesBookContextStats(t *testing.T) {
	t.Parallel()

	text := formatSyncSummary(syncer.Summary{
		DraftsProcessed: 29,
		Updated:         29,
		Sources: []syncer.SourceSummary{
			{
				Name:   "pocketbook",
				Drafts: 29,
				Details: source.Details{
					BookContextSkippedStored: 25,
					BookContextEnriched:      2,
				},
			},
		},
	})

	for _, want := range []string{
		"book context skipped (already in DB): 25",
		"book context enriched: 2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("sync summary missing %q: %q", want, text)
		}
	}
}

func TestFormatSyncSummaryIncludesPartialFailureStats(t *testing.T) {
	t.Parallel()

	text := formatSyncSummary(syncer.Summary{
		Sources: []syncer.SourceSummary{
			{
				Name:   "pocketbook",
				Drafts: 12,
				Details: source.Details{
					BooksSkipped: 1,
					BooksFailed:  2,
					NotesFailed:  3,
				},
			},
			{
				Name:   "google-sheet",
				Drafts: 20,
				Details: source.Details{
					RowParseErrors: 4,
				},
			},
		},
	})

	for _, want := range []string{
		"books skipped: 1",
		"books failed: 2",
		"notes failed: 3",
		"row parse errors: 4",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("sync summary missing %q: %q", want, text)
		}
	}
}

func TestFormatSyncSummaryIncludesSkippedSpreadsheetRows(t *testing.T) {
	t.Parallel()

	text := formatSyncSummary(syncer.Summary{
		Sources: []syncer.SourceSummary{
			{
				Name:   "google-sheet",
				Drafts: 279,
				Details: source.Details{
					RowsSkippedUnchanged: 275,
				},
			},
		},
	})

	if !strings.Contains(text, "rows skipped (unchanged): 275") {
		t.Fatalf("sync summary missing skipped rows: %q", text)
	}
}
