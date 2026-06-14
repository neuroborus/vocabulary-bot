package spreadsheet

import (
	"testing"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func TestParseRows(t *testing.T) {
	t.Parallel()

	rows := [][]string{
		{"word", "translations", "contexts", "note", "tags", "enabled"},
		{"to decelerate", "замедляться, снижать скорость", "The car began to decelerate. The train slowed before the station.", "teacher note", "verb, motion", "true"},
		{"disabled", "выключен", "", "", "", "no"},
		{"bad", "", "", "", "", "maybe"},
	}

	drafts, rowErrors := ParseRows("Vocabulary", rows)
	if len(rowErrors) != 1 {
		t.Fatalf("rowErrors = %d, want 1", len(rowErrors))
	}
	if len(drafts) != 1 {
		t.Fatalf("drafts = %d, want 1", len(drafts))
	}

	draft := drafts[0]
	if draft.Source != vocabulary.SourceGoogleSheet {
		t.Fatalf("Source = %q, want %q", draft.Source, vocabulary.SourceGoogleSheet)
	}
	if draft.RawWord != "to decelerate" {
		t.Fatalf("RawWord = %q", draft.RawWord)
	}
	if len(draft.Translations) != 2 {
		t.Fatalf("translations = %d, want 2", len(draft.Translations))
	}
	if len(draft.Contexts) != 2 {
		t.Fatalf("contexts = %d, want 2", len(draft.Contexts))
	}
	if draft.Anchor.RowNumber != 2 {
		t.Fatalf("RowNumber = %d, want 2", draft.Anchor.RowNumber)
	}
}

func TestParseRowsKeepsSentenceTranslationIntact(t *testing.T) {
	t.Parallel()

	rows := [][]string{
		{"word", "translations", "contexts", "note", "tags", "enabled"},
		{
			"The more people are aware of the existence of deepfakes, the less likely they are to believe everything they see",
			"Чем больше людей знают о существовании фальшивок, тем меньше вероятность того, что они поверят всему, что увидят",
			"", "", "", "true",
		},
	}

	drafts, rowErrors := ParseRows("Vocabulary", rows)
	if len(rowErrors) != 0 {
		t.Fatalf("rowErrors = %#v, want none", rowErrors)
	}
	if len(drafts) != 1 {
		t.Fatalf("drafts = %d, want 1", len(drafts))
	}
	if len(drafts[0].Translations) != 1 {
		t.Fatalf("translations = %#v, want one sentence", drafts[0].Translations)
	}
}
