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

func NewTestHTTPClient(t *testing.T) *http.Client {
	t.Helper()
	httpClient := &http.Client{}
	httpmock.ActivateNonDefault(httpClient)
	t.Cleanup(httpmock.DeactivateAndReset)
	return httpClient
}

func NewTestClient(t *testing.T, httpClient *http.Client) *Client {
	t.Helper()
	client := NewClient(
		WithBaseURL("https://example.test"),
		WithAccessKey("test-access-key"),
		WithWorkspaceId("ws_test"),
		WithHTTPClient(httpClient),
	)
	t.Cleanup(client.Close)
	return client
}

func TestUnitHealth(t *testing.T) {
	t.Run("returns health payload", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(http.MethodGet, "https://example.test/api/v1/_health",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"status": "ok"}))

		client := NewTestClient(t, httpClient)
		got, err := client.Health(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "ok", got["status"])
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
	})

	t.Run("maps api errors", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(http.MethodGet, "https://example.test/api/v1/_health",
			httpmock.NewJsonResponderOrPanic(503, map[string]any{"errorMessage": "unavailable"}))

		client := NewTestClient(t, httpClient)
		_, err := client.Health(context.Background())
		require.Error(t, err)

		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, 503, apiErr.StatusCode)
	})

	t.Run("requires access key", func(t *testing.T) {
		client := NewClient(WithBaseURL("https://example.test"))
		defer client.Close()

		_, err := client.Health(context.Background())
		require.Error(t, err)
		assert.Equal(t, "access_key is required", err.Error())
	})
}

func TestUnitMe(t *testing.T) {
	t.Run("returns principal", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(http.MethodGet, "https://example.test/api/v1/me",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"principalType": "access_key",
				"accessKey": map[string]any{
					"id":          "wkey_1",
					"workspaceId": "ws_test",
					"name":        "Agent runtime",
					"permissions": []string{"documents:read"},
				},
			}))

		client := NewTestClient(t, httpClient)
		got, err := client.Me.Get(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "access_key", got.PrincipalType)
		assert.Equal(t, "ws_test", got.AccessKey.WorkspaceID)
		assert.Equal(t, "Agent runtime", got.AccessKey.Name)
	})
}
