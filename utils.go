// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

var (
	reservedQueryKeys = map[string]struct{}{
		"id": {}, "limit": {}, "offset": {},
	}
	templateVar = regexp.MustCompile(`\{\{(\w+)\}\}`)
)

// QueryParams are optional list/lookup query parameters.
type QueryParams struct {
	ExternalID string
	Labels     map[string]string
	Limit      *int
	Offset     *int
}

func buildQueryParams(q QueryParams) (map[string]string, error) {
	params := map[string]string{}
	if q.ExternalID != "" {
		params["id"] = q.ExternalID
	}
	for key, value := range q.Labels {
		if _, reserved := reservedQueryKeys[key]; reserved {
			return nil, fmt.Errorf("reserved query key: %s", key)
		}
		params[key] = value
	}
	if q.Limit != nil {
		params["limit"] = strconv.Itoa(*q.Limit)
	}
	if q.Offset != nil {
		params["offset"] = strconv.Itoa(*q.Offset)
	}
	return params, nil
}

// StringifyMeta converts client-side metadata to the JSON string the API expects on write.
func StringifyMeta(meta map[string]any) (string, bool) {
	if meta == nil {
		return "", false
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return "", false
	}
	return string(b), true
}

func encodeMessageItem(item MessageInput) map[string]string {
	body := map[string]string{
		"role":    string(item.Role),
		"content": item.Content,
	}
	if meta, ok := StringifyMeta(item.Meta); ok {
		body["meta"] = meta
	}
	return body
}

func encodeMemoryItem(item MemoryInput) map[string]string {
	body := map[string]string{
		"kind":    string(item.Kind),
		"content": item.Content,
	}
	if meta, ok := StringifyMeta(item.Meta); ok {
		body["meta"] = meta
	}
	return body
}

// BuildMessageBatchPayload builds a batch create body for messages.
func BuildMessageBatchPayload(items []MessageInput) map[string]any {
	encoded := make([]map[string]string, len(items))
	for i, item := range items {
		encoded[i] = encodeMessageItem(item)
	}
	return map[string]any{"messages": encoded}
}

// BuildMemoryBatchPayload builds a batch create body for memories.
func BuildMemoryBatchPayload(items []MemoryInput) map[string]any {
	encoded := make([]map[string]string, len(items))
	for i, item := range items {
		encoded[i] = encodeMemoryItem(item)
	}
	return map[string]any{"memories": encoded}
}

func encodeUpdateBody(content string, meta map[string]any, fields map[string]string) map[string]string {
	body := map[string]string{"content": content}
	for key, value := range fields {
		if value != "" {
			body[key] = value
		}
	}
	if encoded, ok := StringifyMeta(meta); ok {
		body["meta"] = encoded
	}
	return body
}

func parseMePrincipal(data map[string]any) (AccessKeyPrincipal, error) {
	principalType, _ := data["principalType"].(string)
	if principalType != "access_key" {
		return AccessKeyPrincipal{}, fmt.Errorf("unknown principalType: %v", data["principalType"])
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return AccessKeyPrincipal{}, err
	}
	var principal AccessKeyPrincipal
	if err := json.Unmarshal(raw, &principal); err != nil {
		return AccessKeyPrincipal{}, err
	}
	return principal, nil
}

// PreparedFile is a file ready for multipart upload.
type PreparedFile struct {
	Filename    string
	Content     []byte
	ContentType string
}

// PrepareFile normalizes a path or in-memory file for upload.
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

// FileChecksum returns the SHA-256 hex digest of content.
func FileChecksum(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// CompilePrompt replaces {{var}} placeholders in prompt content.
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
		} else {
			if err := json.Unmarshal(envelope.Config, &config); err != nil {
				return Prompt{}, err
			}
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

func intPtr(v int) *int { return &v }
