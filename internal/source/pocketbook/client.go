package pocketbook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrNotAuthenticated = errors.New("pocketbook credentials are not configured")

type HTTPError struct {
	StatusCode int
	Status     string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("pocketbook http error %d: %s", e.StatusCode, e.Status)
}

func IsAuthError(err error) bool {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == http.StatusBadRequest ||
			httpErr.StatusCode == http.StatusUnauthorized ||
			httpErr.StatusCode == http.StatusForbidden
	}

	return errors.Is(err, ErrNotAuthenticated)
}

type Client struct {
	baseURL      string
	email        string
	password     string
	refreshToken string
	shopName     string
	httpClient   *http.Client
	store        SessionStore
	logger       *slog.Logger
	now          func() time.Time
}

type ClientOptions struct {
	BaseURL      string
	Email        string
	Password     string
	RefreshToken string
	ShopName     string
	HTTPClient   *http.Client
	SessionStore SessionStore
	Logger       *slog.Logger
	Now          func() time.Time
}

func NewClient(options ClientOptions) *Client {
	if options.BaseURL == "" {
		options.BaseURL = defaultBaseURL
	}
	options.BaseURL = strings.TrimRight(options.BaseURL, "/")

	if options.HTTPClient == nil {
		options.HTTPClient = &http.Client{Timeout: 60 * time.Second}
	}
	if options.SessionStore == nil {
		options.SessionStore = NewMemorySessionStore(Session{})
	}
	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	if options.Now == nil {
		options.Now = time.Now
	}

	return &Client{
		baseURL:      options.BaseURL,
		email:        strings.TrimSpace(options.Email),
		password:     options.Password,
		refreshToken: strings.TrimSpace(options.RefreshToken),
		shopName:     strings.TrimSpace(options.ShopName),
		httpClient:   options.HTTPClient,
		store:        options.SessionStore,
		logger:       options.Logger,
		now:          options.Now,
	}
}

func (c *Client) Authenticate(ctx context.Context) (Session, error) {
	return c.authenticate(ctx, false)
}

func (c *Client) ForceRenew(ctx context.Context) (Session, error) {
	return c.authenticate(ctx, true)
}

func (c *Client) authenticate(ctx context.Context, force bool) (Session, error) {
	session, ok, err := c.store.Load(ctx)
	if err != nil {
		return Session{}, fmt.Errorf("load pocketbook session: %w", err)
	}
	if !ok {
		session = Session{}
	}
	if c.refreshToken != "" && session.RefreshToken == "" {
		session.RefreshToken = c.refreshToken
	}

	if !force && c.isAccessTokenValid(session) {
		return session, nil
	}

	if session.RefreshToken != "" {
		renewed, err := c.renewSession(ctx, session)
		if err == nil {
			if err := c.store.Save(ctx, renewed); err != nil {
				return Session{}, fmt.Errorf("save renewed pocketbook session: %w", err)
			}
			return renewed, nil
		}
		if !IsAuthError(err) {
			return Session{}, fmt.Errorf("refresh pocketbook session: %w", err)
		}

		c.logger.Warn("stored pocketbook refresh token failed; attempting password bootstrap")
	}

	if c.email == "" || c.password == "" {
		return Session{}, ErrNotAuthenticated
	}

	bootstrapped, err := c.bootstrapSession(ctx)
	if err != nil {
		return Session{}, err
	}
	if err := c.store.Save(ctx, bootstrapped); err != nil {
		return Session{}, fmt.Errorf("save bootstrapped pocketbook session: %w", err)
	}

	return bootstrapped, nil
}

func (c *Client) isAccessTokenValid(session Session) bool {
	if session.AccessToken == "" || session.AccessTokenExpiresAt.IsZero() {
		return false
	}

	return session.AccessTokenExpiresAt.After(c.now().UTC().Add(5 * time.Minute))
}

