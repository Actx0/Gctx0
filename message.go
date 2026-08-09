// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"net/http"
)

// Messages is the session message API client.
type Messages struct {
	Resource
}

// NewMessages creates a standalone messages client.
func NewMessages(opts ...Option) *Messages {
	return &Messages{Resource: NewResource(opts...)}
}

// List returns session messages.
func (m *Messages) List(ctx context.Context, agentID, sessionID string, limit, offset int) (*MessageList, error) {
	path, err := m.AgentPath(agentID, "sessions", sessionID, "messages")
	if err != nil {
		return nil, err
	}
	var raw MessageListResponse
	if err := m.Request(ctx, http.MethodGet, path, RequestOptions{Params: PageParams(limit, offset)}, &raw); err != nil {
		return nil, err
	}
	return &MessageList{
		Messages: raw.Messages,
		Limit:    raw.Meta.Limit,
		Offset:   raw.Meta.Offset,
		Total:    raw.Meta.Total,
	}, nil
}

// Get returns a message by ID.
func (m *Messages) Get(ctx context.Context, agentID, sessionID, messageID string) (*Message, error) {
	path, err := m.AgentPath(agentID, "sessions", sessionID, "messages", messageID)
	if err != nil {
		return nil, err
	}
	var message Message
	if err := m.Request(ctx, http.MethodGet, path, RequestOptions{}, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// Search searches session messages.
func (m *Messages) Search(ctx context.Context, agentID, sessionID, query string, limit int) (*MessageSearchResults, error) {
	path, err := m.AgentPath(agentID, "sessions", sessionID, "messages", "search")
	if err != nil {
		return nil, err
	}
	var results MessageSearchResults
	if err := m.Request(ctx, http.MethodPost, path, RequestOptions{
		JSON: map[string]any{"query": query, "limit": limit},
	}, &results); err != nil {
		return nil, err
	}
	return &results, nil
}

// Create creates one message.
func (m *Messages) Create(ctx context.Context, agentID, sessionID string, message MessageInput) (*Message, error) {
	path, err := m.AgentPath(agentID, "sessions", sessionID, "messages")
	if err != nil {
		return nil, err
	}
	var created Message
	if err := m.Request(ctx, http.MethodPost, path, RequestOptions{
		JSON: EncodeMessageItem(message),
	}, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// CreateBatch creates multiple messages.
func (m *Messages) CreateBatch(ctx context.Context, agentID, sessionID string, messages []MessageInput) ([]Message, error) {
	path, err := m.AgentPath(agentID, "sessions", sessionID, "messages", "batch")
	if err != nil {
		return nil, err
	}
	var raw MessageBatchResponse
	if err := m.Request(ctx, http.MethodPost, path, RequestOptions{
		JSON: MessageBatchPayload(messages),
	}, &raw); err != nil {
		return nil, err
	}
	return raw.Messages, nil
}

// Update updates a message.
func (m *Messages) Update(ctx context.Context, agentID, sessionID, messageID, content string, role MessageRole, meta map[string]any) (*Message, error) {
	path, err := m.AgentPath(agentID, "sessions", sessionID, "messages", messageID)
	if err != nil {
		return nil, err
	}
	fields := map[string]string{}
	if role != "" {
		fields["role"] = string(role)
	}
	var updated Message
	if err := m.Request(ctx, http.MethodPut, path, RequestOptions{
		JSON: EncodeUpdateBody(content, meta, fields),
	}, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

// Delete deletes one message.
func (m *Messages) Delete(ctx context.Context, agentID, sessionID, messageID string) error {
	path, err := m.AgentPath(agentID, "sessions", sessionID, "messages", messageID)
	if err != nil {
		return err
	}
	return m.Request(ctx, http.MethodDelete, path, RequestOptions{}, nil)
}

// DeleteBatch deletes multiple messages.
func (m *Messages) DeleteBatch(ctx context.Context, agentID, sessionID string, ids []string) error {
	path, err := m.AgentPath(agentID, "sessions", sessionID, "messages", "batch")
	if err != nil {
		return err
	}
	return m.Request(ctx, http.MethodDelete, path, RequestOptions{
		JSON: map[string]any{"ids": ids},
	}, nil)
}
