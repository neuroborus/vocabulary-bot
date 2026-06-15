package review

import (
	"math"
	"math/rand"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

const (
	pushPriorityBaseline   = 1.0
	notDueWeightMultiplier = 0.12
	recentPushHourPenalty  = 0.08
	recentPushDayPenalty   = 0.35
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

// SelectionOptions tunes weighted random review selection.
type SelectionOptions struct {
	// DocumentPushFactor scales selection weight for spreadsheet-only words and
	// PocketBook PDF notes labeled Document. 0.7 means 30% lower weight.
	DocumentPushFactor float64
	// BookPushFactor scales selection weight for PocketBook book anchors.
	BookPushFactor float64
	// Rand provides randomness for SelectNext. When nil, a time-seeded source is used.
	Rand *rand.Rand
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

func (o SelectionOptions) rng() *rand.Rand {
	if o.Rand != nil {
		return o.Rand
	}

	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

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

// SelectWeight estimates how likely a word is to be picked for /push.
// Higher difficulty and source factors increase weight; due words are favored;
// not-yet-due and very recently pushed words are down-weighted but not excluded.
func SelectWeight(item vocabulary.Item, now time.Time, opts SelectionOptions) float64 {
	weight := PushPriorityScore(item, opts)
	if weight <= 0 {
		weight = pushPriorityBaseline
	}

	weight *= scheduleWeightMultiplier(item, now)
	weight *= recencyWeightMultiplier(item, now)

	if weight < 0 {
		return 0
	}

	return weight
}

func scheduleWeightMultiplier(item vocabulary.Item, now time.Time) float64 {
	if !IsDue(item, now) {
		return notDueWeightMultiplier
	}
	if item.Review.DueAt == nil {
		return 1
	}

	overdue := now.UTC().Sub(item.Review.DueAt.UTC())
	if overdue <= 0 {
		return 1
	}

	days := overdue.Hours() / 24
	return 1 + math.Min(days*0.5, 2)
}

func recencyWeightMultiplier(item vocabulary.Item, now time.Time) float64 {
	if item.Review.LastPushedAt == nil {
		return 1
	}

	since := now.UTC().Sub(item.Review.LastPushedAt.UTC())
	switch {
	case since < time.Hour:
		return recentPushHourPenalty
	case since < 24*time.Hour:
		return recentPushDayPenalty
	default:
		return 1
	}
}

// SelectNext picks a review word at random with probability proportional to SelectWeight.
func SelectNext(items []vocabulary.Item, now time.Time, opts ...SelectionOptions) (vocabulary.Item, bool) {
	options := SelectionOptions{DocumentPushFactor: 1, BookPushFactor: 1}
	if len(opts) > 0 {
		options = opts[0]
	}

	now = now.UTC()
	eligible := make([]vocabulary.Item, 0, len(items))
	weights := make([]float64, 0, len(items))
	for _, item := range items {
		if !IsEligible(item) {
			continue
		}
		weight := SelectWeight(item, now, options)
		if weight <= 0 {
			continue
		}
		eligible = append(eligible, item)
		weights = append(weights, weight)
	}
	if len(eligible) == 0 {
		return vocabulary.Item{}, false
	}

	index := weightedPick(weights, options.rng())
	return eligible[index], true
}

func weightedPick(weights []float64, rng *rand.Rand) int {
	if len(weights) == 1 {
		return 0
	}

	var total float64
	for _, weight := range weights {
		total += weight
	}
	if total <= 0 {
		return rng.Intn(len(weights))
	}

	threshold := rng.Float64() * total
	var running float64
	for index, weight := range weights {
		running += weight
		if threshold < running {
			return index
		}
	}

	return len(weights) - 1
}
