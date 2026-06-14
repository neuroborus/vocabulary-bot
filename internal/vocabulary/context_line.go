package vocabulary

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// CleanContextLine trims book-sentence artifacts such as dangling dialogue quotes.
func CleanContextLine(line string) string {
	line = strings.TrimSpace(line)
	line = trimEdgeQuotes(line, true)
	line = trimEdgeQuotes(line, false)
	return strings.TrimSpace(line)
}

func trimEdgeQuotes(value string, leading bool) string {
	for value != "" {
		if leading {
			r, size := utf8.DecodeRuneInString(value)
			if !isContextQuoteRune(r) {
				break
			}
			value = strings.TrimSpace(value[size:])
			continue
		}

		r, size := utf8.DecodeLastRuneInString(value)
		if !isContextQuoteRune(r) {
			break
		}
		value = strings.TrimSpace(value[:len(value)-size])
	}

	return value
}

func isContextQuoteRune(r rune) bool {
	switch r {
	case '"', '\'', '«', '»', '“', '”', '‘', '’':
		return true
	default:
		return unicode.In(r, unicode.Quotation_Mark)
	}
}
