package telegram

import (
	"fmt"
	"strings"
)

const (
	reviewCallbackPrefix = "review"
	reviewActionEasy     = "easy"
	reviewActionHard     = "hard"
	reviewActionRemove   = "remove"
)

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

func reviewKeyboard(normalizedKey string) InlineKeyboardMarkup {
	return InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "Easy", CallbackData: reviewCallbackData(reviewActionEasy, normalizedKey)},
				{Text: "Hard", CallbackData: reviewCallbackData(reviewActionHard, normalizedKey)},
				{Text: "Remove", CallbackData: reviewCallbackData(reviewActionRemove, normalizedKey)},
			},
		},
	}
}

func reviewCallbackData(action, normalizedKey string) string {
	return fmt.Sprintf("%s:%s:%s", reviewCallbackPrefix, action, normalizedKey)
}

func parseReviewCallback(data string) (action string, normalizedKey string, ok bool) {
	parts := strings.SplitN(data, ":", 3)
	if len(parts) != 3 || parts[0] != reviewCallbackPrefix {
		return "", "", false
	}
	if parts[1] == "" || parts[2] == "" {
		return "", "", false
	}

	return parts[1], parts[2], true
}
