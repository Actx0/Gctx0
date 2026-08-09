// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
)

var templateVar = regexp.MustCompile(`\{\{(\w+)\}\}`)

// PromptResponse is the API wire shape for a prompt version.
type PromptResponse struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Handle        string          `json:"handle"`
	Description   string          `json:"description"`
	Version       int             `json:"version"`
	Type          string          `json:"type"`
	Content       string          `json:"content"`
	CommitHash    string          `json:"commitHash"`
	Status        string          `json:"status"`
	Production    bool            `json:"production"`
	CreatedAt     string          `json:"createdAt"`
	UpdatedAt     string          `json:"updatedAt"`
	Config        json.RawMessage `json:"config"`
	Labels        []string        `json:"labels"`
	CommitMessage *string         `json:"commitMessage"`
	Meta          *string         `json:"meta"`
}

// PromptType is the type of a prompt version.
type PromptType string

// PromptStatus is the lifecycle status of a prompt version.
type PromptStatus string

const (
	PromptTypeText PromptType = "text"
	PromptTypeChat PromptType = "chat"

	PromptStatusActive   PromptStatus = "active"
	PromptStatusArchived PromptStatus = "archived"
)

// PromptWriteOptions are optional fields for prompt create/update.
type PromptWriteOptions struct {
	Name          string
	Description   string
	Type          PromptType
	Config        any
	CommitMessage string
	Meta          any
	Status        PromptStatus
	Production    *bool
}

// PromptInfo is a prompt summary from list/create/get-by-id.
type PromptInfo struct {
	PromptID     string `json:"promptId"`
	Name         string `json:"name"`
	Handle       string `json:"handle"`
	Description  string `json:"description"`
	VersionCount int    `json:"versionCount"`
}

// PromptList is a paginated prompt list.
type PromptList struct {
	Prompts []PromptInfo `json:"prompts"`
	Limit   int          `json:"-"`
	Offset  int          `json:"-"`
	Total   int          `json:"-"`
}

// PromptListResponse is the API list envelope.
type PromptListResponse struct {
	Prompts []PromptInfo `json:"prompts"`
	Meta    ListMeta     `json:"_meta"`
}

// Prompt is a prompt version returned by the API.
type Prompt struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Handle        string         `json:"handle"`
	Description   string         `json:"description"`
	Version       int            `json:"version"`
	Type          string         `json:"type"`
	Content       string         `json:"content"`
	CommitHash    string         `json:"commitHash"`
	Status        string         `json:"status"`
	Production    bool           `json:"production"`
	CreatedAt     string         `json:"createdAt"`
	UpdatedAt     string         `json:"updatedAt"`
	Config        map[string]any `json:"config"`
	Labels        []string       `json:"labels"`
	CommitMessage *string        `json:"commitMessage"`
	Meta          *string        `json:"meta"`
}

// PromptVersion is a backward-compatible alias for Prompt.
type PromptVersion = Prompt

// PromptVersionList is a paginated prompt version list.
type PromptVersionList struct {
	Versions []Prompt `json:"versions"`
	Limit    int      `json:"-"`
	Offset   int      `json:"-"`
	Total    int      `json:"-"`
}

// PromptVersionListResponse is the API version list envelope.
type PromptVersionListResponse struct {
	Versions []Prompt `json:"versions"`
	Meta     ListMeta `json:"_meta"`
}

// Prompts is the workspace prompt API client.
type Prompts struct {
	Resource
}

// NewPrompts creates a standalone prompts client.
func NewPrompts(opts ...Option) *Prompts {
	return &Prompts{Resource: NewResource(opts...)}
}

// List returns workspace prompts.
func (p *Prompts) List(ctx context.Context, limit, offset int) (*PromptList, error) {
	path, err := p.WorkspacePath("prompts")
	if err != nil {
		return nil, err
	}

	var raw PromptListResponse
	if err := p.Request(ctx, http.MethodGet, path, RequestOptions{Params: PageParams(limit, offset)}, &raw); err != nil {
		return nil, err
	}

	return &PromptList{
		Prompts: raw.Prompts,
		Limit:   raw.Meta.Limit,
		Offset:  raw.Meta.Offset,
		Total:   raw.Meta.Total,
	}, nil
}

