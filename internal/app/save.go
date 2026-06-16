package app

import (
	"context"
	"fmt"

	"github.com/neuroborus/vocabulary-bot/internal/config"
	"github.com/neuroborus/vocabulary-bot/internal/openai"
	"github.com/neuroborus/vocabulary-bot/internal/save"
	"github.com/neuroborus/vocabulary-bot/internal/source/spreadsheet"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func buildSaveService(
	ctx context.Context,
	cfg config.Config,
	vocabularyService *vocabulary.Service,
	repository vocabulary.Repository,
) (*save.Service, error) {
	if cfg.OpenAI.APIKey == "" {
		return nil, nil
	}
	if cfg.GoogleSheet.SpreadsheetID == "" || cfg.GoogleSheet.ServiceAccountJSON == "" {
		return nil, nil
	}

	credentialsJSON, err := spreadsheet.ParseServiceAccountCredentials(cfg.GoogleSheet.ServiceAccountJSON)
	if err != nil {
		return nil, fmt.Errorf("save google credentials: %w", err)
	}

	sheetsClient, err := spreadsheet.NewGoogleValuesClient(ctx, credentialsJSON)
	if err != nil {
		return nil, fmt.Errorf("save google sheets client: %w", err)
	}

	return save.NewService(save.ServiceOptions{
		Structurer: openai.NewClient(openai.ClientOptions{
			APIKey:              cfg.OpenAI.APIKey,
			Model:               cfg.OpenAI.Model,
			BaseURL:             cfg.OpenAI.BaseURL,
			TranslationLanguage: cfg.OpenAI.TranslationLanguage,
		}),
		Sheet: spreadsheet.NewRowWriter(
			sheetsClient,
			cfg.GoogleSheet.SpreadsheetID,
			cfg.GoogleSheet.SheetName,
		),
		Vocabulary: vocabularyService,
		Repository: repository,
		SheetName:  cfg.GoogleSheet.SheetName,
	}), nil
}
