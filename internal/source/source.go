package source

import (
	"context"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type Adapter interface {
	Name() string
	Sync(ctx context.Context) ([]vocabulary.Draft, error)
}
