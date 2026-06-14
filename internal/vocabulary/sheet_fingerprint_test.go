package vocabulary

import "testing"

func TestDraftFingerprintIsStableRegardlessOfListOrder(t *testing.T) {
	t.Parallel()

	left := Draft{
		RawWord:      "carve",
		Translations: []string{"вырезать", "резать"},
		Contexts:     []string{"Context B.", "Context A."},
	}
	right := Draft{
		RawWord:      "carve",
		Translations: []string{"резать", "вырезать"},
		Contexts:     []string{"Context A.", "Context B."},
	}

	if DraftFingerprint(left) != DraftFingerprint(right) {
		t.Fatalf("fingerprints differ for reordered values: %q vs %q", DraftFingerprint(left), DraftFingerprint(right))
	}
}

func TestSheetRowUnchangedMatchesStoredAnchorFingerprint(t *testing.T) {
	t.Parallel()

	draft := Draft{
		Source:       SourceGoogleSheet,
		RawWord:      "carve",
		Translations: []string{"вырезать"},
		Anchor: SourceAnchor{
			Source:    SourceGoogleSheet,
			SheetName: "Vocabulary",
			RowNumber: 12,
		},
	}

	item := Item{
		Anchors: []SourceAnchor{{
			Source:         SourceGoogleSheet,
			SheetName:      "Vocabulary",
			RowNumber:      12,
			RowFingerprint: DraftFingerprint(draft),
		}},
	}

	if !SheetRowUnchanged(item, draft) {
		t.Fatal("SheetRowUnchanged() = false, want true")
	}
}

func TestSheetRowUnchangedRequiresStoredFingerprint(t *testing.T) {
	t.Parallel()

	draft := Draft{
		Source:  SourceGoogleSheet,
		RawWord: "carve",
		Anchor: SourceAnchor{
			Source:    SourceGoogleSheet,
			SheetName: "Vocabulary",
			RowNumber: 12,
		},
	}
	item := Item{
		Anchors: []SourceAnchor{{
			Source:    SourceGoogleSheet,
			SheetName: "Vocabulary",
			RowNumber: 12,
		}},
	}

	if SheetRowUnchanged(item, draft) {
		t.Fatal("SheetRowUnchanged() = true, want false without stored fingerprint")
	}
}
