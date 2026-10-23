package vocabulary

import (
	"regexp"
	"strings"
	"unicode"
)

var usageExampleSeparator = regexp.MustCompile(`\s+[-–—]\s+`)

// PartitionUsageExamples moves dictionary usage examples from translation lines
// into context lines. PocketBook often stores them inside the translation block.
func PartitionUsageExamples(lines []string) (translations []string, contexts []string) {
	translations = make([]string, 0, len(lines))
	contexts = make([]string, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if IsUsageExampleLine(line) {
			contexts = appendUniqueText(contexts, FormatUsageExampleLine(line), contextKey)
			continue
		}
		translations = append(translations, line)
	}

	return translations, contexts
}

// NormalizeUsageExamples reclassifies stored translation lines that are usage examples.
func NormalizeUsageExamples(item *Item) {
	translations, contexts := PartitionUsageExamples(item.Translations)
	item.Translations = translations
	item.Contexts = mergeStringsByKey(item.Contexts, contexts, contextKey)
}

// IsUsageExampleLine conservatively classifies storage lines as dictionary
// examples. Presentation can split shorter bilingual lines without reclassifying them.
func IsUsageExampleLine(line string) bool {
	example, _, ok := SplitUsageExampleLine(line)
	return ok && len(Tokenize(example)) >= 3
}

func FormatUsageExampleLine(line string) string {
	example, translation, ok := SplitUsageExampleLine(line)
	if !ok {
		return line
	}

	return example + " — " + translation
}

// SplitUsageExampleLine recognizes an English dictionary example with an explicit
// Cyrillic translation after a spaced hyphen, en dash, or em dash.
func SplitUsageExampleLine(line string) (example, translation string, ok bool) {
	separators := usageExampleSeparator.FindAllStringIndex(line, -1)
	// Work backwards to preserve dashes within the English example. A dash within
	// the translation cannot be the boundary because its left side has Cyrillic.
	for index := len(separators) - 1; index >= 0; index-- {
		separator := separators[index]
		left := strings.TrimSpace(line[:separator[0]])
		right := strings.TrimSpace(line[separator[1]:])
		if containsLatinLetters(left) && !containsCyrillicLetters(left) && containsCyrillicLetters(right) {
			return left, right, true
		}
	}
	return "", "", false
}

func containsLatinLetters(value string) bool {
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}

func containsCyrillicLetters(value string) bool {
	for _, r := range value {
		if unicode.In(r, unicode.Cyrillic) {
			return true
		}
	}
	return false
}

func appendUniqueText(values []string, candidate string, key func(string) string) []string {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return values
	}

	return mergeStringsByKey(values, []string{candidate}, key)
}