func (c *Client) bootstrapSession(ctx context.Context) (Session, error) {
	shops, err := c.GetShops(ctx, c.email)
	if err != nil {
		return Session{}, fmt.Errorf("list pocketbook shops: %w", err)
	}

	shop, err := chooseShop(shops, c.shopName)
	if err != nil {
		return Session{}, err
	}

	tokens, err := c.passwordLogin(ctx, shop)
	if err != nil {
		return Session{}, fmt.Errorf("password bootstrap pocketbook session: %w", err)
	}

	return c.sessionFromTokens(tokens, Session{
		ShopAlias: shop.Alias,
		ShopID:    shop.ShopID,
		ShopName:  shop.Name,
	}), nil
}

func (c *Client) GetShops(ctx context.Context, email string) ([]Shop, error) {
	values := url.Values{}
	values.Set("username", email)
	values.Set("client_id", pocketBookClientID)
	values.Set("client_secret", pocketBookClientSecret)

	endpoint := c.baseURL + "/auth/login?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cache-Control", "no-cache")

	var response shopsResponse
	if err := c.doJSON(req, &response); err != nil {
		return nil, err
	}

	return response.Providers, nil
}

func (c *Client) passwordLogin(ctx context.Context, shop Shop) (authTokens, error) {
	values := url.Values{}
	values.Set("shop_id", firstNonEmpty(shop.ShopID, "1"))
	values.Set("username", c.email)
	values.Set("password", c.password)
	values.Set("client_id", pocketBookClientID)
	values.Set("client_secret", pocketBookClientSecret)
	values.Set("grant_type", "password")
	values.Set("language", "en")

	endpoint := c.baseURL + "/auth/login/" + url.PathEscape(shop.Alias)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return authTokens{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var tokens authTokens
	if err := c.doJSON(req, &tokens); err != nil {
		return authTokens{}, err
	}

	return tokens, nil
}

func (c *Client) renewSession(ctx context.Context, current Session) (Session, error) {
	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", current.RefreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/auth/renew-token", strings.NewReader(values.Encode()))
	if err != nil {
		return Session{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if current.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+current.AccessToken)
	}

	var tokens authTokens
	if err := c.doJSON(req, &tokens); err != nil {
		return Session{}, err
	}

	return c.sessionFromTokens(tokens, current), nil
}

func (c *Client) sessionFromTokens(tokens authTokens, current Session) Session {
	session := current
	session.AccessToken = tokens.AccessToken
	if tokens.RefreshToken != "" {
		session.RefreshToken = tokens.RefreshToken
	}
	if tokens.ExpiresIn > 0 {
		session.AccessTokenExpiresAt = c.now().UTC().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	}

	return session
}

func (c *Client) ListBooks(ctx context.Context) ([]Book, error) {
	var response booksResponse
	if err := c.doAuthorizedJSON(ctx, http.MethodGet, "/books", url.Values{"limit": {"500"}}, nil, &response); err != nil {
		return nil, err
	}

	return response.Items, nil
}

func (c *Client) ListNoteIDs(ctx context.Context, fastHash string) ([]NoteInfo, error) {
	var notes []NoteInfo
	if err := c.doAuthorizedJSON(ctx, http.MethodGet, "/notes", url.Values{"fast_hash": {fastHash}}, nil, &notes); err != nil {
		return nil, err
	}

	return notes, nil
}

func (c *Client) GetNote(ctx context.Context, uuid string, fastHash string) (Note, bool, error) {
	var note Note
	path := "/notes/" + url.PathEscape(uuid)
	err := c.doAuthorizedJSON(ctx, http.MethodGet, path, url.Values{"fast_hash": {fastHash}}, nil, &note)
	if err == nil {
		return note, true, nil
	}

	var httpErr *HTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
		return Note{}, false, nil
	}

	return Note{}, false, err
}

