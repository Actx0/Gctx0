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

func TestMessage(t *testing.T) {
	t.Run("flow", func(t *testing.T) {
		client, ctx := testClient(t)
		session, err := client.Session.Create(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-msg"}, "")
		require.NoError(t, err)

		message, err := client.Message.Create(ctx, defaultAgentID, session.ID, MessageInput{
			Role: MessageRoleUser, Content: "Hello",
			Meta: map[string]any{"source": "test", "channel": "web"},
		})
		require.NoError(t, err)
		assert.Equal(t, MessageRoleUser, message.Role)
		assert.Equal(t, "Hello", message.Content)
		assert.Equal(t, "test", message.Meta["source"])

		listed, err := client.Message.List(ctx, defaultAgentID, session.ID, 50, 0)
		require.NoError(t, err)
		assert.Equal(t, 1, listed.Total)

		fetched, err := client.Message.Get(ctx, defaultAgentID, session.ID, message.ID)
		require.NoError(t, err)
		assert.Equal(t, "Hello", fetched.Content)

		updated, err := client.Message.Update(ctx, defaultAgentID, session.ID, message.ID, "Updated", MessageRoleAssistant, map[string]any{
			"source": "test", "edited": true,
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated", updated.Content)
		assert.Equal(t, MessageRoleAssistant, updated.Role)

		require.NoError(t, client.Message.Delete(ctx, defaultAgentID, session.ID, message.ID))
	})

	t.Run("batch create", func(t *testing.T) {
		client, ctx := testClient(t)
		session, err := client.Session.Create(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-msg-batch"}, "")
		require.NoError(t, err)

		created, err := client.Message.CreateBatch(ctx, defaultAgentID, session.ID, []MessageInput{
			{Role: MessageRoleUser, Content: "Hello", Meta: map[string]any{"source": "batch", "channel": "web"}},
			{Role: MessageRoleAssistant, Content: "Hi there", Meta: map[string]any{"model": "gpt-4", "tokens": 12}},
		})
		require.NoError(t, err)
		assert.Len(t, created, 2)

		ids := []string{created[0].ID, created[1].ID}
		require.NoError(t, client.Message.DeleteBatch(ctx, defaultAgentID, session.ID, ids))

		listed, err := client.Message.List(ctx, defaultAgentID, session.ID, 50, 0)
		require.NoError(t, err)
		assert.Equal(t, 0, listed.Total)
	})

	t.Run("search", func(t *testing.T) {
		client, ctx := testClient(t)
		session, err := client.Session.Create(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-msg-search"}, "")
		require.NoError(t, err)

		created, err := client.Message.CreateBatch(ctx, defaultAgentID, session.ID, []MessageInput{
			{Role: MessageRoleUser, Content: "Let's revisit the pricing discussion."},
			{Role: MessageRoleAssistant, Content: "The trial starts next week."},
		})
		require.NoError(t, err)

		results, err := client.Message.Search(ctx, defaultAgentID, session.ID, "pricing discussion", 1)
		require.NoError(t, err)
		require.Len(t, results.Results, 1)
		assert.Equal(t, created[0].ID, results.Results[0].ID)
		assert.Equal(t, 0.91, results.Results[0].Score)
	})
}

func (ms *mockServer) messageRoute(method, suffix string, body map[string]any, r *http.Request) (int, any) {
	if m := regexp.MustCompile(`^/sessions/([^/]+)/messages/batch$`).FindStringSubmatch(suffix); m != nil {
		sessionID := m[1]
		if _, ok := ms.store.sessions[sessionID]; !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		if method == http.MethodPost {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			created := []map[string]any{}
			items, _ := body["messages"].([]any)
			for _, raw := range items {
				item, _ := raw.(map[string]any)
				message := map[string]any{
					"id": "msg_" + shortID(), "sessionId": sessionID, "role": item["role"],
					"content": item["content"], "meta": parseMeta(item["meta"]), "createdAt": mockTimestamp,
				}
				ms.store.messages[sessionID] = append(ms.store.messages[sessionID], message)
				created = append(created, message)
			}
			return 201, map[string]any{"messages": created}
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
			for _, msg := range ms.store.messages[sessionID] {
				if _, drop := ids[stringify(msg["id"])]; !drop {
					filtered = append(filtered, msg)
				}
			}
			ms.store.messages[sessionID] = filtered
			return 204, nil
		}
	}

	if m := regexp.MustCompile(`^/sessions/([^/]+)/messages/search$`).FindStringSubmatch(suffix); m != nil && method == http.MethodPost {
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
		for _, item := range ms.store.messages[sessionID] {
			if strings.Contains(strings.ToLower(stringify(item["content"])), q) {
				matches = append(matches, map[string]any{
					"id": item["id"], "role": item["role"], "score": 0.91, "text": item["content"],
				})
			}
		}
		if len(matches) > limit {
			matches = matches[:limit]
		}
		return 200, map[string]any{"results": matches}
	}

	if m := regexp.MustCompile(`^/sessions/([^/]+)/messages$`).FindStringSubmatch(suffix); m != nil {
		sessionID := m[1]
		if _, ok := ms.store.sessions[sessionID]; !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			items := ms.store.messages[sessionID]
			return 200, map[string]any{"messages": items, "_meta": mockListMeta(len(items), url.Values{})}
		}
		if method == http.MethodPost {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			message := map[string]any{
				"id": "msg_" + shortID(), "sessionId": sessionID, "role": body["role"],
				"content": body["content"], "meta": parseMeta(body["meta"]), "createdAt": mockTimestamp,
			}
			ms.store.messages[sessionID] = append(ms.store.messages[sessionID], message)
			return 201, message
		}
	}

	if m := regexp.MustCompile(`^/sessions/([^/]+)/messages/([^/]+)$`).FindStringSubmatch(suffix); m != nil {
		sessionID, messageID := m[1], m[2]
		if _, ok := ms.store.sessions[sessionID]; !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		items := ms.store.messages[sessionID]
		var message map[string]any
		idx := -1
		for i, item := range items {
			if stringify(item["id"]) == messageID {
				message = item
				idx = i
				break
			}
		}
		if message == nil {
			return 404, map[string]any{"errorMessage": "Message not found."}
		}
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			return 200, message
		}
		if method == http.MethodPut {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			if body["role"] != nil {
				message["role"] = body["role"]
			}
			message["content"] = body["content"]
			if body["meta"] != nil {
				message["meta"] = parseMeta(body["meta"])
			}
			return 200, message
		}
		if method == http.MethodDelete {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			ms.store.messages[sessionID] = append(items[:idx], items[idx+1:]...)
			return 204, nil
		}
	}
	return 404, map[string]any{"errorMessage": "not found"}
}
