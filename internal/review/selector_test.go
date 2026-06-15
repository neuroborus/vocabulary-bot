package review

import (
	"math/rand"
	"testing"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func testRNG(seed int64) SelectionOptions {
	return SelectionOptions{Rand: rand.New(rand.NewSource(seed))}
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

	if _, ok := SelectNext(items, now, testRNG(1)); ok {
		t.Fatal("SelectNext() = true, want false")
	}
}

func TestSelectNextSkipsNotYetDueWhenBetterDueCandidatesExist(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	overdue := now.Add(-2 * time.Hour)
	future := now.Add(48 * time.Hour)
	lastPush := now.Add(-30 * time.Minute)

	items := []vocabulary.Item{
		{
			NormalizedKey: "hard-later",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled:      true,
				DueAt:        &future,
				HardCount:    5,
				LastPushedAt: &lastPush,
			},
			CreatedAt: now.Add(-24 * time.Hour),
		},
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
	}

	counts := map[string]int{}
	for seed := int64(0); seed < 200; seed++ {
		selected, ok := SelectNext(items, now, testRNG(seed))
		if !ok {
			t.Fatal("SelectNext() = false, want true")
		}
		counts[selected.NormalizedKey]++
	}

	if counts["easy-due"] <= counts["hard-later"] {
		t.Fatalf("due selection counts = %#v, want easy-due favored over not-yet-due hard-later", counts)
	}
}

func TestSelectNextFavorsHigherDifficultyAmongDueWords(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	overdue := now.Add(-time.Hour)

	items := []vocabulary.Item{
		{
			NormalizedKey: "easy-due",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled:   true,
				DueAt:     &overdue,
				EasyCount: 3,
			},
			CreatedAt: now.Add(-72 * time.Hour),
		},
		{
			NormalizedKey: "hard-due",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled:   true,
				DueAt:     &overdue,
				HardCount: 4,
			},
			CreatedAt: now.Add(-24 * time.Hour),
		},
	}

	counts := map[string]int{}
	for seed := int64(0); seed < 300; seed++ {
		selected, ok := SelectNext(items, now, testRNG(seed))
		if !ok {
			t.Fatal("SelectNext() = false, want true")
		}
		counts[selected.NormalizedKey]++
	}

	if counts["hard-due"] <= counts["easy-due"] {
		t.Fatalf("selection counts = %#v, want hard-due favored", counts)
	}
}

func TestSelectNextDeprioritizesDocumentWords(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	overdue := now.Add(-time.Hour)
	opts := SelectionOptions{
		DocumentPushFactor: 0.7,
		BookPushFactor:     1,
		Rand:               rand.New(rand.NewSource(42)),
	}

	items := []vocabulary.Item{
		{
			NormalizedKey: "spreadsheet-hard",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled:   true,
				DueAt:     &overdue,
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
				DueAt:     &overdue,
				HardCount: 8,
			},
			Anchors: []vocabulary.SourceAnchor{{
				Source:      vocabulary.SourcePocketBook,
				SourceLabel: "Necromancer — Fred Saberhagen",
			}},
			CreatedAt: now.Add(-72 * time.Hour),
		},
	}

	counts := map[string]int{}
	for seed := int64(0); seed < 300; seed++ {
		opts.Rand = rand.New(rand.NewSource(seed))
		selected, ok := SelectNext(items, now, opts)
		if !ok {
			t.Fatal("SelectNext() = false, want true")
		}
		counts[selected.NormalizedKey]++
	}

	if counts["book-moderate"] <= counts["spreadsheet-hard"] {
		t.Fatalf("selection counts = %#v, want book-moderate favored over spreadsheet-hard", counts)
	}
}

func TestSelectNextAppliesBookPushFactor(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	overdue := now.Add(-time.Hour)
	opts := SelectionOptions{DocumentPushFactor: 1, BookPushFactor: 0.7}

	items := []vocabulary.Item{
		{
			NormalizedKey: "book-hard",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled:   true,
				DueAt:     &overdue,
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
				DueAt:     &overdue,
				HardCount: 8,
			},
			Anchors: []vocabulary.SourceAnchor{{
				Source: vocabulary.SourceGoogleSheet,
			}},
			CreatedAt: now.Add(-72 * time.Hour),
		},
	}

	counts := map[string]int{}
	for seed := int64(0); seed < 300; seed++ {
		opts.Rand = rand.New(rand.NewSource(seed))
		selected, ok := SelectNext(items, now, opts)
		if !ok {
			t.Fatal("SelectNext() = false, want true")
		}
		counts[selected.NormalizedKey]++
	}

	if counts["sheet-moderate"] <= counts["book-hard"] {
		t.Fatalf("selection counts = %#v, want sheet-moderate favored over discounted book-hard", counts)
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

	counts := map[string]int{}
	for seed := int64(0); seed < 300; seed++ {
		opts.Rand = rand.New(rand.NewSource(seed))
		selected, ok := SelectNext(items, now, opts)
		if !ok {
			t.Fatal("SelectNext() = false, want true")
		}
		counts[selected.NormalizedKey]++
	}

	if counts["sheet-fresh"] <= counts["book-reviewed"] {
		t.Fatalf("selection counts = %#v, want sheet-fresh favored", counts)
	}
}

func TestSelectNextDownWeightsRecentlyPushedWord(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	lastPush := now.Add(-10 * time.Minute)

	items := []vocabulary.Item{
		{
			NormalizedKey: "recent-hard",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled:      true,
				HardCount:    3,
				LastPushedAt: &lastPush,
			},
			CreatedAt: now.Add(-24 * time.Hour),
		},
		{
			NormalizedKey: "other-due",
			Enabled:       true,
			Review: vocabulary.ReviewState{
				Enabled: true,
			},
			CreatedAt: now.Add(-72 * time.Hour),
		},
	}

	counts := map[string]int{}
	for seed := int64(0); seed < 200; seed++ {
		selected, ok := SelectNext(items, now, testRNG(seed))
		if !ok {
			t.Fatal("SelectNext() = false, want true")
		}
		counts[selected.NormalizedKey]++
	}

	if counts["recent-hard"] >= counts["other-due"] {
		t.Fatalf("selection counts = %#v, want recently pushed word down-weighted", counts)
	}
}

func TestSelectWeightUsesDocumentAndBookFactors(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	opts := SelectionOptions{DocumentPushFactor: 0.7, BookPushFactor: 0.1}

	book := vocabulary.Item{
		Enabled: true,
		Review:  vocabulary.ReviewState{Enabled: true, HardCount: 1},
		Anchors: []vocabulary.SourceAnchor{{
			Source:      vocabulary.SourcePocketBook,
			SourceLabel: "Necromancer — Fred Saberhagen",
		}},
	}
	sheet := vocabulary.Item{
		Enabled: true,
		Review:  vocabulary.ReviewState{Enabled: true},
		Anchors: []vocabulary.SourceAnchor{{
			Source: vocabulary.SourceGoogleSheet,
		}},
	}

	bookWeight := SelectWeight(book, now, opts)
	sheetWeight := SelectWeight(sheet, now, opts)
	if bookWeight >= sheetWeight {
		t.Fatalf("weights book=%v sheet=%v, want sheet heavier than discounted book", bookWeight, sheetWeight)
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

func TestWeightedPickRespectsDistribution(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(7))
	counts := [2]int{}
	for index := 0; index < 1000; index++ {
		pick := weightedPick([]float64{3, 1}, rng)
		counts[pick]++
	}

	if counts[0] <= counts[1] {
		t.Fatalf("weighted picks = %#v, want heavier index 0", counts)
	}
}
