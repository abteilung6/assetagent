package llm

import (
	"context"
	"encoding/json"
)

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

type Message struct {
	Role       string
	Content    string
	ToolCalls  []ToolCall
	ToolName   string
	ToolCallID string
}

type Tool struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

type ToolCall struct {
	Name      string
	Arguments json.RawMessage
}

type CompletionRequest struct {
	Messages []Message
	Tools    []Tool
}

type CompletionResponse struct {
	Content   string
	ToolCalls []ToolCall
}

type Provider interface {
	Model() string
	Ping(ctx context.Context) error
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}
