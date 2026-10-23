package review

import (
	"fmt"
	"math"
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

func TestSelectNextNeverSelectsFutureDueWords(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	overdue := now.Add(-2 * time.Hour)
	future := now.Add(48 * time.Hour)
	lastPush := now.Add(-72 * time.Hour)

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

	if counts["hard-later"] != 0 {
		t.Fatalf("selection counts = %#v, future-due words must be excluded", counts)
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

func TestSelectNextExcludesRecentlyPushedWord(t *testing.T) {
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

	if counts["recent-hard"] != 0 {
		t.Fatalf("selection counts = %#v, recently pushed words must be excluded", counts)
	}
}

func TestSelectionEligibilityBoundaries(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	for _, tt := range []struct {
		name         string
		dueOffset    time.Duration
		pushOffset   time.Duration
		itemDisabled bool
		reviewOff    bool
		wantEligible bool
	}{
		{name: "due now and cooldown elapsed", pushOffset: -24 * time.Hour, wantEligible: true},
		{name: "overdue and past cooldown", dueOffset: -time.Hour, pushOffset: -25 * time.Hour, wantEligible: true},
		{name: "future due", dueOffset: time.Nanosecond, pushOffset: -25 * time.Hour},
		{name: "just inside cooldown", pushOffset: -24*time.Hour + time.Nanosecond},
		{name: "just outside cooldown", pushOffset: -24*time.Hour - time.Nanosecond, wantEligible: true},
		{name: "future delivery timestamp", pushOffset: time.Hour},
		{name: "item disabled", pushOffset: -25 * time.Hour, itemDisabled: true},
		{name: "review disabled", pushOffset: -25 * time.Hour, reviewOff: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dueAt := now.Add(tt.dueOffset).UTC()
			lastPush := now.Add(tt.pushOffset).UTC()
			item := vocabulary.Item{
				Enabled: !tt.itemDisabled,
				Review: vocabulary.ReviewState{
					Enabled:      !tt.reviewOff,
					DueAt:        &dueAt,
					LastPushedAt: &lastPush,
					HardCount:    100,
				},
			}
			weight := SelectWeight(item, now, SelectionOptions{})
			if (weight > 0) != tt.wantEligible {
				t.Fatalf("weight = %v, want eligible = %v", weight, tt.wantEligible)
			}
			if _, ok := SelectNext([]vocabulary.Item{item}, now, testRNG(1)); ok != tt.wantEligible {
				t.Fatalf("SelectNext() = %v, want %v", ok, tt.wantEligible)
			}
		})
	}
}

func TestSelectNextEmptyEligiblePool(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	recent := now.Add(-time.Hour)
	items := []vocabulary.Item{
		{Enabled: true, Review: vocabulary.ReviewState{Enabled: true, DueAt: &future}},
		{Enabled: true, Review: vocabulary.ReviewState{Enabled: true, LastPushedAt: &recent}},
		{Enabled: false, Review: vocabulary.ReviewState{Enabled: true}},
	}
	for _, pool := range [][]vocabulary.Item{nil, items} {
		if _, ok := SelectNext(pool, now, testRNG(1)); ok {
			t.Fatal("SelectNext() selected a word from an empty eligible pool")
		}
	}
}

func TestSelectWeightRecognizesHistoricalReviewMarkers(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	lastPush := now.Add(-48 * time.Hour)
	for _, tt := range []struct {
		name   string
		review vocabulary.ReviewState
		want   float64
	}{
		{name: "unseen", want: 10},
		{name: "only timestamp", review: vocabulary.ReviewState{LastPushedAt: &lastPush}, want: 1},
		{name: "only count", review: vocabulary.ReviewState{PushCount: 1}, want: 1},
		{name: "both delivery markers", review: vocabulary.ReviewState{LastPushedAt: &lastPush, PushCount: 1}, want: 1},
		{name: "legacy easy rating", review: vocabulary.ReviewState{EasyCount: 1}, want: 1},
		{name: "legacy hard rating", review: vocabulary.ReviewState{HardCount: 1}, want: 2},
		{name: "legacy schedule", review: vocabulary.ReviewState{DueAt: &now}, want: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.review.Enabled = true
			item := vocabulary.Item{Enabled: true, Review: tt.review}
			if got := SelectWeight(item, now, SelectionOptions{}); got != tt.want {
				t.Fatalf("SelectWeight() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectNextStronglyFavorsUnseenWhileKeepingReviewedSelectable(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	items := []vocabulary.Item{
		{NormalizedKey: "new", Enabled: true, Review: vocabulary.ReviewState{Enabled: true}},
		{NormalizedKey: "reviewed", Enabled: true, Review: vocabulary.ReviewState{Enabled: true, PushCount: 1, DueAt: &now}},
	}
	opts := testRNG(42)
	const draws = 10000
	newCount := 0
	for range draws {
		item, ok := SelectNext(items, now, opts)
		if !ok {
			t.Fatal("SelectNext() = false, want true")
		}
		if item.NormalizedKey == "new" {
			newCount++
		}
	}
	share := float64(newCount) / draws
	if share < 0.88 || share > 0.94 {
		t.Fatalf("unseen share = %.3f, want about 10/11 with reviewed words still selected", share)
	}
}

func TestSourceFactorsApplyToMasteredWords(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	opts := SelectionOptions{DocumentPushFactor: 0.7, BookPushFactor: 0.1, Rand: rand.New(rand.NewSource(7))}
	for _, easyCount := range []int{0, 1, 10} {
		t.Run(fmt.Sprint(easyCount), func(t *testing.T) {
			book := vocabulary.Item{
				NormalizedKey: "book", Enabled: true,
				Review:  vocabulary.ReviewState{Enabled: true, EasyCount: easyCount, PushCount: 1},
				Anchors: []vocabulary.SourceAnchor{{Source: vocabulary.SourcePocketBook, SourceLabel: "Example Book"}},
			}
			document := book
			document.NormalizedKey = "document"
			document.Anchors = []vocabulary.SourceAnchor{{Source: vocabulary.SourceGoogleSheet}}
			bookWeight := SelectWeight(book, now, opts)
			documentWeight := SelectWeight(document, now, opts)
			if math.Abs(bookWeight/documentWeight-1.0/7) > 1e-12 {
				t.Fatalf("book/document weights = %v/%v, want ratio 1/7", bookWeight, documentWeight)
			}
			bookCount := 0
			const draws = 10000
			for range draws {
				item, ok := SelectNext([]vocabulary.Item{book, document}, now, opts)
				if !ok {
					t.Fatal("SelectNext() = false, want true")
				}
				if item.NormalizedKey == "book" {
					bookCount++
				}
			}
			share := float64(bookCount) / draws
			if share < 0.10 || share > 0.15 {
				t.Fatalf("book share = %.3f, want about 1/8", share)
			}
		})
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
