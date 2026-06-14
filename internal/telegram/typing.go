package telegram

import (
	"context"
	"time"
)

const (
	chatActionTyping         = "typing"
	chatActionUploadDocument = "upload_document"
)

func runWithChatAction(ctx context.Context, notifier Notifier, chatID int64, action string, fn func(context.Context) error) error {
	stopAction := keepChatAction(ctx, notifier, chatID, action)
	defer stopAction()

	return fn(ctx)
}

func keepChatAction(ctx context.Context, notifier Notifier, chatID int64, action string) func() {
	send := func() {
		_ = notifier.SendChatAction(ctx, chatID, action)
	}
	send()

	typingCtx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				send()
			}
		}
	}()

	return cancel
}
