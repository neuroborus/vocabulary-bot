package syncer

import (
	"context"
	"testing"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/source"
	"github.com/neuroborus/vocabulary-bot/internal/source/spreadsheet"
	"github.com/neuroborus/vocabulary-bot/internal/storage/memory"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type stubAdapter struct {
	name   string
	drafts []vocabulary.Draft
}

func (a stubAdapter) Name() string { return a.name }

func (a stubAdapter) Sync(context.Context) ([]vocabulary.Draft, error) {
	return append([]vocabulary.Draft(nil), a.drafts...), nil
}

func TestRunSkipsUnchangedSpreadsheetRows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := memory.NewVocabularyRepository()
	service := vocabulary.NewService(repository, func() time.Time {
		return time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	})

	draft := vocabulary.Draft{
		Source:       vocabulary.SourceGoogleSheet,
		RawWord:      "carve",
		Translations: []string{"вырезать"},
		Anchor: vocabulary.SourceAnchor{
			Source:    vocabulary.SourceGoogleSheet,
			SheetName: "Vocabulary",
			RowNumber: 5,
		},
	}

	if _, err := service.MergeDraft(ctx, draft); err != nil {
		t.Fatalf("first merge: %v", err)
	}

	syncService := NewService([]source.Adapter{
		stubAdapter{
			name:   spreadsheet.NewAdapter(spreadsheet.AdapterOptions{}).Name(),
			drafts: []vocabulary.Draft{draft},
		},
	}, service, nil)

	summary, err := syncService.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if summary.DraftsProcessed != 0 {
		t.Fatalf("DraftsProcessed = %d, want 0", summary.DraftsProcessed)
	}
	if len(summary.Sources) != 1 {
		t.Fatalf("sources = %d, want 1", len(summary.Sources))
	}
	if summary.Sources[0].Details.RowsSkippedUnchanged != 1 {
		t.Fatalf("RowsSkippedUnchanged = %d, want 1", summary.Sources[0].Details.RowsSkippedUnchanged)
	}
}

func TestRunMergesChangedSpreadsheetRow(t *testing.T) {
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

	changed := original
	changed.Translations = []string{"вырезать", "гравировать"}

	syncService := NewService([]source.Adapter{
		stubAdapter{
			name:   "google-sheet",
			drafts: []vocabulary.Draft{changed},
		},
	}, service, nil)

	summary, err := syncService.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if summary.DraftsProcessed != 1 {
		t.Fatalf("DraftsProcessed = %d, want 1", summary.DraftsProcessed)
	}
	if summary.Updated != 1 {
		t.Fatalf("Updated = %d, want 1", summary.Updated)
	}
	if summary.Sources[0].Details.RowsSkippedUnchanged != 0 {
		t.Fatalf("RowsSkippedUnchanged = %d, want 0", summary.Sources[0].Details.RowsSkippedUnchanged)
	}
}

func TestMergeDraftUpdatesExistingSheetRowWhenWordChanges(t *testing.T) {
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
	if outcome.Skipped {
		t.Fatal("replacement merge was skipped")
	}
	if !outcome.Updated {
		t.Fatal("replacement merge did not update existing row")
	}

	items, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].Anchors[0].RowFingerprint != vocabulary.DraftFingerprint(replacement) {
		t.Fatalf("RowFingerprint = %q, want %q", items[0].Anchors[0].RowFingerprint, vocabulary.DraftFingerprint(replacement))
	}
	if items[0].DisplayWord != "gauge" {
		t.Fatalf("DisplayWord = %q, want gauge", items[0].DisplayWord)
	}
	if items[0].NormalizedKey != "gauge" {
		t.Fatalf("NormalizedKey = %q, want gauge", items[0].NormalizedKey)
	}
}
