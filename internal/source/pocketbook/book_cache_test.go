package pocketbook

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultBookCacheDirUsesSystemTemp(t *testing.T) {
	t.Parallel()

	dir := DefaultBookCacheDir()
	if !strings.HasPrefix(dir, os.TempDir()) {
		t.Fatalf("dir = %q, want prefix %q", dir, os.TempDir())
	}
	wantSuffix := filepath.Join("vocabulary-bot", "books")
	if !strings.HasSuffix(dir, wantSuffix) {
		t.Fatalf("dir = %q, want suffix %q", dir, wantSuffix)
	}
}

func TestBookCacheReusesExistingFile(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	downloads := 0
	book := Book{
		ID:       "book-1",
		Title:    "Necromancer",
		FastHash: "hash-1",
		Link:     "/download/hash-1.epub",
		MimeType: "application/epub+zip",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloads++
		_, _ = w.Write([]byte("epub-bytes"))
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		BaseURL:      server.URL,
		SessionStore: NewMemorySessionStore(Session{AccessToken: "access", AccessTokenExpiresAt: time.Now().Add(time.Hour)}),
	})
	cache := newTestBookCache(t, cacheDir, 2)

	first, err := cache.Acquire(context.Background(), client, book, server.URL+book.Link)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	second, err := cache.Acquire(context.Background(), client, book, server.URL+book.Link)
	if err != nil {
		t.Fatalf("second acquire: %v", err)
	}

	if first != second {
		t.Fatalf("paths = %q and %q, want same cached file", first, second)
	}
	if downloads != 1 {
		t.Fatalf("downloads = %d, want 1", downloads)
	}

	meta, err := cache.readMeta(cache.metaPath(book))
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	if meta.LastUsedAt.IsZero() {
		t.Fatal("LastUsedAt was not set")
	}
}

func TestBookCacheEvictsOldestByLastUsedAt(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	cacheDir := t.TempDir()
	cache := newTestBookCacheWithNow(t, cacheDir, 2, func() time.Time { return now })

	books := []Book{
		{ID: "book-1", FastHash: "hash-1", MimeType: "application/epub+zip"},
		{ID: "book-2", FastHash: "hash-2", MimeType: "application/epub+zip"},
		{ID: "book-3", FastHash: "hash-3", MimeType: "application/epub+zip"},
	}

	for index, book := range books {
		bookPath := cache.bookPath(book)
		if err := os.WriteFile(bookPath, []byte("book"), 0o600); err != nil {
			t.Fatalf("write book %d: %v", index, err)
		}
		meta := bookCacheMeta{
			BookID:     book.ID,
			FastHash:   book.FastHash,
			FileName:   cachedBookFileName(book),
			LastUsedAt: now.Add(time.Duration(index) * time.Hour),
		}
		data, err := json.Marshal(meta)
		if err != nil {
			t.Fatalf("marshal meta: %v", err)
		}
		if err := os.WriteFile(cache.metaPath(book), data, 0o600); err != nil {
			t.Fatalf("write meta %d: %v", index, err)
		}
	}

	if err := cache.enforceLimit("hash-3"); err != nil {
		t.Fatalf("enforceLimit() error = %v", err)
	}

	if _, err := os.Stat(cache.bookPath(books[0])); !os.IsNotExist(err) {
		t.Fatal("oldest cache entry was not evicted")
	}
	if _, err := os.Stat(cache.bookPath(books[1])); err != nil {
		t.Fatalf("middle cache entry missing: %v", err)
	}
	if _, err := os.Stat(cache.bookPath(books[2])); err != nil {
		t.Fatalf("newest cache entry missing: %v", err)
	}
}

func TestBookCacheRemovesStaleFastHashForSameBookID(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	cache := newTestBookCache(t, cacheDir, 2)

	oldBook := Book{
		ID:       "book-1",
		FastHash: "hash-old",
		MimeType: "application/epub+zip",
	}
	newBook := Book{
		ID:       "book-1",
		FastHash: "hash-new",
		MimeType: "application/epub+zip",
	}

	oldPath := cache.bookPath(oldBook)
	if err := os.WriteFile(oldPath, []byte("old"), 0o600); err != nil {
		t.Fatalf("write old book: %v", err)
	}
	if err := os.WriteFile(cache.metaPath(oldBook), mustMetaJSON(t, oldBook), 0o600); err != nil {
		t.Fatalf("write old meta: %v", err)
	}

	if err := cache.removeStaleVersions(newBook); err != nil {
		t.Fatalf("removeStaleVersions() error = %v", err)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatal("stale cached book was not removed")
	}
	if _, err := os.Stat(cache.metaPath(oldBook)); !os.IsNotExist(err) {
		t.Fatal("stale cached meta was not removed")
	}
}

func newTestBookCache(t *testing.T, dir string, max int) *BookCache {
	t.Helper()

	return newTestBookCacheWithNow(t, dir, max, time.Now)
}

func newTestBookCacheWithNow(t *testing.T, dir string, max int, now func() time.Time) *BookCache {
	t.Helper()

	cache, err := NewBookCache(BookCacheOptions{
		Dir:    dir,
		Max:    max,
		Logger: slog.New(slog.NewTextHandler(testWriter{t: t}, nil)),
		Now:    now,
	})
	if err != nil {
		t.Fatalf("NewBookCache() error = %v", err)
	}

	return cache
}

func mustMetaJSON(t *testing.T, book Book) []byte {
	t.Helper()

	data, err := json.Marshal(bookCacheMeta{
		BookID:     book.ID,
		FastHash:   book.FastHash,
		FileName:   cachedBookFileName(book),
		LastUsedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("marshal meta: %v", err)
	}

	return data
}
