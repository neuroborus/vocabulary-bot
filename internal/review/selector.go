package review

import (
	"math"
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

// SelectionOptions tunes review word priority.
type SelectionOptions struct {
	// DocumentPushFactor scales difficulty for spreadsheet-only words and PocketBook
	// PDF notes labeled Document. 0.7 means 30% lower push priority.
	DocumentPushFactor float64
	// BookPushFactor scales difficulty for PocketBook book anchors.
	BookPushFactor float64
}

func (o SelectionOptions) effectiveDocumentPushFactor() float64 {
	if o.DocumentPushFactor <= 0 {
		return 1
	}

	return o.DocumentPushFactor
}

func (o SelectionOptions) effectiveBookPushFactor() float64 {
	if o.BookPushFactor <= 0 {
		return 1
	}

	return o.BookPushFactor
}

const pushPriorityBaseline = 1.0

// DifficultyScore estimates how hard a word is for the user.
// Higher values mean the word was marked Hard more often than Easy.
func DifficultyScore(item vocabulary.Item) int {
	return item.Review.HardCount - item.Review.EasyCount
}

// PushPriorityScore combines difficulty with a baseline so source factors still
// matter for never-reviewed words, then scales by the source-specific factor.
func PushPriorityScore(item vocabulary.Item, opts SelectionOptions) float64 {
	score := float64(DifficultyScore(item)) + pushPriorityBaseline

	switch {
	case vocabulary.IsDocumentPushCandidate(item):
		return score * opts.effectiveDocumentPushFactor()
	case vocabulary.IsBookPushCandidate(item):
		return score * opts.effectiveBookPushFactor()
	default:
		return score
	}
}

// SelectNext picks the next review word.
// Priority:
// 1. Higher push priority score: (difficulty + 1) scaled by source factor
// 2. Due words over not-yet-due words
// 3. Earlier dueAt
// 4. Older last push / never pushed
// 5. Older createdAt
func SelectNext(items []vocabulary.Item, now time.Time, opts ...SelectionOptions) (vocabulary.Item, bool) {
	options := SelectionOptions{DocumentPushFactor: 1, BookPushFactor: 1}
	if len(opts) > 0 {
		options = opts[0]
	}

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
		return compareSelectPriority(eligible[i], eligible[j], now, options)
	})

	return eligible[0], true
}

func compareSelectPriority(left, right vocabulary.Item, now time.Time, opts SelectionOptions) bool {
	leftScore := PushPriorityScore(left, opts)
	rightScore := PushPriorityScore(right, opts)
	if !scoresEqual(leftScore, rightScore) {
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

func scoresEqual(left, right float64) bool {
	return math.Abs(left-right) < 1e-9
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
