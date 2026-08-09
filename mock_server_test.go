// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"sort"
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

var agentPathRE = regexp.MustCompile(`^/api/v1/workspaces/([^/]+)/agents/([^/]+)(/.*)?$`)

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
			"expiresAt": "2026-08-01T00:00:00Z",
			"createdAt": "2026-07-05T08:00:00Z", "updatedAt": "2026-07-05T08:00:00Z",
		},
	}
}

func (ms *mockServer) agentObject(agentID, name, description string) map[string]any {
	return map[string]any{
		"id": agentID, "workspaceId": defaultWorkspaceID, "name": name, "kind": "unmanaged",
		"promptId": nil, "kbLabels": map[string]string{}, "handle": shortID(),
		"description": description, "status": "active",
		"createdAt": mockTimestamp, "updatedAt": mockTimestamp,
	}
}

func (ms *mockServer) documentObject(documentID, title, filename string, labels []string, content []byte) map[string]any {
	sum := sha256.Sum256(content)
	size := len(content)
	if size == 0 {
		size = 100
	}
	charCount := len(string(content))
	if charCount == 0 {
		charCount = 80
	}
	if labels == nil {
		labels = []string{}
	}
	return map[string]any{
		"id": documentID, "workspaceId": defaultWorkspaceID, "title": title, "filename": filename,
		"contentType": "text/markdown", "checksum": hex.EncodeToString(sum[:]),
		"size": map[string]any{"value": size, "unit": "bytes"}, "charCount": charCount,
		"labels": labels, "chunkingStrategy": "recursive", "chunkSize": 2000, "chunkOverlap": 400,
		"status": "processing", "createdAt": mockTimestamp, "updatedAt": mockTimestamp,
	}
}

func (ms *mockServer) promptSummary(promptID string) map[string]any {
	prompt := ms.store.prompts[promptID]
	versions := ms.store.promptVersions[promptID]
	return map[string]any{
		"promptId": prompt["promptId"], "name": prompt["name"], "handle": prompt["handle"],
		"description": prompt["description"], "versionCount": len(versions),
	}
}

func (ms *mockServer) promptVersionObject(prompt map[string]any, versionID string, version int, typ, content string, config any, commitMessage, meta any, production bool, labels []any) map[string]any {
	parsedConfig := map[string]any{}
	switch c := config.(type) {
	case string:
		if c != "" {
			_ = json.Unmarshal([]byte(c), &parsedConfig)
		}
	case map[string]any:
		parsedConfig = c
	}
	if labels == nil {
		labels = []any{}
	}
	return map[string]any{
		"id": versionID, "name": prompt["name"], "handle": prompt["handle"],
		"description": prompt["description"], "version": version, "type": typ, "content": content,
		"config": parsedConfig, "labels": labels, "commitMessage": commitMessage,
		"commitHash": shortID() + shortID()[:4], "meta": meta, "status": "active",
		"production": production, "createdAt": mockTimestamp, "updatedAt": mockTimestamp,
	}
}

func (ms *mockServer) findPromptByHandle(handle string) map[string]any {
	for _, prompt := range ms.store.prompts {
		if stringify(prompt["handle"]) == handle {
			return prompt
		}
	}
	return nil
}

