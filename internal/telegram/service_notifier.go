package telegram

import "context"

// ServiceNotifier sends operational messages to the configured owner chat.
// Service notifications always use TELEGRAM_ALLOWED_USER_ID, not TELEGRAM_TARGET_CHAT_ID.
type ServiceNotifier struct {
	notifier Notifier
	chatID   int64
}

func NewServiceNotifier(notifier Notifier, allowedUserID int64) *ServiceNotifier {
	return &ServiceNotifier{
		notifier: notifier,
		chatID:   allowedUserID,
	}
}

func (n *ServiceNotifier) Notify(ctx context.Context, text string) error {
	return n.notifier.SendHTMLMessage(ctx, n.chatID, text)
}
