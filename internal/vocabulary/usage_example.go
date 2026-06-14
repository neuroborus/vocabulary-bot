package vocabulary

import (
	"regexp"
	"strings"
	"unicode"
)

var usageExamplePattern = regexp.MustCompile(`^(.+?)\s+-\s+(.+)$`)

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

func IsUsageExampleLine(line string) bool {
	match := usageExamplePattern.FindStringSubmatch(strings.TrimSpace(line))
	if len(match) < 3 {
		return false
	}

	left := strings.TrimSpace(match[1])
	right := strings.TrimSpace(match[2])
	if left == "" || right == "" {
		return false
	}

	return containsLatinLetters(left) && len(Tokenize(left)) >= 3 && containsCyrillicLetters(right)
}

func FormatUsageExampleLine(line string) string {
	match := usageExamplePattern.FindStringSubmatch(strings.TrimSpace(line))
	if len(match) < 3 {
		return line
	}

	return strings.TrimSpace(match[1]) + " — " + strings.TrimSpace(match[2])
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
