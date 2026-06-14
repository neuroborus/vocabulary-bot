package telegram

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

var (
	transcriptionPrefixPattern = regexp.MustCompile(`^\[([^\]]+)\]\s*`)
	posHeaderPattern           = regexp.MustCompile(`(?i)^(noun|verb|adjective|adverb|preposition|conjunction|pronoun|phrase|participle|gerund)\b(?:\s+\d+\)\s*|\s+\d+\)|\s*:\s*|\s+|$)`)
	numberedMeaningPattern     = regexp.MustCompile(`^\d+\)\s*`)
	qualifierPattern           = regexp.MustCompile(`(?i)^(о|по|из|на|для|с|в|при)\s+`)
	examplePairPattern         = regexp.MustCompile(`^(.+?)\s+-\s+(.+)$`)
)

type lexiconDisplay struct {
	Transcription string
	PartsOfSpeech []partOfSpeechDisplay
	SimpleLines   []string
	Contexts      []string
}

type partOfSpeechDisplay struct {
	Label    string
	Meanings []string
}

func formatReviewReminder(item vocabulary.Item, spoilerTranslations bool) string {
	lexicon := parseLexiconDisplay(item.Translations, item.Contexts)

	var builder strings.Builder
	builder.WriteString("<b>")
	builder.WriteString(escapeHTML(item.DisplayWord))
	builder.WriteString("</b>")

	if len(lexicon.Contexts) > 0 {
		builder.WriteString("\n\n<b>Context</b>")
		for _, contextValue := range lexicon.Contexts {
			builder.WriteString("\n• ")
			builder.WriteString(escapeHTML(contextValue))
		}
	}

	translationSection := formatTranslationSection(item, lexicon)
	if translationSection != "" {
		if spoilerTranslations {
			builder.WriteString("\n\n<tg-spoiler>")
			builder.WriteString(translationSection)
			builder.WriteString("</tg-spoiler>")
		} else {
			builder.WriteString("\n\n")
			builder.WriteString(translationSection)
		}
	}

	if sourceLabel := vocabulary.PrimarySourceLabel(item); sourceLabel != "" {
		builder.WriteString("\n\n<i>")
		builder.WriteString(escapeHTML(sourceLabel))
		builder.WriteString("</i>")
	}

	return builder.String()
}

func formatTranslationSection(item vocabulary.Item, lexicon lexiconDisplay) string {
	var builder strings.Builder

	if lexicon.Transcription != "" {
		builder.WriteString("<i>")
		builder.WriteString(escapeHTML("[" + lexicon.Transcription + "]"))
		builder.WriteString("</i>")
	}

	if variants := visibleVariants(item); len(variants) > 0 {
		builder.WriteString("\n\n<b>Variants</b>")
		for _, variant := range variants {
			builder.WriteString("\n• ")
			builder.WriteString(escapeHTML(variant))
		}
	}

	for _, block := range lexicon.PartsOfSpeech {
		builder.WriteString("\n\n<b>")
		builder.WriteString(escapeHTML(block.Label))
		builder.WriteString("</b>")
		for _, meaning := range block.Meanings {
			builder.WriteString("\n• ")
			builder.WriteString(escapeHTML(meaning))
		}
	}

	if len(lexicon.SimpleLines) > 0 {
		builder.WriteString("\n\n<b>Translation</b>")
		for _, line := range lexicon.SimpleLines {
			builder.WriteString("\n• ")
			builder.WriteString(escapeHTML(line))
		}
	}

	return strings.TrimSpace(builder.String())
}

func visibleVariants(item vocabulary.Item) []string {
	displayKey := vocabulary.CompactKey(item.DisplayWord)
	seen := make(map[string]struct{})
	variants := make([]string, 0, len(item.Forms))

	for _, form := range item.Forms {
		value := strings.TrimSpace(form.Value)
		if value == "" || vocabulary.CompactKey(value) == displayKey {
			continue
		}

		key := vocabulary.NormalizeText(value)
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		variants = append(variants, value)
	}

	return variants
}

