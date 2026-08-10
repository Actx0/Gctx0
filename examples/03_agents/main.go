// Create, list, get, update, and delete an agent.
//
//	go run ./examples/03_agents
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

	created, err := client.Agent.Create(ctx, "Mara assistant", "Answers questions about Mara Ellison from the docs knowledge base.", gctx0.AgentWriteOptions{})
	if err != nil {
		log.Fatal(err)
	}
	show("Created", created)
	fmt.Printf("memoryPipeline=%v\n", created.Configs.MemoryPipeline)

	listed, err := client.Agent.List(ctx, 50, 0)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nListed (total=%d)\n", listed.Total)
	fmt.Println("========================================")
	for _, agent := range listed.Agents {
		fmt.Printf("  %s  %s  status=%s  memoryPipeline=%v\n", agent.ID, agent.Name, agent.Status, agent.Configs.MemoryPipeline)
	}

	fetched, err := client.Agent.Get(ctx, created.ID)
	if err != nil {
		log.Fatal(err)
	}
	show("Fetched", fetched)

	disabled := false
	updated, err := client.Agent.Update(ctx, created.ID, "Mara assistant v2", "Updated description for the Mara docs agent.", gctx0.AgentWriteOptions{
		MemoryPipeline: &disabled,
	})
	if err != nil {
		log.Fatal(err)
	}
	show("Updated", updated)

	if err := client.Agent.Delete(ctx, created.ID); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nDeleted %s\n", created.ID)
}
