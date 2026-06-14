package vocabulary_test

import (
	"context"
	"testing"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/storage/memory"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func TestHasBookLikeContextDetectsStoredSentence(t *testing.T) {
	t.Parallel()

	if !vocabulary.HasBookLikeContext([]string{"He had to lean against the wall."}) {
		t.Fatal("expected book-like sentence to be detected")
	}
}

func TestHasBookLikeContextIgnoresUsageExamples(t *testing.T) {
	t.Parallel()

	if vocabulary.HasBookLikeContext([]string{"to lean on a friend's advice - полагаться на совет друга"}) {
		t.Fatal("usage example should not count as stored book context")
	}
}

func TestHasBookLikeContextDetectsBookSentenceWithDashes(t *testing.T) {
	t.Parallel()

	sentence := "But this man - this Guild-master - was nothing so simple as a crackpot."
	if !vocabulary.HasBookLikeContext([]string{sentence}) {
		t.Fatal("book sentence with dashes should count as stored book context")
	}
}

func TestIsUsageExampleLineRejectsEnglishDashSentence(t *testing.T) {
	t.Parallel()

	sentence := "But this man - this Guild-master - was nothing so simple as a crackpot."
	if vocabulary.IsUsageExampleLine(sentence) {
		t.Fatal("English book sentence with dashes should not be a usage example")
	}
}

func TestHasBookLikeContextIgnoresBareHeadword(t *testing.T) {
	t.Parallel()

	if vocabulary.HasBookLikeContext([]string{"lean"}) {
		t.Fatal("bare headword should not count as stored book context")
	}
}

func TestServiceHasStoredBookLikeContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := memory.NewVocabularyRepository()
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	service := vocabulary.NewService(repository, func() time.Time { return now })

	created, err := service.MergeDraft(ctx, vocabulary.Draft{
		Source:  vocabulary.SourcePocketBook,
		RawWord: "lean",
		Contexts: []string{
			"He had to lean against the wall.",
		},
	})
	if err != nil {
		t.Fatalf("merge draft: %v", err)
	}
	if !created.Created {
		t.Fatal("expected created item")
	}

	has, err := service.HasStoredBookLikeContext(ctx, "lean")
	if err != nil {
		t.Fatalf("HasStoredBookLikeContext() error = %v", err)
	}
	if !has {
		t.Fatal("HasStoredBookLikeContext() = false, want true")
	}
}

func TestServiceHasStoredBookLikeContextDetectsDashSentence(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := memory.NewVocabularyRepository()
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	service := vocabulary.NewService(repository, func() time.Time { return now })

	sentence := "But this man - this Guild-master - was nothing so simple as a crackpot."
	created, err := service.MergeDraft(ctx, vocabulary.Draft{
		Source:   vocabulary.SourcePocketBook,
		RawWord:  "crackpot",
		Contexts: []string{sentence},
	})
	if err != nil {
		t.Fatalf("merge draft: %v", err)
	}
	if !created.Created {
		t.Fatal("expected created item")
	}

	has, err := service.HasStoredBookLikeContext(ctx, "crackpot")
	if err != nil {
		t.Fatalf("HasStoredBookLikeContext() error = %v", err)
	}
	if !has {
		t.Fatal("HasStoredBookLikeContext() = false, want true for dash sentence")
	}
}
