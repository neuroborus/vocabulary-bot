package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/storage/memory"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func TestVocabularyRepositoryReplaceRenamesExistingItem(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := memory.NewVocabularyRepository()

	item := memoryTestItem("decelerate")
	if err := repository.Create(ctx, item); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	replacement := memoryTestItem("slowdown")
	replacement.DisplayWord = "slow down"
	if err := repository.Replace(ctx, replacement, "decelerate"); err != nil {
		t.Fatalf("Replace() error = %v", err)
	}

	oldMatches, err := repository.FindByLookupKeys(ctx, []string{"decelerate"})
	if err != nil {
		t.Fatalf("FindByLookupKeys(old) error = %v", err)
	}
	if len(oldMatches) != 0 {
		t.Fatalf("old key matches = %d, want 0", len(oldMatches))
	}

	newMatches, err := repository.FindByLookupKeys(ctx, []string{"slowdown"})
	if err != nil {
		t.Fatalf("FindByLookupKeys(new) error = %v", err)
	}
	if len(newMatches) != 1 {
		t.Fatalf("new key matches = %d, want 1", len(newMatches))
	}
	if newMatches[0].DisplayWord != "slow down" {
		t.Fatalf("DisplayWord = %q, want %q", newMatches[0].DisplayWord, "slow down")
	}
}

func TestVocabularyRepositoryReplaceMissingPreviousKey(t *testing.T) {
	t.Parallel()

	repository := memory.NewVocabularyRepository()

	if err := repository.Replace(context.Background(), memoryTestItem("decelerate"), "missing"); err == nil {
		t.Fatal("Replace() error = nil, want missing previous key error")
	}
}

func TestVocabularyRepositoryReplaceRejectsCollision(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := memory.NewVocabularyRepository()

	if err := repository.Create(ctx, memoryTestItem("carve")); err != nil {
		t.Fatalf("Create(carve) error = %v", err)
	}
	if err := repository.Create(ctx, memoryTestItem("gauge")); err != nil {
		t.Fatalf("Create(gauge) error = %v", err)
	}

	replacement := memoryTestItem("gauge")
	replacement.DisplayWord = "replacement gauge"
	if err := repository.Replace(ctx, replacement, "carve"); err == nil {
		t.Fatal("Replace() error = nil, want collision error")
	}

	items, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2 after rejected replace", len(items))
	}

	matches, err := repository.FindByLookupKeys(ctx, []string{"gauge"})
	if err != nil {
		t.Fatalf("FindByLookupKeys() error = %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("gauge matches = %d, want 1", len(matches))
	}
	if matches[0].DisplayWord != "gauge" {
		t.Fatalf("DisplayWord = %q, want original gauge", matches[0].DisplayWord)
	}
}

func memoryTestItem(normalizedKey string) vocabulary.Item {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	return vocabulary.Item{
		NormalizedKey: normalizedKey,
		LookupKeys:    []string{normalizedKey},
		DisplayWord:   normalizedKey,
		Forms: []vocabulary.Form{
			{
				Value:           normalizedKey,
				NormalizedValue: normalizedKey,
				LookupKeys:      []string{normalizedKey},
				Count:           1,
				FirstSeenAt:     now,
				LastSeenAt:      now,
			},
		},
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