func (ms *mockServer) findPromptVersionByName(handle, version string) map[string]any {
	prompt := ms.findPromptByHandle(handle)
	if prompt == nil {
		return nil
	}
	versionsMap := ms.store.promptVersions[stringify(prompt["promptId"])]
	versions := make([]map[string]any, 0, len(versionsMap))
	for _, v := range versionsMap {
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool {
		return asInt(versions[i]["version"]) < asInt(versions[j]["version"])
	})
	if len(versions) == 0 {
		return nil
	}
	if version == "" || version == "latest" {
		return versions[len(versions)-1]
	}
	if version == "production" {
		for _, item := range versions {
			if item["production"] == true {
				return item
			}
		}
		return versions[0]
	}
	number := -1
	if strings.HasPrefix(version, "v") {
		if n, err := strconv.Atoi(version[1:]); err == nil {
			number = n
		}
	} else if n, err := strconv.Atoi(version); err == nil {
		number = n
	} else {
		return nil
	}
	for _, item := range versions {
		if asInt(item["version"]) == number {
			return item
		}
	}
	return nil
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

func (ms *mockServer) workspaceRoute(method, path string, query url.Values, body map[string]any, form map[string]any, r *http.Request, wsPrefix string) (int, any) {
	if path == wsPrefix+"/agents" {
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			agents := values(ms.store.agents)
			return 200, map[string]any{"agents": agents, "_meta": mockListMeta(len(agents), query)}
		}
		if method == http.MethodPost {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			agentID := "agt_" + shortID()
			agent := ms.agentObject(agentID, stringify(body["name"]), stringify(body["description"]))
			ms.store.agents[agentID] = agent
			return 201, agent
		}
	}

	if status, data := ms.promptRoute(method, path, query, body, r, wsPrefix); status != 404 {
		return status, data
	}

	if path == wsPrefix+"/documents" {
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			docs := values(ms.store.documents)
			return 200, map[string]any{"documents": docs, "_meta": mockListMeta(len(docs), query)}
		}
		if method == http.MethodPost {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			fileInfo, _ := form["file"].(map[string]any)
			filename := "upload.md"
			var content []byte
			if fileInfo != nil {
				if f, ok := fileInfo["filename"].(string); ok && f != "" {
					filename = f
				}
				if c, ok := fileInfo["content"].([]byte); ok {
					content = c
				}
			}
			title := "Untitled"
			if t, ok := form["title"].(string); ok && t != "" {
				title = t
			}
			labels := []string{}
			if raw, ok := form["labels"].(string); ok && raw != "" {
				_ = json.Unmarshal([]byte(raw), &labels)
			}
			documentID := "doc_" + shortID()
			doc := ms.documentObject(documentID, title, filename, labels, content)
			ms.store.documents[documentID] = doc
			return 201, doc
		}
	}

	if path == wsPrefix+"/documents/search" {
		if !ms.authorized(r) {
			return 401, map[string]any{"errorMessage": "Invalid access key."}
		}
		labels, _ := body["labels"].(map[string]any)
		labelMap := map[string]string{}
		for k, v := range labels {
			labelMap[k] = stringify(v)
		}
		return 200, map[string]any{"results": []map[string]any{{
			"documentId": "doc_search_1", "chunkId": "chunk_1", "score": 0.87,
			"text": "Result for: " + stringify(body["query"]), "labels": labelMap,
		}}}
	}

	docPrefix := wsPrefix + "/documents/"
	if strings.HasPrefix(path, docPrefix) && method == http.MethodDelete {
		documentID := path[len(docPrefix):]
		if _, ok := ms.store.documents[documentID]; ok {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			delete(ms.store.documents, documentID)
			return 204, nil
		}
	}
	return 404, map[string]any{"errorMessage": "not found"}
}

