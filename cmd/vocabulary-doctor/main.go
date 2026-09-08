// Command vocabulary-doctor audits the stored vocabulary collection for
// collapsed items: entities whose forms do not share any strong lookup key and
// therefore represent unrelated phrases merged into one record (the weak-key
// collision, e.g. many "X the Y" idioms joined via "the").
//
// It is read-only: it reports the collapsed items and how their forms cluster
// so an agent or operator can decide how to repair each one. It never rewrites
// data. Exit codes let it be used as a check:
//
//	0  no collapsed items
//	1  collapsed items found (report on stdout)
//	2  execution error (message on stderr)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/neuroborus/vocabulary-bot/internal/config"
	mongostorage "github.com/neuroborus/vocabulary-bot/internal/storage/mongo"
	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

type anchorRef struct {
	Source     string `json:"source"`
	SheetName  string `json:"sheetName,omitempty"`
	RowNumber  int    `json:"rowNumber,omitempty"`
	BookTitle  string `json:"bookTitle,omitempty"`
	ExternalID string `json:"externalId,omitempty"`
}

type report struct {
	NormalizedKey string                   `json:"normalizedKey"`
	DisplayWord   string                   `json:"displayWord"`
	ClusterCount  int                      `json:"clusterCount"`
	Clusters      []vocabulary.FormCluster `json:"clusters"`
	Anchors       []anchorRef              `json:"anchors,omitempty"`
}

func main() {
	asJSON := flag.Bool("json", false, "emit the report as JSON for machine consumption")
	timeout := flag.Duration("timeout", 2*time.Minute, "overall timeout")
	flag.Parse()

	code, err := run(*asJSON, *timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vocabulary-doctor: %v\n", err)
		os.Exit(2)
	}
	os.Exit(code)
}

// run returns the process exit code (0 clean, 1 collapsed items found). It keeps
// os.Exit out of the body so the MongoDB disconnect defer always runs.
func run(asJSON bool, timeout time.Duration) (int, error) {
	cfg, err := config.Load()
	if err != nil {
		return 0, fmt.Errorf("load config: %w", err)
	}
	if cfg.MongoDB.URI == "" {
		return 0, fmt.Errorf("MONGODB_URI is not set; nothing to audit")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, err := mongostorage.Connect(ctx, cfg.MongoDB.URI)
	if err != nil {
		return 0, fmt.Errorf("connect mongodb: %w", err)
	}
	defer func() {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = client.Disconnect(disconnectCtx)
	}()

	repository := mongostorage.NewVocabularyRepository(client.Database(cfg.MongoDB.DBName))
	items, err := repository.List(ctx)
	if err != nil {
		return 0, fmt.Errorf("list vocabulary items: %w", err)
	}

	reports := collectReports(items)

	if asJSON {
		if err := writeJSON(reports); err != nil {
			return 0, err
		}
	} else {
		writeText(len(items), reports)
	}

	if len(reports) > 0 {
		return 1, nil
	}

	return 0, nil
}

func collectReports(items []vocabulary.Item) []report {
	reports := make([]report, 0)
	for _, item := range items {
		clusters := vocabulary.ClusterItemForms(item)
		if len(clusters) <= 1 {
			continue
		}
		reports = append(reports, report{
			NormalizedKey: item.NormalizedKey,
			DisplayWord:   item.DisplayWord,
			ClusterCount:  len(clusters),
			Clusters:      clusters,
			Anchors:       anchorRefs(item),
		})
	}

	return reports
}

func anchorRefs(item vocabulary.Item) []anchorRef {
	refs := make([]anchorRef, 0, len(item.Anchors))
	for _, anchor := range item.Anchors {
		refs = append(refs, anchorRef{
			Source:     string(anchor.Source),
			SheetName:  anchor.SheetName,
			RowNumber:  anchor.RowNumber,
			BookTitle:  anchor.BookTitle,
			ExternalID: anchor.ExternalID,
		})
	}

	return refs
}

func writeJSON(reports []report) error {
	encoded, err := json.MarshalIndent(reports, "", "  ")
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	fmt.Println(string(encoded))

	return nil
}

func writeText(scanned int, reports []report) {
	fmt.Printf("Scanned %d items, %d collapsed.\n", scanned, len(reports))
	for _, r := range reports {
		fmt.Printf("\n[collapsed] key=%q display=%q clusters=%d\n", r.NormalizedKey, r.DisplayWord, r.ClusterCount)
		for i, cluster := range r.Clusters {
			fmt.Printf("  cluster %d: %v keys=%v\n", i+1, cluster.Forms, cluster.StrongKeys)
		}
		for _, anchor := range r.Anchors {
			fmt.Printf("  anchor: %s\n", formatAnchor(anchor))
		}
	}
}

func formatAnchor(anchor anchorRef) string {
	switch {
	case anchor.SheetName != "" || anchor.RowNumber != 0:
		return fmt.Sprintf("%s %s row %d", anchor.Source, anchor.SheetName, anchor.RowNumber)
	case anchor.BookTitle != "":
		return fmt.Sprintf("%s book %q", anchor.Source, anchor.BookTitle)
	case anchor.ExternalID != "":
		return fmt.Sprintf("%s external %s", anchor.Source, anchor.ExternalID)
	default:
		return anchor.Source
	}
}
