package spreadsheet

import (
	"context"
	"errors"
	"testing"
)

func TestParseServiceAccountCredentialsAcceptsJSON(t *testing.T) {
	t.Parallel()

	raw := `{"type":"service_account","project_id":"example"}`
	got, err := ParseServiceAccountCredentials(raw)
	if err != nil {
		t.Fatalf("ParseServiceAccountCredentials() error = %v", err)
	}
	if string(got) != raw {
		t.Fatalf("credentials = %q, want %q", string(got), raw)
	}
}

func TestParseServiceAccountCredentialsAcceptsBase64JSON(t *testing.T) {
	t.Parallel()

	raw := `{"type":"service_account","project_id":"example"}`
	encoded := "eyJ0eXBlIjoic2VydmljZV9hY2NvdW50IiwicHJvamVjdF9pZCI6ImV4YW1wbGUifQ=="

	got, err := ParseServiceAccountCredentials(encoded)
	if err != nil {
		t.Fatalf("ParseServiceAccountCredentials() error = %v", err)
	}
	if string(got) != raw {
		t.Fatalf("credentials = %q, want %q", string(got), raw)
	}
}

func TestParseServiceAccountCredentialsRejectsEmpty(t *testing.T) {
	t.Parallel()

	if _, err := ParseServiceAccountCredentials(""); err == nil {
		t.Fatal("expected empty credentials error")
	}
}

func TestAnyMatrixToStrings(t *testing.T) {
	t.Parallel()

	got := anyMatrixToStrings([][]any{
		{"word", 42, true},
		{"lean", nil},
	})
	if len(got) != 2 || got[0][1] != "42" || got[0][2] != "true" || got[1][1] != "" {
		t.Fatalf("anyMatrixToStrings() = %#v", got)
	}
}

type fakeValuesClient struct {
	rows      [][]string
	err       error
	lastRange string
}

func (f *fakeValuesClient) FetchValues(ctx context.Context, spreadsheetID, valueRange string) ([][]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.lastRange = valueRange
	if f.err != nil {
		return nil, f.err
	}

	return f.rows, nil
}

func TestAdapterDefaultsRangeToSheetAToF(t *testing.T) {
	t.Parallel()

	client := &fakeValuesClient{
		rows: [][]string{
			{"word", "translations", "contexts", "note", "tags", "enabled"},
			{"lean", "наклонять", "", "", "", "true"},
		},
	}
	adapter := NewAdapter(AdapterOptions{
		SpreadsheetID: "sheet-1",
		SheetName:     "Vocabulary",
		ValuesClient:  client,
	})

	if _, err := adapter.Sync(context.Background()); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if client.lastRange != "Vocabulary!A:F" {
		t.Fatalf("lastRange = %q, want %q", client.lastRange, "Vocabulary!A:F")
	}
}

func TestAdapterSyncParsesRowsFromSheet(t *testing.T) {
	t.Parallel()

	adapter := NewAdapter(AdapterOptions{
		SpreadsheetID: "sheet-1",
		SheetName:     "Vocabulary",
		Range:         "Vocabulary!A:F",
		ValuesClient: &fakeValuesClient{
			rows: [][]string{
				{"word", "translations", "contexts", "note", "tags", "enabled"},
				{"assessing", "оценивание", "Assessing the risks took longer than we expected.", "", "", "TRUE"},
			},
		},
	})

	drafts, err := adapter.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("drafts = %d, want 1", len(drafts))
	}
	if drafts[0].RawWord != "assessing" {
		t.Fatalf("RawWord = %q", drafts[0].RawWord)
	}
	if drafts[0].Anchor.SheetName != "Vocabulary" {
		t.Fatalf("SheetName = %q", drafts[0].Anchor.SheetName)
	}
}

func TestAdapterSyncReportsRowParseErrors(t *testing.T) {
	t.Parallel()

	adapter := NewAdapter(AdapterOptions{
		SpreadsheetID: "sheet-1",
		SheetName:     "Vocabulary",
		Range:         "Vocabulary!A:F",
		ValuesClient: &fakeValuesClient{
			rows: [][]string{
				{"word", "translations", "contexts", "note", "tags", "enabled"},
				{"assessing", "оценивание", "", "", "", "TRUE"},
				{"bad", "", "", "", "", "maybe"},
			},
		},
	})

	if _, err := adapter.Sync(context.Background()); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	details := adapter.SyncDetails()
	if details.RowParseErrors != 1 {
		t.Fatalf("RowParseErrors = %d, want 1", details.RowParseErrors)
	}
}

func TestAdapterSyncRequiresSpreadsheetID(t *testing.T) {
	t.Parallel()

	adapter := NewAdapter(AdapterOptions{
		ValuesClient: &fakeValuesClient{},
	})

	if _, err := adapter.Sync(context.Background()); err == nil {
		t.Fatal("expected spreadsheet id error")
	}
}

func TestAdapterSyncRequiresCredentialsWhenClientMissing(t *testing.T) {
	t.Parallel()

	adapter := NewAdapter(AdapterOptions{
		SpreadsheetID: "sheet-1",
	})

	if _, err := adapter.Sync(context.Background()); err == nil {
		t.Fatal("expected credentials error")
	}
}

func TestAdapterSyncPropagatesFetchError(t *testing.T) {
	t.Parallel()

	adapter := NewAdapter(AdapterOptions{
		SpreadsheetID: "sheet-1",
		ValuesClient: &fakeValuesClient{
			err: errors.New("api unavailable"),
		},
	})

	if _, err := adapter.Sync(context.Background()); err == nil {
		t.Fatal("expected fetch error")
	}
}
