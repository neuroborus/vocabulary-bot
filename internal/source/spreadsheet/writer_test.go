package spreadsheet

import (
	"context"
	"testing"
)

type appendRecorder struct {
	spreadsheetID string
	valueRange    string
	row           []string
	updatedRange  string
	err           error
}

func (r *appendRecorder) AppendValues(ctx context.Context, spreadsheetID, valueRange string, row []string) (string, error) {
	r.spreadsheetID = spreadsheetID
	r.valueRange = valueRange
	r.row = append([]string(nil), row...)
	if r.err != nil {
		return "", r.err
	}
	if r.updatedRange == "" {
		r.updatedRange = "Vocabulary!A42:F42"
	}
	return r.updatedRange, nil
}

func TestRowWriterAppendVocabularyRow(t *testing.T) {
	t.Parallel()

	recorder := &appendRecorder{}
	writer := NewRowWriter(recorder, "sheet-1", "Vocabulary")

	rowNumber, err := writer.AppendVocabularyRow(context.Background(), "teasel", "чесало", "Pat Teasely walked in.")
	if err != nil {
		t.Fatalf("AppendVocabularyRow() error = %v", err)
	}
	if rowNumber != 42 {
		t.Fatalf("rowNumber = %d, want 42", rowNumber)
	}
	if recorder.spreadsheetID != "sheet-1" {
		t.Fatalf("spreadsheetID = %q", recorder.spreadsheetID)
	}
	if recorder.valueRange != "Vocabulary!A:F" {
		t.Fatalf("valueRange = %q", recorder.valueRange)
	}
	want := []string{"teasel", "чесало", "Pat Teasely walked in.", "", "", "TRUE"}
	for index, value := range want {
		if recorder.row[index] != value {
			t.Fatalf("row[%d] = %q, want %q", index, recorder.row[index], value)
		}
	}
}

func TestParseRowNumber(t *testing.T) {
	t.Parallel()

	if got := parseRowNumber("Vocabulary!A42:F42"); got != 42 {
		t.Fatalf("parseRowNumber() = %d, want 42", got)
	}
}