// Create creates a prompt.
func (p *Prompts) Create(ctx context.Context, name string, typ PromptType, content string, opts PromptWriteOptions) (*PromptInfo, error) {
	opts.Name = name
	opts.Type = typ
	body, err := PromptWriteBody(content, opts)
	if err != nil {
		return nil, err
	}

	path, err := p.WorkspacePath("prompts")
	if err != nil {
		return nil, err
	}

	var info PromptInfo
	if err := p.Request(ctx, http.MethodPost, path, RequestOptions{JSON: body}, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// Get returns prompt info by ID.
func (p *Prompts) Get(ctx context.Context, promptID string) (*PromptInfo, error) {
	path, err := p.WorkspacePath("prompts", promptID)
	if err != nil {
		return nil, err
	}

	var info PromptInfo
	if err := p.Request(ctx, http.MethodGet, path, RequestOptions{}, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// Delete deletes a prompt.
func (p *Prompts) Delete(ctx context.Context, promptID string) error {
	path, err := p.WorkspacePath("prompts", promptID)
	if err != nil {
		return err
	}

	return p.Request(ctx, http.MethodDelete, path, RequestOptions{}, nil)
}

// GetByName returns a prompt version by handle.
func (p *Prompts) GetByName(ctx context.Context, name string, version string) (*Prompt, error) {
	path, err := p.WorkspacePath("promptsByName", name)
	if err != nil {
		return nil, err
	}

	params := map[string]string{}
	if version != "" {
		params["version"] = version
	}

	var prompt Prompt
	if err := p.Request(ctx, http.MethodGet, path, RequestOptions{Params: params}, &prompt); err != nil {
		return nil, err
	}

	return &prompt, nil
}

// ListVersions returns prompt versions.
func (p *Prompts) ListVersions(ctx context.Context, promptID string, limit, offset int) (*PromptVersionList, error) {
	path, err := p.WorkspacePath("prompts", promptID, "versions")
	if err != nil {
		return nil, err
	}

	var raw PromptVersionListResponse
	if err := p.Request(ctx, http.MethodGet, path, RequestOptions{Params: PageParams(limit, offset)}, &raw); err != nil {
		return nil, err
	}

	versions := make([]Prompt, 0, len(raw.Versions))
	for _, version := range raw.Versions {
		versions = append(versions, version)
	}

	return &PromptVersionList{
		Versions: versions,
		Limit:    raw.Meta.Limit,
		Offset:   raw.Meta.Offset,
		Total:    raw.Meta.Total,
	}, nil
}

// CreateVersion creates a new prompt version.
func (p *Prompts) CreateVersion(ctx context.Context, promptID string, typ PromptType, content string, opts PromptWriteOptions) (*Prompt, error) {
	opts.Type = typ
	body, err := PromptWriteBody(content, opts)
	if err != nil {
		return nil, err
	}

	path, err := p.WorkspacePath("prompts", promptID, "versions")
	if err != nil {
		return nil, err
	}

	var prompt Prompt
	if err := p.Request(ctx, http.MethodPost, path, RequestOptions{JSON: body}, &prompt); err != nil {
		return nil, err
	}

	return &prompt, nil
}

// GetVersion returns a prompt version by ID.
func (p *Prompts) GetVersion(ctx context.Context, promptID, versionID string) (*Prompt, error) {
	path, err := p.WorkspacePath("prompts", promptID, "versions", versionID)
	if err != nil {
		return nil, err
	}

	var prompt Prompt
	if err := p.Request(ctx, http.MethodGet, path, RequestOptions{}, &prompt); err != nil {
		return nil, err
	}

	return &prompt, nil
}

// UpdateVersion updates a prompt version.
func (p *Prompts) UpdateVersion(ctx context.Context, promptID, versionID, content string, opts PromptWriteOptions) (*Prompt, error) {
	body, err := PromptWriteBody(content, opts)
	if err != nil {
		return nil, err
	}

	path, err := p.WorkspacePath("prompts", promptID, "versions", versionID)
	if err != nil {
		return nil, err
	}

	var prompt Prompt
	if err := p.Request(ctx, http.MethodPut, path, RequestOptions{JSON: body}, &prompt); err != nil {
		return nil, err
	}

	return &prompt, nil
}

// DeleteVersion deletes a prompt version.
func (p *Prompts) DeleteVersion(ctx context.Context, promptID, versionID string) error {
	path, err := p.WorkspacePath("prompts", promptID, "versions", versionID)
	if err != nil {
		return err
	}

	return p.Request(ctx, http.MethodDelete, path, RequestOptions{}, nil)
}

// Compile replaces {{var}} placeholders on a Prompt.
func (p Prompt) Compile(variables map[string]string) (string, error) {
	var missing string

	out := templateVar.ReplaceAllStringFunc(p.Content, func(match string) string {
		key := templateVar.FindStringSubmatch(match)[1]
		value, ok := variables[key]
		if !ok {
			missing = key
			return match
		}
		return value
	})

	if missing != "" {
		return "", fmt.Errorf("missing template variable: %s", missing)
	}

	return out, nil
}

// UnmarshalJSON decodes a prompt version, accepting config as an object or JSON string.
func (p *Prompt) UnmarshalJSON(data []byte) error {
	var raw PromptResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	config := map[string]any{}
	if len(raw.Config) > 0 && string(raw.Config) != "null" {
		if raw.Config[0] == '"' {
			var asString string
			if err := json.Unmarshal(raw.Config, &asString); err != nil {
				return err
			}
			if asString != "" {
				if err := json.Unmarshal([]byte(asString), &config); err != nil {
					return err
				}
			}
		} else if err := json.Unmarshal(raw.Config, &config); err != nil {
			return err
		}
	}

	labels := raw.Labels
	if labels == nil {
		labels = []string{}
	}

	*p = Prompt{
		ID:            raw.ID,
		Name:          raw.Name,
		Handle:        raw.Handle,
		Description:   raw.Description,
		Version:       raw.Version,
		Type:          raw.Type,
		Content:       raw.Content,
		CommitHash:    raw.CommitHash,
		Status:        raw.Status,
		Production:    raw.Production,
		CreatedAt:     raw.CreatedAt,
		UpdatedAt:     raw.UpdatedAt,
		Config:        config,
		Labels:        labels,
		CommitMessage: raw.CommitMessage,
		Meta:          raw.Meta,
	}

	return nil
}
