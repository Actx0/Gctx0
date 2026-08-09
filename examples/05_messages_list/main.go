// Interactive personal assistant using the full message list as history (+ RAG).
//
//	go run ./examples/05_messages_list
package main

import (
	"context"
	"log"

	"github.com/Actx0/Gctx0"
	"github.com/Actx0/Gctx0/examples/exampleutil"
)

const system = "You are a helpful personal assistant. Use the prior conversation and any provided context to answer. Cite context like [1]. If unsure, say so."

func main() {
	client := exampleutil.NewClient()
	defer client.Close()
	ctx := context.Background()

	setup, err := exampleutil.Bootstrap(ctx, client,
		"Personal assistant (messages list)",
		"Personal assistant using full message list history + RAG.",
		"Personal Assistant Messages List",
		system,
		"personal-assistant-messages-list",
		"Personal assistant — messages list",
	)
	if err != nil {
		log.Fatal(err)
	}
	defer exampleutil.Teardown(ctx, client, setup)

	log.Printf("agent=%s session=%s", setup.AgentID, setup.SessionID)
	log.Println("chat — quit/exit to stop")

	err = exampleutil.ChatLoop(func(text string) error {
		stored, err := client.Message.List(ctx, setup.AgentID, setup.SessionID, 100, 0)
		if err != nil {
			return err
		}
		rag, err := exampleutil.RAGContext(ctx, client, text, 3)
		if err != nil {
			return err
		}
		reply, usage, err := exampleutil.Ask(setup.System, text, exampleutil.HistoryFromMessages(stored.Messages), rag)
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
