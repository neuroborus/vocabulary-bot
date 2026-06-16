package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/logging"
)

const defaultBaseURL = "https://api.openai.com/v1"
const defaultModel = "gpt-4o-mini"
const defaultTranslationLanguage = "Russian"

type VocabularyFields struct {
	Word        string `json:"word"`
	Context     string `json:"context"`
	Translation string `json:"translation"`
}

type Client struct {
	apiKey              string
	model               string
	baseURL             string
	translationLanguage string
	httpClient          *http.Client
}

type ClientOptions struct {
	APIKey              string
	Model               string
	BaseURL             string
	TranslationLanguage string
	HTTPClient          *http.Client
}

func NewClient(options ClientOptions) *Client {
	baseURL := strings.TrimRight(strings.TrimSpace(options.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	model := strings.TrimSpace(options.Model)
	if model == "" {
		model = defaultModel
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}

	translationLanguage := strings.TrimSpace(options.TranslationLanguage)
	if translationLanguage == "" {
		translationLanguage = defaultTranslationLanguage
	}

	return &Client{
		apiKey:              strings.TrimSpace(options.APIKey),
		model:               model,
		baseURL:             baseURL,
		translationLanguage: translationLanguage,
		httpClient:          httpClient,
	}
}

func (c *Client) StructureVocabulary(ctx context.Context, input string) (VocabularyFields, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return VocabularyFields{}, fmt.Errorf("vocabulary input is empty")
	}
	if c.apiKey == "" {
		return VocabularyFields{}, fmt.Errorf("OPENAI_API_KEY is not configured")
	}

	body, err := json.Marshal(chatCompletionRequest{
		Model: c.model,
		ResponseFormat: responseFormat{
			Type: "json_object",
		},
		Messages: []chatMessage{
			{Role: "system", Content: vocabularySystemPrompt(c.translationLanguage)},
			{Role: "user", Content: input},
		},
	})
	if err != nil {
		return VocabularyFields{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return VocabularyFields{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(req)
	if err != nil {
		return VocabularyFields{}, err
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return VocabularyFields{}, err
	}

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return VocabularyFields{}, fmt.Errorf(
			"openai chat completion failed: %s: %s",
			response.Status,
			logging.SanitizeString(parseErrorDescription(payload)),
		)
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(payload, &completion); err != nil {
		return VocabularyFields{}, fmt.Errorf("decode openai response: %w", err)
	}
	if len(completion.Choices) == 0 || strings.TrimSpace(completion.Choices[0].Message.Content) == "" {
		return VocabularyFields{}, fmt.Errorf("openai returned empty completion")
	}

	var fields VocabularyFields
	if err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &fields); err != nil {
		return VocabularyFields{}, fmt.Errorf("decode openai vocabulary json: %w", err)
	}

	fields.Word = strings.TrimSpace(fields.Word)
	fields.Context = strings.TrimSpace(fields.Context)
	fields.Translation = strings.TrimSpace(fields.Translation)
	if fields.Word == "" || fields.Context == "" || fields.Translation == "" {
		return VocabularyFields{}, fmt.Errorf("openai returned incomplete vocabulary fields")
	}

	return fields, nil
}

type chatCompletionRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	ResponseFormat responseFormat `json:"response_format"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *apiError `json:"error"`
}

type apiError struct {
	Message string `json:"message"`
}

func parseErrorDescription(payload []byte) string {
	var response struct {
		Error apiError `json:"error"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return ""
	}

	return strings.TrimSpace(response.Error.Message)
}
