// Upload docs, search them, then delete the uploaded documents.
//
//	go run ./examples/02_docs
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Actx0/Gctx0/examples/util"
)

var labels = map[string]string{"tag": "docs", "team": "platform-team"}

var queries = []string{
	"Where does Mara live?",
	"What kind of work does Mara do?",
	"Who is in Mara's family?",
	"What are Mara's hobbies?",
}

func main() {
	docsDir := filepath.Join("examples", "docs")
	if _, err := os.Stat(docsDir); err != nil {
		docsDir = filepath.Join("docs")
	}
	entries, err := filepath.Glob(filepath.Join(docsDir, "*.txt"))
	if err != nil {
		log.Fatal(err)
	}
	sort.Strings(entries)
	if len(entries) == 0 {
		log.Fatalf("no .txt files in %s", docsDir)
	}

	client := util.NewClient()
	defer client.Close()
	ctx := context.Background()

	listed, err := client.Document.List(ctx, 100, 0)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("remote documents (%d):\n", listed.Total)
	for _, doc := range listed.Documents {
		fmt.Printf("  %s checksum=%s status=%s\n", doc.Filename, doc.Checksum, doc.Status)
	}

	docIDs := make([]string, 0, len(entries))
	fmt.Printf("\nlocal files (%d):\n", len(entries))
	for _, path := range entries {
		existing, err := client.Document.Exists(ctx, path, labels, 50)
		if err != nil {
			log.Fatal(err)
		}
		if existing != nil {
			fmt.Printf("  skip  %s (already uploaded %s)\n", filepath.Base(path), existing.ID)
			docIDs = append(docIDs, existing.ID)
			continue
		}
		title := strings.ReplaceAll(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), "_", " ")
		uploaded, err := client.Document.Upload(ctx, path, title, labels)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  upload %s -> %s checksum=%s\n", filepath.Base(path), uploaded.ID, uploaded.Checksum)
		docIDs = append(docIDs, uploaded.ID)
	}

	fmt.Println("\nwaiting 120 seconds for indexing...")
	time.Sleep(120 * time.Second)

	fmt.Println("\nsearch")
	fmt.Println("========================================")
	for _, query := range queries {
		results, err := client.Document.Search(ctx, query, labels, 3)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("\nquery: %s\n", query)
		fmt.Println("----------------------------------------")
		if len(results.Results) == 0 {
			fmt.Println("  (no hits)")
			continue
		}
		for _, hit := range results.Results {
			fmt.Printf("  [%.2f] %s\n", hit.Score, hit.Text)
		}
	}

	fmt.Println("\ndelete")
	fmt.Println("========================================")
	for _, docID := range docIDs {
		if err := client.Document.Delete(ctx, docID); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  deleted %s\n", docID)
	}
}
