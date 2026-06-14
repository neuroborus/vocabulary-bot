package spreadsheet

import (
	"context"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type Adapter struct {
	sheetName string
}

func NewAdapter(sheetName string) *Adapter {
	if sheetName == "" {
		sheetName = "Vocabulary"
	}

	return &Adapter{sheetName: sheetName}
}

func (a *Adapter) Name() string {
	return "google-sheet"
}

func (a *Adapter) Sync(ctx context.Context) ([]vocabulary.Draft, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return nil, nil
}
