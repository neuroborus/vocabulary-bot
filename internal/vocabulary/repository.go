package vocabulary

import "context"

type Repository interface {
	FindByLookupKeys(ctx context.Context, lookupKeys []string) ([]Item, error)
	FindBySheetRow(ctx context.Context, sheetName string, rowNumber int) (Item, bool, error)
	Create(ctx context.Context, item Item) error
	Update(ctx context.Context, item Item) error
	List(ctx context.Context) ([]Item, error)
}
