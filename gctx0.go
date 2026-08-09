// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"net/http"
)

// Client is the Go client for the Actx0 platform.
type Client struct {
	*BaseClient

	Agent     *Agents
	Document  *Documents
	Knowledge *Documents
	Me        *Me
	Memory    *Memories
	Message   *Messages
	Prompt    *Prompts
	Session   *Sessions
}

// NewClient creates a full Actx0 client.
func NewClient(opts ...Option) *Client {
	base := newBaseClient(opts...)

	agent := &Agents{}
	agent.attachTo(base)
	document := &Documents{}
	document.attachTo(base)
	me := &Me{}
	me.attachTo(base)
	memory := &Memories{}
	memory.attachTo(base)
	message := &Messages{}
	message.attachTo(base)
	prompt := &Prompts{}
	prompt.attachTo(base)
	session := &Sessions{}
	session.attachTo(base)

	return &Client{
		BaseClient: base,
		Agent:      agent,
		Document:   document,
		Knowledge:  document,
		Me:         me,
		Memory:     memory,
		Message:    message,
		Prompt:     prompt,
		Session:    session,
	}
}

// Health checks API health.
func (c *Client) Health(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.request(ctx, http.MethodGet, "/api/v1/_health", RequestOptions{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}
