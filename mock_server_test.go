// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const (
	defaultWorkspaceAccessKey = "test-workspace-access-key"
	defaultWorkspaceID        = "ws-test-1"
	defaultAgentID            = "agt-test-1"
	mockTimestamp             = "2026-07-11T10:00:00Z"
)

type mockStore struct {
	mu             sync.Mutex
	agents         map[string]map[string]any
	documents      map[string]map[string]any
	sessions       map[string]map[string]any
	messages       map[string][]map[string]any
	memories       map[string][]map[string]any
	prompts        map[string]map[string]any
	promptVersions map[string]map[string]map[string]any
}

func shortID() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func randomID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%x", b[:])
}

func defaultAgent(agentID string) map[string]any {
	return map[string]any{
		"id":          agentID,
		"workspaceId": defaultWorkspaceID,
		"name":        "Support bot",
		"kind":        "unmanaged",
		"promptId":    nil,
		"kbLabels":    map[string]string{},
		"handle":      "a8k2m9x1",
		"description": "Handles customer questions",
		"status":      "active",
		"createdAt":   mockTimestamp,
		"updatedAt":   mockTimestamp,
	}
}

func newMockStore() *mockStore {
	promptID := "prm_customer_support"
	prompt := map[string]any{
		"promptId":    promptID,
		"name":        "Customer Support",
		"handle":      "customer-support",
		"description": "Customer Support Prompt",
	}
	v1 := map[string]any{
		"id": "prv_v1", "name": "Customer Support", "handle": "customer-support",
		"description": "Customer Support Prompt", "version": 1, "type": "text",
		"content": "You are a helpful assistant v1\n{{ctx}}", "config": map[string]any{"model": "gpt3"},
		"labels": []any{}, "commitMessage": "initial", "commitHash": "ba506ac20c11",
		"meta": nil, "status": "active", "production": true,
		"createdAt": mockTimestamp, "updatedAt": mockTimestamp,
	}
	v2 := map[string]any{
		"id": "prv_v2", "name": "Customer Support", "handle": "customer-support",
		"description": "Customer Support Prompt", "version": 2, "type": "text",
		"content": "You are a helpful assistant v2\n{{ctx}}", "config": map[string]any{"model": "gpt3"},
		"labels": []any{"latest"}, "commitMessage": "v2", "commitHash": "ba506ac20c12",
		"meta": nil, "status": "active", "production": false,
		"createdAt": mockTimestamp, "updatedAt": mockTimestamp,
	}
	return &mockStore{
		agents:    map[string]map[string]any{defaultAgentID: defaultAgent(defaultAgentID)},
		documents: map[string]map[string]any{},
		sessions:  map[string]map[string]any{},
		messages:  map[string][]map[string]any{},
		memories:  map[string][]map[string]any{},
		prompts:   map[string]map[string]any{promptID: prompt},
		promptVersions: map[string]map[string]map[string]any{
			promptID: {"prv_v1": v1, "prv_v2": v2},
		},
	}
}

type mockServer struct {
	store  *mockStore
	server *httptest.Server
}

func startMockServer(t *testing.T) string {
	t.Helper()
	ms := &mockServer{store: newMockStore()}
	ms.server = httptest.NewServer(http.HandlerFunc(ms.handle))
	t.Cleanup(ms.server.Close)
	return ms.server.URL
}

func (ms *mockServer) authorized(r *http.Request) bool {
	return r.Header.Get("X-Access-Key") == defaultWorkspaceAccessKey
}

func (ms *mockServer) send(w http.ResponseWriter, status int, data any) {
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func mockListMeta(n int, query url.Values) map[string]any {
	limit := 50
	offset := 0
	if v := query.Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			limit = parsed
		}
	}
	if v := query.Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			offset = parsed
		}
	}
	return map[string]any{"limit": limit, "offset": offset, "total": n}
}

func queryLabels(query url.Values) map[string]string {
	labels := map[string]string{}
	for key, values := range query {
		if key == "id" || key == "limit" || key == "offset" {
			continue
		}
		if len(values) > 0 {
			labels[key] = values[0]
		}
	}
	return labels
}

func labelsEqual(a any, b map[string]string) bool {
	am, ok := a.(map[string]string)
	if !ok {
		if generic, ok := a.(map[string]any); ok {
			am = map[string]string{}
			for k, v := range generic {
				am[k] = stringify(v)
			}
		} else {
			return false
		}
	}
	if len(am) != len(b) {
		return false
	}
	for k, v := range b {
		if am[k] != v {
			return false
		}
	}
	return true
}

