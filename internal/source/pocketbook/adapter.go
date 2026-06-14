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
	client             *Client
	logger             *slog.Logger
	bookContextEnabled bool
	bookCache          *BookCache
}

type AdapterOptions struct {
	BaseURL            string
	Email              string
	Password           string
	RefreshToken       string
	ShopName           string
	HTTPClient         *http.Client
	SessionStore       SessionStore
	Logger             *slog.Logger
	Now                func() time.Time
	BookContextEnabled bool
	BookCacheDir       string
	BookCacheMax       int
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

	adapter := &Adapter{
		client:             client,
		logger:             logger,
		bookContextEnabled: options.BookContextEnabled,
	}

	if options.BookContextEnabled {
		bookCache, err := NewBookCache(BookCacheOptions{
			Dir:    options.BookCacheDir,
			Max:    options.BookCacheMax,
			Logger: logger,
			Now:    options.Now,
		})
		if err != nil {
			logger.Warn(
				"pocketbook book cache disabled",
				slog.String("error", err.Error()),
			)
		} else {
			adapter.bookCache = bookCache
		}
	}

	return adapter
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
		if err := validateBookForNotes(book); err != nil {
			a.logger.Warn(
				"pocketbook book skipped because fast_hash is empty",
				slog.String("book_id", book.ID),
				slog.String("title", book.Title),
			)
			continue
		}

		bookDrafts, err := a.syncBook(ctx, book)
		if err != nil {
			a.logger.Error(
				"pocketbook book sync failed",
				slog.String("book_id", book.ID),
				slog.String("title", book.Title),
				slog.String("error", err.Error()),
			)
			continue
		}

		drafts = append(drafts, bookDrafts...)
	}

	return drafts, nil
}

func (a *Adapter) syncBook(ctx context.Context, book Book) ([]vocabulary.Draft, error) {
	noteIDs, err := a.client.ListNoteIDs(ctx, book.FastHash)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}

	notes := fetchBookNotes(ctx, a.client, book, noteIDs, a.logger)
	if len(notes) == 0 {
		return nil, nil
	}

	parsed := parsedDraftsFromNotes(book, notes)
	a.enrichBookDrafts(ctx, book, parsed)

	return draftsFromParsed(parsed), nil
}
