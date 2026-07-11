package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (o *OpenRouter) StreamComplete(
	ctx context.Context,
	req CompletionRequest,
	emit DeltaEmitter,
) (CompletionResponse, error) {
	if o.apiKey == "" {
		return CompletionResponse{}, fmt.Errorf("openrouter api key is not configured")
	}

	payload := openRouterChatRequest{
		Model:           o.model,
		Messages:        toOpenRouterMessages(req.Messages),
		Tools:           toOpenRouterTools(req.Tools),
		Stream:          true,
		ReasoningEffort: "none",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("openrouter stream marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("openrouter stream request: %w", err)
	}
	o.setHeaders(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("openrouter stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return CompletionResponse{}, fmt.Errorf("openrouter stream: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var content strings.Builder
	toolCalls := newOpenRouterStreamToolCalls()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return CompletionResponse{}, err
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk openRouterStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return CompletionResponse{}, fmt.Errorf("openrouter stream decode: %w", err)
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta
		if delta.Content != "" {
			content.WriteString(delta.Content)
			if emit != nil {
				if err := emit(delta.Content); err != nil {
					return CompletionResponse{}, err
				}
			}
		}

		if len(delta.ToolCalls) > 0 {
			toolCalls.add(delta.ToolCalls)
		}
	}

	if err := scanner.Err(); err != nil {
		return CompletionResponse{}, fmt.Errorf("openrouter stream read: %w", err)
	}

	calls := toolCalls.build()
	return CompletionResponse{
		Content:   content.String(),
		ToolCalls: EnsureToolCallIDs(calls),
	}, nil
}

type openRouterStreamChunk struct {
	Choices []struct {
		Delta openRouterStreamDelta `json:"delta"`
	} `json:"choices"`
}

type openRouterStreamDelta struct {
	Content   string                     `json:"content"`
	ToolCalls []openRouterStreamToolCall `json:"tool_calls"`
}

type openRouterStreamToolCall struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openRouterStreamToolCalls struct {
	calls map[int]*ToolCall
	order []int
}

func newOpenRouterStreamToolCalls() *openRouterStreamToolCalls {
	return &openRouterStreamToolCalls{calls: make(map[int]*ToolCall)}
}

func (a *openRouterStreamToolCalls) add(parts []openRouterStreamToolCall) {
	for _, part := range parts {
		call, ok := a.calls[part.Index]
		if !ok {
			call = &ToolCall{}
			a.calls[part.Index] = call
			a.order = append(a.order, part.Index)
		}
		if part.ID != "" {
			call.ID = part.ID
		}
		if part.Function.Name != "" {
			call.Name = part.Function.Name
		}
		if part.Function.Arguments != "" {
			prev := string(call.Arguments)
			call.Arguments = json.RawMessage(prev + part.Function.Arguments)
		}
	}
}

func (a *openRouterStreamToolCalls) build() []ToolCall {
	if len(a.order) == 0 {
		return nil
	}

	out := make([]ToolCall, 0, len(a.order))
	for _, idx := range a.order {
		call := a.calls[idx]
		args := parseToolArguments(call.Arguments)
		out = append(out, ToolCall{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: args,
		})
	}
	return out
}
