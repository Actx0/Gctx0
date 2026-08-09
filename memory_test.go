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

const memoriesURL = "https://example.test/api/v1/workspaces/ws_test/agents/ag_1/sessions/ses_1/memories"

func TestUnitMemory(t *testing.T) {
	t.Run("List", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			`=~^`+memoriesURL+`(?:\?.*)?\z`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"memories": []map[string]any{{"id": "mem_1", "kind": "fact", "content": "Prefers dark mode"}},
				"_meta":    map[string]any{"limit": 50, "offset": 0, "total": 1},
			}),
		)

		listed, err := NewTestClient(
			t,
			httpClient,
		).Memory.List(
			context.Background(),
			"ag_1",
			"ses_1",
			50,
			0,
		)
		require.NoError(t, err)
		require.Len(t, listed.Memories, 1)
		assert.Equal(t, "mem_1", listed.Memories[0].ID)
	})

	t.Run("Get", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			memoriesURL+"/mem_1",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"id": "mem_1", "content": "Prefers dark mode"}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Memory.Get(
			context.Background(),
			"ag_1",
			"ses_1",
			"mem_1",
		)
		require.NoError(t, err)
		assert.Equal(t, "Prefers dark mode", got.Content)
	})

	t.Run("Search", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPost,
			memoriesURL+"/search",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"results": []map[string]any{{"id": "mem_1", "text": "Prefers dark mode", "score": 0.9}},
			}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Memory.Search(
			context.Background(),
			"ag_1",
			"ses_1",
			"dark",
			10,
		)
		require.NoError(t, err)
		require.Len(t, got.Results, 1)
		assert.Equal(t, "mem_1", got.Results[0].ID)
	})

	t.Run("Create", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPost,
			memoriesURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"id": "mem_1", "kind": "fact"}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Memory.Create(
			context.Background(),
			"ag_1",
			"ses_1",
			MemoryInput{
				Kind:    MemoryKindFact,
				Content: "Prefers dark mode",
			},
		)
		require.NoError(t, err)
		assert.Equal(t, "mem_1", got.ID)
	})

	t.Run("CreateBatch", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPost,
			memoriesURL+"/batch",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"memories": []map[string]any{{"id": "mem_1"}, {"id": "mem_2"}},
			}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Memory.CreateBatch(
			context.Background(),
			"ag_1",
			"ses_1",
			[]MemoryInput{
				{Kind: MemoryKindFact, Content: "A"},
				{Kind: MemoryKindPreference, Content: "B"},
			},
		)
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("Update", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPut,
			memoriesURL+"/mem_1",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"id": "mem_1", "content": "Updated"}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Memory.Update(
			context.Background(),
			"ag_1",
			"ses_1",
			"mem_1",
			"Updated",
			MemoryKindFact,
			nil,
		)
		require.NoError(t, err)
		assert.Equal(t, "Updated", got.Content)
	})

	t.Run("Delete", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodDelete,
			memoriesURL+"/mem_1",
			httpmock.NewStringResponder(204, ""),
		)

		err := NewTestClient(
			t,
			httpClient,
		).Memory.Delete(
			context.Background(),
			"ag_1",
			"ses_1",
			"mem_1",
		)
		require.NoError(t, err)
	})

	t.Run("DeleteBatch", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodDelete,
			memoriesURL+"/batch",
			httpmock.NewStringResponder(204, ""),
		)

		err := NewTestClient(
			t,
			httpClient,
		).Memory.DeleteBatch(
			context.Background(),
			"ag_1",
			"ses_1",
			[]string{"mem_1", "mem_2"},
		)
		require.NoError(t, err)
	})
}
