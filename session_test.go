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

const sessionsURL = "https://example.test/api/v1/workspaces/ws_test/agents/ag_1/sessions"

func TestUnitSession(t *testing.T) {
	lookup := SessionLookup{ExternalID: "thread-1"}

	t.Run("Create", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPost,
			`=~^`+sessionsURL+`(?:\?.*)?\z`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"id":         "ses_1",
				"externalId": "thread-1",
				"title":      "Chat",
			}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Session.Create(
			context.Background(),
			"ag_1",
			lookup,
			"Chat",
		)
		require.NoError(t, err)
		assert.Equal(t, "ses_1", got.ID)
	})

	t.Run("List", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			`=~^`+sessionsURL+`(?:\?.*)?\z`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"sessions": []map[string]any{{"id": "ses_1", "title": "Chat"}},
				"_meta":    map[string]any{"limit": 50, "offset": 0, "total": 1},
			}),
		)

		listed, err := NewTestClient(
			t,
			httpClient,
		).Session.List(
			context.Background(),
			"ag_1",
			SessionLookup{},
			50,
			0,
		)
		require.NoError(t, err)
		require.Len(t, listed.Sessions, 1)
		assert.Equal(t, "ses_1", listed.Sessions[0].ID)
	})

	t.Run("Get", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			sessionsURL+"/ses_1",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"id": "ses_1", "title": "Chat"}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Session.Get(
			context.Background(),
			"ag_1",
			"ses_1",
		)
		require.NoError(t, err)
		assert.Equal(t, "Chat", got.Title)
	})

	t.Run("GetByLabels", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			`=~^`+sessionsURL+`/by-labels(?:\?.*)?\z`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"id":         "ses_1",
				"externalId": "thread-1",
			}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Session.GetByLabels(
			context.Background(),
			"ag_1",
			lookup,
		)
		require.NoError(t, err)
		assert.Equal(t, "thread-1", got.ExternalID)
	})

	t.Run("Update", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPut,
			`=~^`+sessionsURL+`/by-labels(?:\?.*)?\z`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"id": "ses_1", "title": "Renamed"}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Session.Update(
			context.Background(),
			"ag_1",
			lookup,
			"Renamed",
			nil,
		)
		require.NoError(t, err)
		assert.Equal(t, "Renamed", got.Title)
	})

	t.Run("Delete", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodDelete,
			`=~^`+sessionsURL+`/by-labels(?:\?.*)?\z`,
			httpmock.NewStringResponder(204, ""),
		)

		err := NewTestClient(
			t,
			httpClient,
		).Session.Delete(
			context.Background(),
			"ag_1",
			lookup,
		)
		require.NoError(t, err)
	})
}
