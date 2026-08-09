// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"net/http"
)

// Agents is the workspace agent API client.
type Agents struct {
	Resource
}

// NewAgents creates a standalone agents client.
func NewAgents(opts ...Option) *Agents {
	return &Agents{Resource: newResource(opts...)}
}

// List returns workspace agents.
func (a *Agents) List(ctx context.Context, limit, offset int) (*AgentList, error) {
	path, err := a.workspacePath("agents")
	if err != nil {
		return nil, err
	}
	var raw AgentListResponse
	if err := a.request(ctx, http.MethodGet, path, RequestOptions{Params: PageParams(limit, offset)}, &raw); err != nil {
		return nil, err
	}
	return &AgentList{
		Agents: raw.Agents,
		Limit:  raw.Meta.Limit,
		Offset: raw.Meta.Offset,
		Total:  raw.Meta.Total,
	}, nil
}

// Get returns an agent by ID.
func (a *Agents) Get(ctx context.Context, agentID string) (*Agent, error) {
	path, err := a.workspacePath("agents", agentID)
	if err != nil {
		return nil, err
	}
	var agent Agent
	if err := a.request(ctx, http.MethodGet, path, RequestOptions{}, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// Create creates an agent.
func (a *Agents) Create(ctx context.Context, name, description string) (*Agent, error) {
	path, err := a.workspacePath("agents")
	if err != nil {
		return nil, err
	}
	var agent Agent
	if err := a.request(ctx, http.MethodPost, path, RequestOptions{
		JSON: map[string]string{"name": name, "description": description},
	}, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// Update updates an agent.
func (a *Agents) Update(ctx context.Context, agentID, name, description string) (*Agent, error) {
	path, err := a.workspacePath("agents", agentID)
	if err != nil {
		return nil, err
	}
	var agent Agent
	if err := a.request(ctx, http.MethodPut, path, RequestOptions{
		JSON: map[string]string{"name": name, "description": description},
	}, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// Delete deletes an agent.
func (a *Agents) Delete(ctx context.Context, agentID string) error {
	path, err := a.workspacePath("agents", agentID)
	if err != nil {
		return err
	}
	return a.request(ctx, http.MethodDelete, path, RequestOptions{}, nil)
}
