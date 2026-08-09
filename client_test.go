// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

func TestAgentListAndGet(t *testing.T) {
	client, ctx := testClient(t)
	agents, err := client.Agent.List(ctx, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if agents.Total < 1 || agents.Agents[0].Name != "Support bot" {
		t.Fatalf("unexpected agents: %+v", agents)
	}
	agent, err := client.Agent.Get(ctx, defaultAgentID)
	if err != nil {
		t.Fatal(err)
	}
	if agent.ID != defaultAgentID || agent.Kind != "unmanaged" {
		t.Fatalf("unexpected agent: %+v", agent)
	}
}

func TestAgentCreateUpdateDelete(t *testing.T) {
	client, ctx := testClient(t)
	created, err := client.Agent.Create(ctx, "Bot", "Test bot")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := client.Agent.Update(ctx, created.ID, "Renamed bot", "Updated description")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Renamed bot" {
		t.Fatalf("name = %q", updated.Name)
	}
	if err := client.Agent.Delete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
}

func TestDocumentListSearchUpload(t *testing.T) {
	client, ctx := testClient(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.md")
	if err := os.WriteFile(path, []byte("# Refund policy\n30 day window."), 0o644); err != nil {
		t.Fatal(err)
	}

	uploaded, err := client.Document.Upload(ctx, path, "Refund policy", map[string]string{
		"team": "support", "category": "policy",
	})
	if err != nil {
		t.Fatal(err)
	}
	if uploaded.Title != "Refund policy" || uploaded.Status != "processing" {
		t.Fatalf("unexpected upload: %+v", uploaded)
	}

	listed, err := client.Document.List(ctx, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if listed.Total != 1 || listed.Documents[0].ID != uploaded.ID {
		t.Fatalf("unexpected list: %+v", listed)
	}

	results, err := client.Document.Search(ctx, "refund policy", map[string]string{"team": "support"}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results.Results) != 1 || results.Results[0].Score != 0.87 {
		t.Fatalf("unexpected search: %+v", results)
	}
	if err := client.Document.Delete(ctx, uploaded.ID); err != nil {
		t.Fatal(err)
	}
}

func TestDocumentExists(t *testing.T) {
	client, ctx := testClient(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.md")
	if err := os.WriteFile(path, []byte("# Refund policy\n30 day window."), 0o644); err != nil {
		t.Fatal(err)
	}
	labels := map[string]string{"team": "support", "category": "policy"}

	found, err := client.Document.Exists(ctx, path, labels, 50)
	if err != nil {
		t.Fatal(err)
	}
	if found != nil {
		t.Fatal("expected nil before upload")
	}

	uploaded, err := client.Document.Upload(ctx, path, "Refund policy", labels)
	if err != nil {
		t.Fatal(err)
	}
	found, err = client.Document.Exists(ctx, path, labels, 50)
	if err != nil {
		t.Fatal(err)
	}
	if found == nil || found.ID != uploaded.ID {
		t.Fatalf("expected uploaded doc, got %+v", found)
	}

	found, err = client.Document.Exists(ctx, path, map[string]string{"team": "other"}, 50)
	if err != nil {
		t.Fatal(err)
	}
	if found != nil {
		t.Fatal("expected nil for different labels")
	}

	if err := os.WriteFile(path, []byte("# Refund policy\nUpdated text."), 0o644); err != nil {
		t.Fatal(err)
	}
	found, err = client.Document.Exists(ctx, path, labels, 50)
	if err != nil {
		t.Fatal(err)
	}
	if found != nil {
		t.Fatal("expected nil for changed content")
	}
	_ = client.Document.Delete(ctx, uploaded.ID)
}

func TestPromptGetLatest(t *testing.T) {
	client, ctx := testClient(t)
	prompt, err := client.Prompt.GetByName(ctx, "customer-support", "")
	if err != nil {
		t.Fatal(err)
	}
	if prompt.Handle != "customer-support" || prompt.Version != 2 {
		t.Fatalf("unexpected prompt: %+v", prompt)
	}
	if prompt.Content != "You are a helpful assistant v2\n{{ctx}}" {
		t.Fatalf("content = %q", prompt.Content)
	}
}

func TestPromptGetWithVersion(t *testing.T) {
	client, ctx := testClient(t)
	prompt, err := client.Prompt.GetByName(ctx, "customer-support", "v1")
	if err != nil {
		t.Fatal(err)
	}
	if prompt.Version != 1 {
		t.Fatalf("version = %d", prompt.Version)
	}
}

func TestPromptGetNamedVersions(t *testing.T) {
	client, ctx := testClient(t)
	latest, err := client.Prompt.GetByName(ctx, "customer-support", "latest")
	if err != nil {
		t.Fatal(err)
	}
	if latest.Version != 2 {
		t.Fatalf("latest version = %d", latest.Version)
	}
	production, err := client.Prompt.GetByName(ctx, "customer-support", "production")
	if err != nil {
		t.Fatal(err)
	}
	if production.Version != 1 {
		t.Fatalf("production version = %d", production.Version)
	}
}

func TestPromptCompile(t *testing.T) {
	client, ctx := testClient(t)
	prompt, err := client.Prompt.GetByName(ctx, "customer-support", "")
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := prompt.Compile(map[string]string{"ctx": "Ahmed"})
	if err != nil {
		t.Fatal(err)
	}
	if compiled != "You are a helpful assistant v2\nAhmed" {
		t.Fatalf("compiled = %q", compiled)
	}
}

func TestPromptGetRequiresWorkspace(t *testing.T) {
	baseURL := startMockServer(t)
	client := NewClient(WithBaseURL(baseURL), WithAccessKey(defaultWorkspaceAccessKey))
	defer client.Close()
	_, err := client.Prompt.GetByName(context.Background(), "customer-support", "")
	if err == nil || err.Error() != "workspace_id is required" {
		t.Fatalf("expected workspace_id error, got %v", err)
	}
}

func TestPromptCRUDAndVersions(t *testing.T) {
	client, ctx := testClient(t)
	prod := false
	created, err := client.Prompt.Create(ctx, "Mara Guide", PromptTypeText, "You know Mara Ellison.", PromptWriteOptions{
		Description:   "Answers questions about Mara",
		Config:        map[string]any{"tone": "friendly"},
		CommitMessage: "initial",
		Meta:          map[string]any{"source": "examples"},
		Production:    &prod,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Handle != "mara-guide" || created.VersionCount != 1 {
		t.Fatalf("unexpected create: %+v", created)
	}

	fetched, err := client.Prompt.Get(ctx, created.PromptID)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.PromptID != created.PromptID {
		t.Fatalf("fetched id = %q", fetched.PromptID)
	}

	production := true
	version, err := client.Prompt.CreateVersion(ctx, created.PromptID, PromptTypeText, "You know Mara Ellison well.", PromptWriteOptions{
		CommitMessage: "v2",
		Production:    &production,
	})
	if err != nil {
		t.Fatal(err)
	}
	if version.Version != 2 || !version.Production {
		t.Fatalf("unexpected version: %+v", version)
	}

	versions, err := client.Prompt.ListVersions(ctx, created.PromptID, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if versions.Total != 2 {
		t.Fatalf("total = %d", versions.Total)
	}

	got, err := client.Prompt.GetVersion(ctx, created.PromptID, version.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "You know Mara Ellison well." {
		t.Fatalf("content = %q", got.Content)
	}

	updated, err := client.Prompt.UpdateVersion(ctx, created.PromptID, version.ID, "You know Mara Ellison very well.", PromptWriteOptions{
		Status:     PromptStatusActive,
		Production: &production,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Content != "You know Mara Ellison very well." {
		t.Fatalf("updated content = %q", updated.Content)
	}

	byName, err := client.Prompt.GetByName(ctx, "mara-guide", "production")
	if err != nil {
		t.Fatal(err)
	}
	if byName.ID != version.ID {
		t.Fatalf("byName id = %q", byName.ID)
	}

	if err := client.Prompt.DeleteVersion(ctx, created.PromptID, version.ID); err != nil {
		t.Fatal(err)
	}
	if err := client.Prompt.Delete(ctx, created.PromptID); err != nil {
		t.Fatal(err)
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

func TestStandaloneKnowledgeClient(t *testing.T) {
	baseURL := startMockServer(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	client := NewKnowledge(
		WithBaseURL(baseURL),
		WithAccessKey(defaultWorkspaceAccessKey),
		WithWorkspaceID(defaultWorkspaceID),
	)
	defer client.Close()

	uploaded, err := client.Upload(context.Background(), path, "Notes", nil)
	if err != nil {
		t.Fatal(err)
	}
	if uploaded.Status != "processing" {
		t.Fatalf("status = %q", uploaded.Status)
	}
	if err := client.Delete(context.Background(), uploaded.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSessionFlow(t *testing.T) {
	client, ctx := testClient(t)
	created, err := client.Session.Create(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-123"}, "Support chat")
	if err != nil {
		t.Fatal(err)
	}
	if created.ExternalID != "thread-123" || created.Title != "Support chat" {
		t.Fatalf("unexpected session: %+v", created)
	}

	fetched, err := client.Session.Get(ctx, defaultAgentID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.ID != created.ID {
		t.Fatalf("fetched id = %q", fetched.ID)
	}

	byLabels, err := client.Session.GetByLabels(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-123"})
	if err != nil {
		t.Fatal(err)
	}
	if byLabels.ID != created.ID {
		t.Fatalf("byLabels id = %q", byLabels.ID)
	}

	listed, err := client.Session.List(ctx, defaultAgentID, SessionLookup{}, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if listed.Total != 1 {
		t.Fatalf("total = %d", listed.Total)
	}

	updated, err := client.Session.Update(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-123"}, "Renamed chat", map[string]string{"userId": "42"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Renamed chat" || updated.Labels["userId"] != "42" {
		t.Fatalf("unexpected update: %+v", updated)
	}
}

func TestMessageFlow(t *testing.T) {
	client, ctx := testClient(t)
	session, err := client.Session.Create(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-msg"}, "")
	if err != nil {
		t.Fatal(err)
	}

	message, err := client.Message.Create(ctx, defaultAgentID, session.ID, MessageInput{
		Role: MessageRoleUser, Content: "Hello",
		Meta: map[string]any{"source": "test", "channel": "web"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if message.Role != MessageRoleUser || message.Content != "Hello" {
		t.Fatalf("unexpected message: %+v", message)
	}
	if message.Meta["source"] != "test" {
		t.Fatalf("meta = %+v", message.Meta)
	}

	listed, err := client.Message.List(ctx, defaultAgentID, session.ID, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if listed.Total != 1 {
		t.Fatalf("total = %d", listed.Total)
	}

	fetched, err := client.Message.Get(ctx, defaultAgentID, session.ID, message.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.Content != "Hello" {
		t.Fatalf("content = %q", fetched.Content)
	}

	updated, err := client.Message.Update(ctx, defaultAgentID, session.ID, message.ID, "Updated", MessageRoleAssistant, map[string]any{
		"source": "test", "edited": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Content != "Updated" || updated.Role != MessageRoleAssistant {
		t.Fatalf("unexpected update: %+v", updated)
	}

	if err := client.Message.Delete(ctx, defaultAgentID, session.ID, message.ID); err != nil {
		t.Fatal(err)
	}
}

func TestMessageBatchCreate(t *testing.T) {
	client, ctx := testClient(t)
	session, err := client.Session.Create(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-msg-batch"}, "")
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.Message.CreateBatch(ctx, defaultAgentID, session.ID, []MessageInput{
		{Role: MessageRoleUser, Content: "Hello", Meta: map[string]any{"source": "batch", "channel": "web"}},
		{Role: MessageRoleAssistant, Content: "Hi there", Meta: map[string]any{"model": "gpt-4", "tokens": 12}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 2 {
		t.Fatalf("len = %d", len(created))
	}
	ids := []string{created[0].ID, created[1].ID}
	if err := client.Message.DeleteBatch(ctx, defaultAgentID, session.ID, ids); err != nil {
		t.Fatal(err)
	}
	listed, err := client.Message.List(ctx, defaultAgentID, session.ID, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if listed.Total != 0 {
		t.Fatalf("total = %d", listed.Total)
	}
}

func TestMessageSearch(t *testing.T) {
	client, ctx := testClient(t)
	session, err := client.Session.Create(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-msg-search"}, "")
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.Message.CreateBatch(ctx, defaultAgentID, session.ID, []MessageInput{
		{Role: MessageRoleUser, Content: "Let's revisit the pricing discussion."},
		{Role: MessageRoleAssistant, Content: "The trial starts next week."},
	})
	if err != nil {
		t.Fatal(err)
	}
	results, err := client.Message.Search(ctx, defaultAgentID, session.ID, "pricing discussion", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results.Results) != 1 || results.Results[0].ID != created[0].ID || results.Results[0].Score != 0.91 {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestMemoryFlow(t *testing.T) {
	client, ctx := testClient(t)
	session, err := client.Session.Create(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-mem"}, "")
	if err != nil {
		t.Fatal(err)
	}
	memory, err := client.Memory.Create(ctx, defaultAgentID, session.ID, MemoryInput{
		Kind: MemoryKindFact, Content: "User is in Cairo",
		Meta: map[string]any{"confidence": 0.9, "source": "onboarding"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if memory.Kind != MemoryKindFact || memory.Content != "User is in Cairo" {
		t.Fatalf("unexpected memory: %+v", memory)
	}

	listed, err := client.Memory.List(ctx, defaultAgentID, session.ID, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if listed.Total != 1 {
		t.Fatalf("total = %d", listed.Total)
	}

	fetched, err := client.Memory.Get(ctx, defaultAgentID, session.ID, memory.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.Content != "User is in Cairo" {
		t.Fatalf("content = %q", fetched.Content)
	}

	updated, err := client.Memory.Update(ctx, defaultAgentID, session.ID, memory.ID, "User is in Cairo, Egypt", "", map[string]any{
		"confidence": 0.95, "verified": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Content != "User is in Cairo, Egypt" {
		t.Fatalf("updated = %q", updated.Content)
	}
	if err := client.Memory.Delete(ctx, defaultAgentID, session.ID, memory.ID); err != nil {
		t.Fatal(err)
	}
}

func TestMemoryBatchCreate(t *testing.T) {
	client, ctx := testClient(t)
	session, err := client.Session.Create(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-mem-batch"}, "")
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.Memory.CreateBatch(ctx, defaultAgentID, session.ID, []MemoryInput{
		{Kind: MemoryKindFact, Content: "User prefers dark mode", Meta: map[string]any{"confidence": 0.95, "source": "onboarding"}},
		{Kind: MemoryKindSummary, Content: "Discussed billing setup"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 2 {
		t.Fatalf("len = %d", len(created))
	}
	ids := []string{created[0].ID, created[1].ID}
	if err := client.Memory.DeleteBatch(ctx, defaultAgentID, session.ID, ids); err != nil {
		t.Fatal(err)
	}
}

func TestMemorySearch(t *testing.T) {
	client, ctx := testClient(t)
	session, err := client.Session.Create(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-mem-search"}, "")
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.Memory.CreateBatch(ctx, defaultAgentID, session.ID, []MemoryInput{
		{Kind: MemoryKindPreference, Content: "User preferences include dark mode."},
		{Kind: MemoryKindFact, Content: "User is in Cairo."},
	})
	if err != nil {
		t.Fatal(err)
	}
	results, err := client.Memory.Search(ctx, defaultAgentID, session.ID, "user preferences", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results.Results) != 1 || results.Results[0].ID != created[0].ID || results.Results[0].Score != 0.88 {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestMetaHelpers(t *testing.T) {
	meta, ok := StringifyMeta(map[string]any{"source": "sdk"})
	if !ok || meta != `{"source":"sdk"}` {
		t.Fatalf("stringify = %q ok=%v", meta, ok)
	}
	if _, ok := StringifyMeta(nil); ok {
		t.Fatal("expected false for nil meta")
	}

	payload := BuildMessageBatchPayload([]MessageInput{
		{Role: MessageRoleUser, Content: "Hi", Meta: map[string]any{"source": "import"}},
	})
	messages := payload["messages"].([]map[string]string)
	if messages[0]["meta"] != `{"source":"import"}` {
		t.Fatalf("message meta = %q", messages[0]["meta"])
	}

	memPayload := BuildMemoryBatchPayload([]MemoryInput{
		{Kind: MemoryKindFact, Content: "Prefers dark mode", Meta: map[string]any{"source": "import"}},
	})
	memories := memPayload["memories"].([]map[string]string)
	if memories[0]["meta"] != `{"source":"import"}` {
		t.Fatalf("memory meta = %q", memories[0]["meta"])
	}
}

func TestSessionDelete(t *testing.T) {
	client, ctx := testClient(t)
	if _, err := client.Session.Create(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-del"}, ""); err != nil {
		t.Fatal(err)
	}
	if err := client.Session.Delete(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-del"}); err != nil {
		t.Fatal(err)
	}
	_, err := client.Session.GetByLabels(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-del"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
}

func TestKnowledgeUploadDictLabels(t *testing.T) {
	client, ctx := testClient(t)
	uploaded, err := client.Knowledge.Upload(ctx, FileBytes{
		Filename: "policy.md", Content: []byte("# Refund policy"), ContentType: "text/markdown",
	}, "Refund policy", map[string]string{"team": "support", "category": "policy"})
	if err != nil {
		t.Fatal(err)
	}
	if len(uploaded.Labels) != 2 {
		t.Fatalf("labels = %+v", uploaded.Labels)
	}
	seen := map[string]bool{}
	for _, label := range uploaded.Labels {
		seen[label] = true
	}
	if !seen["team=support"] || !seen["category=policy"] {
		t.Fatalf("labels = %+v", uploaded.Labels)
	}
	_ = client.Knowledge.Delete(ctx, uploaded.ID)
}
