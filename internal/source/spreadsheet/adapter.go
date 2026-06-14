package spreadsheet

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type Adapter struct {
	spreadsheetID      string
	sheetName          string
	rangeA1            string
	serviceAccountJSON string
	logger             *slog.Logger
	valuesClient       ValuesClient
}

type AdapterOptions struct {
	SpreadsheetID      string
	SheetName          string
	Range              string
	ServiceAccountJSON string
	Logger             *slog.Logger
	ValuesClient       ValuesClient
}

func NewAdapter(options AdapterOptions) *Adapter {
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}

	sheetName := strings.TrimSpace(options.SheetName)
	if sheetName == "" {
		sheetName = "Vocabulary"
	}

	rangeA1 := strings.TrimSpace(options.Range)
	if rangeA1 == "" {
		rangeA1 = sheetName + "!A:F"
	}

	return &Adapter{
		spreadsheetID:      strings.TrimSpace(options.SpreadsheetID),
		sheetName:          sheetName,
		rangeA1:            rangeA1,
		serviceAccountJSON: strings.TrimSpace(options.ServiceAccountJSON),
		logger:             logger,
		valuesClient:       options.ValuesClient,
	}
}

func (a *Adapter) Name() string {
	return "google-sheet"
}

func (a *Adapter) Sync(ctx context.Context) ([]vocabulary.Draft, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if a.spreadsheetID == "" {
		return nil, fmt.Errorf("GOOGLE_SPREADSHEET_ID is not configured")
	}

	client, err := a.client(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := client.FetchValues(ctx, a.spreadsheetID, a.rangeA1)
	if err != nil {
		return nil, err
	}

	drafts, rowErrors := ParseRows(a.sheetName, rows)
	for _, rowError := range rowErrors {
		a.logger.Warn(
			"spreadsheet row parse skipped",
			slog.String("sheet", a.sheetName),
			slog.Int("row", rowError.RowNumber),
			slog.String("problem", rowError.Problem),
		)
	}

	a.logger.Info(
		"spreadsheet rows parsed",
		slog.String("sheet", a.sheetName),
		slog.String("range", a.rangeA1),
		slog.Int("drafts", len(drafts)),
		slog.Int("row_errors", len(rowErrors)),
	)

	return drafts, nil
}

func (a *Adapter) client(ctx context.Context) (ValuesClient, error) {
	if a.valuesClient != nil {
		return a.valuesClient, nil
	}

	credentialsJSON, err := ParseServiceAccountCredentials(a.serviceAccountJSON)
	if err != nil {
		return nil, fmt.Errorf("GOOGLE_SERVICE_ACCOUNT_JSON: %w", err)
	}

	client, err := NewGoogleValuesClient(ctx, credentialsJSON)
	if err != nil {
		return nil, err
	}

	a.valuesClient = client
	return a.valuesClient, nil
}
