package telegram

import (
	"context"
	"log/slog"
	"time"
)

type Bot struct {
	client  *Client
	handler *CommandHandler
	logger  *slog.Logger
}

func NewBot(client *Client, handler *CommandHandler, logger *slog.Logger) *Bot {
	if logger == nil {
		logger = slog.Default()
	}

	return &Bot{
		client:  client,
		handler: handler,
		logger:  logger,
	}
}

func (b *Bot) Poll(ctx context.Context) error {
	offset := 0

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		updates, err := b.client.GetUpdates(ctx, offset, 25)
		if err != nil {
			b.logger.Error("telegram getUpdates failed", slog.String("error", err.Error()))
			if err := sleepWithContext(ctx, 5*time.Second); err != nil {
				return err
			}
			continue
		}

		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
			if update.CallbackQuery != nil {
				if err := b.handler.HandleCallbackQuery(ctx, *update.CallbackQuery); err != nil {
					b.logger.Error("telegram callback failed", slog.String("error", err.Error()))
				}
				continue
			}
			if update.Message == nil {
				continue
			}
			if err := b.handler.HandleMessage(ctx, *update.Message); err != nil {
				b.logger.Error("telegram command failed", slog.String("error", err.Error()))
			}
		}
	}
}

func sleepWithContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
