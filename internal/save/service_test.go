package save

import (
	"context"
	"testing"

	"github.com/neuroborus/vocabulary-bot/internal/openai"
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

func TestSaveFromInputAppendsToSheetOnly(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

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
	})

	result, err := service.SaveFromInput(ctx, "teasel")
	if err != nil {
		t.Fatalf("SaveFromInput() error = %v", err)
	}
	if result.Word != "teasel" {
		t.Fatalf("Word = %q", result.Word)
	}
	if result.RowNumber != 42 {
		t.Fatalf("RowNumber = %d, want 42", result.RowNumber)
	}
	if sheet.word != "teasel" || sheet.translation != "чесало" {
		t.Fatalf("sheet row = %#v", sheet)
	}
	if result.Translation != "чесало" {
		t.Fatalf("Translation = %q", result.Translation)
	}
}
