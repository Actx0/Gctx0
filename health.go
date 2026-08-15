// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"net/http"
)

// Health checks API health.
func (c *Client) Health(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.Request(ctx, http.MethodGet, "/api/v1/public/_health", RequestOptions{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}