func values[V any](m map[string]V) []V {
	out := make([]V, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func (ms *mockServer) promptRoute(method, path string, query url.Values, body map[string]any, r *http.Request, wsPrefix string) (int, any) {
	if path == wsPrefix+"/prompts" {
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			prompts := make([]map[string]any, 0, len(ms.store.prompts))
			for id := range ms.store.prompts {
				prompts = append(prompts, ms.promptSummary(id))
			}
			return 200, map[string]any{"prompts": prompts, "_meta": mockListMeta(len(prompts), query)}
		}
		if method == http.MethodPost {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			promptID := "prm_" + shortID()
			handle := strings.ToLower(strings.ReplaceAll(stringify(body["name"]), " ", "-"))
			desc := ""
			if body["description"] != nil {
				desc = stringify(body["description"])
			}
			prompt := map[string]any{"promptId": promptID, "name": body["name"], "handle": handle, "description": desc}
			versionID := "prv_" + shortID()
			version := ms.promptVersionObject(prompt, versionID, 1, stringify(body["type"]), stringify(body["content"]), body["config"], body["commitMessage"], body["meta"], body["production"] == true, []any{"latest"})
			ms.store.prompts[promptID] = prompt
			ms.store.promptVersions[promptID] = map[string]map[string]any{versionID: version}
			return 201, ms.promptSummary(promptID)
		}
	}

	promptPrefix := wsPrefix + "/prompts/"
	if !strings.HasPrefix(path, promptPrefix) {
		return 404, map[string]any{"errorMessage": "not found"}
	}
	remainder := path[len(promptPrefix):]
	parts := strings.Split(remainder, "/")
	promptID := parts[0]
	if _, ok := ms.store.prompts[promptID]; !ok {
		return 404, map[string]any{"errorMessage": "prompt not found"}
	}

	if len(parts) == 1 {
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			return 200, ms.promptSummary(promptID)
		}
		if method == http.MethodDelete {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			delete(ms.store.prompts, promptID)
			delete(ms.store.promptVersions, promptID)
			return 204, nil
		}
		return 405, map[string]any{"errorMessage": "method not allowed"}
	}
	if parts[1] != "versions" {
		return 404, map[string]any{"errorMessage": "not found"}
	}
	versions := ms.store.promptVersions[promptID]
	if versions == nil {
		versions = map[string]map[string]any{}
		ms.store.promptVersions[promptID] = versions
	}
	prompt := ms.store.prompts[promptID]

	if len(parts) == 2 {
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			items := values(versions)
			sort.Slice(items, func(i, j int) bool { return asInt(items[i]["version"]) < asInt(items[j]["version"]) })
			return 200, map[string]any{"versions": items, "_meta": mockListMeta(len(items), query)}
		}
		if method == http.MethodPost {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			next := 0
			for _, v := range versions {
				if n := asInt(v["version"]); n > next {
					next = n
				}
			}
			next++
			versionID := "prv_" + shortID()
			production := body["production"] == true
			if production {
				for _, item := range versions {
					item["production"] = false
				}
			}
			for _, item := range versions {
				labels, _ := item["labels"].([]any)
				filtered := []any{}
				for _, label := range labels {
					if label != "latest" {
						filtered = append(filtered, label)
					}
				}
				item["labels"] = filtered
			}
			version := ms.promptVersionObject(prompt, versionID, next, stringify(body["type"]), stringify(body["content"]), body["config"], body["commitMessage"], body["meta"], production, []any{"latest"})
			versions[versionID] = version
			return 201, version
		}
		return 405, map[string]any{"errorMessage": "method not allowed"}
	}

	versionID := parts[2]
	current, ok := versions[versionID]
	if !ok {
		return 404, map[string]any{"errorMessage": "version not found"}
	}
	if method == http.MethodGet {
		if !ms.authorized(r) {
			return 401, map[string]any{"errorMessage": "Invalid access key."}
		}
		return 200, current
	}
	if method == http.MethodPut {
		if !ms.authorized(r) {
			return 403, map[string]any{"errorMessage": "Write requires user API key."}
		}
		if body["type"] != nil {
			current["type"] = body["type"]
		}
		current["content"] = body["content"]
		if body["config"] != nil {
			switch c := body["config"].(type) {
			case string:
				parsed := map[string]any{}
				if c != "" {
					_ = json.Unmarshal([]byte(c), &parsed)
				}
				current["config"] = parsed
			default:
				current["config"] = c
			}
		}
		if body["commitMessage"] != nil {
			current["commitMessage"] = body["commitMessage"]
		}
		if body["meta"] != nil {
			current["meta"] = body["meta"]
		}
		if body["status"] != nil {
			current["status"] = body["status"]
		}
		if body["production"] == true {
			for _, item := range versions {
				item["production"] = false
			}
			current["production"] = true
		}
		current["updatedAt"] = mockTimestamp
		return 200, current
	}
	if method == http.MethodDelete {
		if !ms.authorized(r) {
			return 403, map[string]any{"errorMessage": "Write requires user API key."}
		}
		delete(versions, versionID)
		return 204, nil
	}
	return 405, map[string]any{"errorMessage": "method not allowed"}
}

