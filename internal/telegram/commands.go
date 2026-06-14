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
			Description: "Show bot help and available commands",
		},
		{
			Command:     CommandInfo,
			Description: "Show service info and available commands",
		},
		{
			Command:     CommandHealth,
			Description: "Show service health and current word count",
		},
		{
			Command:     CommandSync,
			Description: "Synchronize PocketBook and Google Sheets now",
		},
		{
			Command:     CommandListWords,
			Description: "Export all vocabulary items as JSON",
		},
		{
			Command:     CommandLogs,
			Description: "Send the current application log file",
		},
		{
			Command:     CommandTurnOff,
			Description: "Disable sync and notifications",
		},
		{
			Command:     CommandTurnOn,
			Description: "Enable sync and notifications",
		},
		{
			Command:     CommandPush,
			Description: "Send one review word when review push is implemented",
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
