// Create a prompt, add a version, fetch it by handle, then delete it.
//
//	go run ./examples/04_prompts
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Actx0/Gctx0"
	"github.com/Actx0/Gctx0/examples/exampleutil"
)

func show(label string, value any) {
	fmt.Printf("\n%s\n", label)
	fmt.Println("========================================")
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(value)
}

func main() {
	client := exampleutil.NewClient()
	defer client.Close()
	ctx := context.Background()

	production := true
	created, err := client.Prompt.Create(ctx, "Mara Guide", gctx0.PromptTypeText,
		"You answer questions about Mara Ellison using retrieved context.",
		gctx0.PromptWriteOptions{
			Description:   "System prompt for the Mara docs agent",
			CommitMessage: "initial",
			Production:    &production,
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	show("Created prompt", created)

	version, err := client.Prompt.CreateVersion(ctx, created.PromptID, gctx0.PromptTypeText,
		"You answer questions about Mara Ellison using only the provided context. Cite sources like [1]. If the context is missing an answer, say so.",
		gctx0.PromptWriteOptions{CommitMessage: "add citation rule", Production: &production},
	)
	if err != nil {
		log.Fatal(err)
	}
	show("Created version", version)

	latest, err := client.Prompt.GetByName(ctx, created.Handle, "")
	if err != nil {
		log.Fatal(err)
	}
	show("Fetched by handle", latest)
	compiled, err := latest.Compile(nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\ncompiled: %s\n", compiled)

	if err := client.Prompt.Delete(ctx, created.PromptID); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nDeleted %s\n", created.PromptID)
}
