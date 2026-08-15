// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const defaultBaseURL = "https://app.actx0.com"

// Option configures a client.
type Option func(*config)

type config struct {
	baseURL     string
	timeout     time.Duration
	accessKey   string
	workspaceId string
	httpClient  *http.Client
}

func defaultConfig() config {
	return config{
		baseURL: defaultBaseURL,
		timeout: 30 * time.Second,
	}
}

// WithBaseURL sets the API base URL.
func WithBaseURL(baseURL string) Option {
	return func(c *config) { c.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithTimeout sets the HTTP timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *config) { c.timeout = timeout }
}

// WithAccessKey sets the X-Access-Key value.
func WithAccessKey(accessKey string) Option {
	return func(c *config) { c.accessKey = accessKey }
}

// WithWorkspaceId sets the workspace used for workspace-scoped routes.
func WithWorkspaceId(workspaceId string) Option {
	return func(c *config) { c.workspaceId = workspaceId }
}

// WithHTTPClient sets a custom HTTP client used by Resty.
func WithHTTPClient(client *http.Client) Option {
	return func(c *config) { c.httpClient = client }
}

// BaseClient is the shared Resty transport used by resource clients.
type BaseClient struct {
	baseURL     string
	timeout     time.Duration
	accessKey   string
	workspaceId string
	resty       *resty.Client
}

func NewBaseClient(opts ...Option) *BaseClient {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	var rc *resty.Client
	if cfg.httpClient != nil {
		rc = resty.NewWithClient(cfg.httpClient)
	} else {
		rc = resty.New()
	}
	rc.SetBaseURL(strings.TrimRight(cfg.baseURL, "/")).
		SetTimeout(cfg.timeout).
		SetHeader("X-Access-Key", cfg.accessKey)

	return &BaseClient{
		baseURL:     strings.TrimRight(cfg.baseURL, "/"),
		timeout:     cfg.timeout,
		accessKey:   cfg.accessKey,
		workspaceId: cfg.workspaceId,
		resty:       rc,
	}
}

type RequestOptions struct {
	Params  map[string]string
	JSON    any
	Form    map[string]string
	File    *PreparedFile
	Headers map[string]string
}

// ListMeta is pagination metadata from list responses.
type ListMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

func (c *BaseClient) Request(ctx context.Context, method, path string, opts RequestOptions, out any) error {
	if c.accessKey == "" {
		return fmt.Errorf("access_key is required")
	}

	req := c.resty.R().SetContext(ctx)
	if len(opts.Params) > 0 {
		req.SetQueryParams(opts.Params)
	}
	for key, value := range opts.Headers {
		req.SetHeader(key, value)
	}
	if out != nil {
		req.SetResult(out)
	}

	var resp *resty.Response
	var err error

	switch {
	case opts.File != nil:
		if len(opts.Form) > 0 {
			req.SetFormData(opts.Form)
		}
		req.SetFileReader("file", opts.File.Filename, bytes.NewReader(opts.File.Content))
		resp, err = req.Execute(method, path)
	case opts.JSON != nil:
		req.SetBody(opts.JSON)
		resp, err = req.Execute(method, path)
	default:
		resp, err = req.Execute(method, path)
	}
	if err != nil {
		return err
	}

	if resp.StatusCode() == http.StatusNoContent {
		return nil
	}
	if resp.IsError() {
		var parsed any
		body := resp.Body()
		if len(body) > 0 {
			if jsonErr := json.Unmarshal(body, &parsed); jsonErr != nil {
				parsed = string(body)
			}
		}
		return &APIError{StatusCode: resp.StatusCode(), Body: parsed}
	}
	return nil
}

// Close releases idle HTTP connections.
func (c *BaseClient) Close() {
	if transport, ok := c.resty.GetClient().Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}

// Resource is the shared base for API resource clients.
type Resource struct {
	*BaseClient
}

func NewResource(opts ...Option) Resource {
	return Resource{BaseClient: NewBaseClient(opts...)}
}

func (r *Resource) RequireWorkspace() (string, error) {
	if r.workspaceId == "" {
		return "", fmt.Errorf("workspace_id is required")
	}
	return r.workspaceId, nil
}

func (r *Resource) WorkspacePath(parts ...string) (string, error) {
	ws, err := r.RequireWorkspace()
	if err != nil {
		return "", err
	}
	path := "/api/v1/workspaces/" + ws
	for _, part := range parts {
		path += "/" + part
	}
	return path, nil
}

func (r *Resource) AgentPath(agentID string, parts ...string) (string, error) {
	all := append([]string{"agents", agentID}, parts...)
	return r.WorkspacePath(all...)
}
