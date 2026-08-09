// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"net/http"
)

// Memories is the session memory API client.
type Memories struct {
	Resource
}

// NewMemories creates a standalone memories client.
func NewMemories(opts ...Option) *Memories {
	return &Memories{Resource: newResource(opts...)}
}

// List returns session memories.
func (m *Memories) List(ctx context.Context, agentID, sessionID string, limit, offset int) (*MemoryList, error) {
	path, err := m.agentPath(agentID, "sessions", sessionID, "memories")
	if err != nil {
		return nil, err
	}
	var raw MemoryListResponse
	if err := m.request(ctx, http.MethodGet, path, RequestOptions{Params: PageParams(limit, offset)}, &raw); err != nil {
		return nil, err
	}
	return &MemoryList{
		Memories: raw.Memories,
		Limit:    raw.Meta.Limit,
		Offset:   raw.Meta.Offset,
		Total:    raw.Meta.Total,
	}, nil
}

// Get returns a memory by ID.
func (m *Memories) Get(ctx context.Context, agentID, sessionID, memoryID string) (*Memory, error) {
	path, err := m.agentPath(agentID, "sessions", sessionID, "memories", memoryID)
	if err != nil {
		return nil, err
	}
	var memory Memory
	if err := m.request(ctx, http.MethodGet, path, RequestOptions{}, &memory); err != nil {
		return nil, err
	}
	return &memory, nil
}

// Search searches session memories.
func (m *Memories) Search(ctx context.Context, agentID, sessionID, query string, limit int) (*MemorySearchResults, error) {
	path, err := m.agentPath(agentID, "sessions", sessionID, "memories", "search")
	if err != nil {
		return nil, err
	}
	var results MemorySearchResults
	if err := m.request(ctx, http.MethodPost, path, RequestOptions{
		JSON: map[string]any{"query": query, "limit": limit},
	}, &results); err != nil {
		return nil, err
	}
	return &results, nil
}

// Create creates one memory.
func (m *Memories) Create(ctx context.Context, agentID, sessionID string, memory MemoryInput) (*Memory, error) {
	path, err := m.agentPath(agentID, "sessions", sessionID, "memories")
	if err != nil {
		return nil, err
	}
	var created Memory
	if err := m.request(ctx, http.MethodPost, path, RequestOptions{
		JSON: EncodeMemoryItem(memory),
	}, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// CreateBatch creates multiple memories.
func (m *Memories) CreateBatch(ctx context.Context, agentID, sessionID string, memories []MemoryInput) ([]Memory, error) {
	path, err := m.agentPath(agentID, "sessions", sessionID, "memories", "batch")
	if err != nil {
		return nil, err
	}
	var raw MemoryBatchResponse
	if err := m.request(ctx, http.MethodPost, path, RequestOptions{
		JSON: MemoryBatchPayload(memories),
	}, &raw); err != nil {
		return nil, err
	}
	return raw.Memories, nil
}

// Update updates a memory.
func (m *Memories) Update(ctx context.Context, agentID, sessionID, memoryID, content string, kind MemoryKind, meta map[string]any) (*Memory, error) {
	path, err := m.agentPath(agentID, "sessions", sessionID, "memories", memoryID)
	if err != nil {
		return nil, err
	}
	fields := map[string]string{}
	if kind != "" {
		fields["kind"] = string(kind)
	}
	var updated Memory
	if err := m.request(ctx, http.MethodPut, path, RequestOptions{
		JSON: EncodeUpdateBody(content, meta, fields),
	}, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

// Delete deletes one memory.
func (m *Memories) Delete(ctx context.Context, agentID, sessionID, memoryID string) error {
	path, err := m.agentPath(agentID, "sessions", sessionID, "memories", memoryID)
	if err != nil {
		return err
	}
	return m.request(ctx, http.MethodDelete, path, RequestOptions{}, nil)
}

// DeleteBatch deletes multiple memories.
func (m *Memories) DeleteBatch(ctx context.Context, agentID, sessionID string, ids []string) error {
	path, err := m.agentPath(agentID, "sessions", sessionID, "memories", "batch")
	if err != nil {
		return err
	}
	return m.request(ctx, http.MethodDelete, path, RequestOptions{
		JSON: map[string]any{"ids": ids},
	}, nil)
}
