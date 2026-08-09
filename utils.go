// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func PageParams(limit, offset int) map[string]string {
	return map[string]string{
		"limit":  strconv.Itoa(limit),
		"offset": strconv.Itoa(offset),
	}
}

func LabelParams(externalID string, labels map[string]string) (map[string]string, error) {
	params := map[string]string{}
	if externalID != "" {
		params["id"] = externalID
	}
	for key, value := range labels {
		switch key {
		case "id", "limit", "offset":
			return nil, fmt.Errorf("reserved query key: %s", key)
		}
		params[key] = value
	}
	return params, nil
}

func WithMeta(body map[string]string, meta map[string]any) map[string]string {
	if meta == nil {
		return body
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return body
	}
	body["meta"] = string(b)
	return body
}

func EncodeMessageItem(item MessageInput) map[string]string {
	return WithMeta(map[string]string{
		"role":    string(item.Role),
		"content": item.Content,
	}, item.Meta)
}

func EncodeMemoryItem(item MemoryInput) map[string]string {
	return WithMeta(map[string]string{
		"kind":    string(item.Kind),
		"content": item.Content,
	}, item.Meta)
}

func EncodeUpdateBody(content string, meta map[string]any, fields map[string]string) map[string]string {
	body := map[string]string{"content": content}
	for key, value := range fields {
		if value != "" {
			body[key] = value
		}
	}
	return WithMeta(body, meta)
}

func MessageBatchPayload(items []MessageInput) map[string]any {
	encoded := make([]map[string]string, len(items))
	for i, item := range items {
		encoded[i] = EncodeMessageItem(item)
	}
	return map[string]any{"messages": encoded}
}

func MemoryBatchPayload(items []MemoryInput) map[string]any {
	encoded := make([]map[string]string, len(items))
	for i, item := range items {
		encoded[i] = EncodeMemoryItem(item)
	}
	return map[string]any{"memories": encoded}
}

func EncodeJSONField(value any) (string, bool, error) {
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

func PromptWriteBody(content string, opts PromptWriteOptions) (map[string]any, error) {
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
	if encoded, ok, err := EncodeJSONField(opts.Config); err != nil {
		return nil, err
	} else if ok {
		body["config"] = encoded
	}
	if opts.CommitMessage != "" {
		body["commitMessage"] = opts.CommitMessage
	}
	if encoded, ok, err := EncodeJSONField(opts.Meta); err != nil {
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
