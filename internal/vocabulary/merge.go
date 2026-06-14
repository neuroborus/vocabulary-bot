package vocabulary

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrEmptyWord      = errors.New("empty vocabulary word")
	ErrAmbiguousMatch = errors.New("ambiguous vocabulary match")
)

type Service struct {
	repository Repository
	now        func() time.Time
}

type MergeOutcome struct {
	NormalizedKey string
	LookupKeys    []string
	Created       bool
	Updated       bool
	Skipped       bool
	Ambiguous     bool
}

func NewService(repository Repository, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}

	return &Service{
		repository: repository,
		now:        now,
	}
}

func (s *Service) MergeDraft(ctx context.Context, draft Draft) (MergeOutcome, error) {
	lookupKeys := BuildLookupKeys(draft.RawWord)
	if len(lookupKeys) == 0 {
		return MergeOutcome{}, ErrEmptyWord
	}

	matches, err := s.repository.FindByLookupKeys(ctx, lookupKeys)
	if err != nil {
		return MergeOutcome{}, fmt.Errorf("find vocabulary item: %w", err)
	}

	if len(matches) > 1 {
		return MergeOutcome{
			LookupKeys: lookupKeys,
			Ambiguous:  true,
		}, ErrAmbiguousMatch
	}

	now := s.now().UTC()

	if draft.Source == SourceGoogleSheet && draft.Anchor.RowNumber > 0 {
		item, found, err := s.repository.FindBySheetRow(ctx, draft.Anchor.SheetName, draft.Anchor.RowNumber)
		if err != nil {
			return MergeOutcome{}, fmt.Errorf("find sheet row anchor: %w", err)
		}
		if found {
			if SheetRowUnchanged(item, draft) {
				return MergeOutcome{
					NormalizedKey: item.NormalizedKey,
					LookupKeys:    append([]string(nil), item.LookupKeys...),
					Skipped:       true,
				}, nil
			}

			MergeIntoItem(&item, draft, now)
			if err := s.repository.Update(ctx, item); err != nil {
				return MergeOutcome{}, fmt.Errorf("update vocabulary item: %w", err)
			}

			return MergeOutcome{
				NormalizedKey: item.NormalizedKey,
				LookupKeys:    item.LookupKeys,
				Updated:       true,
			}, nil
		}
	}

	if len(matches) == 0 {
		item, err := NewItemFromDraft(draft, now)
		if err != nil {
			return MergeOutcome{}, err
		}

		if err := s.repository.Create(ctx, item); err != nil {
			return MergeOutcome{}, fmt.Errorf("create vocabulary item: %w", err)
		}

		return MergeOutcome{
			NormalizedKey: item.NormalizedKey,
			LookupKeys:    item.LookupKeys,
			Created:       true,
		}, nil
	}

	item := matches[0]
	MergeIntoItem(&item, draft, now)

	if err := s.repository.Update(ctx, item); err != nil {
		return MergeOutcome{}, fmt.Errorf("update vocabulary item: %w", err)
	}

	return MergeOutcome{
		NormalizedKey: item.NormalizedKey,
		LookupKeys:    item.LookupKeys,
		Updated:       true,
	}, nil
}

