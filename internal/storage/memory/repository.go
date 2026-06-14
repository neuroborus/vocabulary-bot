package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type VocabularyRepository struct {
	mu    sync.RWMutex
	items map[string]vocabulary.Item
}

func NewVocabularyRepository() *VocabularyRepository {
	return &VocabularyRepository{
		items: make(map[string]vocabulary.Item),
	}
}

func (r *VocabularyRepository) FindByLookupKeys(ctx context.Context, lookupKeys []string) ([]vocabulary.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	keySet := make(map[string]struct{}, len(lookupKeys))
	for _, key := range lookupKeys {
		keySet[key] = struct{}{}
	}

	matches := make([]vocabulary.Item, 0)
	seen := make(map[string]struct{})

	for normalizedKey, item := range r.items {
		if _, ok := keySet[item.NormalizedKey]; ok {
			matches = appendMatch(matches, seen, normalizedKey, item)
			continue
		}

		for _, lookupKey := range item.LookupKeys {
			if _, ok := keySet[lookupKey]; ok {
				matches = appendMatch(matches, seen, normalizedKey, item)
				break
			}
		}
	}

	return matches, nil
}

func (r *VocabularyRepository) FindBySheetRow(ctx context.Context, sheetName string, rowNumber int) (vocabulary.Item, bool, error) {
	if err := ctx.Err(); err != nil {
		return vocabulary.Item{}, false, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	sheetName = strings.TrimSpace(sheetName)
	for _, item := range r.items {
		for _, anchor := range item.Anchors {
			if anchor.Source != vocabulary.SourceGoogleSheet {
				continue
			}
			if anchor.RowNumber != rowNumber {
				continue
			}
			if strings.TrimSpace(anchor.SheetName) != sheetName {
				continue
			}

			return cloneItem(item), true, nil
		}
	}

	return vocabulary.Item{}, false, nil
}

func (r *VocabularyRepository) Create(ctx context.Context, item vocabulary.Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[item.NormalizedKey]; ok {
		return fmt.Errorf("vocabulary item %q already exists", item.NormalizedKey)
	}

	r.items[item.NormalizedKey] = cloneItem(item)

	return nil
}

func (r *VocabularyRepository) Update(ctx context.Context, item vocabulary.Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[item.NormalizedKey]; !ok {
		return fmt.Errorf("vocabulary item %q not found", item.NormalizedKey)
	}

	r.items[item.NormalizedKey] = cloneItem(item)

	return nil
}

func (r *VocabularyRepository) Replace(ctx context.Context, item vocabulary.Item, previousNormalizedKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if previousNormalizedKey == "" {
		return fmt.Errorf("previous normalized key is required")
	}

	delete(r.items, previousNormalizedKey)
	r.items[item.NormalizedKey] = cloneItem(item)

	return nil
}

func (r *VocabularyRepository) Delete(ctx context.Context, normalizedKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	normalizedKey = strings.TrimSpace(normalizedKey)
	if normalizedKey == "" {
		return fmt.Errorf("normalized key is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[normalizedKey]; !ok {
		return fmt.Errorf("vocabulary item %q not found", normalizedKey)
	}

	delete(r.items, normalizedKey)

	return nil
}

func (r *VocabularyRepository) List(ctx context.Context) ([]vocabulary.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]vocabulary.Item, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, cloneItem(item))
	}

	return items, nil
}

func appendMatch(matches []vocabulary.Item, seen map[string]struct{}, key string, item vocabulary.Item) []vocabulary.Item {
	if _, ok := seen[key]; ok {
		return matches
	}

	seen[key] = struct{}{}
	return append(matches, cloneItem(item))
}

func cloneItem(item vocabulary.Item) vocabulary.Item {
	item.LookupKeys = append([]string(nil), item.LookupKeys...)
	item.Forms = append([]vocabulary.Form(nil), item.Forms...)
	for index := range item.Forms {
		item.Forms[index].LookupKeys = append([]string(nil), item.Forms[index].LookupKeys...)
	}
	item.Translations = append([]string(nil), item.Translations...)
	item.Contexts = append([]string(nil), item.Contexts...)
	item.Notes = append([]string(nil), item.Notes...)
	item.Tags = append([]string(nil), item.Tags...)
	item.Anchors = append([]vocabulary.SourceAnchor(nil), item.Anchors...)
	for index := range item.Anchors {
		if item.Anchors[index].RowSnapshot != nil {
			snapshot := *item.Anchors[index].RowSnapshot
			snapshot.Translations = append([]string(nil), snapshot.Translations...)
			snapshot.Contexts = append([]string(nil), snapshot.Contexts...)
			snapshot.Notes = append([]string(nil), snapshot.Notes...)
			snapshot.Tags = append([]string(nil), snapshot.Tags...)
			item.Anchors[index].RowSnapshot = &snapshot
		}
	}

	return item
}
