package spreadsheet

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"google.golang.org/api/sheets/v4"
)

var updatedRangeRowPattern = regexp.MustCompile(`!A(\d+)(?::|$)`)

type AppendValuesClient interface {
	AppendValues(ctx context.Context, spreadsheetID, valueRange string, row []string) (string, error)
}

type RowWriter struct {
	client        AppendValuesClient
	spreadsheetID string
	sheetName     string
}

func NewRowWriter(client AppendValuesClient, spreadsheetID, sheetName string) *RowWriter {
	return &RowWriter{
		client:        client,
		spreadsheetID: strings.TrimSpace(spreadsheetID),
		sheetName:     strings.TrimSpace(sheetName),
	}
}

func (w *RowWriter) AppendVocabularyRow(ctx context.Context, word, translation, contextSentence string) (int, error) {
	if w == nil || w.client == nil {
		return 0, fmt.Errorf("spreadsheet writer is not configured")
	}
	if w.spreadsheetID == "" {
		return 0, fmt.Errorf("GOOGLE_SPREADSHEET_ID is not configured")
	}

	sheetName := w.sheetName
	if sheetName == "" {
		sheetName = "Vocabulary"
	}

	row := []string{
		strings.TrimSpace(word),
		strings.TrimSpace(translation),
		strings.TrimSpace(contextSentence),
		"",
		"",
		"TRUE",
	}

	updatedRange, err := w.client.AppendValues(ctx, w.spreadsheetID, sheetName+"!A:F", row)
	if err != nil {
		return 0, err
	}

	return parseRowNumber(updatedRange), nil
}

func (c *GoogleValuesClient) AppendValues(ctx context.Context, spreadsheetID, valueRange string, row []string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	values := make([]interface{}, len(row))
	for index, cell := range row {
		values[index] = cell
	}

	response, err := c.service.Spreadsheets.Values.Append(
		spreadsheetID,
		valueRange,
		&sheets.ValueRange{Values: [][]interface{}{values}},
	).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("append spreadsheet range %q: %w", valueRange, err)
	}
	if response == nil || response.Updates == nil {
		return "", fmt.Errorf("append spreadsheet range %q: empty response", valueRange)
	}

	updatedRange := strings.TrimSpace(response.Updates.UpdatedRange)
	if updatedRange == "" {
		return "", fmt.Errorf("append spreadsheet range %q: missing updated range", valueRange)
	}

	return updatedRange, nil
}

func parseRowNumber(updatedRange string) int {
	match := updatedRangeRowPattern.FindStringSubmatch(updatedRange)
	if len(match) < 2 {
		return 0
	}

	rowNumber, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}

	return rowNumber
}
