package telegram

import (
	"fmt"
	"strings"

	"github.com/neuroborus/vocabulary-bot/internal/save"
	syncer "github.com/neuroborus/vocabulary-bot/internal/sync"
)

const parseModeHTML = "HTML"

func StartupMessage() string {
	return "<b>Vocabulary Bot</b> is online."
}

func formatStartMessage() string {
	var builder strings.Builder
	builder.WriteString("<b>Vocabulary Bot</b>\n")
	builder.WriteString("<i>Imports words from PocketBook and Google Sheets.</i>\n\n")
	builder.WriteString(formatCommandsBlock())
	return builder.String()
}

func formatInfoMessage(health string) string {
	return health + "\n\n" + formatCommandsBlock()
}

func formatHealthMessage(status string, wordCount int, syncEnabled, notificationsEnabled bool, chatID, callerID int64) string {
	message := fmt.Sprintf(
		"<b>Health</b>\n"+
			"Status: %s\n"+
			"Words: <b>%d</b>\n"+
			"Sync: %s\n"+
			"Notifications: %s",
		healthStatusLabel(status),
		wordCount,
		boolLabel(syncEnabled),
		boolLabel(notificationsEnabled),
	)
	if chatID != 0 {
		message += fmt.Sprintf("\nChat ID: <code>%d</code>", chatID)
	}
	if callerID != 0 {
		message += fmt.Sprintf("\nCaller ID: <code>%d</code>", callerID)
	}

	return message
}

func formatSyncSummary(summary syncer.Summary) string {
	var builder strings.Builder
	builder.WriteString("<b>Sync complete</b>\n\n")
	builder.WriteString("<b>Totals</b>\n")
	builder.WriteString(fmt.Sprintf("• Drafts processed: <b>%d</b>\n", summary.DraftsProcessed))
	builder.WriteString(fmt.Sprintf("• Created: <b>%d</b>\n", summary.Created))
	builder.WriteString(fmt.Sprintf("• Updated: <b>%d</b>\n", summary.Updated))
	builder.WriteString(fmt.Sprintf("• Ambiguous: <b>%d</b>\n", summary.Ambiguous))
	builder.WriteString(fmt.Sprintf("• Source errors: <b>%d</b>", summary.SourceErrors))

	if len(summary.Sources) > 0 {
		builder.WriteString("\n\n<b>Sources</b>")
		for _, source := range summary.Sources {
			builder.WriteString("\n• ")
			builder.WriteString(escapeHTML(source.Name))
			builder.WriteString(" — ")
			builder.WriteString(fmt.Sprintf("%d draft", source.Drafts))
			if source.Drafts != 1 {
				builder.WriteString("s")
			}
			if source.Error != "" {
				builder.WriteString("\n  ⚠️ ")
				builder.WriteString(escapeHTML(source.Error))
			}
			if source.Details.BookContextSkippedStored > 0 {
				builder.WriteString("\n  book context skipped (already in DB): ")
				builder.WriteString(fmt.Sprintf("%d", source.Details.BookContextSkippedStored))
			}
			if source.Details.BookContextEnriched > 0 {
				builder.WriteString("\n  book context enriched: ")
				builder.WriteString(fmt.Sprintf("%d", source.Details.BookContextEnriched))
			}
			if source.Details.RowsSkippedUnchanged > 0 {
				builder.WriteString("\n  rows skipped (unchanged): ")
				builder.WriteString(fmt.Sprintf("%d", source.Details.RowsSkippedUnchanged))
			}
			if source.Details.BooksSkipped > 0 {
				builder.WriteString("\n  books skipped: ")
				builder.WriteString(fmt.Sprintf("%d", source.Details.BooksSkipped))
			}
			if source.Details.BooksFailed > 0 {
				builder.WriteString("\n  books failed: ")
				builder.WriteString(fmt.Sprintf("%d", source.Details.BooksFailed))
			}
			if source.Details.NotesFailed > 0 {
				builder.WriteString("\n  notes failed: ")
				builder.WriteString(fmt.Sprintf("%d", source.Details.NotesFailed))
			}
			if source.Details.RowParseErrors > 0 {
				builder.WriteString("\n  row parse errors: ")
				builder.WriteString(fmt.Sprintf("%d", source.Details.RowParseErrors))
			}
		}
	}

	return builder.String()
}

func formatCommandsBlock() string {
	var builder strings.Builder
	builder.WriteString("<b>Commands</b>")

	for _, command := range KnownCommands() {
		builder.WriteString("\n")
		builder.WriteString("<code>")
		builder.WriteString(escapeHTML(command.Command))
		builder.WriteString("</code> — ")
		builder.WriteString(escapeHTML(command.Description))
	}

	return builder.String()
}

func formatSaveConfirmation(result save.Result) string {
	var builder strings.Builder

	if strings.TrimSpace(result.Word) != "" {
		builder.WriteString("<b>Word</b>\n")
		builder.WriteString(escapeHTML(result.Word))
	}
	if strings.TrimSpace(result.Context) != "" {
		builder.WriteString("\n\n<b>Context</b>\n")
		builder.WriteString(escapeHTML(result.Context))
	}
	if strings.TrimSpace(result.Translation) != "" {
		builder.WriteString("\n\n<b>Translation</b>\n")
		builder.WriteString(escapeHTML(result.Translation))
	}

	builder.WriteString("\n\n<tg-spoiler>")
	builder.WriteString("<b>Saved to Google Sheets</b>")
	if result.RowNumber > 0 {
		builder.WriteString("\nRow: ")
		builder.WriteString(fmt.Sprintf("%d", result.RowNumber))
	}
	builder.WriteString("\n\nIt will be imported into local storage on the next scheduled sync. An admin can run /sync in private chat to import it immediately.")
	builder.WriteString("</tg-spoiler>")

	return builder.String()
}

func formatSaveInputRequired(message Message) string {
	if saveReplyHadNoReadableText(message) || isGroupLikeChat(message.Chat.Type) {
		return formatNotice(
			"Nothing to save",
			"The bot could not read the replied message in this group. Send the text with <code>/save</code>, disable <b>Group Privacy</b> in @BotFather, or add the bot as a group admin.",
		)
	}

	return formatNotice(
		"Nothing to save",
		"Reply to the message you want to save with <code>/save</code>, or write the text before or after <code>/save</code>.",
	)
}

func formatNotice(title, body string) string {
	if body == "" {
		return "ℹ️ <b>" + escapeHTML(title) + "</b>"
	}

	return "ℹ️ <b>" + escapeHTML(title) + "</b>\n\n" + body
}

func formatError(title, detail string) string {
	if detail == "" {
		return "⚠️ <b>" + escapeHTML(title) + "</b>"
	}

	return "⚠️ <b>" + escapeHTML(title) + "</b>\n\n<code>" + escapeHTML(detail) + "</code>"
}

func boolLabel(enabled bool) string {
	if enabled {
		return "✅ enabled"
	}

	return "⛔ disabled"
}

func healthStatusLabel(status string) string {
	switch status {
	case "ok":
		return "✅ ok"
	case "degraded":
		return "⚠️ degraded"
	default:
		return escapeHTML(status)
	}
}

func escapeHTML(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(value)
}
