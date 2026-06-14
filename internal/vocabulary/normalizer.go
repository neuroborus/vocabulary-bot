package vocabulary

import (
	"strings"
	"unicode"
)

func NormalizeText(value string) string {
	value = strings.ToLower(value)
	value = strings.Map(normalizeRune, value)
	value = strings.TrimFunc(value, isWrappingRune)
	value = strings.Join(strings.Fields(value), " ")

	return strings.TrimSpace(value)
}

func CompactKey(value string) string {
	normalized := NormalizeText(value)

	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '-' {
			return -1
		}

		return r
	}, normalized)
}

func BuildLookupKeys(value string) []string {
	tokens := Tokenize(value)
	if len(tokens) == 0 {
		return nil
	}

	keys := make([]string, 0, 4)
	keys = appendLookupKey(keys, CompactKey(strings.Join(tokens, " ")))

	if len(tokens) == 2 {
		keys = appendLongerTokenLookupKey(keys, tokens[0], tokens[1])
		return keys
	}

	if len(tokens) > 2 {
		keys = appendLookupKey(keys, CompactKey(strings.Join(tokens[1:], " ")))
		keys = appendLookupKey(keys, CompactKey(strings.Join(tokens[:len(tokens)-1], " ")))
		keys = appendLookupKey(keys, CompactKey(strings.Join(tokens[1:len(tokens)-1], " ")))
	}

	return keys
}

func Tokenize(value string) []string {
	normalized := NormalizeText(value)
	if normalized == "" {
		return nil
	}

	return strings.Fields(normalized)
}

func appendLookupKey(keys []string, candidate string) []string {
	if candidate == "" {
		return keys
	}

	for _, existing := range keys {
		if existing == candidate {
			return keys
		}
	}

	return append(keys, candidate)
}

func appendLongerTokenLookupKey(keys []string, left string, right string) []string {
	leftKey := CompactKey(left)
	rightKey := CompactKey(right)

	if len([]rune(leftKey)) > len([]rune(rightKey)) {
		return appendLookupKey(keys, leftKey)
	}
	if len([]rune(rightKey)) > len([]rune(leftKey)) {
		return appendLookupKey(keys, rightKey)
	}

	return keys
}

func normalizeRune(r rune) rune {
	switch r {
	case '\u2018', '\u2019':
		return '\''
	case '\u2010', '\u2011', '\u2012', '\u2013', '\u2014':
		return '-'
	default:
		return r
	}
}

func isWrappingRune(r rune) bool {
	switch r {
	case '"', '\'', '(', ')', '[', ']', '{', '}', '\u201c', '\u201d', '\u2018', '\u2019':
		return true
	default:
		return unicode.IsSpace(r)
	}
}
