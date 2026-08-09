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

// Sessions is the agent session API client.
type Sessions struct {
	Resource
}

// NewSessions creates a standalone sessions client.
func NewSessions(opts ...Option) *Sessions {
	return &Sessions{Resource: newResource(opts...)}
}

func (s *Sessions) requireLookup(lookup SessionLookup) (map[string]string, error) {
	params, err := buildQueryParams(QueryParams{
		ExternalID: lookup.ExternalID,
		Labels:     lookup.Labels,
	})
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
	params, err := s.requireLookup(lookup)
	if err != nil {
		return nil, err
	}
	path, err := s.agentPath(agentID, "sessions")
	if err != nil {
		return nil, err
	}
	var body any
	if title != "" {
		body = map[string]string{"title": title}
	}
	var session Session
	if err := s.request(ctx, http.MethodPost, path, requestOptions{
		params: params,
		json:   body,
	}, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// List returns sessions for an agent.
func (s *Sessions) List(ctx context.Context, agentID string, lookup SessionLookup, limit, offset int) (*SessionList, error) {
	params, err := buildQueryParams(QueryParams{
		ExternalID: lookup.ExternalID,
		Labels:     lookup.Labels,
		Limit:      intPtr(limit),
		Offset:     intPtr(offset),
	})
	if err != nil {
		return nil, err
	}
	path, err := s.agentPath(agentID, "sessions")
	if err != nil {
		return nil, err
	}
	var raw sessionListResponse
	if err := s.request(ctx, http.MethodGet, path, requestOptions{params: params}, &raw); err != nil {
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
	path, err := s.agentPath(agentID, "sessions", sessionID)
	if err != nil {
		return nil, err
	}
	var session Session
	if err := s.request(ctx, http.MethodGet, path, requestOptions{}, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// GetByLabels returns a session by external ID or labels.
func (s *Sessions) GetByLabels(ctx context.Context, agentID string, lookup SessionLookup) (*Session, error) {
	params, err := s.requireLookup(lookup)
	if err != nil {
		return nil, err
	}
	path, err := s.agentPath(agentID, "sessions", "by-labels")
	if err != nil {
		return nil, err
	}
	var session Session
	if err := s.request(ctx, http.MethodGet, path, requestOptions{params: params}, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// Update updates a session matched by external ID or labels.
func (s *Sessions) Update(ctx context.Context, agentID string, lookup SessionLookup, title string, newLabels map[string]string) (*Session, error) {
	params, err := s.requireLookup(lookup)
	if err != nil {
		return nil, err
	}
	path, err := s.agentPath(agentID, "sessions", "by-labels")
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
	if err := s.request(ctx, http.MethodPut, path, requestOptions{
		params: params,
		json:   body,
	}, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// Delete deletes a session matched by external ID or labels.
func (s *Sessions) Delete(ctx context.Context, agentID string, lookup SessionLookup) error {
	params, err := s.requireLookup(lookup)
	if err != nil {
		return err
	}
	path, err := s.agentPath(agentID, "sessions", "by-labels")
	if err != nil {
		return err
	}
	return s.request(ctx, http.MethodDelete, path, requestOptions{params: params}, nil)
}
