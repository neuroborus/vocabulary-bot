package review

import (
	"testing"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func TestSelectNextPrefersHigherDifficultyScore(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	overdue := now.Add(-2 * time.Hour)
	future := now.Add(48 * time.Hour)

	items := []vocabulary.Item{
		{
			NormalizedKey: "easy-due",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled:   true,
				DueAt:     &overdue,
				EasyCount: 4,
			},
			CreatedAt: now.Add(-72 * time.Hour),
		},
		{
			NormalizedKey: "hard-later",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled:   true,
				DueAt:     &future,
				HardCount: 3,
				EasyCount: 1,
			},
			CreatedAt: now.Add(-24 * time.Hour),
		},
	}

	selected, ok := SelectNext(items, now)
	if !ok {
		t.Fatal("SelectNext() = false, want true")
	}
	if selected.NormalizedKey != "hard-later" {
		t.Fatalf("selected = %q, want hard-later", selected.NormalizedKey)
	}
}

func TestSelectNextUsesDueAsTieBreakerWhenScoresEqual(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	dueSoon := now.Add(-time.Hour)
	dueLater := now.Add(24 * time.Hour)

	items := []vocabulary.Item{
		{
			NormalizedKey: "later",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled: true,
				DueAt:   &dueLater,
			},
			CreatedAt: now.Add(-48 * time.Hour),
		},
		{
			NormalizedKey: "due",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled: true,
				DueAt:   &dueSoon,
			},
			CreatedAt: now.Add(-24 * time.Hour),
		},
	}

	selected, ok := SelectNext(items, now)
	if !ok {
		t.Fatal("SelectNext() = false, want true")
	}
	if selected.NormalizedKey != "due" {
		t.Fatalf("selected = %q, want due", selected.NormalizedKey)
	}
}

func TestSelectNextFallsBackToOldestEligibleWhenScoresAndDueEqual(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	future := now.Add(48 * time.Hour)

	items := []vocabulary.Item{
		{
			NormalizedKey: "newer",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled: true,
				DueAt:   &future,
			},
			CreatedAt: now.Add(-24 * time.Hour),
		},
		{
			NormalizedKey: "older",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled: true,
				DueAt:   &future,
			},
			CreatedAt: now.Add(-72 * time.Hour),
		},
	}

	selected, ok := SelectNext(items, now)
	if !ok {
		t.Fatal("SelectNext() = false, want true")
	}
	if selected.NormalizedKey != "older" {
		t.Fatalf("selected = %q, want older", selected.NormalizedKey)
	}
}

func TestSelectNextSkipsDisabledWords(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	items := []vocabulary.Item{
		{
			NormalizedKey: "disabled",
			Enabled:       false,
			Review:        vocabulary.ReviewState{Enabled: true},
			CreatedAt:     now,
		},
	}

	if _, ok := SelectNext(items, now); ok {
		t.Fatal("SelectNext() = true, want false")
	}
}

func TestSelectNextDeprioritizesDocumentWords(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	future := now.Add(48 * time.Hour)
	opts := SelectionOptions{DocumentPushFactor: 0.7, BookPushFactor: 1}

	items := []vocabulary.Item{
		{
			NormalizedKey: "spreadsheet-hard",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled:   true,
				DueAt:     &future,
				HardCount: 10,
			},
			Anchors: []vocabulary.SourceAnchor{{
				Source: vocabulary.SourceGoogleSheet,
			}},
			CreatedAt: now.Add(-24 * time.Hour),
		},
		{
			NormalizedKey: "book-moderate",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled:   true,
				DueAt:     &future,
				HardCount: 8,
			},
			Anchors: []vocabulary.SourceAnchor{{
				Source:      vocabulary.SourcePocketBook,
				SourceLabel: "Necromancer — Fred Saberhagen",
			}},
			CreatedAt: now.Add(-72 * time.Hour),
		},
	}

	selected, ok := SelectNext(items, now, opts)
	if !ok {
		t.Fatal("SelectNext() = false, want true")
	}
	if selected.NormalizedKey != "book-moderate" {
		t.Fatalf("selected = %q, want book-moderate", selected.NormalizedKey)
	}
}

func TestSelectNextAppliesBookPushFactor(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	future := now.Add(48 * time.Hour)
	opts := SelectionOptions{DocumentPushFactor: 1, BookPushFactor: 0.7}

	items := []vocabulary.Item{
		{
			NormalizedKey: "book-hard",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled:   true,
				DueAt:     &future,
				HardCount: 10,
			},
			Anchors: []vocabulary.SourceAnchor{{
				Source:      vocabulary.SourcePocketBook,
				SourceLabel: "Necromancer — Fred Saberhagen",
			}},
			CreatedAt: now.Add(-24 * time.Hour),
		},
		{
			NormalizedKey: "sheet-moderate",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled:   true,
				DueAt:     &future,
				HardCount: 8,
			},
			Anchors: []vocabulary.SourceAnchor{{
				Source: vocabulary.SourceGoogleSheet,
			}},
			CreatedAt: now.Add(-72 * time.Hour),
		},
	}

	selected, ok := SelectNext(items, now, opts)
	if !ok {
		t.Fatal("SelectNext() = false, want true")
	}
	if selected.NormalizedKey != "sheet-moderate" {
		t.Fatalf("selected = %q, want sheet-moderate", selected.NormalizedKey)
	}
}

func TestSelectNextPrefersSpreadsheetWithLowBookFactorAndZeroDifficulty(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	opts := SelectionOptions{DocumentPushFactor: 1, BookPushFactor: 0.1}

	items := []vocabulary.Item{
		{
			NormalizedKey: "book-reviewed",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled:   true,
				HardCount: 1,
			},
			Anchors: []vocabulary.SourceAnchor{{
				Source:      vocabulary.SourcePocketBook,
				SourceLabel: "Necromancer — Fred Saberhagen",
			}},
			CreatedAt: now.Add(-24 * time.Hour),
		},
		{
			NormalizedKey: "sheet-fresh",
			Enabled:       true,
			Review:        vocabulary.ReviewState{Enabled: true},
			Anchors: []vocabulary.SourceAnchor{{
				Source: vocabulary.SourceGoogleSheet,
			}},
			CreatedAt: now.Add(-72 * time.Hour),
		},
	}

	selected, ok := SelectNext(items, now, opts)
	if !ok {
		t.Fatal("SelectNext() = false, want true")
	}
	if selected.NormalizedKey != "sheet-fresh" {
		t.Fatalf("selected = %q, want sheet-fresh", selected.NormalizedKey)
	}
}

func TestDifficultyScore(t *testing.T) {
	t.Parallel()

	score := DifficultyScore(vocabulary.Item{
		Review: vocabulary.ReviewState{
			HardCount: 3,
			EasyCount: 1,
		},
	})
	if score != 2 {
		t.Fatalf("DifficultyScore() = %d, want 2", score)
	}
}
