// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const documentsURL = "https://example.test/api/v1/workspaces/ws_test/documents"

func TestUnitDocument(t *testing.T) {
	t.Run("List", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			`=~^`+documentsURL+`(?:\?.*)?\z`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"documents": []map[string]any{{"id": "doc_1", "title": "Guide"}},
				"_meta":     map[string]any{"limit": 50, "offset": 0, "total": 1},
			}),
		)

		listed, err := NewTestClient(
			t,
			httpClient,
		).Document.List(
			context.Background(),
			50,
			0,
		)
		require.NoError(t, err)
		require.Len(t, listed.Documents, 1)
		assert.Equal(t, "doc_1", listed.Documents[0].ID)
	})

	t.Run("Exists", func(t *testing.T) {
		content := []byte("hello")
		sum := sha256.Sum256(content)
		checksum := hex.EncodeToString(sum[:])

		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodGet,
			`=~^`+documentsURL+`(?:\?.*)?\z`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"documents": []map[string]any{{
					"id":       "doc_1",
					"filename": "guide.md",
					"checksum": checksum,
					"labels":   []string{"env=test"},
				}},
				"_meta": map[string]any{"limit": 50, "offset": 0, "total": 1},
			}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Document.Exists(
			context.Background(),
			FileBytes{
				Filename: "guide.md",
				Content:  content,
			},
			map[string]string{"env": "test"},
			50,
		)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "doc_1", got.ID)
	})

	t.Run("Upload", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPost,
			documentsURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"id": "doc_1", "title": "Guide"}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Document.Upload(
			context.Background(),
			FileBytes{
				Filename: "guide.md",
				Content:  []byte("hello"),
			},
			"Guide",
			map[string]string{"env": "test"},
		)
		require.NoError(t, err)
		assert.Equal(t, "doc_1", got.ID)
	})

	t.Run("Search", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodPost,
			documentsURL+"/search",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"results": []map[string]any{{"documentId": "doc_1", "text": "hello", "score": 0.9}},
			}),
		)

		got, err := NewTestClient(
			t,
			httpClient,
		).Document.Search(
			context.Background(),
			"hello",
			nil,
			10,
		)
		require.NoError(t, err)
		require.Len(t, got.Results, 1)
		assert.Equal(t, "doc_1", got.Results[0].DocumentID)
	})

	t.Run("Delete", func(t *testing.T) {
		httpClient := NewTestHTTPClient(t)
		httpmock.RegisterResponder(
			http.MethodDelete,
			documentsURL+"/doc_1",
			httpmock.NewStringResponder(204, ""),
		)

		err := NewTestClient(
			t,
			httpClient,
		).Document.Delete(
			context.Background(),
			"doc_1",
		)
		require.NoError(t, err)
	})
}
