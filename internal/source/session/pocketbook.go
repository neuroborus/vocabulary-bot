package session

import (
	"context"
	"time"
)

// PocketBook holds OAuth tokens and shop metadata for PocketBook Cloud.
type PocketBook struct {
	AccessToken          string    `json:"accessToken"`
	RefreshToken         string    `json:"refreshToken"`
	AccessTokenExpiresAt time.Time `json:"accessTokenExpiresAt"`
	ShopAlias            string    `json:"shopAlias"`
	ShopID               string    `json:"shopId,omitempty"`
	ShopName             string    `json:"shopName,omitempty"`
}

// PocketBookStore persists PocketBook auth state across process restarts.
type PocketBookStore interface {
	Load(ctx context.Context) (PocketBook, bool, error)
	Save(ctx context.Context, session PocketBook) error
	Clear(ctx context.Context) error
}
