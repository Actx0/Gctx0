// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

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
	base := NewBaseClient(opts...)
	res := Resource{BaseClient: base}
	document := &Documents{Resource: res}

	return &Client{
		BaseClient: base,
		Agent:      &Agents{Resource: res},
		Document:   document,
		Knowledge:  document,
		Me:         &Me{Resource: res},
		Memory:     &Memories{Resource: res},
		Message:    &Messages{Resource: res},
		Prompt:     &Prompts{Resource: res},
		Session:    &Sessions{Resource: res},
	}
}
