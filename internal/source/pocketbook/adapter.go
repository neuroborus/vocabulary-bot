package pocketbook

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type Adapter struct {
	client *Client
	logger *slog.Logger
}

type AdapterOptions struct {
	BaseURL      string
	Email        string
	Password     string
	RefreshToken string
	ShopName     string
	HTTPClient   *http.Client
	SessionStore SessionStore
	Logger       *slog.Logger
	Now          func() time.Time
}

func NewAdapter(options AdapterOptions) *Adapter {
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}

	client := NewClient(ClientOptions{
		BaseURL:      options.BaseURL,
		Email:        options.Email,
		Password:     options.Password,
		RefreshToken: options.RefreshToken,
		ShopName:     options.ShopName,
		HTTPClient:   options.HTTPClient,
		SessionStore: options.SessionStore,
		Logger:       logger,
		Now:          options.Now,
	})

	return &Adapter{
		client: client,
		logger: logger,
	}
}

func (a *Adapter) Name() string {
	return "pocketbook"
}

func (a *Adapter) Sync(ctx context.Context) ([]vocabulary.Draft, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	books, err := a.client.ListBooks(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pocketbook books: %w", err)
	}

	drafts := make([]vocabulary.Draft, 0)
	for _, book := range books {
		if book.FastHash == "" {
			a.logger.Warn(
				"pocketbook book skipped because fast_hash is empty",
				slog.String("book_id", book.ID),
				slog.String("title", book.Title),
			)
			continue
		}

		noteIDs, err := a.client.ListNoteIDs(ctx, book.FastHash)
		if err != nil {
			a.logger.Error(
				"pocketbook book notes failed",
				slog.String("book_id", book.ID),
				slog.String("title", book.Title),
				slog.String("error", err.Error()),
			)
			continue
		}

		for _, noteID := range noteIDs {
			if noteID.UUID == "" {
				continue
			}

			note, ok, err := a.client.GetNote(ctx, noteID.UUID, book.FastHash)
			if err != nil {
				a.logger.Error(
					"pocketbook note fetch failed",
					slog.String("book_id", book.ID),
					slog.String("note_uuid", noteID.UUID),
					slog.String("error", err.Error()),
				)
				continue
			}
			if !ok {
				continue
			}

			draft, ok := ParseNote(book, note)
			if !ok {
				continue
			}

			drafts = append(drafts, draft)
		}
	}

	return drafts, nil
}
