// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSession(t *testing.T) {
	t.Run("flow", func(t *testing.T) {
		client, ctx := testClient(t)
		created, err := client.Session.Create(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-123"}, "Support chat")
		require.NoError(t, err)
		assert.Equal(t, "thread-123", created.ExternalID)
		assert.Equal(t, "Support chat", created.Title)

		fetched, err := client.Session.Get(ctx, defaultAgentID, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, fetched.ID)

		byLabels, err := client.Session.GetByLabels(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-123"})
		require.NoError(t, err)
		assert.Equal(t, created.ID, byLabels.ID)

		listed, err := client.Session.List(ctx, defaultAgentID, SessionLookup{}, 50, 0)
		require.NoError(t, err)
		assert.Equal(t, 1, listed.Total)

		updated, err := client.Session.Update(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-123"}, "Renamed chat", map[string]string{"userId": "42"})
		require.NoError(t, err)
		assert.Equal(t, "Renamed chat", updated.Title)
		assert.Equal(t, "42", updated.Labels["userId"])
	})

	t.Run("delete", func(t *testing.T) {
		client, ctx := testClient(t)
		_, err := client.Session.Create(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-del"}, "")
		require.NoError(t, err)
		require.NoError(t, client.Session.Delete(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-del"}))

		_, err = client.Session.GetByLabels(ctx, defaultAgentID, SessionLookup{ExternalID: "thread-del"})
		var apiErr *APIError
		assert.True(t, errors.As(err, &apiErr))
	})
}

func (ms *mockServer) sessionRoute(method, suffix string, query url.Values, body map[string]any, r *http.Request, workspaceId, agentID string) (int, any) {
	if suffix == "/sessions" {
		if method == http.MethodPost {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			externalID := query.Get("id")
			labels := queryLabels(query)
			if externalID == "" && len(labels) == 0 {
				return 400, map[string]any{"errorMessage": "id or labels required"}
			}
			if externalID != "" && ms.findSessionByLabels(url.Values{"id": {externalID}}) != nil {
				return 409, map[string]any{"errorMessage": "Session already exists."}
			}
			sessionID := "ses_" + shortID()
			ext := externalID
			if ext == "" {
				ext = randomID()
			}
			title := ""
			if body["title"] != nil {
				title = stringify(body["title"])
			}
			session := map[string]any{
				"id": sessionID, "externalId": ext, "workspaceId": workspaceId, "agentId": agentID,
				"title": title, "status": "active", "labels": labels, "meta": map[string]any{},
				"createdAt": mockTimestamp, "updatedAt": mockTimestamp,
			}
			ms.store.sessions[sessionID] = session
			ms.store.messages[sessionID] = []map[string]any{}
			ms.store.memories[sessionID] = []map[string]any{}
			return 201, session
		}
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			sessions := []map[string]any{}
			for _, s := range ms.store.sessions {
				if stringify(s["agentId"]) != agentID {
					continue
				}
				sessions = append(sessions, s)
			}
			if query.Get("id") != "" {
				filtered := []map[string]any{}
				for _, s := range sessions {
					if stringify(s["externalId"]) == query.Get("id") {
						filtered = append(filtered, s)
					}
				}
				sessions = filtered
			}
			labels := queryLabels(query)
			if len(labels) > 0 {
				filtered := []map[string]any{}
				for _, s := range sessions {
					if labelsEqual(s["labels"], labels) {
						filtered = append(filtered, s)
					}
				}
				sessions = filtered
			}
			return 200, map[string]any{"sessions": sessions, "_meta": mockListMeta(len(sessions), query)}
		}
	}

	if suffix == "/sessions/by-labels" {
		if !ms.authorized(r) {
			return 401, map[string]any{"errorMessage": "Invalid access key."}
		}
		session := ms.findSessionByLabels(query)
		if session == nil {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		if method == http.MethodGet {
			return 200, session
		}
		if method == http.MethodPut {
			if body["title"] != nil {
				session["title"] = body["title"]
			}
			if body["labels"] != nil {
				session["labels"] = body["labels"]
			}
			session["updatedAt"] = mockTimestamp
			return 200, session
		}
		if method == http.MethodDelete {
			id := stringify(session["id"])
			delete(ms.store.sessions, id)
			delete(ms.store.messages, id)
			delete(ms.store.memories, id)
			return 204, nil
		}
	}

	if m := regexp.MustCompile(`^/sessions/([^/]+)$`).FindStringSubmatch(suffix); m != nil && method == http.MethodGet {
		if !ms.authorized(r) {
			return 401, map[string]any{"errorMessage": "Invalid access key."}
		}
		session, ok := ms.store.sessions[m[1]]
		if !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		return 200, session
	}

	return 404, map[string]any{"errorMessage": "not found"}
}

func (ms *mockServer) sessionNestedRoute(method, suffix string, body map[string]any, r *http.Request) (int, any) {
	switch {
	case strings.Contains(suffix, "/messages"):
		return ms.messageRoute(method, suffix, body, r)
	case strings.Contains(suffix, "/memories"):
		return ms.memoryRoute(method, suffix, body, r)
	default:
		return 404, map[string]any{"errorMessage": "not found"}
	}
}
