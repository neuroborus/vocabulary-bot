package review

import (
	"sort"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func IsEligible(item vocabulary.Item) bool {
	return item.Enabled && item.Review.Enabled
}

func IsDue(item vocabulary.Item, now time.Time) bool {
	if !IsEligible(item) {
		return false
	}
	if item.Review.DueAt == nil {
		return true
	}

	return !item.Review.DueAt.After(now.UTC())
}

// DifficultyScore estimates how hard a word is for the user.
// Higher values mean the word was marked Hard more often than Easy.
func DifficultyScore(item vocabulary.Item) int {
	return item.Review.HardCount - item.Review.EasyCount
}

// SelectNext picks the next review word.
// Priority:
// 1. Higher difficulty score (hardCount - easyCount)
// 2. Due words over not-yet-due words
// 3. Earlier dueAt
// 4. Older last push / never pushed
// 5. Older createdAt
func SelectNext(items []vocabulary.Item, now time.Time) (vocabulary.Item, bool) {
	now = now.UTC()
	eligible := make([]vocabulary.Item, 0, len(items))
	for _, item := range items {
		if IsEligible(item) {
			eligible = append(eligible, item)
		}
	}
	if len(eligible) == 0 {
		return vocabulary.Item{}, false
	}

	sort.Slice(eligible, func(i, j int) bool {
		return compareSelectPriority(eligible[i], eligible[j], now)
	})

	return eligible[0], true
}

func compareSelectPriority(left, right vocabulary.Item, now time.Time) bool {
	leftScore := DifficultyScore(left)
	rightScore := DifficultyScore(right)
	if leftScore != rightScore {
		return leftScore > rightScore
	}

	leftDue := IsDue(left, now)
	rightDue := IsDue(right, now)
	if leftDue != rightDue {
		return leftDue
	}

	leftDueNil := left.Review.DueAt == nil
	rightDueNil := right.Review.DueAt == nil
	if leftDueNil != rightDueNil {
		return leftDueNil
	}
	if !leftDueNil && left.Review.DueAt.Before(*right.Review.DueAt) {
		return true
	}
	if !leftDueNil && right.Review.DueAt.Before(*left.Review.DueAt) {
		return false
	}

	if lastPushedEqual(left, right) {
		if left.CreatedAt.Equal(right.CreatedAt) {
			return left.NormalizedKey < right.NormalizedKey
		}
		return left.CreatedAt.Before(right.CreatedAt)
	}
	if compareLastPushed(left, right) {
		return true
	}

	return false
}

func lastPushedEqual(left, right vocabulary.Item) bool {
	leftNil := left.Review.LastPushedAt == nil
	rightNil := right.Review.LastPushedAt == nil
	if leftNil || rightNil {
		return leftNil && rightNil
	}

	return left.Review.LastPushedAt.Equal(*right.Review.LastPushedAt)
}

func compareLastPushed(left, right vocabulary.Item) bool {
	leftNil := left.Review.LastPushedAt == nil
	rightNil := right.Review.LastPushedAt == nil
	if leftNil != rightNil {
		return leftNil
	}
	if leftNil {
		return false
	}
	if left.Review.LastPushedAt.Equal(*right.Review.LastPushedAt) {
		return false
	}

	return left.Review.LastPushedAt.Before(*right.Review.LastPushedAt)
}
