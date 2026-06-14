package telegram

import "context"

const (
	CommandStart     = "/start"
	CommandInfo      = "/info"
	CommandHealth    = "/health"
	CommandSync      = "/sync"
	CommandListWords = "/list_words"
	CommandLogs      = "/logs"
	CommandTurnOff   = "/turn_off"
	CommandTurnOn    = "/turn_on"
	CommandPush      = "/push"
)

type Notifier interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
	SendHTMLMessage(ctx context.Context, chatID int64, text string) error
	SendHTMLMessageWithKeyboard(ctx context.Context, chatID int64, text string, keyboard InlineKeyboardMarkup) error
	EditHTMLMessage(ctx context.Context, chatID int64, messageID int, text string, keyboard InlineKeyboardMarkup) error
	SendChatAction(ctx context.Context, chatID int64, action string) error
	AnswerCallbackQuery(ctx context.Context, callbackQueryID string, text string) error
	SendDocument(ctx context.Context, chatID int64, path string, caption string) error
}

type CommandDescription struct {
	Command     string
	Description string
}

type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

func KnownCommands() []CommandDescription {
	return []CommandDescription{
		{
			Command:     CommandStart,
			Description: "Welcome message and command reference",
		},
		{
			Command:     CommandInfo,
			Description: "Health snapshot plus command reference",
		},
		{
			Command:     CommandHealth,
			Description: "Health status, word count, and enabled flags",
		},
		{
			Command:     CommandSync,
			Description: "Sync PocketBook and Google Sheets into storage now",
		},
		{
			Command:     CommandListWords,
			Description: "Send all vocabulary items as a JSON file",
		},
		{
			Command:     CommandLogs,
			Description: "Send the current log file without clearing it",
		},
		{
			Command:     CommandTurnOff,
			Description: "Disable automatic sync and notifications",
		},
		{
			Command:     CommandTurnOn,
			Description: "Enable automatic sync and notifications",
		},
		{
			Command:     CommandPush,
			Description: "Manually send one review word with Easy/Hard buttons",
		},
	}
}

func BotCommands() []BotCommand {
	known := KnownCommands()
	commands := make([]BotCommand, 0, len(known))

	for _, command := range known {
		commands = append(commands, BotCommand{
			Command:     command.Command[1:],
			Description: command.Description,
		})
	}

	return commands
}
