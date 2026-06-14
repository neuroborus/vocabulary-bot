package pocketbook

import (
	"encoding/json"
	"testing"
)

func TestBookMetadataAcceptsNumericYear(t *testing.T) {
	t.Parallel()

	var response booksResponse
	if err := json.Unmarshal([]byte(`{
		"total": 1,
		"items": [{
			"id": "book-1",
			"title": "Road Book",
			"fast_hash": "hash-1",
			"metadata": {
				"authors": "A. Writer",
				"year": 2024,
				"isbn": 9781234567890
			}
		}]
	}`), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(response.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(response.Items))
	}

	metadata := response.Items[0].Metadata
	if metadata.Year.String() != "2024" {
		t.Fatalf("Year = %q, want 2024", metadata.Year.String())
	}
	if metadata.ISBN.String() != "9781234567890" {
		t.Fatalf("ISBN = %q, want 9781234567890", metadata.ISBN.String())
	}
}
