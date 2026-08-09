// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package exampleutil

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Actx0/Gctx0"
)

const (
	AccessKey   = "227fc70d-151c-4a7f-85e2-20ef147cbcc1"
	WorkspaceID = "adae803a-5b20-41c7-bd9b-304792bccabe"
	BaseURL     = "https://actx0.com"

	OpenRouterURL = "https://openrouter.ai/api/v1/chat/completions"
	DefaultModel  = "anthropic/claude-sonnet-4.6"
)

var DocLabels = map[string]string{"source": "docs", "team": "platform-team"}

// Setup holds resources created for an interactive example.
type Setup struct {
	AgentID           string
	PromptID          string
	System            string
	SessionID         string
	SessionExternalID string
}

// NewClient returns a configured Actx0 client for examples.
func NewClient() *gctx0.Client {
	return gctx0.NewClient(
		gctx0.WithAccessKey(AccessKey),
		gctx0.WithWorkspaceId(WorkspaceID),
		gctx0.WithBaseURL(BaseURL),
	)
}

// Bootstrap creates a prompt, agent, and session.
func Bootstrap(ctx context.Context, client *gctx0.Client, agentName, agentDescription, promptName, promptContent, sessionExternalID, sessionTitle string) (*Setup, error) {
	production := true
	promptInfo, err := client.Prompt.Create(ctx, promptName, gctx0.PromptTypeText, promptContent, gctx0.PromptWriteOptions{
		CommitMessage: "initial",
		Production:    &production,
	})
	if err != nil {
		return nil, err
	}
	prompt, err := client.Prompt.GetByName(ctx, promptInfo.Handle, "")
	if err != nil {
		return nil, err
	}
	agent, err := client.Agent.Create(ctx, agentName, agentDescription)
	if err != nil {
		return nil, err
	}
	session, err := client.Session.Create(ctx, agent.ID, gctx0.SessionLookup{ExternalID: sessionExternalID}, sessionTitle)
	if err != nil {
		return nil, err
	}
	return &Setup{
		AgentID:           agent.ID,
		PromptID:          promptInfo.PromptID,
		System:            prompt.Content,
		SessionID:         session.ID,
		SessionExternalID: sessionExternalID,
	}, nil
}

// Teardown deletes example resources.
func Teardown(ctx context.Context, client *gctx0.Client, setup *Setup) {
	_ = client.Session.Delete(ctx, setup.AgentID, gctx0.SessionLookup{ExternalID: setup.SessionExternalID})
	_ = client.Agent.Delete(ctx, setup.AgentID)
	_ = client.Prompt.Delete(ctx, setup.PromptID)
}

// RAGContext searches docs and formats hits.
func RAGContext(ctx context.Context, client *gctx0.Client, query string, limit int) (string, error) {
	results, err := client.Document.Search(ctx, query, DocLabels, limit)
	if err != nil {
		return "", err
	}
	return FormatHits(results.Results, limit), nil
}

// HistoryFromMessages converts stored messages to chat turns.
func HistoryFromMessages(messages []gctx0.Message) []map[string]string {
	out := make([]map[string]string, 0, len(messages))
	for _, m := range messages {
		out = append(out, map[string]string{"role": string(m.Role), "content": m.Content})
	}
	return out
}

// HistoryFromMessageHits converts message search hits to chat turns.
func HistoryFromMessageHits(hits []gctx0.MessageSearchHit) []map[string]string {
	out := make([]map[string]string, 0, len(hits))
	for _, hit := range hits {
		out = append(out, map[string]string{"role": string(hit.Role), "content": hit.Text})
	}
	return out
}

// HistoryFromMemories converts memories to a single assistant turn.
func HistoryFromMemories(memories []gctx0.Memory) []map[string]string {
	if len(memories) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("Here is what I remember:\n")
	for _, m := range memories {
		fmt.Fprintf(&b, "- [%s] %s\n", m.Kind, m.Content)
	}
	return []map[string]string{{"role": "assistant", "content": strings.TrimRight(b.String(), "\n")}}
}

// HistoryFromMemoryHits converts memory search hits to a single assistant turn.
func HistoryFromMemoryHits(hits []gctx0.MemorySearchHit) []map[string]string {
	if len(hits) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("Here is what I remember:\n")
	for _, hit := range hits {
		fmt.Fprintf(&b, "- [%s] %s\n", hit.Kind, hit.Text)
	}
	return []map[string]string{{"role": "assistant", "content": strings.TrimRight(b.String(), "\n")}}
}

// FormatHits formats document search hits.
func FormatHits(hits []gctx0.SearchHit, limit int) string {
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	if len(hits) == 0 {
		return ""
	}
	parts := make([]string, 0, len(hits))
	for i, hit := range hits {
		parts = append(parts, fmt.Sprintf("[%d] %s", i+1, hit.Text))
	}
	return strings.Join(parts, "\n\n")
}

func buildMessages(system, user, context string, history []map[string]string) []map[string]string {
	if context != "" {
		user = "Context:\n" + context + "\n\nQuestion: " + user
	}
	messages := []map[string]string{{"role": "system", "content": system}}
	messages = append(messages, history...)
	messages = append(messages, map[string]string{"role": "user", "content": user})
	return messages
}

// Ask streams a reply from OpenRouter using history + context.
func Ask(system, user string, history []map[string]string, context string) (string, map[string]any, error) {
	messages := buildMessages(system, user, context, history)
	fmt.Printf("[ctx] history=%d (prior turns) sending=%d (system + history + current user)\n", len(history), len(messages))
	reply, usage, err := StreamResponse(messages)
	if err != nil {
		return "", nil, err
	}
	if usage != nil {
		fmt.Printf("[tokens] prompt=%v completion=%v total=%v\n", usage["prompt_tokens"], usage["completion_tokens"], usage["total_tokens"])
	}
	return reply, usage, nil
}

// StreamResponse streams a chat completion from OpenRouter.
func StreamResponse(messages []map[string]string) (string, map[string]any, error) {
	apiKey := os.Getenv("OPENROUTER_KEY")
	payload := map[string]any{
		"model":          DefaultModel,
		"messages":       messages,
		"stream":         true,
		"stream_options": map[string]any{"include_usage": true},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}
	req, err := http.NewRequest(http.MethodPost, OpenRouterURL, strings.NewReader(string(body)))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/Actx0/Gctx0")
	req.Header.Set("X-Title", "gctx0 examples")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		detail, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("OpenRouter error %d: %s", resp.StatusCode, detail)
	}

	fmt.Print("agent> ")
	var parts []string
	var usage map[string]any
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			break
		}
		var chunk map[string]any
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if u, ok := chunk["usage"].(map[string]any); ok {
			usage = u
		}
		choices, _ := chunk["choices"].([]any)
		if len(choices) == 0 {
			continue
		}
		choice, _ := choices[0].(map[string]any)
		delta, _ := choice["delta"].(map[string]any)
		if content, ok := delta["content"].(string); ok && content != "" {
			fmt.Print(content)
			parts = append(parts, content)
		}
	}
	fmt.Print("\n\n")
	return strings.Join(parts, ""), usage, scanner.Err()
}

// ChatLoop runs a simple stdin chat until quit/exit.
func ChatLoop(handle func(text string) error) error {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("you> ")
		text, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		text = strings.TrimSpace(text)
		if text == "" || strings.EqualFold(text, "quit") || strings.EqualFold(text, "exit") {
			return nil
		}
		if err := handle(text); err != nil {
			return err
		}
	}
}
