### Gctx0 Client

Go client for the Actx0 platform.

#### Install

```bash
go get github.com/Actx0/Gctx0
```

#### Usage

```go
package main

import (
	"context"

	"github.com/Actx0/Gctx0"
)

func main() {
	client := gctx0.NewKnowledge(
		gctx0.WithAccessKey("your-access-key"),
		gctx0.WithWorkspaceId("your-workspace-id"),
	)
	defer client.Close()

	_, _ = client.List(context.Background(), 50, 0)
}
```

Or use the full client when you need multiple API areas:

```go
client := gctx0.NewClient(
	gctx0.WithAccessKey("your-access-key"),
	gctx0.WithWorkspaceId("your-workspace-id"),
)
defer client.Close()

_, _ = client.Health(context.Background())
_, _ = client.Knowledge.List(context.Background(), 50, 0)
```

#### Development

```bash
go test ./...
go build ./examples/...
```

Tests start a local mock API server automatically. To run against your own local Actx0 server:

```bash
GCTX0_BASE_URL=http://127.0.0.1:8000 go test ./...
```
