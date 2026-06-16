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

	remainder := strings.TrimSpace(stripCommandToken(message.Text, CommandSave))
	if remainder == "" {
		return "", errEmptySaveInput
	}

	return remainder, nil
}

func stripCommandToken(text, command string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}

	withoutCommand := make([]string, 0, len(fields))
	removed := false
	for _, field := range fields {
		if !removed && commandTokenMatches(field, command) {
			removed = true
			continue
		}
		withoutCommand = append(withoutCommand, field)
	}

	return strings.TrimSpace(strings.Join(withoutCommand, " "))
}

func commandTokenMatches(token, command string) bool {
	token = strings.TrimSpace(token)
	if token == "" || !strings.HasPrefix(token, "/") {
		return false
	}

	if index := strings.Index(token, "@"); index >= 0 {
		token = token[:index]
	}

	return token == command
}
