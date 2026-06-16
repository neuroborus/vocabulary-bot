package save

import (
	"context"
	"errors"
	"fmt"

	"github.com/neuroborus/vocabulary-bot/internal/openai"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type VocabularyStructurer interface {
	StructureVocabulary(ctx context.Context, input string) (openai.VocabularyFields, error)
}

type SheetAppender interface {
	AppendVocabularyRow(ctx context.Context, word, translation, contextSentence string) (int, error)
}

type Service struct {
	structurer VocabularyStructurer
	sheet      SheetAppender
	vocabulary *vocabulary.Service
	repository vocabulary.Repository
	sheetName  string
}

type ServiceOptions struct {
	Structurer VocabularyStructurer
	Sheet      SheetAppender
	Vocabulary *vocabulary.Service
	Repository vocabulary.Repository
	SheetName  string
}

func NewService(options ServiceOptions) *Service {
	sheetName := options.SheetName
	if sheetName == "" {
		sheetName = "Vocabulary"
	}

	return &Service{
		structurer: options.Structurer,
		sheet:      options.Sheet,
		vocabulary: options.Vocabulary,
		repository: options.Repository,
		sheetName:  sheetName,
	}
}

func (s *Service) SaveFromInput(ctx context.Context, input string) (vocabulary.Item, error) {
	if s == nil || s.structurer == nil || s.sheet == nil || s.vocabulary == nil || s.repository == nil {
		return vocabulary.Item{}, fmt.Errorf("save service is not configured")
	}

	fields, err := s.structurer.StructureVocabulary(ctx, input)
	if err != nil {
		return vocabulary.Item{}, err
	}

	rowNumber, err := s.sheet.AppendVocabularyRow(ctx, fields.Word, fields.Translation, fields.Context)
	if err != nil {
		return vocabulary.Item{}, err
	}

	draft := vocabulary.Draft{
		Source:       vocabulary.SourceGoogleSheet,
		RawWord:      fields.Word,
		Translations: []string{fields.Translation},
		Contexts:     []string{fields.Context},
		Anchor: vocabulary.SourceAnchor{
			Source:    vocabulary.SourceGoogleSheet,
			SheetName: s.sheetName,
			RowNumber: rowNumber,
		},
	}

	outcome, err := s.vocabulary.MergeDraft(ctx, draft)
	if err != nil {
		if errors.Is(err, vocabulary.ErrAmbiguousMatch) {
			return vocabulary.Item{}, fmt.Errorf("merge saved vocabulary: %w", err)
		}
		return vocabulary.Item{}, fmt.Errorf("merge saved vocabulary: %w", err)
	}

	item, err := s.loadItem(ctx, outcome.NormalizedKey, fields.Word)
	if err != nil {
		return vocabulary.Item{}, err
	}

	return item, nil
}

func (s *Service) loadItem(ctx context.Context, normalizedKey, rawWord string) (vocabulary.Item, error) {
	if normalizedKey != "" {
		matches, err := s.repository.FindByLookupKeys(ctx, []string{normalizedKey})
		if err != nil {
			return vocabulary.Item{}, err
		}
		for _, match := range matches {
			if match.NormalizedKey == normalizedKey {
				return match, nil
			}
		}
	}

	matches, err := s.repository.FindByLookupKeys(ctx, vocabulary.BuildLookupKeys(rawWord))
	if err != nil {
		return vocabulary.Item{}, err
	}
	if len(matches) == 0 {
		return vocabulary.Item{}, fmt.Errorf("saved vocabulary item not found")
	}

	return matches[0], nil
}
