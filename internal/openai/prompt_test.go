package openai

import (
	"strings"
	"testing"
)

func TestVocabularySystemPromptUsesTranslationLanguage(t *testing.T) {
	t.Parallel()

	prompt := vocabularySystemPrompt("Ukrainian")
	if !strings.Contains(prompt, "Ukrainian translation") {
		t.Fatalf("prompt = %q, want Ukrainian translation language", prompt)
	}
	if strings.Contains(prompt, "Russian translation") {
		t.Fatalf("prompt = %q, want no hardcoded Russian", prompt)
	}
}
