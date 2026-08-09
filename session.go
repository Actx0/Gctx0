// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"fmt"
	"net/http"
)

// SessionLookup identifies a session by external ID and/or labels.
type SessionLookup struct {
	ExternalID string
	Labels     map[string]string
}

// Session is an agent session.
type Session struct {
	ID          string            `json:"id"`
	ExternalID  string            `json:"externalId"`
	WorkspaceID string            `json:"workspaceId"`
	AgentID     string            `json:"agentId"`
	Title       string            `json:"title"`
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Meta        map[string]any    `json:"meta"`
	CreatedAt   string            `json:"createdAt"`
	UpdatedAt   string            `json:"updatedAt"`
}

// SessionList is a paginated session list.
type SessionList struct {
	Sessions []Session `json:"sessions"`
	Limit    int       `json:"-"`
	Offset   int       `json:"-"`
	Total    int       `json:"-"`
}

// SessionListResponse is the API list envelope.
type SessionListResponse struct {
	Sessions []Session `json:"sessions"`
	Meta     ListMeta  `json:"_meta"`
}

// Sessions is the agent session API client.
type Sessions struct {
	Resource
}

// NewSessions creates a standalone sessions client.
func NewSessions(opts ...Option) *Sessions {
	return &Sessions{Resource: NewResource(opts...)}
}

func (s *Sessions) RequireLookup(lookup SessionLookup) (map[string]string, error) {
	params, err := LabelParams(lookup.ExternalID, lookup.Labels)
	if err != nil {
		return nil, err
	}
	if len(params) == 0 {
		return nil, fmt.Errorf("external_id or labels is required")
	}
	return params, nil
}

// Create creates a session keyed by external ID or labels.
func (s *Sessions) Create(ctx context.Context, agentID string, lookup SessionLookup, title string) (*Session, error) {
	params, err := s.RequireLookup(lookup)
	if err != nil {
		return nil, err
	}

	path, err := s.AgentPath(agentID, "sessions")
	if err != nil {
		return nil, err
	}

	var body any
	if title != "" {
		body = map[string]string{"title": title}
	}

	var session Session
	if err := s.Request(ctx, http.MethodPost, path, RequestOptions{
		Params: params,
		JSON:   body,
	}, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// List returns sessions for an agent.
func (s *Sessions) List(ctx context.Context, agentID string, lookup SessionLookup, limit, offset int) (*SessionList, error) {
	params, err := LabelParams(lookup.ExternalID, lookup.Labels)
	if err != nil {
		return nil, err
	}

	for key, value := range PageParams(limit, offset) {
		params[key] = value
	}

	path, err := s.AgentPath(agentID, "sessions")
	if err != nil {
		return nil, err
	}

	var raw SessionListResponse
	if err := s.Request(ctx, http.MethodGet, path, RequestOptions{Params: params}, &raw); err != nil {
		return nil, err
	}

	return &SessionList{
		Sessions: raw.Sessions,
		Limit:    raw.Meta.Limit,
		Offset:   raw.Meta.Offset,
		Total:    raw.Meta.Total,
	}, nil
}

// Get returns a session by ID.
func (s *Sessions) Get(ctx context.Context, agentID, sessionID string) (*Session, error) {
	path, err := s.AgentPath(agentID, "sessions", sessionID)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := s.Request(ctx, http.MethodGet, path, RequestOptions{}, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// GetByLabels returns a session by external ID or labels.
func (s *Sessions) GetByLabels(ctx context.Context, agentID string, lookup SessionLookup) (*Session, error) {
	params, err := s.RequireLookup(lookup)
	if err != nil {
		return nil, err
	}

	path, err := s.AgentPath(agentID, "sessions", "by-labels")
	if err != nil {
		return nil, err
	}

	var session Session
	if err := s.Request(ctx, http.MethodGet, path, RequestOptions{Params: params}, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// Update updates a session matched by external ID or labels.
func (s *Sessions) Update(ctx context.Context, agentID string, lookup SessionLookup, title string, newLabels map[string]string) (*Session, error) {
	params, err := s.RequireLookup(lookup)
	if err != nil {
		return nil, err
	}

	path, err := s.AgentPath(agentID, "sessions", "by-labels")
	if err != nil {
		return nil, err
	}

	body := map[string]any{}
	if title != "" {
		body["title"] = title
	}
	if newLabels != nil {
		body["labels"] = newLabels
	}

	var session Session
	if err := s.Request(ctx, http.MethodPut, path, RequestOptions{
		Params: params,
		JSON:   body,
	}, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// Delete deletes a session matched by external ID or labels.
func (s *Sessions) Delete(ctx context.Context, agentID string, lookup SessionLookup) error {
	params, err := s.RequireLookup(lookup)
	if err != nil {
		return err
	}

	path, err := s.AgentPath(agentID, "sessions", "by-labels")
	if err != nil {
		return err
	}

	return s.Request(ctx, http.MethodDelete, path, RequestOptions{Params: params}, nil)
}
