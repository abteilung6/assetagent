package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultOllamaTimeout = 5 * time.Minute

type Ollama struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewOllama(baseURL, model string) *Ollama {
	return &Ollama{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		httpClient: &http.Client{
			Timeout: defaultOllamaTimeout,
		},
	}
}

func (o *Ollama) Model() string {
	return o.model
}

func (o *Ollama) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.baseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("ollama ping request: %w", err)
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("ollama ping: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tags ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return fmt.Errorf("ollama ping decode: %w", err)
	}

	if !modelAvailable(tags.Models, o.model) {
		return fmt.Errorf("ollama model %q not found (run: make ollama-pull)", o.model)
	}

	return nil
}

func (o *Ollama) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	think := false
	payload := ollamaChatRequest{
		Model:    o.model,
		Messages: toOllamaMessages(req.Messages),
		Tools:    toOllamaTools(req.Tools),
		Stream:   false,
		Think:    &think,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("ollama chat marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("ollama chat request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("ollama chat: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("ollama chat read: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return CompletionResponse{}, fmt.Errorf("ollama chat: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var chatResp ollamaChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return CompletionResponse{}, fmt.Errorf("ollama chat decode: %w", err)
	}

	return fromOllamaMessage(chatResp.Message), nil
}

func modelAvailable(models []ollamaModelInfo, want string) bool {
	for _, m := range models {
		if m.Name == want || strings.HasPrefix(m.Name, want+":") {
			return true
		}
	}
	return false
}

func toOllamaMessages(messages []Message) []ollamaMessage {
	out := make([]ollamaMessage, len(messages))
	for i, msg := range messages {
		om := ollamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
		if msg.ToolName != "" {
			om.ToolName = msg.ToolName
		}
		if len(msg.ToolCalls) > 0 {
			om.ToolCalls = make([]ollamaToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				om.ToolCalls[j] = ollamaToolCall{
					Function: ollamaToolCallFunction{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				}
			}
		}
		out[i] = om
	}
	return out
}

func toOllamaTools(tools []Tool) []ollamaTool {
	if len(tools) == 0 {
		return nil
	}
	out := make([]ollamaTool, len(tools))
	for i, tool := range tools {
		params := tool.Parameters
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		out[i] = ollamaTool{
			Type: "function",
			Function: ollamaToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  params,
			},
		}
	}
	return out
}

func fromOllamaMessage(msg ollamaMessage) CompletionResponse {
	resp := CompletionResponse{Content: msg.Content}
	if len(msg.ToolCalls) == 0 {
		return resp
	}

	resp.ToolCalls = make([]ToolCall, len(msg.ToolCalls))
	for i, tc := range msg.ToolCalls {
		resp.ToolCalls[i] = ToolCall{
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		}
	}
	return resp
}

type ollamaTagsResponse struct {
	Models []ollamaModelInfo `json:"models"`
}

type ollamaModelInfo struct {
	Name string `json:"name"`
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Tools    []ollamaTool    `json:"tools,omitempty"`
	Stream   bool            `json:"stream"`
	Think    *bool           `json:"think,omitempty"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
}

type ollamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolName  string           `json:"tool_name,omitempty"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}

type ollamaTool struct {
	Type     string            `json:"type"`
	Function ollamaToolFunction `json:"function"`
}

type ollamaToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type ollamaToolCall struct {
	Function ollamaToolCallFunction `json:"function"`
}

type ollamaToolCallFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}
