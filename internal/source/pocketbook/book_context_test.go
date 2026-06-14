package pocketbook

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func TestAdapterEnrichesDictionaryWordWithBookSentence(t *testing.T) {
	t.Parallel()

	body := `<html><body><p>He had to lean against the wall. Then he left.</p></body></html>`
	epubPath := writeTestEPUB(t, body)
	epubData, err := os.ReadFile(epubPath)
	if err != nil {
		t.Fatalf("read test epub: %v", err)
	}

	text, err := FlattenEPUB(epubPath)
	if err != nil {
		t.Fatalf("FlattenEPUB() error = %v", err)
	}
	offset := findSubstringOffset(text, "lean")

	now := time.Date(2026, 6, 14, 8, 0, 0, 0, time.UTC)
	store := NewMemorySessionStore(Session{
		AccessToken:          "access",
		RefreshToken:         "refresh",
		AccessTokenExpiresAt: now.Add(time.Hour),
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/books":
			writeJSON(t, w, booksResponse{
				Items: []Book{{
					ID:       "book-1",
					Title:    "Necromancer",
					FastHash: "hash-1",
					Link:     "/download/book-1.epub",
					MimeType: "application/epub+zip",
				}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/download/book-1.epub":
			w.Header().Set("Content-Type", "application/epub+zip")
			_, _ = w.Write(epubData)
		case r.Method == http.MethodGet && r.URL.Path == "/notes":
			writeJSON(t, w, []NoteInfo{{UUID: "note-1"}})
		case r.Method == http.MethodGet && r.URL.Path == "/notes/note-1":
			writeJSON(t, w, Note{
				UUID:      "note-1",
				Note:      &TextWithTime{Text: "Verb 1) опираться"},
				Quotation: &Quotation{Text: "lean"},
				Mark:      &Mark{Anchor: "pbr:/word?page=11&offs=" + strconv.Itoa(offset)},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	adapter := NewAdapter(AdapterOptions{
		BaseURL:            server.URL,
		Email:              "reader@example.test",
		Password:           "secret",
		SessionStore:       store,
		Logger:             slog.New(slog.NewTextHandler(testWriter{t: t}, nil)),
		Now:                func() time.Time { return now },
		BookContextEnabled: true,
		BookCacheDir:       t.TempDir(),
		BookCacheMax:       2,
	})

	drafts, err := adapter.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("drafts = %d, want 1", len(drafts))
	}

	foundBookSentence := false
	for _, contextValue := range drafts[0].Contexts {
		if contextValue == "He had to lean against the wall." {
			foundBookSentence = true
		}
	}
	if !foundBookSentence {
		t.Fatalf("contexts = %#v, want book sentence", drafts[0].Contexts)
	}
}

func TestEnrichBookDraftsKeepsCachedBookFile(t *testing.T) {
	t.Parallel()

	body := `<html><body><p>He had to lean against the wall.</p></body></html>`
	epubPath := writeTestEPUB(t, body)
	epubData, err := os.ReadFile(epubPath)
	if err != nil {
		t.Fatalf("read test epub: %v", err)
	}

	text, err := FlattenEPUB(epubPath)
	if err != nil {
		t.Fatalf("FlattenEPUB() error = %v", err)
	}
	offset := findSubstringOffset(text, "lean")

	cacheDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/download/book-1.epub" {
			_, _ = w.Write(epubData)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		BaseURL:      server.URL,
		SessionStore: NewMemorySessionStore(Session{AccessToken: "access", AccessTokenExpiresAt: time.Now().Add(time.Hour)}),
	})
	cache, err := NewBookCache(BookCacheOptions{Dir: cacheDir, Max: 2})
	if err != nil {
		t.Fatalf("NewBookCache() error = %v", err)
	}
	adapter := &Adapter{
		client:             client,
		logger:             slog.New(slog.NewTextHandler(testWriter{t: t}, nil)),
		bookContextEnabled: true,
		bookCache:          cache,
	}

	book := Book{
		ID:       "book-1",
		Title:    "Necromancer",
		FastHash: "hash-1",
		Link:     "/download/book-1.epub",
		MimeType: "application/epub+zip",
	}
	parsed := []parsedBookDraft{{
		draft: vocabularyDraftLean(offset),
		note: Note{
			UUID:      "note-1",
			Note:      &TextWithTime{Text: "Verb 1) опираться"},
			Quotation: &Quotation{Text: "lean"},
			Mark:      &Mark{Anchor: "pbr:/word?page=11&offs=" + strconv.Itoa(offset)},
		},
	}}

	adapter.enrichBookDrafts(context.Background(), book, parsed)

	cachedBookPath := filepath.Join(cacheDir, cachedBookFileName(book))
	if _, err := os.Stat(cachedBookPath); err != nil {
		t.Fatalf("cached book file missing: %v", err)
	}
	metaPath := cachedBookPath + ".meta.json"
	if _, err := os.Stat(metaPath); err != nil {
		t.Fatalf("cached book meta missing: %v", err)
	}
}

func vocabularyDraftLean(offset int) vocabulary.Draft {
	return vocabulary.Draft{
		Source:       vocabulary.SourcePocketBook,
		RawWord:      "lean",
		Translations: []string{"Verb 1) опираться"},
		Anchor: vocabulary.SourceAnchor{
			Source:   vocabulary.SourcePocketBook,
			BookID:   "book-1",
			Page:     "11",
			Position: "pbr:/word?page=11&offs=" + strconv.Itoa(offset),
		},
	}
}

func findSubstringOffset(text, needle string) int {
	return strings.Index(text, needle)
}
