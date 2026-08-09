// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrompt(t *testing.T) {
	t.Run("get latest", func(t *testing.T) {
		client, ctx := testClient(t)
		prompt, err := client.Prompt.GetByName(ctx, "customer-support", "")
		require.NoError(t, err)
		assert.Equal(t, "customer-support", prompt.Handle)
		assert.Equal(t, 2, prompt.Version)
		assert.Equal(t, "You are a helpful assistant v2\n{{ctx}}", prompt.Content)
	})

	t.Run("get with version", func(t *testing.T) {
		client, ctx := testClient(t)
		prompt, err := client.Prompt.GetByName(ctx, "customer-support", "v1")
		require.NoError(t, err)
		assert.Equal(t, 1, prompt.Version)
	})

	t.Run("named versions", func(t *testing.T) {
		client, ctx := testClient(t)
		latest, err := client.Prompt.GetByName(ctx, "customer-support", "latest")
		require.NoError(t, err)
		assert.Equal(t, 2, latest.Version)

		production, err := client.Prompt.GetByName(ctx, "customer-support", "production")
		require.NoError(t, err)
		assert.Equal(t, 1, production.Version)
	})

	t.Run("compile", func(t *testing.T) {
		client, ctx := testClient(t)
		prompt, err := client.Prompt.GetByName(ctx, "customer-support", "")
		require.NoError(t, err)
		compiled, err := prompt.Compile(map[string]string{"ctx": "Ahmed"})
		require.NoError(t, err)
		assert.Equal(t, "You are a helpful assistant v2\nAhmed", compiled)
	})

	t.Run("requires workspace", func(t *testing.T) {
		baseURL := startMockServer(t)
		client := NewClient(WithBaseURL(baseURL), WithAccessKey(defaultWorkspaceAccessKey))
		defer client.Close()
		_, err := client.Prompt.GetByName(context.Background(), "customer-support", "")
		require.Error(t, err)
		assert.Equal(t, "workspace_id is required", err.Error())
	})

	t.Run("crud and versions", func(t *testing.T) {
		client, ctx := testClient(t)
		prod := false
		created, err := client.Prompt.Create(ctx, "Mara Guide", PromptTypeText, "You know Mara Ellison.", PromptWriteOptions{
			Description:   "Answers questions about Mara",
			Config:        map[string]any{"tone": "friendly"},
			CommitMessage: "initial",
			Meta:          map[string]any{"source": "examples"},
			Production:    &prod,
		})
		require.NoError(t, err)
		assert.Equal(t, "mara-guide", created.Handle)
		assert.Equal(t, 1, created.VersionCount)

		fetched, err := client.Prompt.Get(ctx, created.PromptID)
		require.NoError(t, err)
		assert.Equal(t, created.PromptID, fetched.PromptID)

		production := true
		version, err := client.Prompt.CreateVersion(ctx, created.PromptID, PromptTypeText, "You know Mara Ellison well.", PromptWriteOptions{
			CommitMessage: "v2",
			Production:    &production,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, version.Version)
		assert.True(t, version.Production)

		versions, err := client.Prompt.ListVersions(ctx, created.PromptID, 50, 0)
		require.NoError(t, err)
		assert.Equal(t, 2, versions.Total)

		got, err := client.Prompt.GetVersion(ctx, created.PromptID, version.ID)
		require.NoError(t, err)
		assert.Equal(t, "You know Mara Ellison well.", got.Content)

		updated, err := client.Prompt.UpdateVersion(ctx, created.PromptID, version.ID, "You know Mara Ellison very well.", PromptWriteOptions{
			Status:     PromptStatusActive,
			Production: &production,
		})
		require.NoError(t, err)
		assert.Equal(t, "You know Mara Ellison very well.", updated.Content)

		byName, err := client.Prompt.GetByName(ctx, "mara-guide", "production")
		require.NoError(t, err)
		assert.Equal(t, version.ID, byName.ID)

		require.NoError(t, client.Prompt.DeleteVersion(ctx, created.PromptID, version.ID))
		require.NoError(t, client.Prompt.Delete(ctx, created.PromptID))
	})
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
