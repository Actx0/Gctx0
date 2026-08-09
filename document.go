// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
)

// FileBytes is an in-memory file upload.
type FileBytes struct {
	Filename    string
	Content     []byte
	ContentType string
}

// DocumentSize is a document size value.
type DocumentSize struct {
	Value int    `json:"value"`
	Unit  string `json:"unit"`
}

// Document is a knowledge-base document.
type Document struct {
	ID               string       `json:"id"`
	WorkspaceID      string       `json:"workspaceId"`
	Title            string       `json:"title"`
	Filename         string       `json:"filename"`
	ContentType      string       `json:"contentType"`
	Checksum         string       `json:"checksum"`
	Size             DocumentSize `json:"size"`
	CharCount        int          `json:"charCount"`
	Labels           []string     `json:"labels"`
	ChunkingStrategy string       `json:"chunkingStrategy"`
	ChunkSize        int          `json:"chunkSize"`
	ChunkOverlap     int          `json:"chunkOverlap"`
	Status           string       `json:"status"`
	CreatedAt        string       `json:"createdAt"`
	UpdatedAt        string       `json:"updatedAt"`
}

// DocumentList is a paginated document list.
type DocumentList struct {
	Documents []Document `json:"documents"`
	Limit     int        `json:"-"`
	Offset    int        `json:"-"`
	Total     int        `json:"-"`
}

// DocumentListResponse is the API list envelope.
type DocumentListResponse struct {
	Documents []Document `json:"documents"`
	Meta      ListMeta   `json:"_meta"`
}

// SearchHit is a document search result.
type SearchHit struct {
	DocumentID string            `json:"documentId"`
	ChunkID    string            `json:"chunkId"`
	Score      float64           `json:"score"`
	Text       string            `json:"text"`
	Labels     map[string]string `json:"labels"`
}

// SearchResults holds document search hits.
type SearchResults struct {
	Results []SearchHit `json:"results"`
}

// PreparedFile is a file ready for multipart upload.
type PreparedFile struct {
	Filename    string
	Content     []byte
	ContentType string
}

// Documents is the workspace knowledge-base (documents) API client.
type Documents struct {
	Resource
}

// Knowledge is a backward-compatible alias for Documents.
type Knowledge = Documents

// NewDocuments creates a standalone documents client.
func NewDocuments(opts ...Option) *Documents {
	return &Documents{Resource: NewResource(opts...)}
}

// NewKnowledge creates a standalone documents client.
func NewKnowledge(opts ...Option) *Documents {
	return NewDocuments(opts...)
}

func PrepareFile(file any) (PreparedFile, error) {
	switch v := file.(type) {
	case string:
		content, err := os.ReadFile(v)
		if err != nil {
			return PreparedFile{}, err
		}
		contentType := "text/plain"
		if filepath.Ext(v) == ".md" {
			contentType = "text/markdown"
		}
		return PreparedFile{
			Filename:    filepath.Base(v),
			Content:     content,
			ContentType: contentType,
		}, nil
	case FileBytes:
		contentType := v.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		return PreparedFile{
			Filename:    v.Filename,
			Content:     v.Content,
			ContentType: contentType,
		}, nil
	case PreparedFile:
		return v, nil
	default:
		return PreparedFile{}, fmt.Errorf("unsupported file input type %T", file)
	}
}

func FileChecksum(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// List returns workspace documents.
func (d *Documents) List(ctx context.Context, limit, offset int) (*DocumentList, error) {
	path, err := d.WorkspacePath("documents")
	if err != nil {
		return nil, err
	}

	var raw DocumentListResponse
	if err := d.Request(ctx, http.MethodGet, path, RequestOptions{Params: PageParams(limit, offset)}, &raw); err != nil {
		return nil, err
	}

	return &DocumentList{
		Documents: raw.Documents,
		Limit:     raw.Meta.Limit,
		Offset:    raw.Meta.Offset,
		Total:     raw.Meta.Total,
	}, nil
}

// Exists returns a matching uploaded document by filename, checksum, and labels.
func (d *Documents) Exists(ctx context.Context, file any, labels map[string]string, pageSize int) (*Document, error) {
	prepared, err := PrepareFile(file)
	if err != nil {
		return nil, err
	}

	checksum := FileChecksum(prepared.Content)
	expected := map[string]struct{}{}
	for key, value := range labels {
		expected[key+"="+value] = struct{}{}
	}
	if pageSize <= 0 {
		pageSize = 50
	}

	offset := 0
	for {
		listed, err := d.List(ctx, pageSize, offset)
		if err != nil {
			return nil, err
		}
		for i := range listed.Documents {
			doc := &listed.Documents[i]
			if doc.Filename != prepared.Filename || doc.Checksum != checksum {
				continue
			}
			if len(doc.Labels) != len(expected) {
				continue
			}
			match := true
			for _, label := range doc.Labels {
				if _, ok := expected[label]; !ok {
					match = false
					break
				}
			}
			if match {
				return doc, nil
			}
		}
		offset += pageSize
		if offset >= listed.Total {
			return nil, nil
		}
	}
}

// Upload uploads a document.
func (d *Documents) Upload(ctx context.Context, file any, title string, labels map[string]string) (*Document, error) {
	prepared, err := PrepareFile(file)
	if err != nil {
		return nil, err
	}

	path, err := d.WorkspacePath("documents")
	if err != nil {
		return nil, err
	}

	form := map[string]string{"title": title}
	if labels != nil {
		keys := make([]string, 0, len(labels))
		for key := range labels {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		encoded := make([]string, 0, len(keys))
		for _, key := range keys {
			encoded = append(encoded, key+"="+labels[key])
		}
		b, err := json.Marshal(encoded)
		if err != nil {
			return nil, err
		}
		form["labels"] = string(b)
	}

	var doc Document
	if err := d.Request(ctx, http.MethodPost, path, RequestOptions{
		Form: form,
		File: &prepared,
	}, &doc); err != nil {
		return nil, err
	}

	return &doc, nil
}

// Search searches documents.
func (d *Documents) Search(ctx context.Context, query string, labels map[string]string, limit int) (*SearchResults, error) {
	path, err := d.WorkspacePath("documents", "search")
	if err != nil {
		return nil, err
	}

	body := map[string]any{"query": query, "limit": limit}
	if labels != nil {
		body["labels"] = labels
	}

	var results SearchResults
	if err := d.Request(ctx, http.MethodPost, path, RequestOptions{JSON: body}, &results); err != nil {
		return nil, err
	}

	return &results, nil
}

// Delete deletes a document.
func (d *Documents) Delete(ctx context.Context, documentID string) error {
	path, err := d.WorkspacePath("documents", documentID)
	if err != nil {
		return err
	}

	return d.Request(ctx, http.MethodDelete, path, RequestOptions{}, nil)
}
