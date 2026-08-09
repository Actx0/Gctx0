// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const agentsURL = "https://example.test/api/v1/workspaces/ws_test/agents"

func TestUnitAgent(t *testing.T) {
	t.Run("List", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			`=~^`+agentsURL+`(?:\?.*)?\z`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"agents": []map[string]any{{"id": "ag_1", "name": "Support"}},
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
	})

	t.Run("Get", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			agentsURL+"/ag_1",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"id": "ag_1", "name": "Support"}),
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
	})

	t.Run("Create", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPost,
			agentsURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"id": "ag_1", "name": "Support"}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Agent.Create(
			context.Background(),
			"Support",
			"desc",
		)
		require.NoError(t, err)
		assert.Equal(t, "ag_1", got.ID)
	})

	t.Run("Update", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPut,
			agentsURL+"/ag_1",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"id": "ag_1", "name": "Renamed"}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Agent.Update(
			context.Background(),
			"ag_1",
			"Renamed",
			"desc",
		)
		require.NoError(t, err)
		assert.Equal(t, "Renamed", got.Name)
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
