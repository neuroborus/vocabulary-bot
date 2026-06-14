package spreadsheet

import (
	"context"
	"fmt"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type ValuesClient interface {
	FetchValues(ctx context.Context, spreadsheetID, valueRange string) ([][]string, error)
}

type GoogleValuesClient struct {
	service *sheets.Service
}

func NewGoogleValuesClient(ctx context.Context, credentialsJSON []byte) (*GoogleValuesClient, error) {
	service, err := sheets.NewService(ctx, option.WithCredentialsJSON(credentialsJSON))
	if err != nil {
		return nil, fmt.Errorf("create google sheets service: %w", err)
	}

	return &GoogleValuesClient{service: service}, nil
}

func (c *GoogleValuesClient) FetchValues(ctx context.Context, spreadsheetID, valueRange string) ([][]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	response, err := c.service.Spreadsheets.Values.Get(spreadsheetID, valueRange).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("fetch spreadsheet range %q: %w", valueRange, err)
	}

	if response.Values == nil {
		return nil, nil
	}

	return anyMatrixToStrings(response.Values), nil
}

func anyMatrixToStrings(values [][]any) [][]string {
	rows := make([][]string, 0, len(values))
	for _, row := range values {
		cells := make([]string, 0, len(row))
		for _, cell := range row {
			if cell == nil {
				cells = append(cells, "")
				continue
			}
			cells = append(cells, fmt.Sprint(cell))
		}
		rows = append(rows, cells)
	}

	return rows
}
