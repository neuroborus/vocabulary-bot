package syncer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/neuroborus/vocabulary-bot/internal/logging"
	"github.com/neuroborus/vocabulary-bot/internal/source"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type Service struct {
	adapters   []source.Adapter
	vocabulary *vocabulary.Service
	logger     *slog.Logger
}

type Summary struct {
	Sources         []SourceSummary
	DraftsProcessed int
	Created         int
	Updated         int
	Ambiguous       int
	SourceErrors    int
}

type SourceSummary struct {
	Name    string
	Drafts  int
	Error   string
	Details source.Details
}

func NewService(adapters []source.Adapter, vocabularyService *vocabulary.Service, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		adapters:   append([]source.Adapter(nil), adapters...),
		vocabulary: vocabularyService,
		logger:     logger,
	}
}

func (s *Service) Run(ctx context.Context) (Summary, error) {
	var summary Summary

	for _, adapter := range s.adapters {
		if err := ctx.Err(); err != nil {
			return summary, err
		}

		drafts, err := adapter.Sync(ctx)
		sourceSummary := SourceSummary{
			Name:   adapter.Name(),
			Drafts: len(drafts),
		}

		if err != nil {
			sourceSummary.Error = logging.SanitizeError(err)
			summary.SourceErrors++
			summary.Sources = append(summary.Sources, sourceSummary)
			s.logger.Error(
				"source sync failed",
				slog.String("source", adapter.Name()),
				slog.String("error", logging.SanitizeError(err)),
			)
			continue
		}

		if detailsProvider, ok := adapter.(source.DetailsProvider); ok {
			sourceSummary.Details = detailsProvider.SyncDetails()
		}

		summary.Sources = append(summary.Sources, sourceSummary)

		for _, draft := range drafts {
			outcome, err := s.vocabulary.MergeDraft(ctx, draft)
			if errors.Is(err, vocabulary.ErrAmbiguousMatch) {
				summary.Ambiguous++
				s.logger.Warn(
					"ambiguous vocabulary merge skipped",
					slog.String("source", adapter.Name()),
					slog.String("word", draft.RawWord),
					slog.Any("lookup_keys", outcome.LookupKeys),
				)
				continue
			}
			if err != nil {
				return summary, fmt.Errorf("merge %s draft %q: %w", adapter.Name(), draft.RawWord, err)
			}

			summary.DraftsProcessed++
			if outcome.Created {
				summary.Created++
			}
			if outcome.Updated {
				summary.Updated++
			}
		}
	}

	return summary, nil
}
