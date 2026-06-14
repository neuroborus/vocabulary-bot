package telegram

import (
	"fmt"
	"strings"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

const (
	reviewCallbackPrefix = "review"
	reviewActionEasy     = "easy"
	reviewActionHard     = "hard"
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
			},
		},
	}
}

func emptyInlineKeyboard() InlineKeyboardMarkup {
	return InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{}}
}

func formatReviewAnswered(item vocabulary.Item, action string, spoilerTranslations bool) string {
	base := formatReviewReminder(item, spoilerTranslations)

	var footer string
	switch action {
	case reviewActionEasy:
		footer = fmt.Sprintf("\n\n<b>✓ Easy</b> — next review in %d day(s).", item.Review.IntervalDays)
	case reviewActionHard:
		footer = "\n\n<b>✓ Hard</b> — next review tomorrow."
	default:
		return base
	}

	return base + footer
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
