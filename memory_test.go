// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemory(t *testing.T) {
	t.Run("flow", func(t *testing.T) {
		client, ctx := testClient(t)
		session, err := client.Session.Create(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-mem"}, "")
		require.NoError(t, err)

		memory, err := client.Memory.Create(ctx, defaultAgentID, session.ID, MemoryInput{
			Kind: MemoryKindFact, Content: "User is in Cairo",
			Meta: map[string]any{"confidence": 0.9, "source": "onboarding"},
		})
		require.NoError(t, err)
		assert.Equal(t, MemoryKindFact, memory.Kind)
		assert.Equal(t, "User is in Cairo", memory.Content)

		listed, err := client.Memory.List(ctx, defaultAgentID, session.ID, 50, 0)
		require.NoError(t, err)
		assert.Equal(t, 1, listed.Total)

		fetched, err := client.Memory.Get(ctx, defaultAgentID, session.ID, memory.ID)
		require.NoError(t, err)
		assert.Equal(t, "User is in Cairo", fetched.Content)

		updated, err := client.Memory.Update(ctx, defaultAgentID, session.ID, memory.ID, "User is in Cairo, Egypt", "", map[string]any{
			"confidence": 0.95, "verified": true,
		})
		require.NoError(t, err)
		assert.Equal(t, "User is in Cairo, Egypt", updated.Content)

		require.NoError(t, client.Memory.Delete(ctx, defaultAgentID, session.ID, memory.ID))
	})

	t.Run("batch create", func(t *testing.T) {
		client, ctx := testClient(t)
		session, err := client.Session.Create(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-mem-batch"}, "")
		require.NoError(t, err)

		created, err := client.Memory.CreateBatch(ctx, defaultAgentID, session.ID, []MemoryInput{
			{Kind: MemoryKindFact, Content: "User prefers dark mode", Meta: map[string]any{"confidence": 0.95, "source": "onboarding"}},
			{Kind: MemoryKindSummary, Content: "Discussed billing setup"},
		})
		require.NoError(t, err)
		assert.Len(t, created, 2)

		ids := []string{created[0].ID, created[1].ID}
		require.NoError(t, client.Memory.DeleteBatch(ctx, defaultAgentID, session.ID, ids))
	})

	t.Run("search", func(t *testing.T) {
		client, ctx := testClient(t)
		session, err := client.Session.Create(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-mem-search"}, "")
		require.NoError(t, err)

		created, err := client.Memory.CreateBatch(ctx, defaultAgentID, session.ID, []MemoryInput{
			{Kind: MemoryKindPreference, Content: "User preferences include dark mode."},
			{Kind: MemoryKindFact, Content: "User is in Cairo."},
		})
		require.NoError(t, err)

		results, err := client.Memory.Search(ctx, defaultAgentID, session.ID, "user preferences", 10)
		require.NoError(t, err)
		require.Len(t, results.Results, 1)
		assert.Equal(t, created[0].ID, results.Results[0].ID)
		assert.Equal(t, 0.88, results.Results[0].Score)
	})
}

func (ms *mockServer) memoryRoute(method, suffix string, body map[string]any, r *http.Request) (int, any) {
	if m := regexp.MustCompile(`^/sessions/([^/]+)/memories/batch$`).FindStringSubmatch(suffix); m != nil {
		sessionID := m[1]
		if _, ok := ms.store.sessions[sessionID]; !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		if method == http.MethodPost {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			created := []map[string]any{}
			items, _ := body["memories"].([]any)
			for _, raw := range items {
				item, _ := raw.(map[string]any)
				memory := map[string]any{
					"id": "mem_" + shortID(), "sessionId": sessionID, "kind": item["kind"],
					"content": item["content"], "meta": parseMeta(item["meta"]),
					"createdAt": mockTimestamp, "updatedAt": mockTimestamp,
				}
				ms.store.memories[sessionID] = append(ms.store.memories[sessionID], memory)
				created = append(created, memory)
			}
			return 201, map[string]any{"memories": created}
		}
		if method == http.MethodDelete {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			ids := map[string]struct{}{}
			for _, raw := range body["ids"].([]any) {
				ids[stringify(raw)] = struct{}{}
			}
			filtered := []map[string]any{}
			for _, mem := range ms.store.memories[sessionID] {
				if _, drop := ids[stringify(mem["id"])]; !drop {
					filtered = append(filtered, mem)
				}
			}
			ms.store.memories[sessionID] = filtered
			return 204, nil
		}
	}

	if m := regexp.MustCompile(`^/sessions/([^/]+)/memories/search$`).FindStringSubmatch(suffix); m != nil && method == http.MethodPost {
		if !ms.authorized(r) {
			return 401, map[string]any{"errorMessage": "Invalid access key."}
		}
		sessionID := m[1]
		if _, ok := ms.store.sessions[sessionID]; !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		queryText := stringify(body["query"])
		limit := asInt(body["limit"])
		if limit == 0 {
			limit = 10
		}
		if queryText == "" || limit < 1 || limit > 100 {
			return 400, map[string]any{"errorMessage": "Invalid search request."}
		}
		matches := []map[string]any{}
		q := strings.ToLower(queryText)
		for _, item := range ms.store.memories[sessionID] {
			if strings.Contains(strings.ToLower(stringify(item["content"])), q) {
				matches = append(matches, map[string]any{
					"id": item["id"], "kind": item["kind"], "score": 0.88, "text": item["content"],
				})
			}
		}
		if len(matches) > limit {
			matches = matches[:limit]
		}
		return 200, map[string]any{"results": matches}
	}

	if m := regexp.MustCompile(`^/sessions/([^/]+)/memories$`).FindStringSubmatch(suffix); m != nil {
		sessionID := m[1]
		if _, ok := ms.store.sessions[sessionID]; !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			items := ms.store.memories[sessionID]
			return 200, map[string]any{"memories": items, "_meta": mockListMeta(len(items), url.Values{})}
		}
		if method == http.MethodPost {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			memory := map[string]any{
				"id": "mem_" + shortID(), "sessionId": sessionID, "kind": body["kind"],
				"content": body["content"], "meta": parseMeta(body["meta"]),
				"createdAt": mockTimestamp, "updatedAt": mockTimestamp,
			}
			ms.store.memories[sessionID] = append(ms.store.memories[sessionID], memory)
			return 201, memory
		}
	}

	if m := regexp.MustCompile(`^/sessions/([^/]+)/memories/([^/]+)$`).FindStringSubmatch(suffix); m != nil {
		sessionID, memoryID := m[1], m[2]
		if _, ok := ms.store.sessions[sessionID]; !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		items := ms.store.memories[sessionID]
		var memory map[string]any
		idx := -1
		for i, item := range items {
			if stringify(item["id"]) == memoryID {
				memory = item
				idx = i
				break
			}
		}
		if memory == nil {
			return 404, map[string]any{"errorMessage": "Memory not found."}
		}
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			return 200, memory
		}
		if method == http.MethodPut {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			if body["kind"] != nil {
				memory["kind"] = body["kind"]
			}
			memory["content"] = body["content"]
			if body["meta"] != nil {
				memory["meta"] = parseMeta(body["meta"])
			}
			memory["updatedAt"] = mockTimestamp
			return 200, memory
		}
		if method == http.MethodDelete {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			ms.store.memories[sessionID] = append(items[:idx], items[idx+1:]...)
			return 204, nil
		}
	}

	return 404, map[string]any{"errorMessage": "not found"}
	return 404, map[string]any{"errorMessage": "not found"}
}
