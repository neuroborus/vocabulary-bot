package telegram

import "testing"

func TestExtractSaveInputFromReply(t *testing.T) {
	t.Parallel()

	input, err := extractSaveInput(Message{
		Text: CommandSave,
		ReplyToMessage: &Message{
			Text: "to decelerate",
		},
	})
	if err != nil {
		t.Fatalf("extractSaveInput() error = %v", err)
	}
	if input != "to decelerate" {
		t.Fatalf("input = %q", input)
	}
}

func TestExtractSaveInputFromSameMessage(t *testing.T) {
	t.Parallel()

	input, err := extractSaveInput(Message{
		Text: "/save@VocabularyBot carve the stone",
	})
	if err != nil {
		t.Fatalf("extractSaveInput() error = %v", err)
	}
	if input != "carve the stone" {
		t.Fatalf("input = %q", input)
	}
}

func TestExtractSaveInputRequiresContent(t *testing.T) {
	t.Parallel()

	_, err := extractSaveInput(Message{Text: CommandSave})
	if err == nil {
		t.Fatal("extractSaveInput() error = nil, want empty input")
	}
}
