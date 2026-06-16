package save

import (
	"context"
	"fmt"

	"github.com/neuroborus/vocabulary-bot/internal/openai"
)

type VocabularyStructurer interface {
	StructureVocabulary(ctx context.Context, input string) (openai.VocabularyFields, error)
}

type SheetAppender interface {
	AppendVocabularyRow(ctx context.Context, word, translation, contextSentence string) (int, error)
}

type Result struct {
	Word        string
	Translation string
	Context     string
	RowNumber   int
}

type Service struct {
	structurer VocabularyStructurer
	sheet      SheetAppender
}

type ServiceOptions struct {
	Structurer VocabularyStructurer
	Sheet      SheetAppender
}

func NewService(options ServiceOptions) *Service {
	return &Service{
		structurer: options.Structurer,
		sheet:      options.Sheet,
	}
}

func (s *Service) SaveFromInput(ctx context.Context, input string) (Result, error) {
	if s == nil || s.structurer == nil || s.sheet == nil {
		return Result{}, fmt.Errorf("save service is not configured")
	}

	fields, err := s.structurer.StructureVocabulary(ctx, input)
	if err != nil {
		return Result{}, err
	}

	rowNumber, err := s.sheet.AppendVocabularyRow(ctx, fields.Word, fields.Translation, fields.Context)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Word:        fields.Word,
		Translation: fields.Translation,
		Context:     fields.Context,
		RowNumber:   rowNumber,
	}, nil
}