func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func (ms *mockServer) findSessionByLabels(query url.Values) map[string]any {
	externalID := query.Get("id")
	labels := queryLabels(query)
	for _, session := range ms.store.sessions {
		if externalID != "" && stringify(session["externalId"]) == externalID {
			return session
		}
		if len(labels) > 0 && labelsEqual(session["labels"], labels) {
			return session
		}
	}
	return nil
}

func (ms *mockServer) handle(w http.ResponseWriter, r *http.Request) {
	ms.store.mu.Lock()
	defer ms.store.mu.Unlock()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		ms.send(w, 500, map[string]any{"errorMessage": err.Error()})
		return
	}
	contentType := r.Header.Get("Content-Type")
	var jsonBody map[string]any
	form := map[string]any{}
	if strings.HasPrefix(contentType, "multipart/form-data") {
		form = parseMultipart(body, contentType)
	} else if len(body) > 0 {
		_ = json.Unmarshal(body, &jsonBody)
	}
	if jsonBody == nil {
		jsonBody = map[string]any{}
	}

	status, data := ms.route(r.Method, r.URL.Path, r.URL.Query(), jsonBody, form, r)
	ms.send(w, status, data)
}

func parseMultipart(body []byte, contentType string) map[string]any {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return map[string]any{}
	}
	boundary := params["boundary"]
	reader := multipart.NewReader(strings.NewReader(string(body)), boundary)
	out := map[string]any{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		name := part.FormName()
		filename := part.FileName()
		content, _ := io.ReadAll(part)
		if filename != "" {
			out[name] = map[string]any{"filename": filename, "content": content}
		} else {
			out[name] = string(content)
		}
		_ = part.Close()
	}
	return out
}

func (ms *mockServer) route(method, path string, query url.Values, body map[string]any, form map[string]any, r *http.Request) (int, any) {
	if method == http.MethodGet && path == "/api/v1/_health" {
		return 200, map[string]any{"status": "ok"}
	}
	if method == http.MethodGet && path == "/api/v1/me" {
		return ms.meResponse(r)
	}

	byNamePrefix := "/api/v1/workspaces/" + defaultWorkspaceID + "/promptsByName/"
	if method == http.MethodGet && strings.HasPrefix(path, byNamePrefix) {
		if !ms.authorized(r) {
			return 401, map[string]any{"errorMessage": "Invalid access key."}
		}
		handle := path[len(byNamePrefix):]
		found := ms.findPromptVersionByName(handle, query.Get("version"))
		if found == nil {
			return 404, map[string]any{"errorMessage": "prompt not found"}
		}
		return 200, found
	}

	agentPrefix := "/api/v1/workspaces/" + defaultWorkspaceID + "/agents/"
	if strings.HasPrefix(path, agentPrefix) {
		return ms.agentRoute(method, path, query, body, r)
	}

	wsPrefix := "/api/v1/workspaces/" + defaultWorkspaceID
	if strings.HasPrefix(path, wsPrefix+"/") {
		status, data := ms.workspaceRoute(method, path, query, body, form, r, wsPrefix)
		if status != 404 {
			return status, data
		}
	}
	return 404, map[string]any{"error": "not found"}
}

func (ms *mockServer) meResponse(r *http.Request) (int, any) {
	accessKey := r.Header.Get("X-Access-Key")
	if accessKey == "" {
		return 403, map[string]any{"errorMessage": "X-Access-Key header is required."}
	}
	if accessKey != defaultWorkspaceAccessKey {
		return 401, map[string]any{"errorMessage": "Invalid access key."}
	}
	return 200, map[string]any{
		"principalType": "access_key",
		"accessKey": map[string]any{
			"id": "wkey_ghi789", "workspaceId": defaultWorkspaceID, "name": "Agent runtime",
			"permissions": []string{"CAN_LIST_AGENTS", "CAN_GET_AGENT"},
			"expiresAt":   "2026-08-01T00:00:00Z",
			"createdAt":   "2026-07-05T08:00:00Z", "updatedAt": "2026-07-05T08:00:00Z",
		},
	}
}

func asInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case float64:
		return int(t)
	case json.Number:
		i, _ := t.Int64()
		return int(i)
	default:
		return 0
	}
}

func values[V any](m map[string]V) []V {
	out := make([]V, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func parseMeta(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	switch t := v.(type) {
	case string:
		out := map[string]any{}
		_ = json.Unmarshal([]byte(t), &out)
		return out
	case map[string]any:
		return t
	default:
		return map[string]any{}
	}
}
