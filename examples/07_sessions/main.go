// Create sessions keyed by external_id or by labels.
//
//	go run ./examples/07_sessions
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Actx0/Gctx0"
	"github.com/Actx0/Gctx0/examples/util"
)

func show(label string, value any) {
	fmt.Printf("\n%s\n", label)
	fmt.Println("========================================")
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(value)
}

func main() {
	client := util.NewClient()
	defer client.Close()
	ctx := context.Background()

	agent, err := client.Agent.Create(ctx, "Sessions demo bot", "Used only to demonstrate session create/lookup.", gctx0.AgentWriteOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("agent=%s\n", agent.ID)

	byExternalID, err := client.Session.Create(ctx, agent.ID, gctx0.SessionLookup{ExternalID: "support-ticket-42"}, "Support ticket #42")
	if err != nil {
		log.Fatal(err)
	}
	show("Created with external_id", byExternalID)

	fetched, err := client.Session.GetByLabels(ctx, agent.ID, gctx0.SessionLookup{ExternalID: "support-ticket-42"})
	if err != nil {
		log.Fatal(err)
	}
	show("Fetched by external_id", fetched)

	byLabels, err := client.Session.Create(ctx, agent.ID, gctx0.SessionLookup{
		Labels: map[string]string{"userId": "u-100", "channel": "web"},
	}, "Web chat for user u-100")
	if err != nil {
		log.Fatal(err)
	}
	show("Created with labels", byLabels)

	fetched, err = client.Session.GetByLabels(ctx, agent.ID, gctx0.SessionLookup{
		Labels: map[string]string{"userId": "u-100", "channel": "web"},
	})
	if err != nil {
		log.Fatal(err)
	}
	show("Fetched by labels", fetched)

	listed, err := client.Session.List(ctx, agent.ID, gctx0.SessionLookup{}, 50, 0)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nListed (total=%d)\n", listed.Total)
	fmt.Println("========================================")
	for _, session := range listed.Sessions {
		fmt.Printf("  %s  title=%q  external_id=%q  labels=%v\n", session.ID, session.Title, session.ExternalID, session.Labels)
	}

	if err := client.Session.Delete(ctx, agent.ID, gctx0.SessionLookup{ExternalID: "support-ticket-42"}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nDeleted session with external_id=support-ticket-42")

	if err := client.Session.Delete(ctx, agent.ID, gctx0.SessionLookup{
		Labels: map[string]string{"userId": "u-100", "channel": "web"},
	}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Deleted session with labels={userId=u-100, channel=web}")

	if err := client.Agent.Delete(ctx, agent.ID); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Deleted agent %s\n", agent.ID)
}
