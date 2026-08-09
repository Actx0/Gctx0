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

const messagesURL = "https://example.test/api/v1/workspaces/ws_test/agents/ag_1/sessions/ses_1/messages"

func TestUnitMessage(t *testing.T) {
	t.Run("List", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			`=~^`+messagesURL+`(?:\?.*)?\z`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"messages": []map[string]any{{"id": "msg_1", "content": "Hi"}},
				"_meta":    map[string]any{"limit": 50, "offset": 0, "total": 1},
			}),
		)

		listed, err := NewTestClient(
			t,
			httpClient,
		).Message.List(
			context.Background(),
			"ag_1",
			"ses_1",
			50,
			0,
		)
		require.NoError(t, err)
		require.Len(t, listed.Messages, 1)
		assert.Equal(t, "msg_1", listed.Messages[0].ID)
	})

	t.Run("Get", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			messagesURL+"/msg_1",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"id": "msg_1", "content": "Hi"}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Message.Get(
			context.Background(),
			"ag_1",
			"ses_1",
			"msg_1",
		)
		require.NoError(t, err)
		assert.Equal(t, "Hi", got.Content)
	})

	t.Run("Search", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPost,
			messagesURL+"/search",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"results": []map[string]any{{"id": "msg_1", "text": "Hi", "score": 0.9}},
			}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Message.Search(
			context.Background(),
			"ag_1",
			"ses_1",
			"Hi",
			10,
		)
		require.NoError(t, err)
		require.Len(t, got.Results, 1)
		assert.Equal(t, "msg_1", got.Results[0].ID)
	})

	t.Run("Create", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPost,
			messagesURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"id": "msg_1", "content": "Hi"}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Message.Create(
			context.Background(),
			"ag_1",
			"ses_1",
			MessageInput{
				Role:    MessageRoleUser,
				Content: "Hi",
			},
		)
		require.NoError(t, err)
		assert.Equal(t, "msg_1", got.ID)
	})

	t.Run("CreateBatch", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPost,
			messagesURL+"/batch",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"messages": []map[string]any{{"id": "msg_1"}, {"id": "msg_2"}},
			}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Message.CreateBatch(
			context.Background(),
			"ag_1",
			"ses_1",
			[]MessageInput{
				{Role: MessageRoleUser, Content: "Hi"},
				{Role: MessageRoleAssistant, Content: "Hello"},
			},
		)
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("Update", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPut,
			messagesURL+"/msg_1",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"id": "msg_1", "content": "Updated"}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Message.Update(
			context.Background(),
			"ag_1",
			"ses_1",
			"msg_1",
			"Updated",
			MessageRoleUser,
			nil,
		)
		require.NoError(t, err)
		assert.Equal(t, "Updated", got.Content)
	})

	t.Run("Delete", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodDelete,
			messagesURL+"/msg_1",
			httpmock.NewStringResponder(204, ""),
		)

		err := NewTestClient(
			t,
			httpClient,
		).Message.Delete(
			context.Background(),
			"ag_1",
			"ses_1",
			"msg_1",
		)
		require.NoError(t, err)
	})

	t.Run("DeleteBatch", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodDelete,
			messagesURL+"/batch",
			httpmock.NewStringResponder(204, ""),
		)

		err := NewTestClient(
			t,
			httpClient,
		).Message.DeleteBatch(
			context.Background(),
			"ag_1",
			"ses_1",
			[]string{"msg_1", "msg_2"},
		)
		require.NoError(t, err)
	})
}
