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

func encodeJSONField(value any) (string, bool, error) {
	if value == nil {
		return "", false, nil
	}
	switch v := value.(type) {
	case string:
		return v, true, nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", false, err
		}
		return string(b), true, nil
	}
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
	body, err := promptWriteBody(content, opts)
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
	var raw json.RawMessage
	if err := p.Request(ctx, http.MethodGet, path, RequestOptions{Params: params}, &raw); err != nil {
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
	body, err := promptWriteBody(content, opts)
	if err != nil {
		return nil, err
	}
	path, err := p.WorkspacePath("prompts", promptID, "versions")
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := p.Request(ctx, http.MethodPost, path, RequestOptions{JSON: body}, &raw); err != nil {
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
	path, err := p.WorkspacePath("prompts", promptID, "versions", versionID)
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := p.Request(ctx, http.MethodGet, path, RequestOptions{}, &raw); err != nil {
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
	path, err := p.WorkspacePath("prompts", promptID, "versions", versionID)
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := p.Request(ctx, http.MethodPut, path, RequestOptions{JSON: body}, &raw); err != nil {
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
	path, err := p.WorkspacePath("prompts", promptID, "versions", versionID)
	if err != nil {
		return err
	}
	return p.Request(ctx, http.MethodDelete, path, RequestOptions{}, nil)
}

// Compile replaces {{var}} placeholders in prompt content.
func CompilePrompt(content string, variables map[string]string) (string, error) {
	var missing string
	out := templateVar.ReplaceAllStringFunc(content, func(match string) string {
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

// Compile replaces {{var}} placeholders on a Prompt.
func (p Prompt) Compile(variables map[string]string) (string, error) {
	return CompilePrompt(p.Content, variables)
}

func normalizePrompt(raw json.RawMessage) (Prompt, error) {
	var envelope struct {
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
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Prompt{}, err
	}

	config := map[string]any{}
	if len(envelope.Config) > 0 && string(envelope.Config) != "null" {
		if envelope.Config[0] == '"' {
			var asString string
			if err := json.Unmarshal(envelope.Config, &asString); err != nil {
				return Prompt{}, err
			}
			if asString != "" {
				if err := json.Unmarshal([]byte(asString), &config); err != nil {
					return Prompt{}, err
				}
			}
		} else if err := json.Unmarshal(envelope.Config, &config); err != nil {
			return Prompt{}, err
		}
	}
	if envelope.Labels == nil {
		envelope.Labels = []string{}
	}

	return Prompt{
		ID:            envelope.ID,
		Name:          envelope.Name,
		Handle:        envelope.Handle,
		Description:   envelope.Description,
		Version:       envelope.Version,
		Type:          envelope.Type,
		Content:       envelope.Content,
		CommitHash:    envelope.CommitHash,
		Status:        envelope.Status,
		Production:    envelope.Production,
		CreatedAt:     envelope.CreatedAt,
		UpdatedAt:     envelope.UpdatedAt,
		Config:        config,
		Labels:        envelope.Labels,
		CommitMessage: envelope.CommitMessage,
		Meta:          envelope.Meta,
	}, nil
}
