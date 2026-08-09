// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

// MemoryKind is the kind of a session memory.
type MemoryKind string

const (
	MemoryKindSummary    MemoryKind = "summary"
	MemoryKindFact       MemoryKind = "fact"
	MemoryKindPreference MemoryKind = "preference"
)

// MessageRole is the role of a session message.
type MessageRole string

const (
	MessageRoleSystem    MessageRole = "system"
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
)

// PromptType is the type of a prompt version.
type PromptType string

const (
	PromptTypeText PromptType = "text"
	PromptTypeChat PromptType = "chat"
)

// PromptStatus is the lifecycle status of a prompt version.
type PromptStatus string

const (
	PromptStatusActive   PromptStatus = "active"
	PromptStatusArchived PromptStatus = "archived"
)

// FileBytes is an in-memory file upload.
type FileBytes struct {
	Filename    string
	Content     []byte
	ContentType string
}

// MessageInput is a message create payload.
type MessageInput struct {
	Role    MessageRole    `json:"role"`
	Content string         `json:"content"`
	Meta    map[string]any `json:"meta,omitempty"`
}

// MemoryInput is a memory create payload.
type MemoryInput struct {
	Kind    MemoryKind     `json:"kind"`
	Content string         `json:"content"`
	Meta    map[string]any `json:"meta,omitempty"`
}

// Agent is a workspace agent.
type Agent struct {
	ID          string            `json:"id"`
	WorkspaceID string            `json:"workspaceId"`
	Name        string            `json:"name"`
	Kind        string            `json:"kind"`
	PromptID    *string           `json:"promptId"`
	KBLabels    map[string]string `json:"kbLabels"`
	Handle      string            `json:"handle"`
	Description string            `json:"description"`
	Status      string            `json:"status"`
	CreatedAt   string            `json:"createdAt"`
	UpdatedAt   string            `json:"updatedAt"`
}

// AgentList is a paginated agent list.
type AgentList struct {
	Agents []Agent `json:"agents"`
	Limit  int     `json:"-"`
	Offset int     `json:"-"`
	Total  int     `json:"-"`
}

type agentListResponse struct {
	Agents []Agent `json:"agents"`
	Meta   listMeta `json:"_meta"`
}

// Session is an agent session.
type Session struct {
	ID          string            `json:"id"`
	ExternalID  string            `json:"externalId"`
	WorkspaceID string            `json:"workspaceId"`
	AgentID     string            `json:"agentId"`
	Title       string            `json:"title"`
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Meta        map[string]any    `json:"meta"`
	CreatedAt   string            `json:"createdAt"`
	UpdatedAt   string            `json:"updatedAt"`
}

// SessionList is a paginated session list.
type SessionList struct {
	Sessions []Session `json:"sessions"`
	Limit    int       `json:"-"`
	Offset   int       `json:"-"`
	Total    int       `json:"-"`
}

type sessionListResponse struct {
	Sessions []Session `json:"sessions"`
	Meta     listMeta  `json:"_meta"`
}

// Message is a session message.
type Message struct {
	ID        string         `json:"id"`
	SessionID string         `json:"sessionId"`
	Role      MessageRole    `json:"role"`
	Content   string         `json:"content"`
	Meta      map[string]any `json:"meta"`
	CreatedAt string         `json:"createdAt"`
}

// MessageList is a paginated message list.
type MessageList struct {
	Messages []Message `json:"messages"`
	Limit    int       `json:"-"`
	Offset   int       `json:"-"`
	Total    int       `json:"-"`
}

type messageListResponse struct {
	Messages []Message `json:"messages"`
	Meta     listMeta  `json:"_meta"`
}

// MessageSearchHit is a message search result.
type MessageSearchHit struct {
	ID    string      `json:"id"`
	Role  MessageRole `json:"role"`
	Score float64     `json:"score"`
	Text  string      `json:"text"`
}

// MessageSearchResults holds message search hits.
type MessageSearchResults struct {
	Results []MessageSearchHit `json:"results"`
}

// Memory is a session memory.
type Memory struct {
	ID        string         `json:"id"`
	SessionID string         `json:"sessionId"`
	Kind      MemoryKind     `json:"kind"`
	Content   string         `json:"content"`
	Meta      map[string]any `json:"meta"`
	CreatedAt string         `json:"createdAt"`
	UpdatedAt string         `json:"updatedAt"`
}

// MemoryList is a paginated memory list.
type MemoryList struct {
	Memories []Memory `json:"memories"`
	Limit    int      `json:"-"`
	Offset   int      `json:"-"`
	Total    int      `json:"-"`
}

