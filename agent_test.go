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

func TestAgent(t *testing.T) {
	t.Run("list and get", func(t *testing.T) {
		client, ctx := testClient(t)
		agents, err := client.Agent.List(ctx, 50, 0)
		require.NoError(t, err)
		require.GreaterOrEqual(t, agents.Total, 1)
		assert.Equal(t, "Support bot", agents.Agents[0].Name)

		agent, err := client.Agent.Get(ctx, defaultAgentID)
		require.NoError(t, err)
		assert.Equal(t, defaultAgentID, agent.ID)
		assert.Equal(t, "unmanaged", agent.Kind)
	})

	t.Run("create update delete", func(t *testing.T) {
		client, ctx := testClient(t)
		created, err := client.Agent.Create(ctx, "Bot", "Test bot")
		require.NoError(t, err)

		updated, err := client.Agent.Update(ctx, created.ID, "Renamed bot", "Updated description")
		require.NoError(t, err)
		assert.Equal(t, "Renamed bot", updated.Name)

		require.NoError(t, client.Agent.Delete(ctx, created.ID))
	})
}

var agentPathRE = regexp.MustCompile(`^/api/v1/workspaces/([^/]+)/agents/([^/]+)(/.*)?$`)

func (ms *mockServer) agentObject(agentID, name, description string) map[string]any {
	return map[string]any{
		"id": agentID, "workspaceId": defaultWorkspaceID, "name": name, "kind": "unmanaged",
		"promptId": nil, "kbLabels": map[string]string{}, "handle": shortID(),
		"description": description, "status": "active",
		"createdAt": mockTimestamp, "updatedAt": mockTimestamp,
	}
}

func (ms *mockServer) agentCollectionRoute(method, path string, query url.Values, body map[string]any, r *http.Request, wsPrefix string) (int, any) {
	if path != wsPrefix+"/agents" {
		return 404, map[string]any{"errorMessage": "not found"}
	}
	if method == http.MethodGet {
		if !ms.authorized(r) {
			return 401, map[string]any{"errorMessage": "Invalid access key."}
		}
		agents := values(ms.store.agents)
		return 200, map[string]any{"agents": agents, "_meta": mockListMeta(len(agents), query)}
	}
	if method == http.MethodPost {
		if !ms.authorized(r) {
			return 403, map[string]any{"errorMessage": "Write requires user API key."}
		}
		agentID := "agt_" + shortID()
		agent := ms.agentObject(agentID, stringify(body["name"]), stringify(body["description"]))
		ms.store.agents[agentID] = agent
		return 201, agent
	}
	return 404, map[string]any{"errorMessage": "not found"}
}

func (ms *mockServer) agentRoute(method, path string, query url.Values, body map[string]any, r *http.Request) (int, any) {
	match := agentPathRE.FindStringSubmatch(path)
	if match == nil {
		return 404, map[string]any{"errorMessage": "not found"}
	}
	workspaceId, agentID, suffix := match[1], match[2], match[3]
	if workspaceId != defaultWorkspaceID {
		return 404, map[string]any{"errorMessage": "workspace not found"}
	}

	if suffix == "" {
		agent, ok := ms.store.agents[agentID]
		if !ok {
			return 404, map[string]any{"errorMessage": "Agent not found."}
		}
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			return 200, agent
		}
		if method == http.MethodPut {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			agent["name"] = body["name"]
			agent["description"] = body["description"]
			agent["updatedAt"] = mockTimestamp
			return 200, agent
		}
		if method == http.MethodDelete {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			delete(ms.store.agents, agentID)
			return 204, nil
		}
	}

	if strings.HasPrefix(suffix, "/sessions") {
		if strings.Contains(suffix, "/messages") || strings.Contains(suffix, "/memories") {
			return ms.sessionNestedRoute(method, suffix, body, r)
		}
		return ms.sessionRoute(method, suffix, query, body, r, workspaceId, agentID)
	}
	return 404, map[string]any{"errorMessage": "not found"}
}
