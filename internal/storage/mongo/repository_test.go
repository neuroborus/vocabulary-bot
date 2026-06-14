package mongo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/source/session"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
)

func setupTestDatabase(t *testing.T) (*mongodriver.Database, func()) {
	t.Helper()

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://127.0.0.1:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client, err := Connect(ctx, uri)
	if err != nil {
		t.Skipf("mongodb not available at %s: %v", uri, err)
	}

	dbName := fmt.Sprintf("vocabulary_bot_test_%d", time.Now().UnixNano())
	database := client.Database(dbName)

	cleanup := func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dropCancel()
		_ = database.Drop(dropCtx)
		_ = client.Disconnect(dropCtx)
	}

	return database, cleanup
}

func sampleVocabularyItem(now time.Time) vocabulary.Item {
	return vocabulary.Item{
		NormalizedKey: "decelerate",
		LookupKeys:    []string{"decelerate", "to decelerate"},
		DisplayWord:   "to decelerate",
		Forms: []vocabulary.Form{
			{
				Value:           "to decelerate",
				NormalizedValue: "decelerate",
				LookupKeys:      []string{"decelerate", "to decelerate"},
				Count:           1,
				FirstSeenAt:     now,
				LastSeenAt:      now,
			},
		},
		Translations: []string{"замедлять"},
		Contexts:     []string{"The car began to decelerate."},
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestConnectRejectsInvalidURI(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := Connect(ctx, "://invalid-uri")
	if err == nil {
		t.Fatal("Connect() error = nil, want failure")
	}
}

func TestVocabularyRepositoryCreateFindUpdateList(t *testing.T) {
	database, cleanup := setupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	repository := NewVocabularyRepository(database)
	if err := repository.EnsureIndexes(ctx); err != nil {
		t.Fatalf("EnsureIndexes() error = %v", err)
	}

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	item := sampleVocabularyItem(now)

	if err := repository.Create(ctx, item); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	byLookupKey, err := repository.FindByLookupKeys(ctx, []string{"to decelerate"})
	if err != nil {
		t.Fatalf("FindByLookupKeys() error = %v", err)
	}
	if len(byLookupKey) != 1 {
		t.Fatalf("FindByLookupKeys() len = %d, want 1", len(byLookupKey))
	}
	if byLookupKey[0].DisplayWord != "to decelerate" {
		t.Fatalf("DisplayWord = %q, want %q", byLookupKey[0].DisplayWord, "to decelerate")
	}

	item.DisplayWord = "decelerate sharply"
	item.UpdatedAt = now.Add(time.Hour)
	if err := repository.Update(ctx, item); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	items, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("List() len = %d, want 1", len(items))
	}
	if items[0].DisplayWord != "decelerate sharply" {
		t.Fatalf("updated DisplayWord = %q", items[0].DisplayWord)
	}
}

func TestVocabularyRepositoryUpdateMissingItem(t *testing.T) {
	database, cleanup := setupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	repository := NewVocabularyRepository(database)

	err := repository.Update(ctx, sampleVocabularyItem(time.Now().UTC()))
	if err == nil {
		t.Fatal("Update() error = nil, want not found")
	}
}

func TestVocabularyRepositoryDelete(t *testing.T) {
	database, cleanup := setupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	repository := NewVocabularyRepository(database)

	now := time.Now().UTC()
	item := sampleVocabularyItem(now)
	if err := repository.Create(ctx, item); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repository.Delete(ctx, item.NormalizedKey); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	items, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("List() len = %d, want 0 after delete", len(items))
	}
}

func TestVocabularyRepositoryEnsureIndexesEnforcesUniqueNormalizedKey(t *testing.T) {
	database, cleanup := setupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	repository := NewVocabularyRepository(database)
	if err := repository.EnsureIndexes(ctx); err != nil {
		t.Fatalf("EnsureIndexes() error = %v", err)
	}

	now := time.Now().UTC()
	item := sampleVocabularyItem(now)
	if err := repository.Create(ctx, item); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	duplicate := item
	duplicate.DisplayWord = "duplicate display"
	if err := repository.Create(ctx, duplicate); err == nil {
		t.Fatal("Create() duplicate normalizedKey error = nil, want failure")
	}
}