func NewItemFromDraft(draft Draft, now time.Time) (Item, error) {
	rawWord := strings.TrimSpace(draft.RawWord)
	lookupKeys := BuildLookupKeys(rawWord)
	if len(lookupKeys) == 0 {
		return Item{}, ErrEmptyWord
	}

	anchor := normalizeAnchor(draft, now)

	item := Item{
		NormalizedKey: lookupKeys[0],
		LookupKeys:    append([]string(nil), lookupKeys...),
		DisplayWord:   rawWord,
		Forms: []Form{
			newForm(rawWord, now),
		},
		Translations: cleanTextList(draft.Translations, translationKey),
		Contexts:     cleanTextList(draft.Contexts, contextKey),
		Notes:        cleanTextList(draft.Notes, comparableTextKey),
		Tags:         cleanTextList(draft.Tags, comparableTextKey),
		Anchors:      nil,
		Enabled:      true,
		Review: ReviewState{
			Enabled:      true,
			IntervalDays: 1,
			Ease:         2.5,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if anchor.Source != "" {
		item.Anchors = []SourceAnchor{anchor}
	}

	NormalizeUsageExamples(&item)

	return item, nil
}

func MergeIntoItem(item *Item, draft Draft, now time.Time) {
	rawWord := strings.TrimSpace(draft.RawWord)
	form := newForm(rawWord, now)

	mergeForm(item, form, now)
	item.LookupKeys = mergeStringsByKey(item.LookupKeys, form.LookupKeys, func(value string) string { return value })
	item.DisplayWord = chooseDisplayWord(item.DisplayWord, rawWord, item.Forms)
	item.Translations = mergeStringsByKey(item.Translations, draft.Translations, translationKey)
	item.Contexts = mergeStringsByKey(item.Contexts, draft.Contexts, contextKey)
	item.Notes = mergeStringsByKey(item.Notes, draft.Notes, comparableTextKey)
	item.Tags = mergeStringsByKey(item.Tags, draft.Tags, comparableTextKey)

	anchor := normalizeAnchor(draft, now)
	if anchor.Source != "" {
		mergeAnchor(item, anchor, now)
	}

	item.UpdatedAt = now
	NormalizeUsageExamples(item)
}

func newForm(rawWord string, now time.Time) Form {
	return Form{
		Value:           strings.TrimSpace(rawWord),
		NormalizedValue: NormalizeText(rawWord),
		LookupKeys:      BuildLookupKeys(rawWord),
		Count:           1,
		FirstSeenAt:     now,
		LastSeenAt:      now,
	}
}

func mergeForm(item *Item, incoming Form, now time.Time) {
	incomingKey := formKey(incoming.Value)

	for index := range item.Forms {
		if formKey(item.Forms[index].Value) == incomingKey {
			item.Forms[index].Count++
			item.Forms[index].LastSeenAt = now
			item.Forms[index].LookupKeys = mergeStringsByKey(
				item.Forms[index].LookupKeys,
				incoming.LookupKeys,
				func(value string) string { return value },
			)

			return
		}
	}

	item.Forms = append(item.Forms, incoming)
}

func chooseDisplayWord(current string, incoming string, forms []Form) string {
	current = strings.TrimSpace(current)
	incoming = strings.TrimSpace(incoming)

	if current == "" {
		return incoming
	}
	if incoming == "" {
		return current
	}

	currentTokens := len(Tokenize(current))
	incomingTokens := len(Tokenize(incoming))
	if incomingTokens > currentTokens {
		return incoming
	}
	if incomingTokens < currentTokens {
		return current
	}

	currentHasLeadingArticle := hasLeadingArticle(current)
	incomingHasLeadingArticle := hasLeadingArticle(incoming)
	if incomingHasLeadingArticle && !currentHasLeadingArticle {
		return incoming
	}
	if currentHasLeadingArticle && !incomingHasLeadingArticle {
		return current
	}

	currentHasTrailingWords := hasTrailingKnownForm(current, forms)
	incomingHasTrailingWords := hasTrailingKnownForm(incoming, forms)
	if incomingHasTrailingWords && !currentHasTrailingWords {
		return incoming
	}
	if currentHasTrailingWords && !incomingHasTrailingWords {
		return current
	}

	currentCount := formCount(forms, current)
	incomingCount := formCount(forms, incoming)
	if incomingCount > currentCount {
		return incoming
	}

	return current
}

func hasLeadingArticle(value string) bool {
	tokens := Tokenize(value)
	if len(tokens) == 0 {
		return false
	}

	switch tokens[0] {
	case "a", "an", "the":
		return true
	default:
		return false
	}
}

func hasTrailingKnownForm(value string, forms []Form) bool {
	tokens := Tokenize(value)
	if len(tokens) < 2 {
		return false
	}

	for _, form := range forms {
		formTokens := Tokenize(form.Value)
		if len(formTokens) == 0 || len(formTokens) >= len(tokens) {
			continue
		}
		if hasTokenPrefix(tokens, formTokens) {
			return true
		}
	}

	return false
}

func hasTokenPrefix(tokens []string, prefix []string) bool {
	if len(prefix) > len(tokens) {
		return false
	}

	for index, token := range prefix {
		if tokens[index] != token {
			return false
		}
	}

	return true
}

func formCount(forms []Form, value string) int {
	key := formKey(value)
	for _, form := range forms {
		if formKey(form.Value) == key {
			return form.Count
		}
	}

	return 0
}

func normalizeAnchor(draft Draft, now time.Time) SourceAnchor {
	anchor := draft.Anchor
	if anchor.Source == "" {
		anchor.Source = draft.Source
	}
	if anchor.FirstSeenAt.IsZero() {
		anchor.FirstSeenAt = now
	}
	if anchor.LastSeenAt.IsZero() {
		anchor.LastSeenAt = now
	}
	if anchor.Source == SourceGoogleSheet {
		anchor.RowFingerprint = DraftFingerprint(draft)
	}

	return anchor
}

func mergeAnchor(item *Item, incoming SourceAnchor, now time.Time) {
	incomingKey := anchorKey(incoming)

	for index := range item.Anchors {
		if anchorKey(item.Anchors[index]) == incomingKey {
			if item.Anchors[index].FirstSeenAt.IsZero() {
				item.Anchors[index].FirstSeenAt = incoming.FirstSeenAt
			}
			item.Anchors[index].LastSeenAt = now
			if incoming.SourceLabel != "" {
				item.Anchors[index].SourceLabel = incoming.SourceLabel
			}
			if incoming.BookTitle != "" {
				item.Anchors[index].BookTitle = incoming.BookTitle
			}
			if incoming.Author != "" {
				item.Anchors[index].Author = incoming.Author
			}
			if incoming.RowFingerprint != "" {
				item.Anchors[index].RowFingerprint = incoming.RowFingerprint
			}
			return
		}
	}

	item.Anchors = append(item.Anchors, incoming)
}

func cleanTextList(values []string, key func(string) string) []string {
	return mergeStringsByKey(nil, values, key)
}

func mergeStringsByKey(existing []string, incoming []string, key func(string) string) []string {
	result := append([]string(nil), existing...)
	seen := make(map[string]struct{}, len(result))

	for _, value := range result {
		if normalized := key(value); normalized != "" {
			seen[normalized] = struct{}{}
		}
	}

	for _, value := range incoming {
		cleaned := strings.TrimSpace(value)
		if cleaned == "" {
			continue
		}

		normalized := key(cleaned)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}

		result = append(result, cleaned)
		seen[normalized] = struct{}{}
	}

	return result
}

func formKey(value string) string {
	return CompactKey(value)
}

func translationKey(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func contextKey(value string) string {
	return comparableTextKey(value)
}

func comparableTextKey(value string) string {
	return NormalizeText(value)
}

func anchorKey(anchor SourceAnchor) string {
	source := string(anchor.Source)
	if anchor.ExternalID != "" {
		return source + "|external:" + anchor.ExternalID
	}

	return fmt.Sprintf(
		"%s|row:%d|sheet:%s|book:%s|page:%s|position:%s",
		source,
		anchor.RowNumber,
		anchor.SheetName,
		anchor.BookID,
		anchor.Page,
		anchor.Position,
	)
}
