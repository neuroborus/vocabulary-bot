package vocabulary

import (
	"strings"
	"time"
)

const spreadsheetSourceFallback = "Google Sheets"

func PrimarySourceLabel(item Item) string {
	if label := bestSourceLabel(item, isPocketBookBookAnchor); label != "" {
		return label
	}
	if label := bestSourceLabel(item, isPocketBookDocumentAnchor); label != "" {
		return label
	}
	if label := bestSourceLabel(item, isGoogleSheetAnchor); label != "" {
		return label
	}

	return ""
}

func bestSourceLabel(item Item, match func(SourceAnchor) bool) string {
	var (
		label  string
		seenAt time.Time
	)

	for _, anchor := range item.Anchors {
		if !match(anchor) {
			continue
		}

		candidate := anchorDisplayLabel(anchor)
		if candidate == "" {
			continue
		}

		anchorSeenAt := anchor.LastSeenAt
		if anchorSeenAt.IsZero() {
			anchorSeenAt = anchor.FirstSeenAt
		}
		if label == "" || anchorSeenAt.After(seenAt) {
			label = candidate
			seenAt = anchorSeenAt
		}
	}

	return label
}

func anchorDisplayLabel(anchor SourceAnchor) string {
	if candidate := strings.TrimSpace(anchor.SourceLabel); candidate != "" {
		return candidate
	}
	if candidate := legacyPocketBookSourceLabel(anchor); candidate != "" {
		return candidate
	}

	return spreadsheetSourceLabel(anchor)
}

func spreadsheetSourceLabel(anchor SourceAnchor) string {
	if anchor.Source != SourceGoogleSheet {
		return ""
	}
	if sheetName := strings.TrimSpace(anchor.SheetName); sheetName != "" {
		return sheetName
	}

	return spreadsheetSourceFallback
}

func isPocketBookBookAnchor(anchor SourceAnchor) bool {
	if anchor.Source != SourcePocketBook {
		return false
	}

	return anchorSourceLabel(anchor) != DocumentSourceLabel
}

func isPocketBookDocumentAnchor(anchor SourceAnchor) bool {
	return anchor.Source == SourcePocketBook && anchorSourceLabel(anchor) == DocumentSourceLabel
}

func isGoogleSheetAnchor(anchor SourceAnchor) bool {
	return anchor.Source == SourceGoogleSheet
}

func legacyPocketBookSourceLabel(anchor SourceAnchor) string {
	if anchor.Source != SourcePocketBook {
		return ""
	}

	title := strings.TrimSpace(anchor.BookTitle)
	author := strings.TrimSpace(anchor.Author)
	switch {
	case title != "" && author != "":
		return title + " — " + author
	case title != "":
		return title
	case author != "":
		return author
	default:
		return ""
	}
}
