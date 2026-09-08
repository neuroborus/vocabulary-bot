package vocabulary_test

import (
	"context"
	"testing"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/storage/memory"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func TestMergeDraftPreservesFormsAndPrefersRicherDisplayWord(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 8, 0, 0, 0, time.UTC)
	repository := memory.NewVocabularyRepository()
	service := vocabulary.NewService(repository, func() time.Time { return now })

	created, err := service.MergeDraft(context.Background(), vocabulary.Draft{
		Source:       vocabulary.SourceGoogleSheet,
		RawWord:      "decelerate",
		Translations: []string{"замедляться"},
		Anchor: vocabulary.SourceAnchor{
			Source:    vocabulary.SourceGoogleSheet,
			RowNumber: 2,
			SheetName: "Vocabulary",
		},
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if !created.Created {
		t.Fatalf("first merge should create an item")
	}

	updated, err := service.MergeDraft(context.Background(), vocabulary.Draft{
		Source:       vocabulary.SourcePocketBook,
		RawWord:      "to decelerate",
		Translations: []string{"снижать скорость"},
		Contexts:     []string{"The car began to decelerate rapidly"},
		Anchor: vocabulary.SourceAnchor{
			Source:     vocabulary.SourcePocketBook,
			ExternalID: "note-123",
			BookTitle:  "Some Book",
		},
	})
	if err != nil {
		t.Fatalf("merge richer draft: %v", err)
	}
	if !updated.Updated {
		t.Fatalf("second merge should update the existing item")
	}

	items, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}

	item := items[0]
	if item.DisplayWord != "to decelerate" {
		t.Fatalf("DisplayWord = %q, want %q", item.DisplayWord, "to decelerate")
	}
	if len(item.Forms) != 2 {
		t.Fatalf("forms = %d, want 2", len(item.Forms))
	}
	if len(item.Translations) != 2 {
		t.Fatalf("translations = %d, want 2", len(item.Translations))
	}
	if len(item.Contexts) != 1 {
		t.Fatalf("contexts = %d, want 1", len(item.Contexts))
	}
	if !contains(item.LookupKeys, "todecelerate") || !contains(item.LookupKeys, "decelerate") {
		t.Fatalf("lookup keys do not include merged form keys: %#v", item.LookupKeys)
	}
}

func TestMergeDraftPrefersLeadingArticleDisplayWord(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 8, 0, 0, 0, time.UTC)
	repository := memory.NewVocabularyRepository()
	service := vocabulary.NewService(repository, func() time.Time { return now })

	drafts := []vocabulary.Draft{
		{
			Source:  vocabulary.SourceGoogleSheet,
			RawWord: "apple",
			Anchor: vocabulary.SourceAnchor{
				Source:    vocabulary.SourceGoogleSheet,
				RowNumber: 2,
				SheetName: "Vocabulary",
			},
		},
		{
			Source:  vocabulary.SourceGoogleSheet,
			RawWord: "apple an",
			Anchor: vocabulary.SourceAnchor{
				Source:    vocabulary.SourceGoogleSheet,
				RowNumber: 3,
				SheetName: "Vocabulary",
			},
		},
		{
			Source:  vocabulary.SourceGoogleSheet,
			RawWord: "an apple",
			Anchor: vocabulary.SourceAnchor{
				Source:    vocabulary.SourceGoogleSheet,
				RowNumber: 4,
				SheetName: "Vocabulary",
			},
		},
	}

	for _, draft := range drafts {
		if _, err := service.MergeDraft(context.Background(), draft); err != nil {
			t.Fatalf("merge %q: %v", draft.RawWord, err)
		}
	}

	items, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}

	if items[0].DisplayWord != "an apple" {
		t.Fatalf("DisplayWord = %q, want %q", items[0].DisplayWord, "an apple")
	}
}

func TestMergeDraftKeepsTrailingWordsInDisplayWord(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 8, 0, 0, 0, time.UTC)
	repository := memory.NewVocabularyRepository()
	service := vocabulary.NewService(repository, func() time.Time { return now })

	drafts := []vocabulary.Draft{
		{
			Source:  vocabulary.SourceGoogleSheet,
			RawWord: "decelerate",
			Anchor: vocabulary.SourceAnchor{
				Source:    vocabulary.SourceGoogleSheet,
				RowNumber: 2,
				SheetName: "Vocabulary",
			},
		},
		{
			Source:  vocabulary.SourceGoogleSheet,
			RawWord: "to decelerate",
			Anchor: vocabulary.SourceAnchor{
				Source:    vocabulary.SourceGoogleSheet,
				RowNumber: 3,
				SheetName: "Vocabulary",
			},
		},
		{
			Source:  vocabulary.SourceGoogleSheet,
			RawWord: "decelerate to",
			Anchor: vocabulary.SourceAnchor{
				Source:    vocabulary.SourceGoogleSheet,
				RowNumber: 4,
				SheetName: "Vocabulary",
			},
		},
	}

	for _, draft := range drafts {
		if _, err := service.MergeDraft(context.Background(), draft); err != nil {
			t.Fatalf("merge %q: %v", draft.RawWord, err)
		}
	}

	items, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}

	if items[0].DisplayWord != "decelerate to" {
		t.Fatalf("DisplayWord = %q, want %q", items[0].DisplayWord, "decelerate to")
	}
}

func TestMergeDraftKeepsPhrasesSharingOnlyFunctionWordSeparate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 8, 8, 0, 0, 0, time.UTC)
	repository := memory.NewVocabularyRepository()
	service := vocabulary.NewService(repository, func() time.Time { return now })

	phrases := []struct {
		word        string
		translation string
		row         int
	}{
		{"fit the bill", "подходить под описание", 2},
		{"In the beginning", "вначале", 3},
		{"At the end", "в конце", 4},
		{"learn the ropes", "освоить основы", 5},
		{"get the sack", "уволиться", 6},
	}

	for _, phrase := range phrases {
		outcome, err := service.MergeDraft(context.Background(), vocabulary.Draft{
			Source:       vocabulary.SourceGoogleSheet,
			RawWord:      phrase.word,
			Translations: []string{phrase.translation},
			Anchor: vocabulary.SourceAnchor{
				Source:    vocabulary.SourceGoogleSheet,
				RowNumber: phrase.row,
				SheetName: "Vocabulary",
			},
		})
		if err != nil {
			t.Fatalf("merge %q: %v", phrase.word, err)
		}
		if !outcome.Created {
			t.Fatalf("merge %q should create a separate item, outcome = %+v", phrase.word, outcome)
		}
	}

	items, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != len(phrases) {
		t.Fatalf("items = %d, want %d (phrases must not collapse on the shared %q key)", len(items), len(phrases), "the")
	}

	// Re-syncing an unchanged row must not raise a false ambiguity even though
	// every item still stores the weak "the" candidate key.
	resync, err := service.MergeDraft(context.Background(), vocabulary.Draft{
		Source:       vocabulary.SourceGoogleSheet,
		RawWord:      "fit the bill",
		Translations: []string{"подходить под описание"},
		Anchor: vocabulary.SourceAnchor{
			Source:    vocabulary.SourceGoogleSheet,
			RowNumber: 2,
			SheetName: "Vocabulary",
		},
	})
	if err != nil {
		t.Fatalf("resync fit the bill: %v", err)
	}
	if resync.Ambiguous {
		t.Fatalf("resync must not be ambiguous, outcome = %+v", resync)
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}

	return false
}
