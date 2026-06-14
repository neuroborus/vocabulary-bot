package pocketbook

import (
	"context"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type Adapter struct{}

func NewAdapter() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Name() string {
	return "pocketbook"
}

func (a *Adapter) Sync(ctx context.Context) ([]vocabulary.Draft, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return nil, nil
}
