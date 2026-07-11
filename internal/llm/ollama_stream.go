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

func (o *Ollama) StreamComplete(
	ctx context.Context,
	req CompletionRequest,
	emit DeltaEmitter,
) (CompletionResponse, error) {
	think := false
	payload := ollamaChatRequest{
		Model:    o.model,
		Messages: toOllamaMessages(req.Messages),
		Tools:    toOllamaTools(req.Tools),
		Stream:   true,
		Think:    &think,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("ollama stream marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("ollama stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("ollama stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return CompletionResponse{}, fmt.Errorf("ollama stream: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var final ollamaMessage
	var usage Usage
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return CompletionResponse{}, err
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var chunk ollamaStreamChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			return CompletionResponse{}, fmt.Errorf("ollama stream decode: %w", err)
		}

		if chunk.Message.Content != "" && emit != nil {
			if err := emit(chunk.Message.Content); err != nil {
				return CompletionResponse{}, err
			}
		}

		if chunk.Done {
			final = chunk.Message
			usage = Usage{
				PromptTokens:     chunk.PromptEvalCount,
				CompletionTokens: chunk.EvalCount,
				TotalTokens:      chunk.PromptEvalCount + chunk.EvalCount,
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return CompletionResponse{}, fmt.Errorf("ollama stream read: %w", err)
	}

	completion := fromOllamaMessage(final)
	completion.ToolCalls = EnsureToolCallIDs(completion.ToolCalls)
	completion.Usage = usage
	return completion, nil
}

type ollamaStreamChunk struct {
	Message         ollamaMessage `json:"message"`
	Done            bool          `json:"done"`
	PromptEvalCount int           `json:"prompt_eval_count,omitempty"`
	EvalCount       int           `json:"eval_count,omitempty"`
}
