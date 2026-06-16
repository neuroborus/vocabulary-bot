package save

import (
	"context"
	"testing"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/openai"
	"github.com/neuroborus/vocabulary-bot/internal/storage/memory"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type fakeStructurer struct {
	fields openai.VocabularyFields
	err    error
}

func (f *fakeStructurer) StructureVocabulary(ctx context.Context, input string) (openai.VocabularyFields, error) {
	if f.err != nil {
		return openai.VocabularyFields{}, f.err
	}
	return f.fields, nil
}

type fakeSheetAppender struct {
	rowNumber       int
	word            string
	translation     string
	contextSentence string
}

func (f *fakeSheetAppender) AppendVocabularyRow(ctx context.Context, word, translation, contextSentence string) (int, error) {
	f.word = word
	f.translation = translation
	f.contextSentence = contextSentence
	if f.rowNumber == 0 {
		f.rowNumber = 42
	}
	return f.rowNumber, nil
}

func TestSaveFromInputAppendsAndMerges(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := memory.NewVocabularyRepository()
	vocabularyService := vocabulary.NewService(repository, func() time.Time {
		return time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	})

	structurer := &fakeStructurer{
		fields: openai.VocabularyFields{
			Word:        "teasel",
			Context:     "Pat Teasely walked in.",
			Translation: "чесало",
		},
	}
	sheet := &fakeSheetAppender{}
	service := NewService(ServiceOptions{
		Structurer: structurer,
		Sheet:      sheet,
		Vocabulary: vocabularyService,
		Repository: repository,
		SheetName:  "Vocabulary",
	})

	item, err := service.SaveFromInput(ctx, "teasel")
	if err != nil {
		t.Fatalf("SaveFromInput() error = %v", err)
	}
	if item.DisplayWord != "teasel" {
		t.Fatalf("DisplayWord = %q", item.DisplayWord)
	}
	if sheet.word != "teasel" || sheet.translation != "чесало" {
		t.Fatalf("sheet row = %#v", sheet)
	}
	if len(item.Translations) != 1 || item.Translations[0] != "чесало" {
		t.Fatalf("translations = %#v", item.Translations)
	}
}
