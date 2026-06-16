package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStructureVocabularyParsesCompletionJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}

		var request chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.ResponseFormat.Type != "json_object" {
			t.Fatalf("response format = %q", request.ResponseFormat.Type)
		}
		if request.Messages[len(request.Messages)-1].Content != "teasel" {
			t.Fatalf("user content = %q", request.Messages[len(request.Messages)-1].Content)
		}

		payload, err := json.Marshal(chatCompletionResponse{
			Choices: []struct {
				Message chatMessage `json:"message"`
			}{
				{
					Message: chatMessage{
						Content: `{"word":"teasel","context":"Pat Teasely walked in.","translation":"чесало"}`,
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("marshal response: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		APIKey:  "test-key",
		BaseURL: server.URL + "/v1",
	})
	fields, err := client.StructureVocabulary(context.Background(), "teasel")
	if err != nil {
		t.Fatalf("StructureVocabulary() error = %v", err)
	}
	if fields.Word != "teasel" || fields.Context == "" || fields.Translation != "чесало" {
		t.Fatalf("fields = %#v", fields)
	}
}

func TestStructureVocabularyRejectsIncompleteJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"word\":\"only\"}"}}]}`))
	}))
	defer server.Close()

	client := NewClient(ClientOptions{APIKey: "test-key", BaseURL: server.URL + "/v1"})
	_, err := client.StructureVocabulary(context.Background(), "only")
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("error = %v, want incomplete fields", err)
	}
}
