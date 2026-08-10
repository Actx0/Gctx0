// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"net/http"
)

// AgentConfigs holds nested agent configuration.
type AgentConfigs struct {
	MemoryPipeline bool `json:"memoryPipeline"`
}

// Agent is a workspace agent.
type Agent struct {
	ID          string            `json:"id"`
	WorkspaceID string            `json:"workspaceId"`
	Name        string            `json:"name"`
	Kind        string            `json:"kind"`
	PromptID    *string           `json:"promptId"`
	KBLabels    map[string]string `json:"kbLabels"`
	Handle      string            `json:"handle"`
	Description string            `json:"description"`
	Status      string            `json:"status"`
	Configs     AgentConfigs      `json:"configs"`
	CreatedAt   string            `json:"createdAt"`
	UpdatedAt   string            `json:"updatedAt"`
}

// AgentList is a paginated agent list.
type AgentList struct {
	Agents []Agent `json:"agents"`
	Limit  int     `json:"-"`
	Offset int     `json:"-"`
	Total  int     `json:"-"`
}

// AgentListResponse is the API list envelope.
type AgentListResponse struct {
	Agents []Agent  `json:"agents"`
	Meta   ListMeta `json:"_meta"`
}

// AgentWriteOptions are optional fields for agent create/update.
type AgentWriteOptions struct {
	MemoryPipeline *bool
}

// Agents is the workspace agent API client.
type Agents struct {
	Resource
}

// NewAgents creates a standalone agents client.
func NewAgents(opts ...Option) *Agents {
	return &Agents{Resource: NewResource(opts...)}
}

// List returns workspace agents.
func (a *Agents) List(ctx context.Context, limit, offset int) (*AgentList, error) {
	path, err := a.WorkspacePath("agents")
	if err != nil {
		return nil, err
	}

	var raw AgentListResponse
	if err := a.Request(ctx, http.MethodGet, path, RequestOptions{Params: PageParams(limit, offset)}, &raw); err != nil {
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
	path, err := a.WorkspacePath("agents", agentID)
	if err != nil {
		return nil, err
	}

	var agent Agent
	if err := a.Request(ctx, http.MethodGet, path, RequestOptions{}, &agent); err != nil {
		return nil, err
	}

	return &agent, nil
}

// Create creates an agent.
func (a *Agents) Create(ctx context.Context, name, description string, opts AgentWriteOptions) (*Agent, error) {
	path, err := a.WorkspacePath("agents")
	if err != nil {
		return nil, err
	}

	var agent Agent
	if err := a.Request(ctx, http.MethodPost, path, RequestOptions{
		JSON: AgentWriteBody(name, description, opts),
	}, &agent); err != nil {
		return nil, err
	}

	return &agent, nil
}

// Update updates an agent.
func (a *Agents) Update(ctx context.Context, agentID, name, description string, opts AgentWriteOptions) (*Agent, error) {
	path, err := a.WorkspacePath("agents", agentID)
	if err != nil {
		return nil, err
	}

	var agent Agent
	if err := a.Request(ctx, http.MethodPut, path, RequestOptions{
		JSON: AgentWriteBody(name, description, opts),
	}, &agent); err != nil {
		return nil, err
	}

	return &agent, nil
}

// Delete deletes an agent.
func (a *Agents) Delete(ctx context.Context, agentID string) error {
	path, err := a.WorkspacePath("agents", agentID)
	if err != nil {
		return err
	}

	return a.Request(ctx, http.MethodDelete, path, RequestOptions{}, nil)
}
