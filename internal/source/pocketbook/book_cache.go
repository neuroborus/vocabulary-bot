package pocketbook

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/logging"
)

const bookCacheAppDirName = "vocabulary-bot-cache"

type BookCache struct {
	dir    string
	max    int
	now    func() time.Time
	logger *slog.Logger
}

type BookCacheOptions struct {
	Dir    string
	Max    int
	Logger *slog.Logger
	Now    func() time.Time
}

type bookCacheMeta struct {
	BookID     string    `json:"bookId"`
	FastHash   string    `json:"fastHash"`
	FileName   string    `json:"fileName"`
	Title      string    `json:"title,omitempty"`
	LastUsedAt time.Time `json:"lastUsedAt"`
}

func NewBookCache(options BookCacheOptions) (*BookCache, error) {
	dir := strings.TrimSpace(options.Dir)
	if dir == "" {
		dir = DefaultBookCacheDir()
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create pocketbook book cache dir: %w", err)
	}

	now := options.Now
	if now == nil {
		now = time.Now
	}

	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &BookCache{
		dir:    dir,
		max:    options.Max,
		now:    now,
		logger: logger,
	}, nil
}

func DefaultBookCacheDir() string {
	return filepath.Join(os.TempDir(), bookCacheAppDirName, "books")
}

func (c *BookCache) Acquire(ctx context.Context, client *Client, book Book, downloadURL string) (string, error) {
	if err := c.removeStaleVersions(book); err != nil {
		return "", err
	}

	bookPath := c.bookPath(book)
	metaPath := c.metaPath(book)

	if c.hasValidEntry(bookPath, metaPath, book) {
		if err := c.touch(metaPath, book); err != nil {
			return "", err
		}
		c.logger.Info(
			"pocketbook book cache hit",
			slog.String("book_id", book.ID),
			slog.String("fast_hash", book.FastHash),
			slog.String("path", bookPath),
		)
		return bookPath, nil
	}

	if err := client.DownloadFile(ctx, downloadURL, bookPath); err != nil {
		return "", err
	}

	if err := c.writeMeta(metaPath, book); err != nil {
		_ = os.Remove(bookPath)
		return "", err
	}

	if err := c.enforceLimit(book.FastHash); err != nil {
		c.logger.Warn(
			"pocketbook book cache eviction failed",
			slog.String("book_id", book.ID),
			slog.String("error", logging.SanitizeError(err)),
		)
	}

	c.logger.Info(
		"pocketbook book cached",
		slog.String("book_id", book.ID),
		slog.String("fast_hash", book.FastHash),
		slog.String("path", bookPath),
	)
	return bookPath, nil
}

func (c *BookCache) bookPath(book Book) string {
	return filepath.Join(c.dir, cachedBookFileName(book))
}

func (c *BookCache) metaPath(book Book) string {
	return filepath.Join(c.dir, cachedBookFileName(book)+".meta.json")
}

func cachedBookFileName(book Book) string {
	key := strings.TrimSpace(book.FastHash)
	if key == "" {
		key = book.ID
	}

	return sanitizeBookFileName(key) + bookFileExtension(book)
}

func (c *BookCache) hasValidEntry(bookPath, metaPath string, book Book) bool {
	meta, err := c.readMeta(metaPath)
	if err != nil || meta.FastHash != book.FastHash {
		return false
	}

	info, err := os.Stat(bookPath)
	return err == nil && info.Size() > 0
}

func (c *BookCache) readMeta(path string) (bookCacheMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return bookCacheMeta{}, err
	}

	var meta bookCacheMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return bookCacheMeta{}, err
	}

	return meta, nil
}

func (c *BookCache) writeMeta(path string, book Book) error {
	meta := bookCacheMeta{
		BookID:     book.ID,
		FastHash:   book.FastHash,
		FileName:   cachedBookFileName(book),
		Title:      firstNonEmpty(book.Title, book.Metadata.Title),
		LastUsedAt: c.now().UTC(),
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

func (c *BookCache) touch(metaPath string, book Book) error {
	meta, err := c.readMeta(metaPath)
	if err != nil {
		return err
	}

	meta.LastUsedAt = c.now().UTC()
	meta.Title = firstNonEmpty(book.Title, book.Metadata.Title)

	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	return os.WriteFile(metaPath, data, 0o600)
}

func (c *BookCache) removeStaleVersions(book Book) error {
	entries, err := c.listEntries()
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.BookID != book.ID || entry.FastHash == book.FastHash {
			continue
		}

		c.deleteEntry(entry)
		c.logger.Info(
			"pocketbook book cache removed stale version",
			slog.String("book_id", book.ID),
			slog.String("old_fast_hash", entry.FastHash),
			slog.String("new_fast_hash", book.FastHash),
		)
	}

	return nil
}

func (c *BookCache) enforceLimit(keepFastHash string) error {
	if c.max <= 0 {
		return nil
	}

	entries, err := c.listEntries()
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LastUsedAt.Before(entries[j].LastUsedAt)
	})

	for len(entries) > c.max {
		removed := false
		for i, entry := range entries {
			if entry.FastHash == keepFastHash {
				continue
			}

			c.deleteEntry(entry)
			c.logger.Info(
				"pocketbook book cache evicted oldest entry",
				slog.String("book_id", entry.BookID),
				slog.String("fast_hash", entry.FastHash),
				slog.Time("last_used_at", entry.LastUsedAt),
			)
			entries = append(entries[:i], entries[i+1:]...)
			removed = true
			break
		}
		if !removed {
			break
		}
	}

	return nil
}

func (c *BookCache) listEntries() ([]bookCacheMeta, error) {
	matches, err := filepath.Glob(filepath.Join(c.dir, "*.meta.json"))
	if err != nil {
		return nil, err
	}

	entries := make([]bookCacheMeta, 0, len(matches))
	for _, match := range matches {
		meta, err := c.readMeta(match)
		if err != nil {
			continue
		}
		entries = append(entries, meta)
	}

	return entries, nil
}

func (c *BookCache) deleteEntry(meta bookCacheMeta) {
	if meta.FileName != "" {
		_ = os.Remove(filepath.Join(c.dir, meta.FileName))
		_ = os.Remove(filepath.Join(c.dir, meta.FileName+".meta.json"))
	}
}
