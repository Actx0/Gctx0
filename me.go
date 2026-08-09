// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"net/http"
)

// Me is the key introspection API client.
type Me struct {
	Resource
}

// NewMe creates a standalone me client.
func NewMe(opts ...Option) *Me {
	return &Me{Resource: newResource(opts...)}
}

// Get returns the current access-key principal.
func (m *Me) Get(ctx context.Context) (*AccessKeyPrincipal, error) {
	var raw map[string]any
	if err := m.request(ctx, http.MethodGet, "/api/v1/me", requestOptions{}, &raw); err != nil {
		return nil, err
	}
	principal, err := parseMePrincipal(raw)
	if err != nil {
		return nil, err
	}
	return &principal, nil
}
