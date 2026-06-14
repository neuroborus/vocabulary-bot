package pocketbook

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type parsedBookDraft struct {
	draft vocabulary.Draft
	note  Note
}

func needsBookSentenceContext(draft vocabulary.Draft, note Note) bool {
	position := firstNonEmpty(draft.Anchor.Position, markAnchor(note))
	if !IsDictionaryWordAnchor(position) {
		return false
	}

	selectedText := cleanNoteText(quotationText(note))
	if looksLikeContext(selectedText) {
		return false
	}

	for _, contextValue := range draft.Contexts {
		if looksLikeContext(contextValue) && !vocabulary.IsUsageExampleLine(contextValue) {
			return false
		}
	}

	return true
}

func isEPUBBook(book Book) bool {
	mimeType := strings.ToLower(strings.TrimSpace(book.MimeType))
	if strings.Contains(mimeType, "epub") {
		return true
	}

	for _, candidate := range []string{book.Link, book.Path, book.Title} {
		if strings.HasSuffix(strings.ToLower(strings.TrimSpace(candidate)), ".epub") {
			return true
		}
	}

	return false
}

func bookDownloadURL(book Book) string {
	link := strings.TrimSpace(book.Link)
	if link != "" {
		return link
	}

	return strings.TrimSpace(book.Path)
}

func (a *Adapter) enrichBookDrafts(ctx context.Context, book Book, drafts []parsedBookDraft) {
	if !a.bookContextEnabled || len(drafts) == 0 {
		return
	}

	pending := make([]int, 0, len(drafts))
	for index := range drafts {
		if needsBookSentenceContext(drafts[index].draft, drafts[index].note) {
			pending = append(pending, index)
		}
	}
	if len(pending) == 0 {
		return
	}

	if !supportsBookContext(book) {
		a.logger.Info(
			"pocketbook book context enrichment skipped because book format is unsupported",
			slog.String("book_id", book.ID),
			slog.String("title", book.Title),
			slog.String("mime_type", book.MimeType),
		)
		return
	}

	downloadURL := bookDownloadURL(book)
	if downloadURL == "" {
		a.logger.Warn(
			"pocketbook book context enrichment skipped because download link is empty",
			slog.String("book_id", book.ID),
			slog.String("title", book.Title),
		)
		return
	}

	tempDir, err := os.MkdirTemp("", "vocabulary-bot-book-*")
	if err != nil {
		a.logger.Warn(
			"pocketbook book temp dir failed",
			slog.String("book_id", book.ID),
			slog.String("error", err.Error()),
		)
		return
	}
	defer func() {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			a.logger.Warn(
				"pocketbook book temp dir cleanup failed",
				slog.String("book_id", book.ID),
				slog.String("path", tempDir),
				slog.String("error", removeErr.Error()),
			)
		}
	}()

	bookPath := joinBookPath(tempDir, book)
	if err := a.client.DownloadFile(ctx, downloadURL, bookPath); err != nil {
		a.logger.Warn(
			"pocketbook book download failed",
			slog.String("book_id", book.ID),
			slog.String("title", book.Title),
			slog.String("error", err.Error()),
		)
		return
	}

	bookText, err := FlattenBookFile(book, bookPath)
	if err != nil {
		a.logger.Warn(
			"pocketbook book flatten failed",
			slog.String("book_id", book.ID),
			slog.String("title", book.Title),
			slog.String("format", bookTextFormat(book)),
			slog.String("error", err.Error()),
		)
		return
	}

	a.logger.Info(
		"pocketbook book text loaded for context enrichment",
		slog.String("book_id", book.ID),
		slog.String("title", book.Title),
		slog.String("format", bookTextFormat(book)),
		slog.Int("pending_words", len(pending)),
		slog.Int("text_length", len(bookText)),
	)

	for _, index := range pending {
		entry := &drafts[index]
		position := firstNonEmpty(entry.draft.Anchor.Position, markAnchor(entry.note))
		anchor, ok := ParseAnchorPosition(position)
		if !ok || anchor.Offset <= 0 {
			continue
		}

		resolvedOffset, ok := resolveWordOffset(bookText, entry.draft.RawWord, anchor.Offset)
		if !ok {
			a.logger.Info(
				"pocketbook book context word not found in book text",
				slog.String("book_id", book.ID),
				slog.String("word", entry.draft.RawWord),
				slog.Int("offset", anchor.Offset),
			)
			continue
		}
		if resolvedOffset != anchor.Offset {
			a.logger.Info(
				"pocketbook book context offset fallback used",
				slog.String("book_id", book.ID),
				slog.String("word", entry.draft.RawWord),
				slog.Int("anchor_offset", anchor.Offset),
				slog.Int("resolved_offset", resolvedOffset),
			)
		}

		sentence, ok := ExtractSentenceAtOffset(bookText, resolvedOffset)
		if !ok {
			continue
		}

		entry.draft.Contexts = appendUniqueContext(entry.draft.Contexts, sentence)
		a.logger.Info(
			"pocketbook book sentence context added",
			slog.String("book_id", book.ID),
			slog.String("word", entry.draft.RawWord),
			slog.Int("offset", resolvedOffset),
		)
	}
}

func sanitizeBookFileName(bookID string) string {
	bookID = strings.TrimSpace(bookID)
	if bookID == "" {
		return "book"
	}

	var builder strings.Builder
	builder.Grow(len(bookID))
	for _, r := range bookID {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}

	if builder.Len() == 0 {
		return "book"
	}

	return builder.String()
}

func parsedDraftsFromNotes(book Book, notes []Note) []parsedBookDraft {
	drafts := make([]parsedBookDraft, 0, len(notes))
	for _, note := range notes {
		draft, ok := ParseNote(book, note)
		if !ok {
			continue
		}
		drafts = append(drafts, parsedBookDraft{
			draft: draft,
			note:  note,
		})
	}

	return drafts
}

func draftsFromParsed(parsed []parsedBookDraft) []vocabulary.Draft {
	drafts := make([]vocabulary.Draft, 0, len(parsed))
	for _, entry := range parsed {
		drafts = append(drafts, entry.draft)
	}

	return drafts
}

func fetchBookNotes(ctx context.Context, client *Client, book Book, noteIDs []NoteInfo, logger *slog.Logger) []Note {
	notes := make([]Note, 0, len(noteIDs))
	for _, noteID := range noteIDs {
		if noteID.UUID == "" {
			continue
		}

		note, ok, err := client.GetNote(ctx, noteID.UUID, book.FastHash)
		if err != nil {
			logger.Error(
				"pocketbook note fetch failed",
				slog.String("book_id", book.ID),
				slog.String("note_uuid", noteID.UUID),
				slog.String("error", err.Error()),
			)
			continue
		}
		if !ok {
			continue
		}

		notes = append(notes, note)
	}

	return notes
}

func validateBookForNotes(book Book) error {
	if book.FastHash == "" {
		return fmt.Errorf("fast_hash is empty")
	}

	return nil
}
