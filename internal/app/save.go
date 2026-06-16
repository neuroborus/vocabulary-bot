package app

import (
	"context"
	"fmt"

	"github.com/neuroborus/vocabulary-bot/internal/config"
	"github.com/neuroborus/vocabulary-bot/internal/openai"
	"github.com/neuroborus/vocabulary-bot/internal/save"
	"github.com/neuroborus/vocabulary-bot/internal/source/spreadsheet"
)

func buildSaveService(ctx context.Context, cfg config.Config) (*save.Service, error) {
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
	}), nil
}

func saveDisabledReason(cfg config.Config) string {
	switch {
	case cfg.OpenAI.APIKey == "":
		return "OPENAI_API_KEY is not set"
	case cfg.GoogleSheet.SpreadsheetID == "":
		return "GOOGLE_SPREADSHEET_ID is not set"
	case cfg.GoogleSheet.ServiceAccountJSON == "":
		return "GOOGLE_SERVICE_ACCOUNT_JSON is not set"
	default:
		return "save service prerequisites are not configured"
	}
}
