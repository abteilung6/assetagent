package llm

import (
	"context"
	"fmt"
)

type DeltaEmitter func(content string) error

type StreamingProvider interface {
	Provider
	StreamComplete(ctx context.Context, req CompletionRequest, emit DeltaEmitter) (CompletionResponse, error)
}

func StreamComplete(
	ctx context.Context,
	provider Provider,
	req CompletionRequest,
	emit DeltaEmitter,
) (CompletionResponse, error) {
	if sp, ok := provider.(StreamingProvider); ok {
		return sp.StreamComplete(ctx, req, emit)
	}

	resp, err := provider.Complete(ctx, req)
	if err != nil {
		return CompletionResponse{}, err
	}

	if resp.Content != "" && emit != nil {
		if err := emit(resp.Content); err != nil {
			return CompletionResponse{}, err
		}
	}

	return resp, nil
}

func EnsureToolCallIDs(calls []ToolCall) []ToolCall {
	out := make([]ToolCall, len(calls))
	for i, call := range calls {
		out[i] = call
		if out[i].ID == "" {
			out[i].ID = fmt.Sprintf("call_%d", i)
		}
	}
	return out
}
