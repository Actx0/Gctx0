// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"encoding/json"
	"net/http"
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

func promptWriteBody(content string, opts PromptWriteOptions) (map[string]any, error) {
	body := map[string]any{"content": content}
	if opts.Name != "" {
		body["name"] = opts.Name
	}
	if opts.Description != "" {
		body["description"] = opts.Description
	}
	if opts.Type != "" {
		body["type"] = opts.Type
	}
	if encoded, ok, err := encodeJSONField(opts.Config); err != nil {
		return nil, err
	} else if ok {
		body["config"] = encoded
	}
	if opts.CommitMessage != "" {
		body["commitMessage"] = opts.CommitMessage
	}
	if encoded, ok, err := encodeJSONField(opts.Meta); err != nil {
		return nil, err
	} else if ok {
		body["meta"] = encoded
	}
	if opts.Status != "" {
		body["status"] = opts.Status
	}
	if opts.Production != nil {
		body["production"] = *opts.Production
	}
	return body, nil
}

// Prompts is the workspace prompt API client.
type Prompts struct {
	Resource
}

// NewPrompts creates a standalone prompts client.
func NewPrompts(opts ...Option) *Prompts {
	return &Prompts{Resource: newResource(opts...)}
}

// List returns workspace prompts.
func (p *Prompts) List(ctx context.Context, limit, offset int) (*PromptList, error) {
	params, err := buildQueryParams(QueryParams{Limit: intPtr(limit), Offset: intPtr(offset)})
	if err != nil {
		return nil, err
	}
	path, err := p.workspacePath("prompts")
	if err != nil {
		return nil, err
	}
	var raw promptListResponse
	if err := p.request(ctx, http.MethodGet, path, requestOptions{params: params}, &raw); err != nil {
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
	body, err := promptWriteBody(content, opts)
	if err != nil {
		return nil, err
	}
	path, err := p.workspacePath("prompts")
	if err != nil {
		return nil, err
	}
	var info PromptInfo
	if err := p.request(ctx, http.MethodPost, path, requestOptions{json: body}, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Get returns prompt info by ID.
func (p *Prompts) Get(ctx context.Context, promptID string) (*PromptInfo, error) {
	path, err := p.workspacePath("prompts", promptID)
	if err != nil {
		return nil, err
	}
	var info PromptInfo
	if err := p.request(ctx, http.MethodGet, path, requestOptions{}, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Delete deletes a prompt.
func (p *Prompts) Delete(ctx context.Context, promptID string) error {
	path, err := p.workspacePath("prompts", promptID)
	if err != nil {
		return err
	}
	return p.request(ctx, http.MethodDelete, path, requestOptions{}, nil)
}

// GetByName returns a prompt version by handle.
func (p *Prompts) GetByName(ctx context.Context, name string, version string) (*Prompt, error) {
	path, err := p.workspacePath("promptsByName", name)
	if err != nil {
		return nil, err
	}
	params := map[string]string{}
	if version != "" {
		params["version"] = version
	}
	var raw json.RawMessage
	if err := p.request(ctx, http.MethodGet, path, requestOptions{params: params}, &raw); err != nil {
		return nil, err
	}
	prompt, err := normalizePrompt(raw)
	if err != nil {
		return nil, err
	}
	return &prompt, nil
}

// ListVersions returns prompt versions.
func (p *Prompts) ListVersions(ctx context.Context, promptID string, limit, offset int) (*PromptVersionList, error) {
	params, err := buildQueryParams(QueryParams{Limit: intPtr(limit), Offset: intPtr(offset)})
	if err != nil {
		return nil, err
	}
	path, err := p.workspacePath("prompts", promptID, "versions")
	if err != nil {
		return nil, err
	}
	var raw promptVersionListResponse
	if err := p.request(ctx, http.MethodGet, path, requestOptions{params: params}, &raw); err != nil {
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
	body, err := promptWriteBody(content, opts)
	if err != nil {
		return nil, err
	}
	path, err := p.workspacePath("prompts", promptID, "versions")
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := p.request(ctx, http.MethodPost, path, requestOptions{json: body}, &raw); err != nil {
		return nil, err
	}
	prompt, err := normalizePrompt(raw)
	if err != nil {
		return nil, err
	}
	return &prompt, nil
}

// GetVersion returns a prompt version by ID.
func (p *Prompts) GetVersion(ctx context.Context, promptID, versionID string) (*Prompt, error) {
	path, err := p.workspacePath("prompts", promptID, "versions", versionID)
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := p.request(ctx, http.MethodGet, path, requestOptions{}, &raw); err != nil {
		return nil, err
	}
	prompt, err := normalizePrompt(raw)
	if err != nil {
		return nil, err
	}
	return &prompt, nil
}

// UpdateVersion updates a prompt version.
func (p *Prompts) UpdateVersion(ctx context.Context, promptID, versionID, content string, opts PromptWriteOptions) (*Prompt, error) {
	body, err := promptWriteBody(content, opts)
	if err != nil {
		return nil, err
	}
	path, err := p.workspacePath("prompts", promptID, "versions", versionID)
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := p.request(ctx, http.MethodPut, path, requestOptions{json: body}, &raw); err != nil {
		return nil, err
	}
	prompt, err := normalizePrompt(raw)
	if err != nil {
		return nil, err
	}
	return &prompt, nil
}

// DeleteVersion deletes a prompt version.
func (p *Prompts) DeleteVersion(ctx context.Context, promptID, versionID string) error {
	path, err := p.workspacePath("prompts", promptID, "versions", versionID)
	if err != nil {
		return err
	}
	return p.request(ctx, http.MethodDelete, path, requestOptions{}, nil)
}
