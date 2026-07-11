package chat_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/abteilung6/assetagent/internal/chat"
	"github.com/abteilung6/assetagent/internal/llm"
)

type fakeStreamLLM struct {
	responses []llm.CompletionResponse
	chunks    [][]string
	calls     int
}

func (f *fakeStreamLLM) Model() string { return "fake-stream" }

func (f *fakeStreamLLM) Ping(ctx context.Context) error { return nil }

func (f *fakeStreamLLM) Complete(ctx context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	if f.calls >= len(f.responses) {
		return llm.CompletionResponse{}, errors.New("no more fake responses")
	}
	resp := f.responses[f.calls]
	f.calls++
	return resp, nil
}

func (f *fakeStreamLLM) StreamComplete(
	ctx context.Context,
	req llm.CompletionRequest,
	emit llm.DeltaEmitter,
) (llm.CompletionResponse, error) {
	if f.calls >= len(f.responses) {
		return llm.CompletionResponse{}, errors.New("no more fake responses")
	}

	resp := f.responses[f.calls]
	chunks := f.chunks[f.calls]
	f.calls++

	for _, chunk := range chunks {
		if emit != nil {
			if err := emit(chunk); err != nil {
				return llm.CompletionResponse{}, err
			}
		}
	}

	return resp, nil
}

func TestService_Stream_directAnswer(t *testing.T) {
	llmClient := &fakeStreamLLM{
		responses: []llm.CompletionResponse{{Content: "Hello there."}},
		chunks:    [][]string{{"Hello", " there", "."}},
	}
	svc := chat.NewService(llmClient, &fakeTools{}, chat.DefaultConfig())

	var events []string
	var deltas []string
	err := svc.Stream(context.Background(), []chat.Message{{
		Role:    llm.RoleUser,
		Content: "Hi",
	}}, func(event string, data any) error {
		events = append(events, event)
		if event == chat.StreamEventDelta {
			raw, err := json.Marshal(data)
			if err != nil {
				return err
			}
			var payload struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal(raw, &payload); err != nil {
				return err
			}
			deltas = append(deltas, payload.Content)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	if len(events) < 2 {
		t.Fatalf("events = %v", events)
	}
	if events[len(events)-1] != chat.StreamEventDone {
		t.Fatalf("last event = %q, want done", events[len(events)-1])
	}
	if len(deltas) != 3 {
		t.Fatalf("deltas = %v", deltas)
	}
}

func TestService_Stream_toolLoop(t *testing.T) {
	llmClient := &fakeStreamLLM{
		responses: []llm.CompletionResponse{
			{
				ToolCalls: []llm.ToolCall{{
					ID:        "call_1",
					Name:      "get_cashflow",
					Arguments: json.RawMessage(`{"from":"2025-12-01","to":"2025-12-31"}`),
				}},
			},
			{Content: "You spent 15.98 EUR in December."},
		},
		chunks: [][]string{
			nil,
			{"You spent ", "15.98 EUR in December."},
		},
	}
	tools := &fakeTools{
		result: json.RawMessage(`{"ok":true,"expenses":"15.98","currency":"EUR"}`),
	}
	svc := chat.NewService(llmClient, tools, chat.DefaultConfig())

	var events []string
	err := svc.Stream(context.Background(), []chat.Message{{
		Role:    llm.RoleUser,
		Content: "How much did I spend in December?",
	}}, func(event string, data any) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	want := []string{
		chat.StreamEventToolStart,
		chat.StreamEventToolResult,
		chat.StreamEventDelta,
		chat.StreamEventDelta,
		chat.StreamEventDone,
	}
	if len(events) != len(want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
	for i, event := range want {
		if events[i] != event {
			t.Fatalf("events[%d] = %q, want %q", i, events[i], event)
		}
	}
}

func TestService_Stream_emitsErrorEvent(t *testing.T) {
	svc := chat.NewService(&fakeStreamLLM{}, &fakeTools{}, chat.DefaultConfig())

	var events []string
	err := svc.Stream(context.Background(), nil, func(event string, data any) error {
		events = append(events, event)
		return nil
	})
	if !errors.Is(err, chat.ErrEmptyMessages) {
		t.Fatalf("error = %v, want ErrEmptyMessages", err)
	}
	if len(events) != 1 || events[0] != chat.StreamEventError {
		t.Fatalf("events = %v", events)
	}
}
