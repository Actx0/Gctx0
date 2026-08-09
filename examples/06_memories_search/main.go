// Interactive personal assistant using memory search hits as history (+ RAG).
//
//	go run ./examples/06_memories_search
package main

import (
	"context"
	"log"

	"github.com/Actx0/Gctx0"
	"github.com/Actx0/Gctx0/examples/exampleutil"
)

const system = "You are a helpful personal assistant. Use what you remember about the user and any provided context to answer. Cite context like [1]. If unsure, say so."

func main() {
	client := exampleutil.NewClient()
	defer client.Close()
	ctx := context.Background()

	setup, err := exampleutil.Bootstrap(ctx, client,
		"Personal assistant (memories search)",
		"Personal assistant using memory search history + RAG.",
		"Personal Assistant Memories Search",
		system,
		"personal-assistant-memories-search",
		"Personal assistant — memories search",
	)
	if err != nil {
		log.Fatal(err)
	}
	defer exampleutil.Teardown(ctx, client, setup)

	log.Printf("agent=%s session=%s", setup.AgentID, setup.SessionID)
	log.Println("chat — quit/exit to stop")

	err = exampleutil.ChatLoop(func(text string) error {
		hits, err := client.Memory.Search(ctx, setup.AgentID, setup.SessionID, text, 5)
		if err != nil {
			return err
		}
		rag, err := exampleutil.RAGContext(ctx, client, text, 3)
		if err != nil {
			return err
		}
		reply, usage, err := exampleutil.Ask(setup.System, text, exampleutil.HistoryFromMemoryHits(hits.Results), rag)
		if err != nil {
			return err
		}
		if reply == "" {
			return nil
		}
		meta := map[string]any{"model": exampleutil.DefaultModel, "usage": usage}
		_, err = client.Message.CreateBatch(ctx, setup.AgentID, setup.SessionID, []gctx0.MessageInput{
			{Role: gctx0.MessageRoleUser, Content: text},
			{Role: gctx0.MessageRoleAssistant, Content: reply, Meta: meta},
		})
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
}
