package chat

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/abteilung6/assetagent/internal/llm"
	"github.com/abteilung6/assetagent/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

func LastUserMessage(messages []Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == llm.RoleUser {
			return messages[i].Content
		}
	}
	return ""
}

func StartHTTPChatSpan(
	ctx context.Context,
	provider string,
	model string,
	messageCount int,
	streaming bool,
	lastUser string,
) (context.Context, func()) {
	attrs := []attribute.KeyValue{
		attribute.String("chat.provider", provider),
		attribute.String("chat.model", model),
		attribute.Int("chat.message_count", messageCount),
		attribute.Bool("chat.streaming", streaming),
	}
	ctx, end := telemetry.StartSpan(ctx, "chat.request", attrs...)

	if provider != "" {
		telemetry.SetTraceMetadata(ctx, "provider", provider)
	}
	if model != "" {
		telemetry.SetTraceMetadata(ctx, "model", model)
	}
	telemetry.SetTraceMetadata(ctx, "message_count", strconv.Itoa(messageCount))
	telemetry.SetTraceMetadata(ctx, "streaming", strconv.FormatBool(streaming))
	telemetry.SetTraceInput(ctx, lastUser)

	return ctx, end
}

func startLLMSpan(ctx context.Context, model string, turn int) (context.Context, func(llm.CompletionResponse)) {
	ctx, end := telemetry.StartSpan(ctx, "llm.completion",
		attribute.String("llm.model", model),
		attribute.Int("chat.turn", turn),
	)
	telemetry.SetObservationMetadata(ctx, "turn", strconv.Itoa(turn))

	return ctx, func(resp llm.CompletionResponse) {
		telemetry.RecordLLMUsage(
			ctx,
			model,
			resp.Usage.PromptTokens,
			resp.Usage.CompletionTokens,
			len(resp.ToolCalls),
			len(resp.Content),
		)
		end()
	}
}

func startToolSpan(
	ctx context.Context,
	name string,
	input json.RawMessage,
	detail telemetry.TraceDetail,
) (context.Context, func(error, json.RawMessage)) {
	ctx, end := telemetry.StartSpan(ctx, "tool."+name,
		attribute.String("tool.name", name),
		attribute.String("gen_ai.tool.name", name),
	)
	telemetry.SetObservationMetadata(ctx, "tool_name", name)

	start := time.Now()
	redactedInput := telemetry.RedactToolInput(detail, input)
	if redactedInput != "" {
		telemetry.SetAttributes(ctx, attribute.String("tool.input", redactedInput))
		telemetry.SetObservationMetadata(ctx, "input", redactedInput)
	}

	return ctx, func(err error, result json.RawMessage) {
		latencyMS := time.Since(start).Milliseconds()
		telemetry.SetAttributes(ctx, attribute.Int64("tool.latency_ms", latencyMS))
		telemetry.SetObservationMetadata(ctx, "latency_ms", strconv.FormatInt(latencyMS, 10))

		redactedResult := telemetry.RedactToolResult(detail, result)
		if redactedResult != "" {
			telemetry.SetAttributes(ctx, attribute.String("tool.result", redactedResult))
			telemetry.SetObservationMetadata(ctx, "result", redactedResult)
		}
		if err != nil {
			telemetry.RecordError(ctx, err)
			telemetry.SetObservationMetadata(ctx, "error", err.Error())
		}
		end()
	}
}

func finishChatTrace(ctx context.Context, answer string, toolCalls []ToolCall, llmTurns int) {
	telemetry.SetTraceOutput(ctx, answer)

	names := make([]string, 0, len(toolCalls))
	seen := make(map[string]struct{}, len(toolCalls))
	for _, call := range toolCalls {
		if call.Name == "" {
			continue
		}
		if _, ok := seen[call.Name]; ok {
			continue
		}
		seen[call.Name] = struct{}{}
		names = append(names, call.Name)
	}

	if len(names) > 0 {
		telemetry.SetTraceMetadata(ctx, "tools_used", strings.Join(names, ","))
	}
	telemetry.SetTraceMetadata(ctx, "tool_count", strconv.Itoa(len(toolCalls)))
	telemetry.SetTraceMetadata(ctx, "llm_turns", strconv.Itoa(llmTurns))
}

func traceID(ctx context.Context) string {
	return telemetry.TraceID(ctx)
}
