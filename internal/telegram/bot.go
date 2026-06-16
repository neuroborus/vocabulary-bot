package telegram

import (
	"context"
	"log/slog"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/logging"
)

type Bot struct {
	client               *Client
	handler              *CommandHandler
	allowlist            ChatAllowlist
	leaveDisallowedChats bool
	logger               *slog.Logger
}

func NewBot(client *Client, handler *CommandHandler, allowlist ChatAllowlist, leaveDisallowedChats bool, logger *slog.Logger) *Bot {
	if logger == nil {
		logger = slog.Default()
	}

	return &Bot{
		client:               client,
		handler:              handler,
		allowlist:            allowlist,
		leaveDisallowedChats: leaveDisallowedChats,
		logger:               logger,
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
			b.logger.Error("telegram getUpdates failed", slog.String("error", logging.SanitizeError(err)))
			if err := sleepWithContext(ctx, 5*time.Second); err != nil {
				return err
			}
			continue
		}

		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
			if err := b.handleUpdate(ctx, update); err != nil {
				b.logger.Error("telegram update failed", slog.String("error", logging.SanitizeError(err)))
			}
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update Update) error {
	if update.MyChatMember != nil {
		return b.handleMyChatMember(ctx, *update.MyChatMember)
	}
	if update.CallbackQuery != nil {
		if update.CallbackQuery.Message != nil {
			if rejected, err := b.rejectDisallowedChat(ctx, update.CallbackQuery.Message.Chat); rejected || err != nil {
				return err
			}
		}
		return b.handler.HandleCallbackQuery(ctx, *update.CallbackQuery)
	}
	if update.Message == nil {
		return nil
	}
	if rejected, err := b.rejectDisallowedChat(ctx, update.Message.Chat); rejected || err != nil {
		return err
	}

	return b.handler.HandleMessage(ctx, *update.Message)
}

func (b *Bot) handleMyChatMember(ctx context.Context, update ChatMemberUpdated) error {
	if b.allowlist.Allows(update.Chat.ID) {
		return nil
	}

	switch update.NewChatMember.Status {
	case "member", "administrator", "restricted":
		return b.leaveDisallowedChat(ctx, update.Chat)
	default:
		return nil
	}
}

func (b *Bot) rejectDisallowedChat(ctx context.Context, chat Chat) (bool, error) {
	if b.allowlist.Allows(chat.ID) {
		return false, nil
	}

	err := b.leaveDisallowedChat(ctx, chat)
	return true, err
}

func (b *Bot) leaveDisallowedChat(ctx context.Context, chat Chat) error {
	b.logger.Warn(
		"telegram chat is not allowlisted",
		slog.Int64("chat_id", chat.ID),
		slog.String("chat_type", chat.Type),
		slog.Bool("leave_enabled", b.leaveDisallowedChats),
	)

	if !b.leaveDisallowedChats || !isGroupLikeChat(chat.Type) {
		return nil
	}

	if err := b.client.LeaveChat(ctx, chat.ID); err != nil {
		b.logger.Error(
			"telegram leave chat failed",
			slog.Int64("chat_id", chat.ID),
			slog.String("error", logging.SanitizeError(err)),
		)
		return err
	}

	b.logger.Info(
		"left disallowed telegram chat",
		slog.Int64("chat_id", chat.ID),
		slog.String("chat_type", chat.Type),
	)

	return nil
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
