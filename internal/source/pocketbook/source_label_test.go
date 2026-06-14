package pocketbook

import "testing"

func TestPocketbookSourceLabelForBook(t *testing.T) {
	t.Parallel()

	label := pocketbookSourceLabel(Book{
		Title:    "Necromancer",
		MimeType: "application/x-fictionbook+xml",
		Metadata: BookMetadata{
			Title:   "Necromancer",
			Authors: "Fred Saberhagen",
		},
	})
	if label != "Necromancer — Fred Saberhagen" {
		t.Fatalf("label = %q", label)
	}
}

func TestPocketbookSourceLabelForDocument(t *testing.T) {
	t.Parallel()

	label := pocketbookSourceLabel(Book{
		Title:    "Manual.pdf",
		MimeType: "application/pdf",
		Metadata: BookMetadata{
			Title:   "Manual",
			Authors: "Someone",
		},
	})
	if label != documentSourceLabel {
		t.Fatalf("label = %q, want %q", label, documentSourceLabel)
	}
}

func TestPocketbookSourceLabelFallsBackToTitleAndAuthor(t *testing.T) {
	t.Parallel()

	label := pocketbookSourceLabel(Book{
		Title:    "Road Book",
		Metadata: BookMetadata{Authors: "A. Writer"},
	})
	if label != "Road Book — A. Writer" {
		t.Fatalf("label = %q", label)
	}
}
