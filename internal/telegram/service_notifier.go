package telegram

import "context"

// ServiceNotifier sends operational messages to the configured admin chat.
// Service notifications always use TELEGRAM_ADMIN_ID, not TELEGRAM_TARGET_CHANNEL_ID.
type ServiceNotifier struct {
	notifier Notifier
	chatID   int64
}

func NewServiceNotifier(notifier Notifier, adminID int64) *ServiceNotifier {
	return &ServiceNotifier{
		notifier: notifier,
		chatID:   adminID,
	}
}

func (n *ServiceNotifier) Notify(ctx context.Context, text string) error {
	return n.notifier.SendHTMLMessage(ctx, n.chatID, text)
}
