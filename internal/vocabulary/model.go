package vocabulary

import "time"

type Source string

const (
	SourcePocketBook  Source = "pocketbook"
	SourceGoogleSheet Source = "google-sheet"
)

type Item struct {
	NormalizedKey string         `json:"normalizedKey" bson:"normalizedKey"`
	LookupKeys    []string       `json:"lookupKeys" bson:"lookupKeys"`
	DisplayWord   string         `json:"displayWord" bson:"displayWord"`
	Forms         []Form         `json:"forms" bson:"forms"`
	Translations  []string       `json:"translations" bson:"translations"`
	Contexts      []string       `json:"contexts" bson:"contexts"`
	Notes         []string       `json:"notes" bson:"notes"`
	Tags          []string       `json:"tags" bson:"tags"`
	Anchors       []SourceAnchor `json:"anchors" bson:"anchors"`
	Enabled       bool           `json:"enabled" bson:"enabled"`
	Review        ReviewState    `json:"review" bson:"review"`
	CreatedAt     time.Time      `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt" bson:"updatedAt"`
}

type Form struct {
	Value           string    `json:"value" bson:"value"`
	NormalizedValue string    `json:"normalizedValue" bson:"normalizedValue"`
	LookupKeys      []string  `json:"lookupKeys" bson:"lookupKeys"`
	Count           int       `json:"count" bson:"count"`
	FirstSeenAt     time.Time `json:"firstSeenAt" bson:"firstSeenAt"`
	LastSeenAt      time.Time `json:"lastSeenAt" bson:"lastSeenAt"`
}

type SourceAnchor struct {
	Source      Source    `json:"source" bson:"source"`
	ExternalID  string    `json:"externalId,omitempty" bson:"externalId,omitempty"`
	RowNumber   int       `json:"rowNumber,omitempty" bson:"rowNumber,omitempty"`
	SheetName   string    `json:"sheetName,omitempty" bson:"sheetName,omitempty"`
	BookID      string    `json:"bookId,omitempty" bson:"bookId,omitempty"`
	BookTitle   string    `json:"bookTitle,omitempty" bson:"bookTitle,omitempty"`
	Author      string    `json:"author,omitempty" bson:"author,omitempty"`
	Page        string    `json:"page,omitempty" bson:"page,omitempty"`
	Position    string    `json:"position,omitempty" bson:"position,omitempty"`
	FirstSeenAt time.Time `json:"firstSeenAt" bson:"firstSeenAt"`
	LastSeenAt  time.Time `json:"lastSeenAt" bson:"lastSeenAt"`
}

type ReviewState struct {
	Enabled      bool       `json:"enabled" bson:"enabled"`
	DueAt        *time.Time `json:"dueAt,omitempty" bson:"dueAt,omitempty"`
	LastPushedAt *time.Time `json:"lastPushedAt,omitempty" bson:"lastPushedAt,omitempty"`
	IntervalDays int        `json:"intervalDays" bson:"intervalDays"`
	Ease         float64    `json:"ease" bson:"ease"`
	EasyCount    int        `json:"easyCount" bson:"easyCount"`
	HardCount    int        `json:"hardCount" bson:"hardCount"`
	PushCount    int        `json:"pushCount" bson:"pushCount"`
}

type Draft struct {
	Source       Source
	RawWord      string
	Translations []string
	Contexts     []string
	Notes        []string
	Tags         []string
	Anchor       SourceAnchor
}
