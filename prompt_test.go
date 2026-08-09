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

const (
	promptsURL       = "https://example.test/api/v1/workspaces/ws_test/prompts"
	promptsByNameURL = "https://example.test/api/v1/workspaces/ws_test/promptsByName"
)

func TestUnitPrompt(t *testing.T) {
	t.Run("List", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			`=~^`+promptsURL+`(?:\?.*)?\z`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"prompts": []map[string]any{{"promptId": "prm_1", "handle": "support"}},
				"_meta":   map[string]any{"limit": 50, "offset": 0, "total": 1},
			}),
		)

		listed, err := NewTestClient(
			t,
			httpClient,
		).Prompt.List(
			context.Background(),
			50,
			0,
		)
		require.NoError(t, err)
		require.Len(t, listed.Prompts, 1)
		assert.Equal(t, "prm_1", listed.Prompts[0].PromptID)
	})

	t.Run("Create", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPost,
			promptsURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"promptId": "prm_1", "handle": "support"}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Prompt.Create(
			context.Background(),
			"Support",
			PromptTypeText,
			"Hi",
			PromptWriteOptions{},
		)
		require.NoError(t, err)
		assert.Equal(t, "prm_1", got.PromptID)
	})

	t.Run("Get", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			promptsURL+"/prm_1",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"promptId": "prm_1", "handle": "support"}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Prompt.Get(
			context.Background(),
			"prm_1",
		)
		require.NoError(t, err)
		assert.Equal(t, "support", got.Handle)
	})

	t.Run("Delete", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodDelete,
			promptsURL+"/prm_1",
			httpmock.NewStringResponder(204, ""),
		)

		err := NewTestClient(
			t,
			httpClient,
		).Prompt.Delete(
			context.Background(),
			"prm_1",
		)
		require.NoError(t, err)
	})

	t.Run("GetByName", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			`=~^`+promptsByNameURL+`/support(?:\?.*)?\z`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"id":      "prv_1",
				"handle":  "support",
				"version": 2,
				"content": "Hello {{name}}",
			}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Prompt.GetByName(
			context.Background(),
			"support",
			"",
		)
		require.NoError(t, err)
		assert.Equal(t, 2, got.Version)
	})

	t.Run("ListVersions", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			`=~^`+promptsURL+`/prm_1/versions(?:\?.*)?\z`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"versions": []map[string]any{{"id": "prv_1", "version": 1, "content": "v1"}},
				"_meta":    map[string]any{"limit": 50, "offset": 0, "total": 1},
			}),
		)

		listed, err := NewTestClient(
			t,
			httpClient,
		).Prompt.ListVersions(
			context.Background(),
			"prm_1",
			50,
			0,
		)
		require.NoError(t, err)
		require.Len(t, listed.Versions, 1)
		assert.Equal(t, "prv_1", listed.Versions[0].ID)
	})

	t.Run("CreateVersion", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPost,
			promptsURL+"/prm_1/versions",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"id": "prv_2", "version": 2, "content": "v2"}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Prompt.CreateVersion(
			context.Background(),
			"prm_1",
			PromptTypeText,
			"v2",
			PromptWriteOptions{},
		)
		require.NoError(t, err)
		assert.Equal(t, 2, got.Version)
	})

	t.Run("GetVersion", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			promptsURL+"/prm_1/versions/prv_1",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"id": "prv_1", "version": 1, "content": "v1"}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Prompt.GetVersion(
			context.Background(),
			"prm_1",
			"prv_1",
		)
		require.NoError(t, err)
		assert.Equal(t, "v1", got.Content)
	})

	t.Run("UpdateVersion", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPut,
			promptsURL+"/prm_1/versions/prv_1",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"id": "prv_1", "content": "updated"}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Prompt.UpdateVersion(
			context.Background(),
			"prm_1",
			"prv_1",
			"updated",
			PromptWriteOptions{},
		)
		require.NoError(t, err)
		assert.Equal(t, "updated", got.Content)
	})

	t.Run("DeleteVersion", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodDelete,
			promptsURL+"/prm_1/versions/prv_1",
			httpmock.NewStringResponder(204, ""),
		)

		err := NewTestClient(
			t,
			httpClient,
		).Prompt.DeleteVersion(
			context.Background(),
			"prm_1",
			"prv_1",
		)
		require.NoError(t, err)
	})

	t.Run("Compile", func(t *testing.T) {
		got, err := Prompt{
			Content: "Hello {{name}}",
		}.Compile(
			map[string]string{"name": "Ada"},
		)
		require.NoError(t, err)
		assert.Equal(t, "Hello Ada", got)
	})
}
