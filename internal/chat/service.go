package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/abteilung6/assetagent/internal/llm"
	"github.com/abteilung6/assetagent/internal/telemetry"
)

const defaultMaxTurns = 5

var (
	ErrEmptyMessages    = errors.New("messages must not be empty")
	ErrInvalidRole      = errors.New("message role must be system, user, or assistant")
	ErrMaxTurnsExceeded = errors.New("chat exceeded maximum tool turns")
)

const defaultSystemPrompt = `You are a personal finance assistant for the user's bank transactions.
Use the available tools before answering. Only state numbers that come from tool results.
Do not provide investment advice.

Tool selection (prefer trusted / transfer-aware tools and stored plan artifacts):
- get_baseline(): current FinancialBaseline (income, costs, free cashflow). Prefer for monthly budget / "can I afford" plan questions.
- get_money_review(): latest Money Review summary and findings. Prefer when explaining the monthly review.
- get_forecast(): latest 90-day liquidity forecast summary. Prefer for runway / projected cash questions. Do not invent scenario results — those are run in the Plan UI.
- suggest_review_categories(): Needs review category queue with keyword suggestions. Prefer when helping clear category review. Read-only — direct the user to Apply suggested categories on Needs review to save.
- get_cashflow_v2(from, to): household income, expenses, and net with confirmed internal transfers excluded, plus an evidence contract. Prefer this for totals and "how much did I spend/earn" questions.
- get_recurring_costs(from?, to?): regular bills and subscriptions (rent, insurance, Netflix). Prefer for monthly cost questions.
- get_spending_changes(from, to): compare the given period to the equal-length window immediately before it.
- get_anomalies(from, to): notable amount steps, uncertain recurring series, and large one-off expenses.
- get_cashflow(from, to): legacy raw totals (includes internal transfers). Use only if the user explicitly wants unadjusted bank sums.
- get_top_counterparties(from, to, limit?): who the user spent the most with in a range.
- search_transactions(q, from?, to?, limit?): find specific transactions matching text. Requires q. Max limit 50. Not for period spending totals.

Important: baseline, money review, and forecast tools are read-only. Never claim you created, confirmed, or adjusted a baseline/review/forecast — tell the user to use the Baseline, Reviews, or Plan pages.

Date rules:
- Dates are inclusive YYYY-MM-DD; from must be on or before to.
- Calendar year YYYY: from=YYYY-01-01, to=YYYY-12-31.
- Named month (e.g. "June 2025", "December 2025"): use only that month — from YYYY-MM-01 through the month's last day. Never expand a month question to year-to-date or the full year.
- "Top N" or "top 5": pass limit as a JSON number (e.g. 5), not a string.
- If the period is ambiguous, ask which dates — do not guess or swap from/to.

When a tool returns {"error":...}, correct the arguments and retry before asking the user to confirm invalid ranges.

Result interpretation:
- Only results containing "error" indicate failure.
- Successful results include "ok":true. Empty lists, zero expenses, or total=0 are valid — they mean no matching spending in that period.
- When available is false, explain that the artifact is missing and point the user to the relevant page — do not invent numbers.
- When ok is true, answer from the data. Do not say the tool failed and do not ask the user to reconfirm dates.
- When evidence_ids are present, you may mention that figures exclude confirmed transfers or come from detected recurring series — do not invent ids.`

type ToolRunner interface {
	Tools() []llm.Tool
	Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error)
}

type Config struct {
	MaxTurns     int
	SystemPrompt string
	TraceDetail  telemetry.TraceDetail
}

func DefaultConfig() Config {
	return Config{
		MaxTurns:     defaultMaxTurns,
		SystemPrompt: defaultSystemPrompt,
		TraceDetail:  telemetry.TraceDetailMetadata,
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
		Content: systemPrompt(s.cfg.SystemPrompt),
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
		ctx, endLLM := startLLMSpan(ctx, s.llm.Model(), turn)
		resp, err := s.llm.Complete(ctx, llm.CompletionRequest{
			Messages: conversation,
			Tools:    s.tools.Tools(),
		})
		if err != nil {
			telemetry.RecordError(ctx, err)
			endLLM(llm.CompletionResponse{})
			return Result{}, err
		}
		endLLM(resp)

		if len(resp.ToolCalls) == 0 {
			finishChatTrace(ctx, resp.Content, toolCalls, turn+1)
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
			ctx, endTool := startToolSpan(ctx, call.Name, call.Arguments, s.cfg.TraceDetail)
			result, err := s.tools.Execute(ctx, call.Name, call.Arguments)
			if err != nil {
				result = encodeToolError(err)
			}
			endTool(err, result)

			toolCalls = append(toolCalls, ToolCall{
				Name:   call.Name,
				Input:  append(json.RawMessage(nil), call.Arguments...),
				Result: append(json.RawMessage(nil), result...),
			})

			conversation = append(conversation, llm.Message{
				Role:       llm.RoleTool,
				ToolCallID: call.ID,
				ToolName:   call.Name,
				Content:    string(result),
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

func systemPrompt(base string) string {
	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s\nToday's date is %s.", base, today)
}

func encodeToolError(err error) json.RawMessage {
	payload, marshalErr := json.Marshal(map[string]string{"error": err.Error()})
	if marshalErr != nil {
		return json.RawMessage(`{"error":"tool execution failed"}`)
	}
	return payload
}
