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

func TestExtractSaveInputBeforeCommand(t *testing.T) {
	t.Parallel()

	input, err := extractSaveInput(Message{
		Text: "carve the stone /save@VocabularyBot",
	})
	if err != nil {
		t.Fatalf("extractSaveInput() error = %v", err)
	}
	if input != "carve the stone" {
		t.Fatalf("input = %q", input)
	}
}

func TestExtractSaveInputAroundCommand(t *testing.T) {
	t.Parallel()

	input, err := extractSaveInput(Message{
		Text: "carve /save the stone",
	})
	if err != nil {
		t.Fatalf("extractSaveInput() error = %v", err)
	}
	if input != "carve the stone" {
		t.Fatalf("input = %q", input)
	}
}

func TestExtractSaveInputFromQuote(t *testing.T) {
	t.Parallel()

	input, err := extractSaveInput(Message{
		Text: CommandSave,
		ReplyToMessage: &Message{
			MessageID: 99,
		},
		Quote: &TextQuote{
			Text: "fit the bill - подходить под описание",
		},
	})
	if err != nil {
		t.Fatalf("extractSaveInput() error = %v", err)
	}
	if input != "fit the bill - подходить под описание" {
		t.Fatalf("input = %q", input)
	}
}

func TestExtractSaveInputFromReplyCaption(t *testing.T) {
	t.Parallel()

	input, err := extractSaveInput(Message{
		Text: CommandSave,
		ReplyToMessage: &Message{
			Caption: "photo caption text",
		},
	})
	if err != nil {
		t.Fatalf("extractSaveInput() error = %v", err)
	}
	if input != "photo caption text" {
		t.Fatalf("input = %q", input)
	}
}

func TestSaveReplyHadNoReadableText(t *testing.T) {
	t.Parallel()

	if !saveReplyHadNoReadableText(Message{
		Text:           CommandSave,
		ReplyToMessage: &Message{},
	}) {
		t.Fatal("saveReplyHadNoReadableText() = false, want true for empty reply")
	}

	if saveReplyHadNoReadableText(Message{Text: CommandSave}) {
		t.Fatal("saveReplyHadNoReadableText() = true, want false without reply")
	}
}

func TestExtractSaveInputRequiresContent(t *testing.T) {
	t.Parallel()

	_, err := extractSaveInput(Message{Text: CommandSave})
	if err == nil {
		t.Fatal("extractSaveInput() error = nil, want empty input")
	}
}
