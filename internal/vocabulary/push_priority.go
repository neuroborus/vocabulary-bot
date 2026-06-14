package vocabulary

import "strings"

const DocumentSourceLabel = "Document"

// IsDocumentPushCandidate reports whether a word should use the lower document
// push factor during review selection.
//
// Candidates are spreadsheet-only items and PocketBook PDF notes labeled Document.
// Words that also have a non-document PocketBook book anchor stay at full priority.
func IsDocumentPushCandidate(item Item) bool {
	if len(item.Anchors) == 0 {
		return false
	}

	onlyGoogleSheet := true
	hasExplicitDocument := false

	for _, anchor := range item.Anchors {
		switch anchor.Source {
		case SourceGoogleSheet:
			continue
		case SourcePocketBook:
			onlyGoogleSheet = false
			if anchorSourceLabel(anchor) != DocumentSourceLabel {
				return false
			}
			hasExplicitDocument = true
		default:
			return false
		}
	}

	return onlyGoogleSheet || hasExplicitDocument
}

// IsBookPushCandidate reports whether a word should use the book push factor.
//
// Candidates are PocketBook items with a non-Document book anchor.
func IsBookPushCandidate(item Item) bool {
	for _, anchor := range item.Anchors {
		if anchor.Source != SourcePocketBook {
			continue
		}
		if anchorSourceLabel(anchor) == DocumentSourceLabel {
			continue
		}
		if anchorSourceLabel(anchor) != "" || strings.TrimSpace(anchor.BookTitle) != "" {
			return true
		}
	}

	return false
}

func anchorSourceLabel(anchor SourceAnchor) string {
	if label := strings.TrimSpace(anchor.SourceLabel); label != "" {
		return label
	}

	return legacyPocketBookSourceLabel(anchor)
}
