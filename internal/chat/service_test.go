package chat_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/abteilung6/assetagent/internal/chat"
	"github.com/abteilung6/assetagent/internal/llm"
)

type fakeLLM struct {
	responses []llm.CompletionResponse
	calls     int
	lastReq   llm.CompletionRequest
}

func (f *fakeLLM) Model() string { return "fake" }

func (f *fakeLLM) Ping(ctx context.Context) error { return nil }

func (f *fakeLLM) Complete(ctx context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	f.lastReq = req
	if f.calls >= len(f.responses) {
		return llm.CompletionResponse{}, errors.New("no more fake responses")
	}
	resp := f.responses[f.calls]
	f.calls++
	return resp, nil
}

type fakeTools struct {
	result json.RawMessage
	err    error
	calls  []struct {
		name string
		args json.RawMessage
	}
}

func (f *fakeTools) Tools() []llm.Tool {
	return []llm.Tool{{
		Name:        "get_cashflow",
		Description: "Cashflow",
		Parameters:  json.RawMessage(`{"type":"object"}`),
	}}
}

func (f *fakeTools) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	f.calls = append(f.calls, struct {
		name string
		args json.RawMessage
	}{name: name, args: append(json.RawMessage(nil), args...)})
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func TestService_Chat_directAnswer(t *testing.T) {
	llmClient := &fakeLLM{
		responses: []llm.CompletionResponse{{
			Content: "Hello there.",
		}},
	}
	svc := chat.NewService(llmClient, &fakeTools{}, chat.DefaultConfig())

	result, err := svc.Chat(context.Background(), []chat.Message{{
		Role:    llm.RoleUser,
		Content: "Hi",
	}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if result.Answer != "Hello there." {
		t.Fatalf("answer = %q", result.Answer)
	}
	if len(result.ToolCalls) != 0 {
		t.Fatalf("tool calls = %+v", result.ToolCalls)
	}
	if len(llmClient.lastReq.Tools) != 1 {
		t.Fatalf("tools = %+v", llmClient.lastReq.Tools)
	}
}

func TestService_Chat_toolLoop(t *testing.T) {
	llmClient := &fakeLLM{
		responses: []llm.CompletionResponse{
			{
				ToolCalls: []llm.ToolCall{{
					Name:      "get_cashflow",
					Arguments: json.RawMessage(`{"from":"2025-12-01","to":"2025-12-31"}`),
				}},
			},
			{
				Content: "You spent 15.98 EUR in December.",
			},
		},
	}
	tools := &fakeTools{
		result: json.RawMessage(`{"income":"0","expenses":"15.98","net":"-15.98","currency":"EUR"}`),
	}
	svc := chat.NewService(llmClient, tools, chat.DefaultConfig())

	result, err := svc.Chat(context.Background(), []chat.Message{{
		Role:    llm.RoleUser,
		Content: "How much did I spend in December?",
	}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if result.Answer != "You spent 15.98 EUR in December." {
		t.Fatalf("answer = %q", result.Answer)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("tool calls = %+v", result.ToolCalls)
	}
	if result.ToolCalls[0].Name != "get_cashflow" {
		t.Fatalf("tool name = %q", result.ToolCalls[0].Name)
	}
	if len(tools.calls) != 1 {
		t.Fatalf("tool execute calls = %d", len(tools.calls))
	}
	if llmClient.calls != 2 {
		t.Fatalf("llm calls = %d, want 2", llmClient.calls)
	}
}

func TestService_Chat_rejectsEmptyMessages(t *testing.T) {
	svc := chat.NewService(&fakeLLM{}, &fakeTools{}, chat.DefaultConfig())

	_, err := svc.Chat(context.Background(), nil)
	if !errors.Is(err, chat.ErrEmptyMessages) {
		t.Fatalf("error = %v, want ErrEmptyMessages", err)
	}
}

func TestService_Chat_rejectsInvalidRole(t *testing.T) {
	svc := chat.NewService(&fakeLLM{}, &fakeTools{}, chat.DefaultConfig())

	_, err := svc.Chat(context.Background(), []chat.Message{{
		Role:    "tool",
		Content: "secret",
	}})
	if !errors.Is(err, chat.ErrInvalidRole) {
		t.Fatalf("error = %v, want ErrInvalidRole", err)
	}
}

func TestService_Chat_toolErrorReturnedToModel(t *testing.T) {
	llmClient := &fakeLLM{
		responses: []llm.CompletionResponse{
			{
				ToolCalls: []llm.ToolCall{{
					Name:      "get_cashflow",
					Arguments: json.RawMessage(`{}`),
				}},
			},
			{
				Content: "Which month would you like me to check?",
			},
		},
	}
	tools := &fakeTools{err: errors.New("from and to are required (YYYY-MM-DD)")}
	svc := chat.NewService(llmClient, tools, chat.DefaultConfig())

	result, err := svc.Chat(context.Background(), []chat.Message{{
		Role:    llm.RoleUser,
		Content: "How much did I spend?",
	}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if result.Answer != "Which month would you like me to check?" {
		t.Fatalf("answer = %q", result.Answer)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("tool calls = %+v", result.ToolCalls)
	}
	var toolResult map[string]string
	if err := json.Unmarshal(result.ToolCalls[0].Result, &toolResult); err != nil {
		t.Fatalf("unmarshal tool result: %v", err)
	}
	if toolResult["error"] == "" {
		t.Fatalf("tool result = %+v, want error field", toolResult)
	}
}

func TestService_Chat_maxTurnsExceeded(t *testing.T) {
	llmClient := &fakeLLM{
		responses: []llm.CompletionResponse{
			{ToolCalls: []llm.ToolCall{{Name: "get_cashflow", Arguments: json.RawMessage(`{"from":"2025-12-01","to":"2025-12-31"}`)}}},
			{ToolCalls: []llm.ToolCall{{Name: "get_cashflow", Arguments: json.RawMessage(`{"from":"2025-11-01","to":"2025-11-30"}`)}}},
			{ToolCalls: []llm.ToolCall{{Name: "get_cashflow", Arguments: json.RawMessage(`{"from":"2025-10-01","to":"2025-10-31"}`)}}},
			{ToolCalls: []llm.ToolCall{{Name: "get_cashflow", Arguments: json.RawMessage(`{"from":"2025-09-01","to":"2025-09-30"}`)}}},
			{ToolCalls: []llm.ToolCall{{Name: "get_cashflow", Arguments: json.RawMessage(`{"from":"2025-08-01","to":"2025-08-31"}`)}}},
		},
	}
	svc := chat.NewService(llmClient, &fakeTools{
		result: json.RawMessage(`{"income":"0","expenses":"1","net":"-1","currency":"EUR"}`),
	}, chat.Config{MaxTurns: 5, SystemPrompt: chat.DefaultConfig().SystemPrompt})

	_, err := svc.Chat(context.Background(), []chat.Message{{
		Role:    llm.RoleUser,
		Content: "loop",
	}})
	if !errors.Is(err, chat.ErrMaxTurnsExceeded) {
		t.Fatalf("error = %v, want ErrMaxTurnsExceeded", err)
	}
}

func TestDefaultConfig_prefersTrustedMoneyTools(t *testing.T) {
	prompt := chat.DefaultConfig().SystemPrompt
	for _, want := range []string{
		"get_baseline",
		"get_money_review",
		"get_forecast",
		"suggest_review_categories",
		"get_cashflow_v2",
		"get_recurring_costs",
		"get_spending_changes",
		"get_anomalies",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("system prompt missing %q", want)
		}
	}
}
