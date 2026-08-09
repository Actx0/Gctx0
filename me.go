// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"net/http"
)

// AccessKeyInfo describes an access key principal.
type AccessKeyInfo struct {
	ID          string   `json:"id"`
	WorkspaceID string   `json:"workspaceId"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
	ExpiresAt   *string  `json:"expiresAt,omitempty"`
}

// MePrincipal is returned by /api/v1/me.
type MePrincipal struct {
	PrincipalType string        `json:"principalType"`
	AccessKey     AccessKeyInfo `json:"accessKey"`
}

// Me is the key introspection API client.
type Me struct {
	Resource
}

// NewMe creates a standalone me client.
func NewMe(opts ...Option) *Me {
	return &Me{Resource: NewResource(opts...)}
}

// Get returns the current principal for the access key.
func (m *Me) Get(ctx context.Context) (*MePrincipal, error) {
	var principal MePrincipal
	if err := m.Request(ctx, http.MethodGet, "/api/v1/me", RequestOptions{}, &principal); err != nil {
		return nil, err
	}
	return &principal, nil
}
