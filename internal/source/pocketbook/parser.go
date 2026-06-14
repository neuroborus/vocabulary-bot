package pocketbook

import (
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)

func ParseNote(book Book, note Note) (vocabulary.Draft, bool) {
	selectedText := cleanNoteText(quotationText(note))
	noteText := cleanNoteBlock(commentText(note))
	if noteText == "" {
		return vocabulary.Draft{}, false
	}

	parsed := parseNoteText(noteText)
	word := firstNonEmpty(parsed.word, selectedText)
	translations := parsed.translations
	contexts := parsed.contexts

	if parsed.word == "" {
		if !looksLikeDictionaryWord(selectedText) || !looksLikeTranslationBlock(noteText) {
			return vocabulary.Draft{}, false
		}
		translations = splitTranslations(noteText)
	}

	if word == "" || !looksLikeDictionaryWord(word) {
		return vocabulary.Draft{}, false
	}
	if len(translations) == 0 && len(contexts) == 0 {
		return vocabulary.Draft{}, false
	}

	if selectedText != "" && vocabulary.CompactKey(selectedText) != vocabulary.CompactKey(word) && looksLikeContext(selectedText) && !containsNormalized(contexts, selectedText) {
		contexts = append(contexts, selectedText)
	}

	draft := vocabulary.Draft{
		Source:       vocabulary.SourcePocketBook,
		RawWord:      word,
		Translations: translations,
		Contexts:     contexts,
		Anchor: vocabulary.SourceAnchor{
			Source:     vocabulary.SourcePocketBook,
			ExternalID: note.UUID,
			BookID:     book.ID,
			BookTitle:  firstNonEmpty(book.Title, book.Metadata.Title),
			Author:     book.Metadata.Authors,
			Page:       pageFromAnchor(markAnchor(note)),
			Position:   markAnchor(note),
		},
	}

	return draft, true
}

type parsedNoteText struct {
	word         string
	translations []string
	contexts     []string
}

func parseNoteText(value string) parsedNoteText {
	var parsed parsedNoteText

	lines := splitLines(value)
	for _, line := range lines {
		key, rest, ok := splitLabel(line)
		if !ok {
			continue
		}

		switch key {
		case "word", "term", "selected", "selection":
			if parsed.word == "" {
				parsed.word = cleanNoteText(rest)
			}
		case "translation", "translations", "meaning", "meanings":
			parsed.translations = append(parsed.translations, splitTranslations(rest)...)
		case "context", "contexts", "example", "examples", "sentence", "sentences":
			parsed.contexts = append(parsed.contexts, splitContexts(rest)...)
		}
	}

	if parsed.word == "" && len(lines) == 1 {
		left, right, ok := splitInlinePair(lines[0])
		if ok && looksLikeDictionaryWord(left) {
			parsed.word = cleanNoteText(left)
			parsed.translations = append(parsed.translations, splitTranslations(right)...)
		}
	}

	return parsed
}

func splitLabel(line string) (string, string, bool) {
	candidates := []string{":", "："}
	for _, separator := range candidates {
		left, right, ok := strings.Cut(line, separator)
		if !ok {
			continue
		}

		key := normalizeLabel(left)
		if key == "" {
			return "", "", false
		}

		return key, right, true
	}

	return "", "", false
}

func splitInlinePair(line string) (string, string, bool) {
	separators := []string{" — ", " – ", " - ", " = "}
	for _, separator := range separators {
		left, right, ok := strings.Cut(line, separator)
		if ok && strings.TrimSpace(left) != "" && strings.TrimSpace(right) != "" {
			return left, right, true
		}
	}

	return "", "", false
}

func normalizeLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Trim(value, " \t\r\n#*-")

	switch value {
	case "word", "term", "selected", "selection",
		"translation", "translations", "meaning", "meanings",
		"context", "contexts", "example", "examples", "sentence", "sentences":
		return value
	default:
		return ""
	}
}

func splitTranslations(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.NewReplacer(";", "\n", ",", "\n", "•", "\n").Replace(value)

	return splitAndClean(value, "\n")
}

func splitContexts(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "|||", "\n")

	return splitAndClean(value, "\n")
}

func splitLines(value string) []string {
	return splitAndClean(value, "\n")
}

func splitAndClean(value string, separator string) []string {
	parts := strings.Split(value, separator)
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		cleaned := cleanNoteText(part)
		if cleaned == "" {
			continue
		}

		key := vocabulary.NormalizeText(cleaned)
		if _, ok := seen[key]; ok {
			continue
		}

		result = append(result, cleaned)
		seen[key] = struct{}{}
	}

	return result
}

func cleanNoteText(value string) string {
	value = html.UnescapeString(value)
	value = htmlTagPattern.ReplaceAllString(value, " ")
	value = strings.ReplaceAll(value, "\u00a0", " ")
	value = strings.TrimSpace(value)
	value = strings.Join(strings.Fields(value), " ")

	return value
}

func cleanNoteBlock(value string) string {
	value = html.UnescapeString(value)
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "<br>", "\n")
	value = strings.ReplaceAll(value, "<br/>", "\n")
	value = strings.ReplaceAll(value, "<br />", "\n")
	value = htmlTagPattern.ReplaceAllString(value, " ")

	lines := splitAndClean(value, "\n")
	return strings.Join(lines, "\n")
}

func quotationText(note Note) string {
	if note.Quotation == nil {
		return ""
	}

	return note.Quotation.Text
}

func commentText(note Note) string {
	if note.Note == nil {
		return ""
	}

	return note.Note.Text
}

func markAnchor(note Note) string {
	if note.Mark == nil {
		return ""
	}

	return note.Mark.Anchor
}

func looksLikeDictionaryWord(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || utf8.RuneCountInString(value) > 80 {
		return false
	}

	tokens := vocabulary.Tokenize(value)
	return len(tokens) > 0 && len(tokens) <= 8
}

func looksLikeTranslationBlock(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || utf8.RuneCountInString(value) > 500 {
		return false
	}

	return strings.Count(value, ".") <= 3
}

func looksLikeContext(value string) bool {
	tokens := vocabulary.Tokenize(value)
	return len(tokens) > 8 || strings.ContainsAny(value, ".!?")
}

func containsNormalized(values []string, candidate string) bool {
	candidateKey := vocabulary.NormalizeText(candidate)
	if candidateKey == "" {
		return true
	}

	for _, value := range values {
		if vocabulary.NormalizeText(value) == candidateKey {
			return true
		}
	}

	return false
}

func pageFromAnchor(anchor string) string {
	if anchor == "" {
		return ""
	}

	if parsed, err := url.Parse(anchor); err == nil {
		if page := parsed.Query().Get("page"); page != "" {
			return page
		}
	}

	index := strings.Index(anchor, "page=")
	if index < 0 {
		return ""
	}

	start := index + len("page=")
	end := start
	for end < len(anchor) && anchor[end] >= '0' && anchor[end] <= '9' {
		end++
	}
	if end == start {
		return ""
	}

	return anchor[start:end]
}
