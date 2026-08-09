// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocument(t *testing.T) {
	t.Run("list search upload", func(t *testing.T) {
		client, ctx := testClient(t)
		dir := t.TempDir()
		path := filepath.Join(dir, "policy.md")
		require.NoError(t, os.WriteFile(path, []byte("# Refund policy\n30 day window."), 0o644))

		uploaded, err := client.Document.Upload(ctx, path, "Refund policy", map[string]string{
			"team": "support", "category": "policy",
		})
		require.NoError(t, err)
		assert.Equal(t, "Refund policy", uploaded.Title)
		assert.Equal(t, "processing", uploaded.Status)

		listed, err := client.Document.List(ctx, 50, 0)
		require.NoError(t, err)
		assert.Equal(t, 1, listed.Total)
		assert.Equal(t, uploaded.ID, listed.Documents[0].ID)

		results, err := client.Document.Search(ctx, "refund policy", map[string]string{"team": "support"}, 5)
		require.NoError(t, err)
		require.Len(t, results.Results, 1)
		assert.Equal(t, 0.87, results.Results[0].Score)

		require.NoError(t, client.Document.Delete(ctx, uploaded.ID))
	})

	t.Run("exists", func(t *testing.T) {
		client, ctx := testClient(t)
		dir := t.TempDir()
		path := filepath.Join(dir, "policy.md")
		require.NoError(t, os.WriteFile(path, []byte("# Refund policy\n30 day window."), 0o644))
		labels := map[string]string{"team": "support", "category": "policy"}

		found, err := client.Document.Exists(ctx, path, labels, 50)
		require.NoError(t, err)
		assert.Nil(t, found)

		uploaded, err := client.Document.Upload(ctx, path, "Refund policy", labels)
		require.NoError(t, err)

		found, err = client.Document.Exists(ctx, path, labels, 50)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, uploaded.ID, found.ID)

		found, err = client.Document.Exists(ctx, path, map[string]string{"team": "other"}, 50)
		require.NoError(t, err)
		assert.Nil(t, found)

		require.NoError(t, os.WriteFile(path, []byte("# Refund policy\nUpdated text."), 0o644))
		found, err = client.Document.Exists(ctx, path, labels, 50)
		require.NoError(t, err)
		assert.Nil(t, found)

		_ = client.Document.Delete(ctx, uploaded.ID)
	})
}

func TestKnowledge(t *testing.T) {
	t.Run("standalone client", func(t *testing.T) {
		baseURL := startMockServer(t)
		dir := t.TempDir()
		path := filepath.Join(dir, "notes.txt")
		require.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))
		client := NewKnowledge(
			WithBaseURL(baseURL),
			WithAccessKey(defaultWorkspaceAccessKey),
			WithWorkspaceID(defaultWorkspaceID),
		)
		defer client.Close()

		uploaded, err := client.Upload(context.Background(), path, "Notes", nil)
		require.NoError(t, err)
		assert.Equal(t, "processing", uploaded.Status)
		require.NoError(t, client.Delete(context.Background(), uploaded.ID))
	})

	t.Run("upload dict labels", func(t *testing.T) {
		client, ctx := testClient(t)
		uploaded, err := client.Knowledge.Upload(ctx, FileBytes{
			Filename: "policy.md", Content: []byte("# Refund policy"), ContentType: "text/markdown",
		}, "Refund policy", map[string]string{"team": "support", "category": "policy"})
		require.NoError(t, err)
		assert.Len(t, uploaded.Labels, 2)
		assert.Contains(t, uploaded.Labels, "team=support")
		assert.Contains(t, uploaded.Labels, "category=policy")
		_ = client.Knowledge.Delete(ctx, uploaded.ID)
	})
}

func (ms *mockServer) documentObject(documentID, title, filename string, labels []string, content []byte) map[string]any {
	sum := sha256.Sum256(content)
	size := len(content)
	if size == 0 {
		size = 100
	}
	charCount := len(string(content))
	if charCount == 0 {
		charCount = 80
	}
	if labels == nil {
		labels = []string{}
	}
	return map[string]any{
		"id": documentID, "workspaceId": defaultWorkspaceID, "title": title, "filename": filename,
		"contentType": "text/markdown", "checksum": hex.EncodeToString(sum[:]),
		"size": map[string]any{"value": size, "unit": "bytes"}, "charCount": charCount,
		"labels": labels, "chunkingStrategy": "recursive", "chunkSize": 2000, "chunkOverlap": 400,
		"status": "processing", "createdAt": mockTimestamp, "updatedAt": mockTimestamp,
	}
}

func (ms *mockServer) workspaceRoute(method, path string, query url.Values, body map[string]any, form map[string]any, r *http.Request, wsPrefix string) (int, any) {
	if path == wsPrefix+"/agents" {
		return ms.agentCollectionRoute(method, path, query, body, r, wsPrefix)
	}

	if strings.Contains(path, "/prompts") {
		return ms.promptRoute(method, path, query, body, r, wsPrefix)
	}

	if path == wsPrefix+"/documents" {
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			docs := values(ms.store.documents)
			return 200, map[string]any{"documents": docs, "_meta": mockListMeta(len(docs), query)}
		}
		if method == http.MethodPost {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			fileInfo, _ := form["file"].(map[string]any)
			filename := "upload.md"
			var content []byte
			if fileInfo != nil {
				if f, ok := fileInfo["filename"].(string); ok && f != "" {
					filename = f
				}
				if c, ok := fileInfo["content"].([]byte); ok {
					content = c
				}
			}
			title := "Untitled"
			if t, ok := form["title"].(string); ok && t != "" {
				title = t
			}
			labels := []string{}
			if raw, ok := form["labels"].(string); ok && raw != "" {
				_ = json.Unmarshal([]byte(raw), &labels)
			}
			documentID := "doc_" + shortID()
			doc := ms.documentObject(documentID, title, filename, labels, content)
			ms.store.documents[documentID] = doc
			return 201, doc
		}
	}

	if path == wsPrefix+"/documents/search" {
		if !ms.authorized(r) {
			return 401, map[string]any{"errorMessage": "Invalid access key."}
		}
		labels, _ := body["labels"].(map[string]any)
		labelMap := map[string]string{}
		for k, v := range labels {
			labelMap[k] = stringify(v)
		}
		return 200, map[string]any{"results": []map[string]any{{
			"documentId": "doc_search_1", "chunkId": "chunk_1", "score": 0.87,
			"text": "Result for: " + stringify(body["query"]), "labels": labelMap,
		}}}
	}

	docPrefix := wsPrefix + "/documents/"
	if strings.HasPrefix(path, docPrefix) && method == http.MethodDelete {
		documentID := path[len(docPrefix):]
		if _, ok := ms.store.documents[documentID]; ok {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			delete(ms.store.documents, documentID)
			return 204, nil
		}
	}
	return 404, map[string]any{"errorMessage": "not found"}
}
