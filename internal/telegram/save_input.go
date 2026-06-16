package telegram

import (
	"errors"
	"strings"
)

var errEmptySaveInput = errors.New("save input is empty")

func extractSaveInput(message Message) (string, error) {
	if message.ReplyToMessage != nil {
		replyText := strings.TrimSpace(message.ReplyToMessage.Text)
		if replyText != "" {
			return replyText, nil
		}
	}

	remainder := strings.TrimSpace(stripLeadingCommand(message.Text, CommandSave))
	if remainder == "" {
		return "", errEmptySaveInput
	}

	return remainder, nil
}

func stripLeadingCommand(text, command string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}

	first := fields[0]
	if !strings.HasPrefix(first, command) {
		return text
	}

	return strings.TrimSpace(strings.TrimPrefix(text, first))
}
