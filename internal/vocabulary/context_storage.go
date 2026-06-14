package vocabulary

import (
	"context"
	"strings"
)

func HasBookLikeContext(contexts []string) bool {
	for _, value := range contexts {
		if IsUsageExampleLine(value) {
			continue
		}

		cleaned := CleanContextLine(value)
		if cleaned == "" {
			continue
		}

		if looksLikeBookSentence(cleaned) {
			return true
		}
	}

	return false
}

func looksLikeBookSentence(value string) bool {
	tokens := Tokenize(value)
	return len(tokens) > 8 || strings.ContainsAny(value, ".!?")
}

func (s *Service) HasStoredBookLikeContext(ctx context.Context, rawWord string) (bool, error) {
	lookupKeys := BuildLookupKeys(rawWord)
	if len(lookupKeys) == 0 {
		return false, nil
	}

	matches, err := s.repository.FindByLookupKeys(ctx, lookupKeys)
	if err != nil {
		return false, err
	}
	if len(matches) != 1 {
		return false, nil
	}

	return HasBookLikeContext(matches[0].Contexts), nil
}
