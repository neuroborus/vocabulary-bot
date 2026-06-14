package vocabulary

import (
	"testing"
	"time"
)

func TestIsDocumentPushCandidateForSpreadsheetOnlyItem(t *testing.T) {
	t.Parallel()

	item := Item{
		Anchors: []SourceAnchor{{
			Source:    SourceGoogleSheet,
			SheetName: "Vocabulary",
			RowNumber: 2,
		}},
	}

	if !IsDocumentPushCandidate(item) {
		t.Fatal("IsDocumentPushCandidate() = false, want true for spreadsheet-only item")
	}
}

func TestIsDocumentPushCandidateForPocketBookDocument(t *testing.T) {
	t.Parallel()

	item := Item{
		Anchors: []SourceAnchor{{
			Source:      SourcePocketBook,
			SourceLabel: DocumentSourceLabel,
		}},
	}

	if !IsDocumentPushCandidate(item) {
		t.Fatal("IsDocumentPushCandidate() = false, want true for PocketBook Document")
	}
}

func TestIsDocumentPushCandidateSkipsBookAnchors(t *testing.T) {
	t.Parallel()

	item := Item{
		Anchors: []SourceAnchor{{
			Source:      SourcePocketBook,
			SourceLabel: "Necromancer — Fred Saberhagen",
		}},
	}

	if IsDocumentPushCandidate(item) {
		t.Fatal("IsDocumentPushCandidate() = true, want false for book anchor")
	}
}

func TestIsDocumentPushCandidateSkipsMergedSpreadsheetAndBookItem(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	item := Item{
		Anchors: []SourceAnchor{
			{
				Source:     SourceGoogleSheet,
				SheetName:  "Vocabulary",
				RowNumber:  5,
				LastSeenAt: now,
			},
			{
				Source:      SourcePocketBook,
				SourceLabel: "Necromancer — Fred Saberhagen",
				LastSeenAt:  now.Add(time.Hour),
			},
		},
	}

	if IsDocumentPushCandidate(item) {
		t.Fatal("IsDocumentPushCandidate() = true, want false when a book anchor exists")
	}
	if !IsBookPushCandidate(item) {
		t.Fatal("IsBookPushCandidate() = false, want true when a book anchor exists")
	}
}

func TestIsBookPushCandidateSkipsSpreadsheetOnlyItem(t *testing.T) {
	t.Parallel()

	item := Item{
		Anchors: []SourceAnchor{{
			Source:    SourceGoogleSheet,
			SheetName: "Vocabulary",
			RowNumber: 2,
		}},
	}

	if IsBookPushCandidate(item) {
		t.Fatal("IsBookPushCandidate() = true, want false for spreadsheet-only item")
	}
}