func parseLexiconDisplay(translations, contexts []string) lexiconDisplay {
	lines := make([]string, 0, len(translations))
	for _, value := range translations {
		lines = append(lines, splitDisplayLines(value)...)
	}

	display := lexiconDisplay{
		Contexts: dedupeDisplayLines(cleanContextLines(contexts)),
	}

	var currentPOS *partOfSpeechDisplay
	appendMeaning := func(meaning string) {
		meaning = strings.TrimSpace(meaning)
		if meaning == "" {
			return
		}
		if currentPOS == nil {
			display.SimpleLines = append(display.SimpleLines, meaning)
			return
		}
		currentPOS.Meanings = append(currentPOS.Meanings, meaning)
	}

	for _, line := range lines {
		line = cleanDisplayLine(line)
		if line == "" {
			continue
		}

		if transcription := extractTranscription(&line); transcription != "" && display.Transcription == "" {
			display.Transcription = transcription
		}
		if line == "" {
			continue
		}

		if isExampleLine(line) {
			display.Contexts = appendUniqueLine(display.Contexts, formatExampleLine(line))
			continue
		}

		if pos, remainder, ok := extractPartOfSpeech(line); ok {
			currentPOS = &partOfSpeechDisplay{Label: pos}
			display.PartsOfSpeech = append(display.PartsOfSpeech, *currentPOS)
			currentPOS = &display.PartsOfSpeech[len(display.PartsOfSpeech)-1]
			line = remainder
		}

		if qualifierPattern.MatchString(line) && currentPOS != nil && len(currentPOS.Meanings) > 0 {
			lastIndex := len(currentPOS.Meanings) - 1
			currentPOS.Meanings[lastIndex] += " (" + line + ")"
			continue
		}

		if numberedMeaningPattern.MatchString(line) {
			appendMeaning(strings.TrimSpace(numberedMeaningPattern.ReplaceAllString(line, "")))
			continue
		}

		appendMeaning(line)
	}

	display.Contexts = dedupeDisplayLines(cleanContextLines(display.Contexts))
	return display
}

func titleCase(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return value
	}

	return strings.ToUpper(value[:1]) + value[1:]
}

func splitDisplayLines(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.Split(value, "\n")
}

func cleanDisplayLine(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "•")
	value = strings.TrimPrefix(value, "·")
	value = strings.TrimSpace(value)
	return value
}

func extractTranscription(line *string) string {
	match := transcriptionPrefixPattern.FindStringSubmatch(*line)
	if len(match) < 2 {
		return ""
	}

	*line = strings.TrimSpace(transcriptionPrefixPattern.ReplaceAllString(*line, ""))
	return match[1]
}

func extractPartOfSpeech(line string) (string, string, bool) {
	match := posHeaderPattern.FindStringSubmatch(line)
	if len(match) < 2 {
		return "", line, false
	}

	label := titleCase(match[1])
	remainder := strings.TrimSpace(line[len(match[0]):])
	if numberedMeaningPattern.MatchString(remainder) {
		remainder = strings.TrimSpace(numberedMeaningPattern.ReplaceAllString(remainder, ""))
	}

	return label, remainder, true
}

func isExampleLine(line string) bool {
	match := examplePairPattern.FindStringSubmatch(line)
	if len(match) < 3 {
		return false
	}

	left := strings.TrimSpace(match[1])
	right := strings.TrimSpace(match[2])
	if left == "" || right == "" {
		return false
	}

	return containsLatinLetters(left) && !posHeaderPattern.MatchString(left) && !numberedMeaningPattern.MatchString(left)
}

func formatExampleLine(line string) string {
	match := examplePairPattern.FindStringSubmatch(line)
	if len(match) < 3 {
		return line
	}

	return strings.TrimSpace(match[1]) + " — " + strings.TrimSpace(match[2])
}

func containsLatinLetters(value string) bool {
	for _, r := range value {
		if unicode.In(r, unicode.Latin) && unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func appendUniqueLine(values []string, line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return values
	}

	key := vocabulary.NormalizeText(line)
	for _, existing := range values {
		if vocabulary.NormalizeText(existing) == key {
			return values
		}
	}

	return append(values, line)
}

func dedupeDisplayLines(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = appendUniqueLine(result, cleanDisplayLine(value))
	}
	return result
}

func cleanContextLines(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		cleaned := vocabulary.CleanContextLine(value)
		if cleaned == "" {
			continue
		}
		result = append(result, cleaned)
	}
	return result
}
