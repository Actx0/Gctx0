// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnitPageParams(t *testing.T) {
	t.Run("builds limit and offset", func(t *testing.T) {
		got := PageParams(50, 10)
		assert.Equal(t, map[string]string{"limit": "50", "offset": "10"}, got)
	})

	t.Run("zero values", func(t *testing.T) {
		got := PageParams(0, 0)
		assert.Equal(t, map[string]string{"limit": "0", "offset": "0"}, got)
	})
}

func TestUnitLabelParams(t *testing.T) {
	t.Run("external id only", func(t *testing.T) {
		got, err := LabelParams("thread-1", nil)
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"id": "thread-1"}, got)
	})

	t.Run("labels only", func(t *testing.T) {
		got, err := LabelParams("", map[string]string{"userId": "u-1", "channel": "web"})
		assert.NoError(t, err)
		assert.Equal(t, "u-1", got["userId"])
		assert.Equal(t, "web", got["channel"])
	})

	t.Run("external id and labels", func(t *testing.T) {
		got, err := LabelParams("thread-1", map[string]string{"userId": "u-1"})
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"id": "thread-1", "userId": "u-1"}, got)
	})

	t.Run("rejects reserved keys", func(t *testing.T) {
		for _, key := range []string{"id", "limit", "offset"} {
			_, err := LabelParams("", map[string]string{key: "x"})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "reserved query key")
		}
	})
}

func TestUnitWithMeta(t *testing.T) {
	t.Run("nil meta leaves body unchanged", func(t *testing.T) {
		body := map[string]string{"content": "hello"}
		got := WithMeta(body, nil)
		assert.Equal(t, map[string]string{"content": "hello"}, got)
		_, ok := got["meta"]
		assert.False(t, ok)
	})

	t.Run("encodes meta as json string", func(t *testing.T) {
		got := WithMeta(map[string]string{"content": "hello"}, map[string]any{"source": "sdk"})
		assert.Equal(t, "hello", got["content"])
		assert.JSONEq(t, `{"source":"sdk"}`, got["meta"])
	})
}

func TestUnitEncodeMessageItem(t *testing.T) {
	t.Run("without meta", func(t *testing.T) {
		got := EncodeMessageItem(MessageInput{Role: MessageRoleUser, Content: "Hi"})
		assert.Equal(t, map[string]string{"role": "user", "content": "Hi"}, got)
	})

	t.Run("with meta", func(t *testing.T) {
		got := EncodeMessageItem(MessageInput{
			Role:    MessageRoleAssistant,
			Content: "Hello",
			Meta:    map[string]any{"source": "import"},
		})
		assert.Equal(t, "assistant", got["role"])
		assert.Equal(t, "Hello", got["content"])
		assert.JSONEq(t, `{"source":"import"}`, got["meta"])
	})
}

func TestUnitEncodeMemoryItem(t *testing.T) {
	t.Run("without meta", func(t *testing.T) {
		got := EncodeMemoryItem(MemoryInput{Kind: MemoryKindFact, Content: "Prefers dark mode"})
		assert.Equal(t, map[string]string{"kind": "fact", "content": "Prefers dark mode"}, got)
	})

	t.Run("with meta", func(t *testing.T) {
		got := EncodeMemoryItem(MemoryInput{
			Kind:    MemoryKindPreference,
			Content: "Quiet hours",
			Meta:    map[string]any{"source": "import"},
		})
		assert.Equal(t, "preference", got["kind"])
		assert.JSONEq(t, `{"source":"import"}`, got["meta"])
	})
}

func TestUnitEncodeUpdateBody(t *testing.T) {
	t.Run("content and fields", func(t *testing.T) {
		got := EncodeUpdateBody("updated", nil, map[string]string{"role": "user", "empty": ""})
		assert.Equal(t, map[string]string{"content": "updated", "role": "user"}, got)
	})

	t.Run("with meta", func(t *testing.T) {
		got := EncodeUpdateBody("updated", map[string]any{"k": "v"}, nil)
		assert.Equal(t, "updated", got["content"])
		assert.JSONEq(t, `{"k":"v"}`, got["meta"])
	})
}

func TestUnitMessageBatchPayload(t *testing.T) {
	t.Run("encodes messages", func(t *testing.T) {
		got := MessageBatchPayload([]MessageInput{
			{Role: MessageRoleUser, Content: "Hi", Meta: map[string]any{"source": "import"}},
		})
		messages, ok := got["messages"].([]map[string]string)
		assert.True(t, ok)
		assert.Len(t, messages, 1)
		assert.Equal(t, "user", messages[0]["role"])
		assert.JSONEq(t, `{"source":"import"}`, messages[0]["meta"])
	})
}

func TestUnitMemoryBatchPayload(t *testing.T) {
	t.Run("encodes memories", func(t *testing.T) {
		got := MemoryBatchPayload([]MemoryInput{
			{Kind: MemoryKindFact, Content: "Prefers dark mode"},
		})
		memories, ok := got["memories"].([]map[string]string)
		assert.True(t, ok)
		assert.Len(t, memories, 1)
		assert.Equal(t, "fact", memories[0]["kind"])
		assert.Equal(t, "Prefers dark mode", memories[0]["content"])
	})
}

func TestUnitEncodeJSONField(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		value, ok, err := EncodeJSONField(nil)
		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Empty(t, value)
	})

	t.Run("string passthrough", func(t *testing.T) {
		value, ok, err := EncodeJSONField(`{"a":1}`)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, `{"a":1}`, value)
	})

	t.Run("object marshal", func(t *testing.T) {
		value, ok, err := EncodeJSONField(map[string]any{"tone": "friendly"})
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.JSONEq(t, `{"tone":"friendly"}`, value)
	})
}

func TestUnitPromptWriteBody(t *testing.T) {
	t.Run("content only", func(t *testing.T) {
		got, err := PromptWriteBody("hello", PromptWriteOptions{})
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{"content": "hello"}, got)
	})

	t.Run("optional fields", func(t *testing.T) {
		prod := true
		got, err := PromptWriteBody("hello", PromptWriteOptions{
			Name:          "Guide",
			Description:   "desc",
			Type:          PromptTypeText,
			Config:        map[string]any{"tone": "friendly"},
			CommitMessage: "initial",
			Meta:          map[string]any{"source": "sdk"},
			Status:        PromptStatusActive,
			Production:    &prod,
		})
		assert.NoError(t, err)
		assert.Equal(t, "hello", got["content"])
		assert.Equal(t, "Guide", got["name"])
		assert.Equal(t, "desc", got["description"])
		assert.Equal(t, PromptTypeText, got["type"])
		assert.JSONEq(t, `{"tone":"friendly"}`, got["config"].(string))
		assert.Equal(t, "initial", got["commitMessage"])
		assert.JSONEq(t, `{"source":"sdk"}`, got["meta"].(string))
		assert.Equal(t, PromptStatusActive, got["status"])
		assert.Equal(t, true, got["production"])
	})
}
