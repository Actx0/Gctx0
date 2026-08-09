// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"errors"
	"os"
	"testing"
)

func testClient(t *testing.T) (*Client, context.Context) {
	t.Helper()
	baseURL := os.Getenv("GCTX0_BASE_URL")
	if baseURL == "" {
		baseURL = startMockServer(t)
	}
	client := NewClient(
		WithBaseURL(baseURL),
		WithAccessKey(defaultWorkspaceAccessKey),
		WithWorkspaceID(defaultWorkspaceID),
	)
	t.Cleanup(client.Close)
	return client, context.Background()
}

func TestHealth(t *testing.T) {
	client, ctx := testClient(t)
	got, err := client.Health(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got["status"] != "ok" {
		t.Fatalf("status = %v", got["status"])
	}
}

func TestMeAccessKey(t *testing.T) {
	client, ctx := testClient(t)
	principal, err := client.Me.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if principal.PrincipalType != "access_key" {
		t.Fatalf("principalType = %q", principal.PrincipalType)
	}
	if principal.AccessKey.WorkspaceID != defaultWorkspaceID {
		t.Fatalf("workspaceId = %q", principal.AccessKey.WorkspaceID)
	}
	if principal.AccessKey.Name != "Agent runtime" {
		t.Fatalf("name = %q", principal.AccessKey.Name)
	}
}

func TestMeInvalidAccessKey(t *testing.T) {
	baseURL := startMockServer(t)
	client := NewClient(
		WithBaseURL(baseURL),
		WithAccessKey("bad-access-key"),
		WithWorkspaceID(defaultWorkspaceID),
	)
	defer client.Close()

	_, err := client.Me.Get(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
}

func TestStandaloneMeClient(t *testing.T) {
	baseURL := startMockServer(t)
	client := NewMe(WithBaseURL(baseURL), WithAccessKey(defaultWorkspaceAccessKey))
	defer client.Close()
	principal, err := client.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if principal.PrincipalType != "access_key" {
		t.Fatalf("principalType = %q", principal.PrincipalType)
	}
}

func TestResourcesShareHTTPClient(t *testing.T) {
	client, _ := testClient(t)
	if client.Agent.resty != client.resty {
		t.Fatal("agent resty client not shared")
	}
	if client.Document.resty != client.resty || client.Knowledge.resty != client.resty {
		t.Fatal("document resty client not shared")
	}
	if client.Me.resty != client.resty || client.Memory.resty != client.resty {
		t.Fatal("me/memory resty client not shared")
	}
	if client.Message.resty != client.resty || client.Prompt.resty != client.resty || client.Session.resty != client.resty {
		t.Fatal("message/prompt/session resty client not shared")
	}
}

func TestMetaHelpers(t *testing.T) {
	body := WithMeta(map[string]string{"content": "x"}, map[string]any{"source": "sdk"})
	if body["meta"] != `{"source":"sdk"}` {
		t.Fatalf("meta = %q", body["meta"])
	}
	if _, ok := WithMeta(map[string]string{}, nil)["meta"]; ok {
		t.Fatal("expected no meta for nil")
	}

	payload := MessageBatchPayload([]MessageInput{
		{Role: MessageRoleUser, Content: "Hi", Meta: map[string]any{"source": "import"}},
	})
	messages := payload["messages"].([]map[string]string)
	if messages[0]["meta"] != `{"source":"import"}` {
		t.Fatalf("message meta = %q", messages[0]["meta"])
	}

	memPayload := MemoryBatchPayload([]MemoryInput{
		{Kind: MemoryKindFact, Content: "Prefers dark mode", Meta: map[string]any{"source": "import"}},
	})
	memories := memPayload["memories"].([]map[string]string)
	if memories[0]["meta"] != `{"source":"import"}` {
		t.Fatalf("memory meta = %q", memories[0]["meta"])
	}
}
