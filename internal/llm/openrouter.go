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

const (
	defaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"
	defaultOpenRouterTimeout = 5 * time.Minute
)

type OpenRouter struct {
	baseURL    string
	apiKey     string
	model      string
	appURL     string
	appName    string
	httpClient *http.Client
}

func NewOpenRouter(baseURL, apiKey, model, appURL, appName string) *OpenRouter {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = defaultOpenRouterBaseURL
	}

	return &OpenRouter{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		appURL:  appURL,
		appName: appName,
		httpClient: &http.Client{
			Timeout: defaultOpenRouterTimeout,
		},
	}
}

func (o *OpenRouter) Model() string {
	return o.model
}

func (o *OpenRouter) Ping(ctx context.Context) error {
	if o.apiKey == "" {
		return fmt.Errorf("openrouter api key is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("openrouter ping request: %w", err)
	}
	o.setHeaders(req)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("openrouter unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("openrouter ping: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var models openRouterModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return fmt.Errorf("openrouter ping decode: %w", err)
	}

	if !openRouterModelAvailable(models.Data, o.model) {
		return fmt.Errorf("openrouter model %q not found in catalog", o.model)
	}

	return nil
}

func (o *OpenRouter) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	if o.apiKey == "" {
		return CompletionResponse{}, fmt.Errorf("openrouter api key is not configured")
	}

	payload := openRouterChatRequest{
		Model:           o.model,
		Messages:        toOpenRouterMessages(req.Messages),
		Tools:           toOpenRouterTools(req.Tools),
		Stream:          false,
		ReasoningEffort: "none",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("openrouter chat marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("openrouter chat request: %w", err)
	}
	o.setHeaders(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("openrouter chat: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("openrouter chat read: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return CompletionResponse{}, fmt.Errorf("openrouter chat: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var chatResp openRouterChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return CompletionResponse{}, fmt.Errorf("openrouter chat decode: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return CompletionResponse{}, fmt.Errorf("openrouter chat: empty choices")
	}

	return fromOpenRouterMessage(chatResp.Choices[0].Message, chatResp.Usage), nil
}

func (o *OpenRouter) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	if o.appURL != "" {
		req.Header.Set("HTTP-Referer", o.appURL)
	}
	if o.appName != "" {
		req.Header.Set("X-Title", o.appName)
	}
}

func toOpenRouterMessages(messages []Message) []openRouterMessage {
	out := make([]openRouterMessage, len(messages))
	for i, msg := range messages {
		om := openRouterMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
		if msg.ToolCallID != "" {
			om.ToolCallID = msg.ToolCallID
		}
		if len(msg.ToolCalls) > 0 {
			om.ToolCalls = make([]openRouterToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				om.ToolCalls[j] = openRouterToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: openRouterToolCallFunction{
						Name:      tc.Name,
						Arguments: stringifyToolArguments(tc.Arguments),
					},
				}
			}
		}
		out[i] = om
	}
	return out
}

func toOpenRouterTools(tools []Tool) []openRouterTool {
	if len(tools) == 0 {
		return nil
	}

	out := make([]openRouterTool, len(tools))
	for i, tool := range tools {
		params := tool.Parameters
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		out[i] = openRouterTool{
			Type: "function",
			Function: openRouterToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  params,
			},
		}
	}
	return out
}

func fromOpenRouterMessage(msg openRouterMessage, usage openRouterUsage) CompletionResponse {
	resp := CompletionResponse{
		Content: msg.Content,
		Usage: Usage{
			PromptTokens:     usage.PromptTokens,
			CompletionTokens: usage.CompletionTokens,
			TotalTokens:      usage.TotalTokens,
		},
	}
	if len(msg.ToolCalls) == 0 {
		return resp
	}

	resp.ToolCalls = make([]ToolCall, len(msg.ToolCalls))
	for i, tc := range msg.ToolCalls {
		resp.ToolCalls[i] = ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: parseToolArguments(json.RawMessage(tc.Function.Arguments)),
		}
	}
	return resp
}

func stringifyToolArguments(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		if asString == "" {
			return "{}"
		}
		return asString
	}

	return string(raw)
}

func parseToolArguments(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}

	var asObject json.RawMessage
	if err := json.Unmarshal(raw, &asObject); err == nil && len(asObject) > 0 && asObject[0] == '{' {
		return asObject
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return json.RawMessage(asString)
	}

	return raw
}

func openRouterModelAvailable(models []openRouterModelInfo, want string) bool {
	for _, model := range models {
		if model.ID == want {
			return true
		}
	}
	return false
}

type openRouterModelsResponse struct {
	Data []openRouterModelInfo `json:"data"`
}

type openRouterModelInfo struct {
	ID string `json:"id"`
}

type openRouterChatRequest struct {
	Model           string             `json:"model"`
	Messages        []openRouterMessage `json:"messages"`
	Tools           []openRouterTool   `json:"tools,omitempty"`
	Stream          bool               `json:"stream"`
	ReasoningEffort string             `json:"reasoning_effort,omitempty"`
}

type openRouterChatResponse struct {
	Choices []struct {
		Message openRouterMessage `json:"message"`
	} `json:"choices"`
	Usage openRouterUsage `json:"usage"`
}

type openRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openRouterMessage struct {
	Role       string               `json:"role"`
	Content    string               `json:"content"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
	ToolCalls  []openRouterToolCall `json:"tool_calls,omitempty"`
}

type openRouterTool struct {
	Type     string                 `json:"type"`
	Function openRouterToolFunction `json:"function"`
}

type openRouterToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type openRouterToolCall struct {
	ID       string                       `json:"id"`
	Type     string                       `json:"type"`
	Function openRouterToolCallFunction `json:"function"`
}

type openRouterToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
