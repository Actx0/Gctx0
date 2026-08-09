// Create a client and print access key info via /api/v1/me.
//
//	go run ./examples/01_me
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Actx0/Gctx0/examples/exampleutil"
)

func main() {
	client := exampleutil.NewClient()
	defer client.Close()

	me, err := client.Me.Get(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Access key info")
	fmt.Println("========================================")
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(me.AccessKey)
}
