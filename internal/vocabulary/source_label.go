package vocabulary

import (
	"strings"
	"time"
)

func PrimarySourceLabel(item Item) string {
	var (
		label  string
		seenAt time.Time
	)

	for _, anchor := range item.Anchors {
		candidate := strings.TrimSpace(anchor.SourceLabel)
		if candidate == "" {
			candidate = legacyPocketBookSourceLabel(anchor)
		}
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