func TestPocketBookSessionStoreRoundTrip(t *testing.T) {
	database, cleanup := setupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	store := NewPocketBookSessionStore(database)

	loaded, ok, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load() empty error = %v", err)
	}
	if ok {
		t.Fatalf("Load() empty ok = true, want false")
	}

	expiresAt := time.Date(2026, 6, 14, 13, 0, 0, 0, time.UTC)
	savedSession := session.PocketBook{
		AccessToken:          "FAKE_ACCESS_TOKEN_FOR_TEST_ONLY",
		RefreshToken:         "FAKE_REFRESH_TOKEN_FOR_TEST_ONLY",
		AccessTokenExpiresAt: expiresAt,
		ShopAlias:            "example-shop",
		ShopID:               "shop-123",
		ShopName:             "Example Shop",
	}
	if err := store.Save(ctx, savedSession); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, ok, err = store.Load(ctx)
	if err != nil {
		t.Fatalf("Load() saved error = %v", err)
	}
	if !ok {
		t.Fatal("Load() saved ok = false, want true")
	}
	if loaded != savedSession {
		t.Fatalf("loaded session = %#v, want %#v", loaded, savedSession)
	}

	savedSession.ShopName = "Updated Shop"
	if err := store.Save(ctx, savedSession); err != nil {
		t.Fatalf("Save() update error = %v", err)
	}

	loaded, ok, err = store.Load(ctx)
	if err != nil {
		t.Fatalf("Load() updated error = %v", err)
	}
	if !ok {
		t.Fatal("Load() updated ok = false, want true")
	}
	if loaded.ShopName != "Updated Shop" {
		t.Fatalf("ShopName = %q, want %q", loaded.ShopName, "Updated Shop")
	}
}

func TestPocketBookSessionStoreClear(t *testing.T) {
	database, cleanup := setupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	store := NewPocketBookSessionStore(database)

	if err := store.Clear(ctx); err != nil {
		t.Fatalf("Clear() empty error = %v", err)
	}

	if err := store.Save(ctx, session.PocketBook{
		AccessToken:  "FAKE_ACCESS_TOKEN_FOR_TEST_ONLY",
		RefreshToken: "FAKE_REFRESH_TOKEN_FOR_TEST_ONLY",
		ShopAlias:    "example-shop",
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := store.Clear(ctx); err != nil {
		t.Fatalf("Clear() saved error = %v", err)
	}

	_, ok, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load() after clear error = %v", err)
	}
	if ok {
		t.Fatal("Load() after clear ok = true, want false")
	}
}

func TestRepositoryRespectsCancelledContext(t *testing.T) {
	database, cleanup := setupTestDatabase(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repository := NewVocabularyRepository(database)
	store := NewPocketBookSessionStore(database)

	if _, err := repository.List(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("List() error = %v, want context.Canceled", err)
	}
	if err := repository.Create(ctx, sampleVocabularyItem(time.Now().UTC())); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create() error = %v, want context.Canceled", err)
	}
	if err := repository.Update(ctx, sampleVocabularyItem(time.Now().UTC())); !errors.Is(err, context.Canceled) {
		t.Fatalf("Update() error = %v, want context.Canceled", err)
	}
	if err := repository.Delete(ctx, "decelerate"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete() error = %v, want context.Canceled", err)
	}
	if _, err := repository.FindByLookupKeys(ctx, []string{"decelerate"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("FindByLookupKeys() error = %v, want context.Canceled", err)
	}
	if _, _, err := store.Load(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Load() error = %v, want context.Canceled", err)
	}
	if err := store.Save(ctx, session.PocketBook{ShopAlias: "example-shop"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Save() error = %v, want context.Canceled", err)
	}
	if err := store.Clear(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Clear() error = %v, want context.Canceled", err)
	}
}