func (ms *mockServer) agentRoute(method, path string, query url.Values, body map[string]any, r *http.Request) (int, any) {
	match := agentPathRE.FindStringSubmatch(path)
	if match == nil {
		return 404, map[string]any{"errorMessage": "not found"}
	}
	workspaceID, agentID, suffix := match[1], match[2], match[3]
	if workspaceID != defaultWorkspaceID {
		return 404, map[string]any{"errorMessage": "workspace not found"}
	}

	if suffix == "" {
		agent, ok := ms.store.agents[agentID]
		if !ok {
			return 404, map[string]any{"errorMessage": "Agent not found."}
		}
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			return 200, agent
		}
		if method == http.MethodPut {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			agent["name"] = body["name"]
			agent["description"] = body["description"]
			agent["updatedAt"] = mockTimestamp
			return 200, agent
		}
		if method == http.MethodDelete {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			delete(ms.store.agents, agentID)
			return 204, nil
		}
	}

	if suffix == "/sessions" {
		if method == http.MethodPost {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			externalID := query.Get("id")
			labels := queryLabels(query)
			if externalID == "" && len(labels) == 0 {
				return 400, map[string]any{"errorMessage": "id or labels required"}
			}
			if externalID != "" && ms.findSessionByLabels(url.Values{"id": {externalID}}) != nil {
				return 409, map[string]any{"errorMessage": "Session already exists."}
			}
			sessionID := "ses_" + shortID()
			ext := externalID
			if ext == "" {
				ext = randomID()
			}
			title := ""
			if body["title"] != nil {
				title = stringify(body["title"])
			}
			session := map[string]any{
				"id": sessionID, "externalId": ext, "workspaceId": workspaceID, "agentId": agentID,
				"title": title, "status": "active", "labels": labels, "meta": map[string]any{},
				"createdAt": mockTimestamp, "updatedAt": mockTimestamp,
			}
			ms.store.sessions[sessionID] = session
			ms.store.messages[sessionID] = []map[string]any{}
			ms.store.memories[sessionID] = []map[string]any{}
			return 201, session
		}
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			sessions := []map[string]any{}
			for _, s := range ms.store.sessions {
				if stringify(s["agentId"]) != agentID {
					continue
				}
				sessions = append(sessions, s)
			}
			if query.Get("id") != "" {
				filtered := []map[string]any{}
				for _, s := range sessions {
					if stringify(s["externalId"]) == query.Get("id") {
						filtered = append(filtered, s)
					}
				}
				sessions = filtered
			}
			labels := queryLabels(query)
			if len(labels) > 0 {
				filtered := []map[string]any{}
				for _, s := range sessions {
					if labelsEqual(s["labels"], labels) {
						filtered = append(filtered, s)
					}
				}
				sessions = filtered
			}
			return 200, map[string]any{"sessions": sessions, "_meta": mockListMeta(len(sessions), query)}
		}
	}

	if suffix == "/sessions/by-labels" {
		if !ms.authorized(r) {
			return 401, map[string]any{"errorMessage": "Invalid access key."}
		}
		session := ms.findSessionByLabels(query)
		if session == nil {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		if method == http.MethodGet {
			return 200, session
		}
		if method == http.MethodPut {
			if body["title"] != nil {
				session["title"] = body["title"]
			}
			if body["labels"] != nil {
				session["labels"] = body["labels"]
			}
			session["updatedAt"] = mockTimestamp
			return 200, session
		}
		if method == http.MethodDelete {
			id := stringify(session["id"])
			delete(ms.store.sessions, id)
			delete(ms.store.messages, id)
			delete(ms.store.memories, id)
			return 204, nil
		}
	}

	if m := regexp.MustCompile(`^/sessions/([^/]+)$`).FindStringSubmatch(suffix); m != nil && method == http.MethodGet {
		if !ms.authorized(r) {
			return 401, map[string]any{"errorMessage": "Invalid access key."}
		}
		session, ok := ms.store.sessions[m[1]]
		if !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		return 200, session
	}

	return ms.sessionNestedRoute(method, suffix, body, r)
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

func (ms *mockServer) sessionNestedRoute(method, suffix string, body map[string]any, r *http.Request) (int, any) {
	if m := regexp.MustCompile(`^/sessions/([^/]+)/messages/batch$`).FindStringSubmatch(suffix); m != nil {
		sessionID := m[1]
		if _, ok := ms.store.sessions[sessionID]; !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		if method == http.MethodPost {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			created := []map[string]any{}
			items, _ := body["messages"].([]any)
			for _, raw := range items {
				item, _ := raw.(map[string]any)
				message := map[string]any{
					"id": "msg_" + shortID(), "sessionId": sessionID, "role": item["role"],
					"content": item["content"], "meta": parseMeta(item["meta"]), "createdAt": mockTimestamp,
				}
				ms.store.messages[sessionID] = append(ms.store.messages[sessionID], message)
				created = append(created, message)
			}
			return 201, map[string]any{"messages": created}
		}
		if method == http.MethodDelete {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			ids := map[string]struct{}{}
			for _, raw := range body["ids"].([]any) {
				ids[stringify(raw)] = struct{}{}
			}
			filtered := []map[string]any{}
			for _, msg := range ms.store.messages[sessionID] {
				if _, drop := ids[stringify(msg["id"])]; !drop {
					filtered = append(filtered, msg)
				}
			}
			ms.store.messages[sessionID] = filtered
			return 204, nil
		}
	}

	if m := regexp.MustCompile(`^/sessions/([^/]+)/messages/search$`).FindStringSubmatch(suffix); m != nil && method == http.MethodPost {
		if !ms.authorized(r) {
			return 401, map[string]any{"errorMessage": "Invalid access key."}
		}
		sessionID := m[1]
		if _, ok := ms.store.sessions[sessionID]; !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		queryText := stringify(body["query"])
		limit := asInt(body["limit"])
		if limit == 0 {
			limit = 10
		}
		if queryText == "" || limit < 1 || limit > 100 {
			return 400, map[string]any{"errorMessage": "Invalid search request."}
		}
		matches := []map[string]any{}
		q := strings.ToLower(queryText)
		for _, item := range ms.store.messages[sessionID] {
			if strings.Contains(strings.ToLower(stringify(item["content"])), q) {
				matches = append(matches, map[string]any{
					"id": item["id"], "role": item["role"], "score": 0.91, "text": item["content"],
				})
			}
		}
		if len(matches) > limit {
			matches = matches[:limit]
		}
		return 200, map[string]any{"results": matches}
	}

	if m := regexp.MustCompile(`^/sessions/([^/]+)/messages$`).FindStringSubmatch(suffix); m != nil {
		sessionID := m[1]
		if _, ok := ms.store.sessions[sessionID]; !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			items := ms.store.messages[sessionID]
			return 200, map[string]any{"messages": items, "_meta": mockListMeta(len(items), url.Values{})}
		}
		if method == http.MethodPost {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			message := map[string]any{
				"id": "msg_" + shortID(), "sessionId": sessionID, "role": body["role"],
				"content": body["content"], "meta": parseMeta(body["meta"]), "createdAt": mockTimestamp,
			}
			ms.store.messages[sessionID] = append(ms.store.messages[sessionID], message)
			return 201, message
		}
	}

	if m := regexp.MustCompile(`^/sessions/([^/]+)/messages/([^/]+)$`).FindStringSubmatch(suffix); m != nil {
		sessionID, messageID := m[1], m[2]
		if _, ok := ms.store.sessions[sessionID]; !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		items := ms.store.messages[sessionID]
		var message map[string]any
		idx := -1
		for i, item := range items {
			if stringify(item["id"]) == messageID {
				message = item
				idx = i
				break
			}
		}
		if message == nil {
			return 404, map[string]any{"errorMessage": "Message not found."}
		}
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			return 200, message
		}
		if method == http.MethodPut {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			if body["role"] != nil {
				message["role"] = body["role"]
			}
			message["content"] = body["content"]
			if body["meta"] != nil {
				message["meta"] = parseMeta(body["meta"])
			}
			return 200, message
		}
		if method == http.MethodDelete {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			ms.store.messages[sessionID] = append(items[:idx], items[idx+1:]...)
			return 204, nil
		}
	}

	if m := regexp.MustCompile(`^/sessions/([^/]+)/memories/batch$`).FindStringSubmatch(suffix); m != nil {
		sessionID := m[1]
		if _, ok := ms.store.sessions[sessionID]; !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		if method == http.MethodPost {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			created := []map[string]any{}
			items, _ := body["memories"].([]any)
			for _, raw := range items {
				item, _ := raw.(map[string]any)
				memory := map[string]any{
					"id": "mem_" + shortID(), "sessionId": sessionID, "kind": item["kind"],
					"content": item["content"], "meta": parseMeta(item["meta"]),
					"createdAt": mockTimestamp, "updatedAt": mockTimestamp,
				}
				ms.store.memories[sessionID] = append(ms.store.memories[sessionID], memory)
				created = append(created, memory)
			}
			return 201, map[string]any{"memories": created}
		}
		if method == http.MethodDelete {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			ids := map[string]struct{}{}
			for _, raw := range body["ids"].([]any) {
				ids[stringify(raw)] = struct{}{}
			}
			filtered := []map[string]any{}
			for _, mem := range ms.store.memories[sessionID] {
				if _, drop := ids[stringify(mem["id"])]; !drop {
					filtered = append(filtered, mem)
				}
			}
			ms.store.memories[sessionID] = filtered
			return 204, nil
		}
	}

	if m := regexp.MustCompile(`^/sessions/([^/]+)/memories/search$`).FindStringSubmatch(suffix); m != nil && method == http.MethodPost {
		if !ms.authorized(r) {
			return 401, map[string]any{"errorMessage": "Invalid access key."}
		}
		sessionID := m[1]
		if _, ok := ms.store.sessions[sessionID]; !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		queryText := stringify(body["query"])
		limit := asInt(body["limit"])
		if limit == 0 {
			limit = 10
		}
		if queryText == "" || limit < 1 || limit > 100 {
			return 400, map[string]any{"errorMessage": "Invalid search request."}
		}
		matches := []map[string]any{}
		q := strings.ToLower(queryText)
		for _, item := range ms.store.memories[sessionID] {
			if strings.Contains(strings.ToLower(stringify(item["content"])), q) {
				matches = append(matches, map[string]any{
					"id": item["id"], "kind": item["kind"], "score": 0.88, "text": item["content"],
				})
			}
		}
		if len(matches) > limit {
			matches = matches[:limit]
		}
		return 200, map[string]any{"results": matches}
	}

	if m := regexp.MustCompile(`^/sessions/([^/]+)/memories$`).FindStringSubmatch(suffix); m != nil {
		sessionID := m[1]
		if _, ok := ms.store.sessions[sessionID]; !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			items := ms.store.memories[sessionID]
			return 200, map[string]any{"memories": items, "_meta": mockListMeta(len(items), url.Values{})}
		}
		if method == http.MethodPost {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			memory := map[string]any{
				"id": "mem_" + shortID(), "sessionId": sessionID, "kind": body["kind"],
				"content": body["content"], "meta": parseMeta(body["meta"]),
				"createdAt": mockTimestamp, "updatedAt": mockTimestamp,
			}
			ms.store.memories[sessionID] = append(ms.store.memories[sessionID], memory)
			return 201, memory
		}
	}

	if m := regexp.MustCompile(`^/sessions/([^/]+)/memories/([^/]+)$`).FindStringSubmatch(suffix); m != nil {
		sessionID, memoryID := m[1], m[2]
		if _, ok := ms.store.sessions[sessionID]; !ok {
			return 404, map[string]any{"errorMessage": "Session not found."}
		}
		items := ms.store.memories[sessionID]
		var memory map[string]any
		idx := -1
		for i, item := range items {
			if stringify(item["id"]) == memoryID {
				memory = item
				idx = i
				break
			}
		}
		if memory == nil {
			return 404, map[string]any{"errorMessage": "Memory not found."}
		}
		if method == http.MethodGet {
			if !ms.authorized(r) {
				return 401, map[string]any{"errorMessage": "Invalid access key."}
			}
			return 200, memory
		}
		if method == http.MethodPut {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			if body["kind"] != nil {
				memory["kind"] = body["kind"]
			}
			memory["content"] = body["content"]
			if body["meta"] != nil {
				memory["meta"] = parseMeta(body["meta"])
			}
			memory["updatedAt"] = mockTimestamp
			return 200, memory
		}
		if method == http.MethodDelete {
			if !ms.authorized(r) {
				return 403, map[string]any{"errorMessage": "Write requires user API key."}
			}
			ms.store.memories[sessionID] = append(items[:idx], items[idx+1:]...)
			return 204, nil
		}
	}

	return 404, map[string]any{"errorMessage": "not found"}
}
