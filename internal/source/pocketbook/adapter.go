package pocketbook

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/logging"
	"github.com/neuroborus/vocabulary-bot/internal/source"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type BookContextResolver interface {
	HasStoredBookLikeContext(ctx context.Context, rawWord string) (bool, error)
}

type Adapter struct {
	client              *Client
	logger              *slog.Logger
	bookContextEnabled  bool
	bookCache           *BookCache
	bookContextResolver BookContextResolver
	syncStats           syncStats
}

type syncStats struct {
	bookContextStats
	booksSkipped int
	booksFailed  int
	notesFailed  int
}

type bookContextStats struct {
	skippedStored int
	enriched      int
}

type AdapterOptions struct {
	BaseURL             string
	Email               string
	Password            string
	RefreshToken        string
	ShopName            string
	HTTPClient          *http.Client
	SessionStore        SessionStore
	Logger              *slog.Logger
	Now                 func() time.Time
	BookContextEnabled  bool
	BookCacheDir        string
	BookCacheMax        int
	BookContextResolver BookContextResolver
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
		client:              client,
		logger:              logger,
		bookContextEnabled:  options.BookContextEnabled,
		bookContextResolver: options.BookContextResolver,
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
				slog.String("error", logging.SanitizeError(err)),
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

func (a *Adapter) SyncDetails() source.Details {
	return source.Details{
		BookContextSkippedStored: a.syncStats.skippedStored,
		BookContextEnriched:      a.syncStats.enriched,
		BooksSkipped:             a.syncStats.booksSkipped,
		BooksFailed:              a.syncStats.booksFailed,
		NotesFailed:              a.syncStats.notesFailed,
	}
}

func (a *Adapter) Sync(ctx context.Context) ([]vocabulary.Draft, error) {
	a.syncStats = syncStats{}
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
			a.syncStats.booksSkipped++
			continue
		}

		bookDrafts, err := a.syncBook(ctx, book)
		if err != nil {
			a.logger.Error(
				"pocketbook book sync failed",
				slog.String("book_id", book.ID),
				slog.String("title", book.Title),
				slog.String("error", logging.SanitizeError(err)),
			)
			a.syncStats.booksFailed++
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

	notes, notesFailed := fetchBookNotes(ctx, a.client, book, noteIDs, a.logger)
	a.syncStats.notesFailed += notesFailed
	if len(notes) == 0 {
		return nil, nil
	}

	parsed := parsedDraftsFromNotes(book, notes)
	a.enrichBookDrafts(ctx, book, parsed)

	return draftsFromParsed(parsed), nil
}
