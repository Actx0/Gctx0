// Interactive personal assistant using memory search hits as history (+ RAG).
//
//	go run ./examples/06_memories_search
package main

import (
	"context"
	"log"

	"github.com/Actx0/Gctx0"
	"github.com/Actx0/Gctx0/examples/util"
)

const system = "You are a helpful personal assistant. Use what you remember about the user and any provided context to answer. Cite context like [1]. If unsure, say so."

func main() {
	client := util.NewClient()
	defer client.Close()
	ctx := context.Background()

	setup, err := util.Bootstrap(ctx, client,
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
	defer util.Teardown(ctx, client, setup)

	log.Printf("agent=%s session=%s", setup.AgentID, setup.SessionID)
	log.Println("chat — quit/exit to stop")

	err = util.ChatLoop(func(text string) error {
		hits, err := client.Memory.Search(ctx, setup.AgentID, setup.SessionID, text, 5)
		if err != nil {
			return err
		}
		rag, err := util.RAGContext(ctx, client, text, 3)
		if err != nil {
			return err
		}
		reply, usage, err := util.Ask(setup.System, text, util.HistoryFromMemoryHits(hits.Results), rag)
		if err != nil {
			return err
		}
		if reply == "" {
			return nil
		}
		meta := map[string]any{"model": util.DefaultModel, "usage": usage}
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
