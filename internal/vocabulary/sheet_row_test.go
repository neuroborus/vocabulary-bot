package vocabulary_test

import (
	"context"
	"testing"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/storage/memory"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func TestMergeDraftReplacesSheetRowWhenWordChanges(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := memory.NewVocabularyRepository()
	service := vocabulary.NewService(repository, func() time.Time {
		return time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	})

	original := vocabulary.Draft{
		Source:       vocabulary.SourceGoogleSheet,
		RawWord:      "carve",
		Translations: []string{"вырезать"},
		Anchor: vocabulary.SourceAnchor{
			Source:    vocabulary.SourceGoogleSheet,
			SheetName: "Vocabulary",
			RowNumber: 5,
		},
	}
	if _, err := service.MergeDraft(ctx, original); err != nil {
		t.Fatalf("seed merge: %v", err)
	}

	replacement := vocabulary.Draft{
		Source:       vocabulary.SourceGoogleSheet,
		RawWord:      "gauge",
		Translations: []string{"измерять"},
		Anchor: vocabulary.SourceAnchor{
			Source:    vocabulary.SourceGoogleSheet,
			SheetName: "Vocabulary",
			RowNumber: 5,
		},
	}
	outcome, err := service.MergeDraft(ctx, replacement)
	if err != nil {
		t.Fatalf("replacement merge: %v", err)
	}
	if outcome.Skipped || !outcome.Updated {
		t.Fatalf("outcome = %#v, want updated sheet row", outcome)
	}
	if outcome.NormalizedKey != "gauge" {
		t.Fatalf("NormalizedKey = %q, want %q", outcome.NormalizedKey, "gauge")
	}

	items, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}

	item := items[0]
	if item.DisplayWord != "gauge" {
		t.Fatalf("DisplayWord = %q, want %q", item.DisplayWord, "gauge")
	}
	if len(item.Forms) != 1 || item.Forms[0].Value != "gauge" {
		t.Fatalf("forms = %#v, want only gauge", item.Forms)
	}
	if len(item.Translations) != 1 || item.Translations[0] != "измерять" {
		t.Fatalf("translations = %#v, want [измерять]", item.Translations)
	}
	if containsValue(item.LookupKeys, "carve") {
		t.Fatalf("lookup keys still contain carve: %#v", item.LookupKeys)
	}
	if item.Anchors[0].RowFingerprint != vocabulary.DraftFingerprint(replacement) {
		t.Fatalf("RowFingerprint = %q, want %q", item.Anchors[0].RowFingerprint, vocabulary.DraftFingerprint(replacement))
	}
}

func TestMergeDraftReplacesSheetRowTranslationsWithoutKeepingRemovedValues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := memory.NewVocabularyRepository()
	service := vocabulary.NewService(repository, func() time.Time {
		return time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	})

	original := vocabulary.Draft{
		Source:       vocabulary.SourceGoogleSheet,
		RawWord:      "carve",
		Translations: []string{"вырезать", "гравировать"},
		Anchor: vocabulary.SourceAnchor{
			Source:    vocabulary.SourceGoogleSheet,
			SheetName: "Vocabulary",
			RowNumber: 5,
		},
	}
	if _, err := service.MergeDraft(ctx, original); err != nil {
		t.Fatalf("seed merge: %v", err)
	}

	replacement := vocabulary.Draft{
		Source:       vocabulary.SourceGoogleSheet,
		RawWord:      "carve",
		Translations: []string{"вырезать"},
		Anchor: vocabulary.SourceAnchor{
			Source:    vocabulary.SourceGoogleSheet,
			SheetName: "Vocabulary",
			RowNumber: 5,
		},
	}
	if _, err := service.MergeDraft(ctx, replacement); err != nil {
		t.Fatalf("replacement merge: %v", err)
	}

	items, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if len(items[0].Translations) != 1 || items[0].Translations[0] != "вырезать" {
		t.Fatalf("translations = %#v, want only вырезать", items[0].Translations)
	}
}

func TestMergeDraftKeepsPocketBookDataWhenSheetRowChanges(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := memory.NewVocabularyRepository()
	service := vocabulary.NewService(repository, func() time.Time {
		return time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	})

	if _, err := service.MergeDraft(ctx, vocabulary.Draft{
		Source:       vocabulary.SourcePocketBook,
		RawWord:      "carve",
		Translations: []string{"высекать"},
		Contexts:     []string{"He carved the wood."},
		Anchor: vocabulary.SourceAnchor{
			Source:     vocabulary.SourcePocketBook,
			ExternalID: "note-1",
			BookTitle:  "Some Book",
		},
	}); err != nil {
		t.Fatalf("seed pocketbook merge: %v", err)
	}

	if _, err := service.MergeDraft(ctx, vocabulary.Draft{
		Source:       vocabulary.SourceGoogleSheet,
		RawWord:      "carve",
		Translations: []string{"вырезать"},
		Anchor: vocabulary.SourceAnchor{
			Source:    vocabulary.SourceGoogleSheet,
			SheetName: "Vocabulary",
			RowNumber: 5,
		},
	}); err != nil {
		t.Fatalf("seed sheet merge: %v", err)
	}

	if _, err := service.MergeDraft(ctx, vocabulary.Draft{
		Source:       vocabulary.SourceGoogleSheet,
		RawWord:      "gauge",
		Translations: []string{"измерять"},
		Anchor: vocabulary.SourceAnchor{
			Source:    vocabulary.SourceGoogleSheet,
			SheetName: "Vocabulary",
			RowNumber: 5,
		},
	}); err != nil {
		t.Fatalf("replacement merge: %v", err)
	}

	items, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}

	item := items[0]
	if item.DisplayWord != "gauge" {
		t.Fatalf("DisplayWord = %q, want %q", item.DisplayWord, "gauge")
	}
	if len(item.Forms) != 2 {
		t.Fatalf("forms = %d, want carve + gauge", len(item.Forms))
	}
	if len(item.Contexts) != 1 || item.Contexts[0] != "He carved the wood." {
		t.Fatalf("contexts = %#v, want pocketbook context preserved", item.Contexts)
	}
	if len(item.Translations) != 2 {
		t.Fatalf("translations = %#v, want pocketbook + sheet translation", item.Translations)
	}
	if !containsValue(item.Translations, "высекать") || !containsValue(item.Translations, "измерять") {
		t.Fatalf("translations = %#v, want pocketbook and new sheet values", item.Translations)
	}
	if containsValue(item.Translations, "вырезать") {
		t.Fatalf("translations = %#v, stale sheet translation must be removed", item.Translations)
	}
}

func containsValue(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}

	return false
}
