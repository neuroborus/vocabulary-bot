package vocabulary

import (
	"testing"
	"time"
)

func TestPrimarySourceLabelPrefersLatestAnchor(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	label := PrimarySourceLabel(Item{
		Anchors: []SourceAnchor{
			{
				Source:      SourcePocketBook,
				SourceLabel: "Older Book — Author A",
				LastSeenAt:  now.Add(-time.Hour),
			},
			{
				Source:      SourcePocketBook,
				SourceLabel: "Necromancer — Fred Saberhagen",
				LastSeenAt:  now,
			},
		},
	})
	if label != "Necromancer — Fred Saberhagen" {
		t.Fatalf("label = %q", label)
	}
}

func TestPrimarySourceLabelUsesLegacyBookTitleAndAuthor(t *testing.T) {
	t.Parallel()

	label := PrimarySourceLabel(Item{
		Anchors: []SourceAnchor{
			{
				Source:    SourcePocketBook,
				BookTitle: "Road Book",
				Author:    "A. Writer",
			},
		},
	})
	if label != "Road Book — A. Writer" {
		t.Fatalf("label = %q", label)
	}
}