func (c *Client) DownloadFile(ctx context.Context, fileURL string, destination string) error {
	fileURL = strings.TrimSpace(fileURL)
	if fileURL == "" {
		return fmt.Errorf("download url is empty")
	}

	if !strings.Contains(fileURL, "://") {
		fileURL = strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(fileURL, "/")
	}

	session, err := c.Authenticate(ctx)
	if err != nil {
		return err
	}

	if err := c.downloadToFile(ctx, fileURL, destination, session.AccessToken); err == nil {
		return nil
	} else if !isAuthHTTPError(err) {
		return err
	}

	if err := c.downloadToFile(ctx, fileURL, destination, ""); err == nil {
		return nil
	} else if !isAuthHTTPError(err) {
		return err
	}

	if _, renewErr := c.ForceRenew(ctx); renewErr != nil {
		return fmt.Errorf("renew pocketbook session after download auth failure: %w", renewErr)
	}

	session, err = c.Authenticate(ctx)
	if err != nil {
		return err
	}

	return c.downloadToFile(ctx, fileURL, destination, session.AccessToken)
}

func isAuthHTTPError(err error) bool {
	var httpErr *HTTPError
	return errors.As(err, &httpErr) &&
		(httpErr.StatusCode == http.StatusUnauthorized || httpErr.StatusCode == http.StatusForbidden)
}

func (c *Client) downloadToFile(ctx context.Context, fileURL, destination, accessToken string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return err
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", "vocabulary-bot/0.1")

	response, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return &HTTPError{StatusCode: response.StatusCode, Status: response.Status}
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return fmt.Errorf("create download dir: %w", err)
	}

	tempDestination := destination + ".part"
	file, err := os.OpenFile(tempDestination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create download file: %w", err)
	}

	_, copyErr := io.Copy(file, response.Body)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(tempDestination)
		return fmt.Errorf("write download file: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tempDestination)
		return fmt.Errorf("close download file: %w", closeErr)
	}

	if err := os.Rename(tempDestination, destination); err != nil {
		_ = os.Remove(tempDestination)
		return fmt.Errorf("finalize download file: %w", err)
	}

	return nil
}

func (c *Client) doAuthorizedJSON(ctx context.Context, method string, path string, query url.Values, body []byte, target any) error {
	var lastAuthErr error

	for attempt := 0; attempt < 2; attempt++ {
		session, err := c.Authenticate(ctx)
		if err != nil {
			return err
		}

		endpoint := c.baseURL + path
		if len(query) > 0 {
			endpoint += "?" + query.Encode()
		}

		req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+session.AccessToken)
		req.Header.Set("Cache-Control", "no-cache")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		if err := c.doJSON(req, target); err != nil {
			if IsAuthError(err) && attempt == 0 {
				lastAuthErr = err
				if _, renewErr := c.ForceRenew(ctx); renewErr != nil {
					return fmt.Errorf("renew pocketbook session after auth failure: %w", renewErr)
				}
				continue
			}

			return err
		}

		return nil
	}

	return lastAuthErr
}

func (c *Client) doJSON(req *http.Request, target any) error {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "vocabulary-bot/0.1")

	response, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return &HTTPError{
			StatusCode: response.StatusCode,
			Status:     response.Status,
		}
	}

	if target == nil {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil
	}

	if rawTarget, ok := target.(*json.RawMessage); ok {
		data, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		*rawTarget = data
		return nil
	}

	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(target); err != nil {
		return err
	}

	return nil
}

func chooseShop(shops []Shop, preferred string) (Shop, error) {
	if len(shops) == 0 {
		return Shop{}, errors.New("pocketbook account returned no shops")
	}

	preferred = strings.ToLower(strings.TrimSpace(preferred))
	if preferred == "" {
		return shops[0], nil
	}

	for _, shop := range shops {
		if strings.Contains(strings.ToLower(shop.Name), preferred) ||
			strings.Contains(strings.ToLower(shop.Alias), preferred) ||
			strings.EqualFold(shop.ShopID, preferred) {
			return shop, nil
		}
	}

	return Shop{}, fmt.Errorf("pocketbook shop %q was not found", preferred)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}
