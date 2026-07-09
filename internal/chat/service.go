package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/abteilung6/assetagent/internal/llm"
)

const defaultMaxTurns = 5

var (
	ErrEmptyMessages    = errors.New("messages must not be empty")
	ErrInvalidRole      = errors.New("message role must be system, user, or assistant")
	ErrMaxTurnsExceeded = errors.New("chat exceeded maximum tool turns")
)

const defaultSystemPrompt = `You are a personal finance assistant for the user's bank transactions.
Use the available tools to look up cashflow, counterparties, and transactions before answering.
Only state numbers that come from tool results.
If tools cannot answer the question, say so clearly.
Do not provide investment advice.`

type ToolRunner interface {
	Tools() []llm.Tool
	Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error)
}

type Config struct {
	MaxTurns     int
	SystemPrompt string
}

func DefaultConfig() Config {
	return Config{
		MaxTurns:     defaultMaxTurns,
		SystemPrompt: defaultSystemPrompt,
	}
}

type Service struct {
	llm   llm.Provider
	tools ToolRunner
	cfg   Config
}

func NewService(provider llm.Provider, tools ToolRunner, cfg Config) *Service {
	if cfg.MaxTurns <= 0 {
		cfg.MaxTurns = defaultMaxTurns
	}
	if cfg.SystemPrompt == "" {
		cfg.SystemPrompt = defaultSystemPrompt
	}

	return &Service{
		llm:   provider,
		tools: tools,
		cfg:   cfg,
	}
}

type Message struct {
	Role    string
	Content string
}

type ToolCall struct {
	Name   string
	Input  json.RawMessage
	Result json.RawMessage
}

type Result struct {
	Answer    string
	ToolCalls []ToolCall
}

func (s *Service) Chat(ctx context.Context, messages []Message) (Result, error) {
	if len(messages) == 0 {
		return Result{}, ErrEmptyMessages
	}

	conversation := []llm.Message{{
		Role:    llm.RoleSystem,
		Content: s.cfg.SystemPrompt,
	}}

	for _, msg := range messages {
		if err := validateInputMessage(msg); err != nil {
			return Result{}, err
		}
		conversation = append(conversation, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	toolCalls := make([]ToolCall, 0)

	for turn := 0; turn < s.cfg.MaxTurns; turn++ {
		resp, err := s.llm.Complete(ctx, llm.CompletionRequest{
			Messages: conversation,
			Tools:    s.tools.Tools(),
		})
		if err != nil {
			return Result{}, err
		}

		if len(resp.ToolCalls) == 0 {
			return Result{
				Answer:    resp.Content,
				ToolCalls: toolCalls,
			}, nil
		}

		conversation = append(conversation, llm.Message{
			Role:      llm.RoleAssistant,
			ToolCalls: resp.ToolCalls,
		})

		for _, call := range resp.ToolCalls {
			result, err := s.tools.Execute(ctx, call.Name, call.Arguments)
			if err != nil {
				return Result{}, fmt.Errorf("execute tool %q: %w", call.Name, err)
			}

			toolCalls = append(toolCalls, ToolCall{
				Name:   call.Name,
				Input:  append(json.RawMessage(nil), call.Arguments...),
				Result: append(json.RawMessage(nil), result...),
			})

			conversation = append(conversation, llm.Message{
				Role:     llm.RoleTool,
				ToolName: call.Name,
				Content:  string(result),
			})
		}
	}

	return Result{}, ErrMaxTurnsExceeded
}

func validateInputMessage(msg Message) error {
	switch msg.Role {
	case llm.RoleSystem, llm.RoleUser, llm.RoleAssistant:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidRole, msg.Role)
	}
}
