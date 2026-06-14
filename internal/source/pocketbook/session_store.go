package pocketbook

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Session struct {
	AccessToken          string    `json:"accessToken"`
	RefreshToken         string    `json:"refreshToken"`
	AccessTokenExpiresAt time.Time `json:"accessTokenExpiresAt"`
	ShopAlias            string    `json:"shopAlias"`
	ShopID               string    `json:"shopId,omitempty"`
	ShopName             string    `json:"shopName,omitempty"`
}

type SessionStore interface {
	Load(ctx context.Context) (Session, bool, error)
	Save(ctx context.Context, session Session) error
	Clear(ctx context.Context) error
}

type MemorySessionStore struct {
	mu      sync.RWMutex
	session Session
	ok      bool
}

func NewMemorySessionStore(session Session) *MemorySessionStore {
	return &MemorySessionStore{
		session: session,
		ok:      session.AccessToken != "" || session.RefreshToken != "",
	}
}

func (s *MemorySessionStore) Load(ctx context.Context) (Session, bool, error) {
	if err := ctx.Err(); err != nil {
		return Session{}, false, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.session, s.ok, nil
}

func (s *MemorySessionStore) Save(ctx context.Context, session Session) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.session = session
	s.ok = true

	return nil
}

func (s *MemorySessionStore) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.session = Session{}
	s.ok = false

	return nil
}

type FileSessionStore struct {
	path string
}

func NewFileSessionStore(path string) *FileSessionStore {
	if path == "" {
		path = DefaultSessionPath()
	}

	return &FileSessionStore{path: path}
}

func DefaultSessionPath() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "vocabulary-bot", "pocketbook-session.json")
	}

	return filepath.Join(os.TempDir(), "vocabulary-bot", "pocketbook-session.json")
}

func (s *FileSessionStore) Load(ctx context.Context) (Session, bool, error) {
	if err := ctx.Err(); err != nil {
		return Session{}, false, err
	}

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Session{}, false, nil
	}
	if err != nil {
		return Session{}, false, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return Session{}, false, err
	}

	return session, session.AccessToken != "" || session.RefreshToken != "", nil
}

func (s *FileSessionStore) Save(ctx context.Context, session Session) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, ".pocketbook-session-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return err
	}

	removeTmp = false
	return nil
}

func (s *FileSessionStore) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}
