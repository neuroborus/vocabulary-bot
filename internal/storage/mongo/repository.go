package mongo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/source/session"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	VocabularyCollectionName         = "vocabulary_items"
	PocketBookSessionsCollectionName = "pocketbook_sessions"
	pocketBookSessionID              = "default"
)

func Connect(ctx context.Context, uri string) (*mongodriver.Client, error) {
	client, err := mongodriver.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}

	return client, nil
}

type VocabularyRepository struct {
	collection *mongodriver.Collection
}

func NewVocabularyRepository(database *mongodriver.Database) *VocabularyRepository {
	return &VocabularyRepository{
		collection: database.Collection(VocabularyCollectionName),
	}
}

func (r *VocabularyRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongodriver.IndexModel{
		{
			Keys:    bson.D{{Key: "normalizedKey", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "lookupKeys", Value: 1}},
		},
	})

	return err
}

func (r *VocabularyRepository) FindByLookupKeys(ctx context.Context, lookupKeys []string) ([]vocabulary.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cursor, err := r.collection.Find(ctx, bson.M{
		"$or": bson.A{
			bson.M{"normalizedKey": bson.M{"$in": lookupKeys}},
			bson.M{"lookupKeys": bson.M{"$in": lookupKeys}},
		},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []vocabulary.Item
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *VocabularyRepository) FindBySheetRow(ctx context.Context, sheetName string, rowNumber int) (vocabulary.Item, bool, error) {
	if err := ctx.Err(); err != nil {
		return vocabulary.Item{}, false, err
	}

	var item vocabulary.Item
	err := r.collection.FindOne(ctx, bson.M{
		"anchors": bson.M{
			"$elemMatch": bson.M{
				"source":    vocabulary.SourceGoogleSheet,
				"sheetName": strings.TrimSpace(sheetName),
				"rowNumber": rowNumber,
			},
		},
	}).Decode(&item)
	if errors.Is(err, mongodriver.ErrNoDocuments) {
		return vocabulary.Item{}, false, nil
	}
	if err != nil {
		return vocabulary.Item{}, false, err
	}

	return item, true, nil
}

func (r *VocabularyRepository) Create(ctx context.Context, item vocabulary.Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := r.collection.InsertOne(ctx, item); err != nil {
		return err
	}

	return nil
}

func (r *VocabularyRepository) Update(ctx context.Context, item vocabulary.Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	result, err := r.collection.ReplaceOne(ctx, bson.M{"normalizedKey": item.NormalizedKey}, item)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("vocabulary item %q not found", item.NormalizedKey)
	}

	return nil
}

func (r *VocabularyRepository) Replace(ctx context.Context, item vocabulary.Item, previousNormalizedKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(previousNormalizedKey) == "" {
		return fmt.Errorf("previous normalized key is required")
	}

	result, err := r.collection.ReplaceOne(ctx, bson.M{"normalizedKey": previousNormalizedKey}, item)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("vocabulary item %q not found", previousNormalizedKey)
	}

	return nil
}

func (r *VocabularyRepository) List(ctx context.Context) ([]vocabulary.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cursor, err := r.collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []vocabulary.Item
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}

	return items, nil
}

type PocketBookSessionStore struct {
	collection *mongodriver.Collection
}

type pocketBookSessionDocument struct {
	ID                   string    `bson:"_id"`
	AccessToken          string    `bson:"accessToken"`
	RefreshToken         string    `bson:"refreshToken"`
	AccessTokenExpiresAt time.Time `bson:"accessTokenExpiresAt"`
	ShopAlias            string    `bson:"shopAlias"`
	ShopID               string    `bson:"shopId,omitempty"`
	ShopName             string    `bson:"shopName,omitempty"`
	UpdatedAt            time.Time `bson:"updatedAt"`
}

func NewPocketBookSessionStore(database *mongodriver.Database) *PocketBookSessionStore {
	return &PocketBookSessionStore{
		collection: database.Collection(PocketBookSessionsCollectionName),
	}
}

func (s *PocketBookSessionStore) Load(ctx context.Context) (session.PocketBook, bool, error) {
	if err := ctx.Err(); err != nil {
		return session.PocketBook{}, false, err
	}

	var document pocketBookSessionDocument
	err := s.collection.FindOne(ctx, bson.M{"_id": pocketBookSessionID}).Decode(&document)
	if err == nil {
		return document.toSession(), true, nil
	}
	if errors.Is(err, mongodriver.ErrNoDocuments) {
		return session.PocketBook{}, false, nil
	}

	return session.PocketBook{}, false, err
}

func (s *PocketBookSessionStore) Save(ctx context.Context, pocketbookSession session.PocketBook) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	document := pocketBookSessionDocument{
		ID:                   pocketBookSessionID,
		AccessToken:          pocketbookSession.AccessToken,
		RefreshToken:         pocketbookSession.RefreshToken,
		AccessTokenExpiresAt: pocketbookSession.AccessTokenExpiresAt,
		ShopAlias:            pocketbookSession.ShopAlias,
		ShopID:               pocketbookSession.ShopID,
		ShopName:             pocketbookSession.ShopName,
		UpdatedAt:            time.Now().UTC(),
	}

	_, err := s.collection.ReplaceOne(
		ctx,
		bson.M{"_id": pocketBookSessionID},
		document,
		options.Replace().SetUpsert(true),
	)

	return err
}

func (s *PocketBookSessionStore) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	_, err := s.collection.DeleteOne(ctx, bson.M{"_id": pocketBookSessionID})
	return err
}

func (d pocketBookSessionDocument) toSession() session.PocketBook {
	return session.PocketBook{
		AccessToken:          d.AccessToken,
		RefreshToken:         d.RefreshToken,
		AccessTokenExpiresAt: d.AccessTokenExpiresAt,
		ShopAlias:            d.ShopAlias,
		ShopID:               d.ShopID,
		ShopName:             d.ShopName,
	}
}
