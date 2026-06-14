package review

import (
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func MarkPushed(item *vocabulary.Item, now time.Time) {
	value := now.UTC()
	item.Review.LastPushedAt = &value
	item.Review.PushCount++
}

func MarkEasy(item *vocabulary.Item, now time.Time) {
	MarkPushed(item, now)
	item.Review.EasyCount++
	if item.Review.IntervalDays <= 0 {
		item.Review.IntervalDays = 1
	}
	item.Review.IntervalDays *= 2

	dueAt := now.UTC().AddDate(0, 0, item.Review.IntervalDays)
	item.Review.DueAt = &dueAt
}

func MarkHard(item *vocabulary.Item, now time.Time) {
	MarkPushed(item, now)
	item.Review.HardCount++
	item.Review.IntervalDays = 1

	dueAt := now.UTC().AddDate(0, 0, item.Review.IntervalDays)
	item.Review.DueAt = &dueAt
}