type memoryListResponse struct {
	Memories []Memory `json:"memories"`
	Meta     listMeta `json:"_meta"`
}

// MemorySearchHit is a memory search result.
type MemorySearchHit struct {
	ID    string     `json:"id"`
	Kind  MemoryKind `json:"kind"`
	Score float64    `json:"score"`
	Text  string     `json:"text"`
}

// MemorySearchResults holds memory search hits.
type MemorySearchResults struct {
	Results []MemorySearchHit `json:"results"`
}

// DocumentSize is a document size value.
type DocumentSize struct {
	Value int    `json:"value"`
	Unit  string `json:"unit"`
}

// Document is a knowledge-base document.
type Document struct {
	ID               string       `json:"id"`
	WorkspaceID      string       `json:"workspaceId"`
	Title            string       `json:"title"`
	Filename         string       `json:"filename"`
	ContentType      string       `json:"contentType"`
	Checksum         string       `json:"checksum"`
	Size             DocumentSize `json:"size"`
	CharCount        int          `json:"charCount"`
	Labels           []string     `json:"labels"`
	ChunkingStrategy string       `json:"chunkingStrategy"`
	ChunkSize        int          `json:"chunkSize"`
	ChunkOverlap     int          `json:"chunkOverlap"`
	Status           string       `json:"status"`
	CreatedAt        string       `json:"createdAt"`
	UpdatedAt        string       `json:"updatedAt"`
}

// DocumentList is a paginated document list.
type DocumentList struct {
	Documents []Document `json:"documents"`
	Limit     int        `json:"-"`
	Offset    int        `json:"-"`
	Total     int        `json:"-"`
}

type documentListResponse struct {
	Documents []Document `json:"documents"`
	Meta      listMeta   `json:"_meta"`
}

// SearchHit is a document search result.
type SearchHit struct {
	DocumentID string            `json:"documentId"`
	ChunkID    string            `json:"chunkId"`
	Score      float64           `json:"score"`
	Text       string            `json:"text"`
	Labels     map[string]string `json:"labels"`
}

// SearchResults holds document search hits.
type SearchResults struct {
	Results []SearchHit `json:"results"`
}

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

// AccessKeyPrincipal is returned by /api/v1/me for access keys.
type AccessKeyPrincipal struct {
	PrincipalType string        `json:"principalType"`
	AccessKey     AccessKeyInfo `json:"accessKey"`
}

// MePrincipal is currently always an access-key principal.
type MePrincipal = AccessKeyPrincipal

// PromptInfo is a prompt summary from list/create/get-by-id.
type PromptInfo struct {
	PromptID     string `json:"promptId"`
	Name         string `json:"name"`
	Handle       string `json:"handle"`
	Description  string `json:"description"`
	VersionCount int    `json:"versionCount"`
}

// PromptList is a paginated prompt list.
type PromptList struct {
	Prompts []PromptInfo `json:"prompts"`
	Limit   int          `json:"-"`
	Offset  int          `json:"-"`
	Total   int          `json:"-"`
}

type promptListResponse struct {
	Prompts []PromptInfo `json:"prompts"`
	Meta    listMeta     `json:"_meta"`
}

// Prompt is a prompt version returned by the API.
type Prompt struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Handle        string         `json:"handle"`
	Description   string         `json:"description"`
	Version       int            `json:"version"`
	Type          string         `json:"type"`
	Content       string         `json:"content"`
	CommitHash    string         `json:"commitHash"`
	Status        string         `json:"status"`
	Production    bool           `json:"production"`
	CreatedAt     string         `json:"createdAt"`
	UpdatedAt     string         `json:"updatedAt"`
	Config        map[string]any `json:"config"`
	Labels        []string       `json:"labels"`
	CommitMessage *string        `json:"commitMessage"`
	Meta          *string        `json:"meta"`
}

// PromptVersion is a backward-compatible alias for Prompt.
type PromptVersion = Prompt

// PromptVersionList is a paginated prompt version list.
type PromptVersionList struct {
	Versions []Prompt `json:"versions"`
	Limit    int      `json:"-"`
	Offset   int      `json:"-"`
	Total    int      `json:"-"`
}

type promptVersionListResponse struct {
	Versions []Prompt `json:"versions"`
	Meta     listMeta `json:"_meta"`
}

type listMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type messageBatchResponse struct {
	Messages []Message `json:"messages"`
}

type memoryBatchResponse struct {
	Memories []Memory `json:"memories"`
}
