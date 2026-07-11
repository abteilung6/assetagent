package chat

import (
	"context"
	"encoding/json"

	"github.com/abteilung6/assetagent/internal/llm"
)

const (
	StreamEventDelta      = "delta"
	StreamEventToolStart  = "tool_start"
	StreamEventToolResult = "tool_result"
	StreamEventDone       = "done"
	StreamEventError      = "error"
)

type StreamWriter func(event string, data any) error

type deltaPayload struct {
	Content string `json:"content"`
}

type toolStartPayload struct {
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type toolResultPayload struct {
	Name   string          `json:"name"`
	Result json.RawMessage `json:"result"`
}

type streamDonePayload struct {
	Answer    string     `json:"answer"`
	ToolCalls []ToolCall `json:"tool_calls"`
}

type streamErrorPayload struct {
	Message string `json:"message"`
}

func (s *Service) Stream(ctx context.Context, messages []Message, write StreamWriter) error {
	if len(messages) == 0 {
		return writeStreamError(write, ErrEmptyMessages)
	}

	conversation := []llm.Message{{
		Role:    llm.RoleSystem,
		Content: systemPrompt(s.cfg.SystemPrompt),
	}}

	for _, msg := range messages {
		if err := validateInputMessage(msg); err != nil {
			return writeStreamError(write, err)
		}
		conversation = append(conversation, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	toolCalls := make([]ToolCall, 0)

	for turn := 0; turn < s.cfg.MaxTurns; turn++ {
		resp, err := llm.StreamComplete(ctx, s.llm, llm.CompletionRequest{
			Messages: conversation,
			Tools:    s.tools.Tools(),
		}, func(delta string) error {
			return write(StreamEventDelta, deltaPayload{Content: delta})
		})
		if err != nil {
			return writeStreamError(write, err)
		}

		if len(resp.ToolCalls) == 0 {
			return write(StreamEventDone, streamDonePayload{
				Answer:    resp.Content,
				ToolCalls: toolCalls,
			})
		}

		conversation = append(conversation, llm.Message{
			Role:      llm.RoleAssistant,
			ToolCalls: resp.ToolCalls,
		})

		for _, call := range resp.ToolCalls {
			input := append(json.RawMessage(nil), call.Arguments...)
			if err := write(StreamEventToolStart, toolStartPayload{
				Name:  call.Name,
				Input: input,
			}); err != nil {
				return err
			}

			result, err := s.tools.Execute(ctx, call.Name, call.Arguments)
			if err != nil {
				result = encodeToolError(err)
			}

			toolCalls = append(toolCalls, ToolCall{
				Name:   call.Name,
				Input:  input,
				Result: append(json.RawMessage(nil), result...),
			})

			if err := write(StreamEventToolResult, toolResultPayload{
				Name:   call.Name,
				Result: result,
			}); err != nil {
				return err
			}

			conversation = append(conversation, llm.Message{
				Role:       llm.RoleTool,
				ToolCallID: call.ID,
				ToolName:   call.Name,
				Content:    string(result),
			})
		}
	}

	return writeStreamError(write, ErrMaxTurnsExceeded)
}

func writeStreamError(write StreamWriter, err error) error {
	if write == nil {
		return err
	}
	_ = write(StreamEventError, streamErrorPayload{Message: err.Error()})
	return err
}

func (s *RegistryService) Stream(
	ctx context.Context,
	provider string,
	model string,
	messages []Message,
	write StreamWriter,
) error {
	var providerID llm.ProviderID
	if provider != "" {
		var err error
		providerID, err = llm.ParseProviderID(provider)
		if err != nil {
			return writeStreamError(write, err)
		}
	}

	resolved, err := s.registry.Resolve(ctx, providerID, model)
	if err != nil {
		return writeStreamError(write, err)
	}

	svc := NewService(resolved, s.tools, s.cfg)
	return svc.Stream(ctx, messages, write)
}
