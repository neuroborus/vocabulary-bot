package vocabulary

import (
	"sort"
	"strings"
)

const sheetFingerprintSeparator = "\x1e"

// DraftFingerprint returns a stable content fingerprint for a spreadsheet draft row.
func DraftFingerprint(draft Draft) string {
	parts := []string{strings.TrimSpace(draft.RawWord)}
	parts = append(parts, normalizedFingerprintValues(draft.Translations, translationKey)...)
	parts = append(parts, normalizedFingerprintValues(draft.Contexts, contextKey)...)
	parts = append(parts, normalizedFingerprintValues(draft.Notes, comparableTextKey)...)
	parts = append(parts, normalizedFingerprintValues(draft.Tags, comparableTextKey)...)

	return strings.Join(parts, sheetFingerprintSeparator)
}

// SheetRowUnchanged reports whether the stored sheet anchor already matches the draft.
func SheetRowUnchanged(item Item, draft Draft) bool {
	if draft.Source != SourceGoogleSheet || draft.Anchor.RowNumber <= 0 {
		return false
	}

	fingerprint := DraftFingerprint(draft)
	sheetName := strings.TrimSpace(draft.Anchor.SheetName)

	for _, anchor := range item.Anchors {
		if anchor.Source != SourceGoogleSheet {
			continue
		}
		if anchor.RowNumber != draft.Anchor.RowNumber {
			continue
		}
		if strings.TrimSpace(anchor.SheetName) != sheetName {
			continue
		}

		return anchor.RowFingerprint != "" && anchor.RowFingerprint == fingerprint
	}

	return false
}

func normalizedFingerprintValues(values []string, key func(string) string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))

	for _, value := range values {
		candidate := key(strings.TrimSpace(value))
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		normalized = append(normalized, candidate)
	}

	sort.Strings(normalized)

	return normalized
}
