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

func TestAdapterBootstrapsAfterStoredRefreshTokenFails(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 8, 0, 0, 0, time.UTC)
	store := NewMemorySessionStore(Session{
		AccessToken:          "old-access",
		RefreshToken:         "old-refresh",
		AccessTokenExpiresAt: now.Add(-time.Hour),
		ShopAlias:            "main",
		ShopID:               "1",
		ShopName:             "Main Shop",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/renew-token":
			if got := r.FormValue("refresh_token"); got != "old-refresh" {
				t.Fatalf("refresh_token = %q, want old-refresh", got)
			}
			http.Error(w, "expired", http.StatusUnauthorized)
		case r.Method == http.MethodGet && r.URL.Path == "/auth/login":
			writeJSON(t, w, shopsResponse{
				Providers: []Shop{{Alias: "main", Name: "Main Shop", ShopID: "1"}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/auth/login/main":
			if got := r.FormValue("username"); got != "reader@example.test" {
				t.Fatalf("username = %q", got)
			}
			if got := r.FormValue("password"); got != "secret" {
				t.Fatalf("password = %q", got)
			}
			writeJSON(t, w, authTokens{
				AccessToken:  "new-access",
				RefreshToken: "new-refresh",
				ExpiresIn:    3600,
				TokenType:    "Bearer",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/books":
			requireBearer(t, r, "new-access")
			writeJSON(t, w, booksResponse{
				Total: 1,
				Items: []Book{{
					ID:       "book-1",
					Title:    "Road Book",
					FastHash: "hash-1",
					Metadata: BookMetadata{Authors: "A. Writer"},
				}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/notes":
			requireBearer(t, r, "new-access")
			if got := r.URL.Query().Get("fast_hash"); got != "hash-1" {
				t.Fatalf("fast_hash = %q, want hash-1", got)
			}
			writeJSON(t, w, []NoteInfo{{UUID: "note-1"}})
		case r.Method == http.MethodGet && r.URL.Path == "/notes/note-1":
			requireBearer(t, r, "new-access")
			writeJSON(t, w, Note{
				UUID:      "note-1",
				Note:      &TextWithTime{Text: "замедляться, снижать скорость"},
				Quotation: &Quotation{Text: "to decelerate"},
				Mark:      &Mark{Anchor: "pbr:/page?page=36&offs=123"},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	adapter := NewAdapter(AdapterOptions{
		BaseURL:      server.URL,
		Email:        "reader@example.test",
		Password:     "secret",
		SessionStore: store,
		Logger:       slog.New(slog.NewTextHandler(testWriter{t: t}, nil)),
		Now:          func() time.Time { return now },
	})

	drafts, err := adapter.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("drafts = %d, want 1", len(drafts))
	}

	draft := drafts[0]
	if draft.RawWord != "to decelerate" {
		t.Fatalf("RawWord = %q, want to decelerate", draft.RawWord)
	}
	if len(draft.Translations) != 2 {
		t.Fatalf("translations = %#v, want 2 translations", draft.Translations)
	}
	if draft.Anchor.ExternalID != "note-1" || draft.Anchor.BookTitle != "Road Book" || draft.Anchor.Page != "36" {
		t.Fatalf("anchor = %#v", draft.Anchor)
	}

	saved, ok, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load saved session: %v", err)
	}
	if !ok {
		t.Fatalf("stored session missing")
	}
	if saved.RefreshToken != "new-refresh" {
		t.Fatalf("stored refresh token = %q, want new-refresh", saved.RefreshToken)
	}
	if saved.AccessToken != "new-access" {
		t.Fatalf("stored access token = %q, want new-access", saved.AccessToken)
	}
}

func TestClientRetriesAuthorizedRequestAfterAccessTokenUnauthorized(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 8, 0, 0, 0, time.UTC)
	store := NewMemorySessionStore(Session{
		AccessToken:          "stale-access",
		RefreshToken:         "valid-refresh",
		AccessTokenExpiresAt: now.Add(time.Hour),
		ShopAlias:            "main",
	})
	bookRequests := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/books":
			bookRequests++
			if bookRequests == 1 {
				requireBearer(t, r, "stale-access")
				http.Error(w, "stale", http.StatusUnauthorized)
				return
			}
			requireBearer(t, r, "fresh-access")
			writeJSON(t, w, booksResponse{Items: []Book{{ID: "book-1", Title: "Book", FastHash: "hash-1"}}})
		case r.Method == http.MethodPost && r.URL.Path == "/auth/renew-token":
			requireBearer(t, r, "stale-access")
			if got := r.FormValue("refresh_token"); got != "valid-refresh" {
				t.Fatalf("refresh_token = %q", got)
			}
			writeJSON(t, w, authTokens{
				AccessToken:  "fresh-access",
				RefreshToken: "fresh-refresh",
				ExpiresIn:    3600,
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		BaseURL:      server.URL,
		SessionStore: store,
		Now:          func() time.Time { return now },
		Logger:       slog.New(slog.NewTextHandler(testWriter{t: t}, nil)),
	})

	books, err := client.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("ListBooks() error = %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("books = %d, want 1", len(books))
	}
	if bookRequests != 2 {
		t.Fatalf("bookRequests = %d, want 2", bookRequests)
	}

	saved, _, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if saved.AccessToken != "fresh-access" || saved.RefreshToken != "fresh-refresh" {
		t.Fatalf("saved session = %#v", saved)
	}
}

func TestParseNoteSupportsLabeledDictionaryNote(t *testing.T) {
	t.Parallel()

	book := Book{ID: "book-1", Title: "Road Book", Metadata: BookMetadata{Authors: "A. Writer"}}
	note := Note{
		UUID: "note-1",
		Note: &TextWithTime{Text: strings.Join([]string{
			"Word: decelerate",
			"Translations: замедляться, снижать скорость",
			"Context: The car began to decelerate rapidly.",
		}, "\n")},
		Quotation: &Quotation{Text: "The car began to decelerate rapidly."},
		Mark:      &Mark{Anchor: "pbr:/page?page=8"},
	}

	draft, ok := ParseNote(book, note)
	if !ok {
		t.Fatalf("ParseNote() skipped note")
	}
	if draft.RawWord != "decelerate" {
		t.Fatalf("RawWord = %q", draft.RawWord)
	}
	if len(draft.Translations) != 2 {
		t.Fatalf("translations = %#v", draft.Translations)
	}
	if len(draft.Contexts) != 1 {
		t.Fatalf("contexts = %#v", draft.Contexts)
	}
	if draft.Anchor.Page != "8" {
		t.Fatalf("Page = %q, want 8", draft.Anchor.Page)
	}
}

func TestParseNoteSkipsPlainLongHighlightComment(t *testing.T) {
	t.Parallel()

	_, ok := ParseNote(Book{ID: "book-1", Title: "Book"}, Note{
		UUID:      "note-1",
		Note:      &TextWithTime{Text: "This paragraph is important for the argument."},
		Quotation: &Quotation{Text: "This is a long highlighted paragraph that should not become a vocabulary word."},
	})
	if ok {
		t.Fatalf("ParseNote() accepted a plain long highlight comment")
	}
}

func TestFileSessionStoreRoundTripUsesOwnerOnlyPermissions(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "session.json")
	store := NewFileSessionStore(path)
	session := Session{
		AccessToken:          "access",
		RefreshToken:         "refresh",
		AccessTokenExpiresAt: time.Date(2026, 6, 14, 8, 0, 0, 0, time.UTC),
		ShopAlias:            "main",
	}

	if err := store.Save(context.Background(), session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat session file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("session file mode = %#o, want 0600", got)
	}

	loaded, ok, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !ok {
		t.Fatalf("Load() ok = false")
	}
	if loaded.RefreshToken != session.RefreshToken || loaded.AccessToken != session.AccessToken {
		t.Fatalf("loaded session = %#v", loaded)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write json: %v", err)
	}
}

func requireBearer(t *testing.T, r *http.Request, token string) {
	t.Helper()

	got := r.Header.Get("Authorization")
	want := "Bearer " + token
	if got != want {
		t.Fatalf("Authorization = %q, want %q", got, want)
	}
}

type testWriter struct {
	t *testing.T
}

func (w testWriter) Write(data []byte) (int, error) {
	w.t.Helper()
	w.t.Log(strings.TrimSpace(string(data)))

	return len(data), nil
}
