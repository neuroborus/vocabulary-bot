package telegram

import "context"

const (
	CommandHealth    = "/health"
	CommandListWords = "/list-words"
	CommandTurnOff   = "/turn-off"
	CommandTurnOn    = "/turn-on"
	CommandSync      = "/sync"
	CommandPush      = "/push"
	CommandLogs      = "/logs"
)

type Notifier interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
	SendDocument(ctx context.Context, chatID int64, path string, caption string) error
}

func KnownCommands() []string {
	return []string{
		CommandHealth,
		CommandListWords,
		CommandTurnOff,
		CommandTurnOn,
		CommandSync,
		CommandPush,
		CommandLogs,
	}
}
