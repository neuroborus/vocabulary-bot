package pocketbook

import (
	"encoding/json"
	"time"
)

const (
	defaultBaseURL     = "https://cloud.pocketbook.digital/api/v1.0"
	pocketBookClientID = "qNAx1RDb"
	// Public client secret observed in PocketBook's first-party app flow and
	// reused by community importers. Treat it as a public OAuth client value,
	// not as an application secret.
	pocketBookClientSecret = "K3YYSjCgDJNoWKdGVOyO1mrROp3MMZqqRNXNXTmh"
)

type Shop struct {
	Alias  string `json:"alias"`
	Name   string `json:"name"`
	ShopID string `json:"shop_id"`
}

type shopsResponse struct {
	Providers []Shop `json:"providers"`
}

type authTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type Book struct {
	ID          string       `json:"id"`
	Path        string       `json:"path"`
	Title       string       `json:"title"`
	MimeType    string       `json:"mime_type"`
	CreatedAt   string       `json:"created_at"`
	FastHash    string       `json:"fast_hash"`
	ReadStatus  string       `json:"read_status"`
	Collections string       `json:"collections"`
	Metadata    BookMetadata `json:"metadata"`
	Raw         json.RawMessage
}

type BookMetadata struct {
	Title   string      `json:"title"`
	Authors string      `json:"authors"`
	Year    string      `json:"year"`
	ISBN    string      `json:"isbn"`
	Cover   []BookCover `json:"cover"`
}

type BookCover struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Path   string `json:"path"`
}

type booksResponse struct {
	Total int    `json:"total"`
	Items []Book `json:"items"`
}

type NoteInfo struct {
	Type    string       `json:"type"`
	UUID    string       `json:"uuid"`
	Updated FlexibleTime `json:"updated"`
	Raw     json.RawMessage
}

type Note struct {
	UUID      string         `json:"uuid"`
	Color     *ValueWithTime `json:"color"`
	Type      *ValueWithTime `json:"type"`
	Note      *TextWithTime  `json:"note"`
	Quotation *Quotation     `json:"quotation"`
	Mark      *Mark          `json:"mark"`
	Raw       json.RawMessage
}

type ValueWithTime struct {
	Value   string       `json:"value"`
	Updated FlexibleTime `json:"updated"`
}

type TextWithTime struct {
	Text    string       `json:"text"`
	Updated FlexibleTime `json:"updated"`
}

type Quotation struct {
	Begin   string       `json:"begin"`
	End     string       `json:"end"`
	Text    string       `json:"text"`
	Updated FlexibleTime `json:"updated"`
}

type Mark struct {
	Anchor  string       `json:"anchor"`
	Created FlexibleTime `json:"created"`
	Updated FlexibleTime `json:"updated"`
}

type FlexibleTime struct {
	Time time.Time
}

func (t *FlexibleTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || len(data) == 0 {
		return nil
	}

	var number float64
	if err := json.Unmarshal(data, &number); err == nil {
		if number > 0 {
			sec, frac := int64(number), number-float64(int64(number))
			t.Time = time.Unix(sec, int64(frac*1_000_000_000)).UTC()
		}
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	if text == "" {
		return nil
	}

	if parsed, err := time.Parse(time.RFC3339Nano, text); err == nil {
		t.Time = parsed.UTC()
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, text); err == nil {
		t.Time = parsed.UTC()
		return nil
	}

	return nil
}

func (t FlexibleTime) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}

	return json.Marshal(t.Time.Format(time.RFC3339Nano))
}
