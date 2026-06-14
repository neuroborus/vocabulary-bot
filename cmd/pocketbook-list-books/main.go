package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/config"
	"github.com/neuroborus/vocabulary-bot/internal/source/pocketbook"
)

func main() {
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	if !cfg.PocketBook.Enabled {
		fmt.Fprintln(os.Stderr, "POCKETBOOK_SYNC_ENABLED is false")
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

	encoded, err := json.MarshalIndent(books, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
}
