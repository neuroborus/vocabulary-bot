package pocketbook

import (
	"fmt"
	"path/filepath"
	"strings"
)

func supportsBookContext(book Book) bool {
	return bookTextFormat(book) != ""
}

func bookTextFormat(book Book) string {
	mimeType := strings.ToLower(strings.TrimSpace(book.MimeType))
	switch {
	case strings.Contains(mimeType, "epub"):
		return "epub"
	case strings.Contains(mimeType, "fictionbook"), strings.Contains(mimeType, "fb2"):
		return "fb2"
	}

	for _, candidate := range []string{book.Link, book.Path, book.Title} {
		lower := strings.ToLower(strings.TrimSpace(candidate))
		switch {
		case strings.HasSuffix(lower, ".epub"):
			return "epub"
		case strings.HasSuffix(lower, ".fb2"):
			return "fb2"
		}
	}

	return ""
}

func bookFileExtension(book Book) string {
	switch bookTextFormat(book) {
	case "epub":
		return ".epub"
	case "fb2":
		return ".fb2"
	default:
		return ".book"
	}
}

func FlattenBookFile(book Book, bookPath string) (string, error) {
	switch bookTextFormat(book) {
	case "epub":
		return FlattenEPUB(bookPath)
	case "fb2":
		return FlattenFB2(bookPath)
	default:
		return "", fmt.Errorf("unsupported book format for %q", book.Title)
	}
}

func localBookBasename(book Book) string {
	return sanitizeBookFileName(book.ID) + bookFileExtension(book)
}

func joinBookPath(dir string, book Book) string {
	return filepath.Join(dir, localBookBasename(book))
}
