package review

import (
	"testing"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func TestMarkPushedCountsDeliveries(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	item := vocabulary.Item{}
	MarkPushed(&item, now)
	MarkPushed(&item, now.Add(24*time.Hour))
	if item.Review.PushCount != 2 {
		t.Fatalf("PushCount = %d, want 2", item.Review.PushCount)
	}
	if item.Review.LastPushedAt == nil || !item.Review.LastPushedAt.Equal(now.Add(24*time.Hour)) || item.Review.LastPushedAt.Location() != time.UTC {
		t.Fatalf("LastPushedAt = %v, want latest delivery in UTC", item.Review.LastPushedAt)
	}
}

func TestRatingsOnlyChangeReviewSchedule(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	lastPush := now.Add(-6 * time.Hour).UTC()
	for _, tt := range []struct {
		name     string
		mark     func(*vocabulary.Item, time.Time)
		interval int
		wantDays int
		wantEasy int
		wantHard int
	}{
		{name: "first easy", mark: MarkEasy, wantDays: 2, wantEasy: 1},
		{name: "easy doubles interval", mark: MarkEasy, interval: 4, wantDays: 8, wantEasy: 1},
		{name: "hard resets interval", mark: MarkHard, interval: 4, wantDays: 1, wantHard: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, pushed := range []bool{false, true} {
				item := vocabulary.Item{Review: vocabulary.ReviewState{IntervalDays: tt.interval}}
				if pushed {
					item.Review.LastPushedAt = &lastPush
					item.Review.PushCount = 7
				}
				before := item.Review
				tt.mark(&item, now)
				if item.Review.PushCount != before.PushCount || item.Review.LastPushedAt != before.LastPushedAt {
					t.Fatalf("rating changed delivery state: before = %+v, after = %+v", before, item.Review)
				}
				if item.Review.EasyCount != tt.wantEasy || item.Review.HardCount != tt.wantHard || item.Review.IntervalDays != tt.wantDays {
					t.Fatalf("review = %+v, want easy/hard/interval = %d/%d/%d", item.Review, tt.wantEasy, tt.wantHard, tt.wantDays)
				}
				wantDue := now.UTC().AddDate(0, 0, tt.wantDays)
				if item.Review.DueAt == nil || !item.Review.DueAt.Equal(wantDue) || item.Review.DueAt.Location() != time.UTC {
					t.Fatalf("DueAt = %v, want %v in UTC", item.Review.DueAt, wantDue)
				}
			}
		})
	}
}
