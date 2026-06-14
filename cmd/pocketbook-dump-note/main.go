package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/config"
	"github.com/neuroborus/vocabulary-bot/internal/source/pocketbook"
)

func main() {
	var (
		wordFlag = flag.String("word", "", "match notes whose quotation or note text contains this word")
		uuidFlag = flag.String("uuid", "", "fetch one note by UUID")
		limit    = flag.Int("limit", 5, "max notes to print")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	if !cfg.PocketBook.Enabled {
		fmt.Fprintln(os.Stderr, "POCKETBOOK_ENABLED is false")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	store := pocketbook.NewFileSessionStore(cfg.PocketBook.TokenPath)
	client := pocketbook.NewClient(pocketbook.ClientOptions{
		BaseURL:      cfg.PocketBook.BaseURL,
		Email:        cfg.PocketBook.Email,
		Password:     cfg.PocketBook.Password,
		RefreshToken: cfg.PocketBook.RefreshToken,
		ShopName:     cfg.PocketBook.ShopName,
		SessionStore: store,
	})

	books, err := client.ListBooks(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list books: %v\n", err)
		os.Exit(1)
	}

	needle := strings.ToLower(strings.TrimSpace(*wordFlag))
	uuid := strings.TrimSpace(*uuidFlag)
	printed := 0

	for _, book := range books {
		if book.FastHash == "" {
			continue
		}

		noteIDs, err := client.ListNoteIDs(ctx, book.FastHash)
		if err != nil {
			fmt.Fprintf(os.Stderr, "list notes for %q: %v\n", book.Title, err)
			continue
		}

		for _, info := range noteIDs {
			if info.UUID == "" {
				continue
			}
			if uuid != "" && !strings.EqualFold(info.UUID, uuid) {
				continue
			}

			note, ok, err := client.GetNote(ctx, info.UUID, book.FastHash)
			if err != nil || !ok {
				continue
			}

			if uuid == "" && needle != "" && !noteMatches(note, needle) {
				continue
			}

			draft, parsed := pocketbook.ParseNote(book, note)
			summary := map[string]any{
				"bookTitle": book.Title,
				"bookId":    book.ID,
				"noteUuid":  note.UUID,
				"parsed":    parsed,
				"draft": map[string]any{
					"rawWord":      draft.RawWord,
					"translations": draft.Translations,
					"contexts":     draft.Contexts,
					"anchor":       draft.Anchor,
				},
				"quotation": note.Quotation,
				"note":      note.Note,
				"mark":      note.Mark,
			}

			encoded, err := json.MarshalIndent(summary, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "encode: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(encoded))
			fmt.Println("---")

			printed++
			if uuid != "" || printed >= *limit {
				return
			}
		}
	}

	if printed == 0 {
		fmt.Fprintln(os.Stderr, "no matching notes found")
		os.Exit(1)
	}
}

func noteMatches(note pocketbook.Note, needle string) bool {
	if needle == "" {
		return true
	}

	fields := []string{
		quotationText(note),
		commentText(note),
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), needle) {
			return true
		}
	}
	return false
}

func quotationText(note pocketbook.Note) string {
	if note.Quotation == nil {
		return ""
	}
	return note.Quotation.Text
}

func commentText(note pocketbook.Note) string {
	if note.Note == nil {
		return ""
	}
	return note.Note.Text
}
