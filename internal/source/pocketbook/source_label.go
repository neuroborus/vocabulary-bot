package pocketbook

import "strings"

const documentSourceLabel = "Document"

func pocketbookSourceLabel(book Book) string {
	if isDocumentBook(book) {
		return documentSourceLabel
	}

	title := strings.TrimSpace(firstNonEmpty(book.Title, book.Metadata.Title))
	author := strings.TrimSpace(book.Metadata.Authors)
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

func isDocumentBook(book Book) bool {
	for _, candidate := range []string{book.MimeType, book.Title, book.Link, book.Path} {
		lower := strings.ToLower(strings.TrimSpace(candidate))
		if strings.Contains(lower, "pdf") || strings.HasSuffix(lower, ".pdf") {
			return true
		}
	}

	return false
}
