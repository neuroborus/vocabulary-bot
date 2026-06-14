package spreadsheet

import (
	"fmt"
	"strings"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type RowError struct {
	RowNumber int
	Problem   string
}

func ParseRows(sheetName string, rows [][]string) ([]vocabulary.Draft, []RowError) {
	if len(rows) == 0 {
		return nil, nil
	}

	header := headerIndex(rows[0])
	wordIndex, ok := lookupHeader(header, "word")
	if !ok {
		return nil, []RowError{{RowNumber: 1, Problem: "missing required word column"}}
	}

	translationIndex, _ := lookupHeader(header, "translations", "translation", "meaning", "meanings")
	contextIndex, _ := lookupHeader(header, "contexts", "context", "examples", "example")
	noteIndex, _ := lookupHeader(header, "note", "notes")
	tagIndex, _ := lookupHeader(header, "tags", "tag")
	enabledIndex, _ := lookupHeader(header, "enabled")

	drafts := make([]vocabulary.Draft, 0, len(rows)-1)
	rowErrors := make([]RowError, 0)

	for rowOffset, row := range rows[1:] {
		rowNumber := rowOffset + 2
		word := strings.TrimSpace(cell(row, wordIndex))
		if word == "" {
			continue
		}

		enabled, err := parseEnabled(cell(row, enabledIndex))
		if err != nil {
			rowErrors = append(rowErrors, RowError{
				RowNumber: rowNumber,
				Problem:   err.Error(),
			})
			continue
		}
		if !enabled {
			continue
		}

		drafts = append(drafts, vocabulary.Draft{
			Source:       vocabulary.SourceGoogleSheet,
			RawWord:      word,
			Translations: splitAndClean(cell(row, translationIndex), ","),
			Contexts:     splitAndClean(cell(row, contextIndex), "."),
			Notes:        splitAndClean(cell(row, noteIndex), "\n"),
			Tags:         splitAndClean(cell(row, tagIndex), ","),
			Anchor: vocabulary.SourceAnchor{
				Source:    vocabulary.SourceGoogleSheet,
				RowNumber: rowNumber,
				SheetName: sheetName,
			},
		})
	}

	return drafts, rowErrors
}

func headerIndex(row []string) map[string]int {
	result := make(map[string]int, len(row))

	for index, value := range row {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" {
			continue
		}

		result[key] = index
	}

	return result
}

func lookupHeader(header map[string]int, names ...string) (int, bool) {
	for _, name := range names {
		if index, ok := header[name]; ok {
			return index, true
		}
	}

	return -1, false
}

func cell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}

	return row[index]
}

func parseEnabled(value string) (bool, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return true, nil
	}

	switch value {
	case "true", "yes", "1", "on":
		return true, nil
	case "false", "no", "0", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid enabled value %q", value)
	}
}

func splitAndClean(value string, separator string) []string {
	parts := strings.Split(value, separator)
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		cleaned := strings.TrimSpace(part)
		if cleaned == "" {
			continue
		}

		key := strings.ToLower(strings.Join(strings.Fields(cleaned), " "))
		if _, ok := seen[key]; ok {
			continue
		}

		result = append(result, cleaned)
		seen[key] = struct{}{}
	}

	return result
}
