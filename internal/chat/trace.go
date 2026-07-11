package chat

import (
	"context"
	"encoding/json"
	"time"

	"github.com/abteilung6/assetagent/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

func StartHTTPChatSpan(
	ctx context.Context,
	provider string,
	model string,
	messageCount int,
	streaming bool,
) (context.Context, func()) {
	attrs := []attribute.KeyValue{
		attribute.String("chat.provider", provider),
		attribute.String("chat.model", model),
		attribute.Int("chat.message_count", messageCount),
		attribute.Bool("chat.streaming", streaming),
	}
	return telemetry.StartSpan(ctx, "chat.request", attrs...)
}

func startLLMSpan(ctx context.Context, model string, turn int) (context.Context, func()) {
	return telemetry.StartSpan(ctx, "llm.completion",
		attribute.String("llm.model", model),
		attribute.Int("chat.turn", turn),
	)
}

func startToolSpan(
	ctx context.Context,
	name string,
	input json.RawMessage,
	detail telemetry.TraceDetail,
) (context.Context, func(error, json.RawMessage)) {
	ctx, end := telemetry.StartSpan(ctx, "tool."+name,
		attribute.String("tool.name", name),
	)

	start := time.Now()
	redactedInput := telemetry.RedactToolInput(detail, input)
	if redactedInput != "" {
		telemetry.SetAttributes(ctx, attribute.String("tool.input", redactedInput))
	}

	return ctx, func(err error, result json.RawMessage) {
		telemetry.SetAttributes(ctx, attribute.Int64("tool.latency_ms", time.Since(start).Milliseconds()))
		redactedResult := telemetry.RedactToolResult(detail, result)
		if redactedResult != "" {
			telemetry.SetAttributes(ctx, attribute.String("tool.result", redactedResult))
		}
		if err != nil {
			telemetry.RecordError(ctx, err)
		}
		end()
	}
}

func traceID(ctx context.Context) string {
	return telemetry.TraceID(ctx)
}
