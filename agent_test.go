// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const agentsURL = "https://example.test/api/v1/workspaces/ws_test/agents"

func sampleAgent(id, name string, memoryPipeline bool) map[string]any {
	return map[string]any{
		"id":          id,
		"workspaceId": "ws_test",
		"name":        name,
		"kind":        "unmanaged",
		"promptId":    nil,
		"kbLabels":    map[string]any{},
		"handle":      "abcd-1234",
		"description": "desc",
		"status":      "active",
		"configs":     map[string]any{"memoryPipeline": memoryPipeline},
		"createdAt":   "2026-08-10T18:00:00Z",
		"updatedAt":   "2026-08-10T18:00:00Z",
	}
}

func TestUnitAgent(t *testing.T) {
	t.Run("List", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			`=~^`+agentsURL+`(?:\?.*)?\z`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"agents": []map[string]any{sampleAgent("ag_1", "Support", false)},
				"_meta":  map[string]any{"limit": 50, "offset": 0, "total": 1},
			}),
		)

		listed, err := NewTestClient(
			t,
			httpClient,
		).Agent.List(
			context.Background(),
			50,
			0,
		)
		require.NoError(t, err)
		require.Len(t, listed.Agents, 1)
		assert.Equal(t, "ag_1", listed.Agents[0].ID)
		assert.False(t, listed.Agents[0].Configs.MemoryPipeline)
	})

	t.Run("Get", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			agentsURL+"/ag_1",
			httpmock.NewJsonResponderOrPanic(200, sampleAgent("ag_1", "Support", true)),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Agent.Get(
			context.Background(),
			"ag_1",
		)
		require.NoError(t, err)
		assert.Equal(t, "Support", got.Name)
		assert.True(t, got.Configs.MemoryPipeline)
	})

	t.Run("Create", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPost,
			agentsURL,
			func(req *http.Request) (*http.Response, error) {
				var body map[string]any
				require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
				assert.Equal(t, "Support", body["name"])
				assert.Equal(t, "desc", body["description"])
				_, hasConfigs := body["configs"]
				assert.False(t, hasConfigs)
				return httpmock.NewJsonResponse(201, sampleAgent("ag_1", "Support", false))
			},
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Agent.Create(
			context.Background(),
			"Support",
			"desc",
			AgentWriteOptions{},
		)
		require.NoError(t, err)
		assert.Equal(t, "ag_1", got.ID)
		assert.False(t, got.Configs.MemoryPipeline)
	})

	t.Run("CreateWithMemoryPipeline", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPost,
			agentsURL,
			func(req *http.Request) (*http.Response, error) {
				var body map[string]any
				require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
				configs, ok := body["configs"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, true, configs["memoryPipeline"])
				return httpmock.NewJsonResponse(201, sampleAgent("ag_2", "Pipeline", true))
			},
		)

		enabled := true
		got, err := NewTestClient(
			t,
			httpClient,
		).Agent.Create(
			context.Background(),
			"Pipeline",
			"desc",
			AgentWriteOptions{MemoryPipeline: &enabled},
		)
		require.NoError(t, err)
		assert.True(t, got.Configs.MemoryPipeline)
	})

	t.Run("Update", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPut,
			agentsURL+"/ag_1",
			func(req *http.Request) (*http.Response, error) {
				var body map[string]any
				require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
				assert.Equal(t, "Renamed", body["name"])
				configs, ok := body["configs"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, false, configs["memoryPipeline"])
				return httpmock.NewJsonResponse(200, sampleAgent("ag_1", "Renamed", false))
			},
		)

		disabled := false
		got, err := NewTestClient(
			t,
			httpClient,
		).Agent.Update(
			context.Background(),
			"ag_1",
			"Renamed",
			"desc",
			AgentWriteOptions{MemoryPipeline: &disabled},
		)
		require.NoError(t, err)
		assert.Equal(t, "Renamed", got.Name)
		assert.False(t, got.Configs.MemoryPipeline)
	})

	t.Run("Delete", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodDelete,
			agentsURL+"/ag_1",
			httpmock.NewStringResponder(204, ""),
		)

		err := NewTestClient(
			t,
			httpClient,
		).Agent.Delete(
			context.Background(),
			"ag_1",
		)
		require.NoError(t, err)
	})
}

func TestUnitAgentWriteBody(t *testing.T) {
	t.Run("omits configs by default", func(t *testing.T) {
		got := AgentWriteBody("Bot", "desc", AgentWriteOptions{})
		assert.Equal(t, map[string]any{
			"name":        "Bot",
			"description": "desc",
		}, got)
	})

	t.Run("includes nested memoryPipeline", func(t *testing.T) {
		enabled := true
		got := AgentWriteBody("Bot", "desc", AgentWriteOptions{MemoryPipeline: &enabled})
		assert.Equal(t, map[string]any{
			"name":        "Bot",
			"description": "desc",
			"configs": map[string]any{
				"memoryPipeline": true,
			},
		}, got)
	})
}
