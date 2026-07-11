package telemetry

import (
	"context"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/attribute"
)

const maxTraceTextLen = 500

func TruncateText(value string, max int) string {
	if max <= 0 {
		return ""
	}
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "…"
}

func SetTraceMetadata(ctx context.Context, key, value string) {
	if value == "" {
		return
	}
	SetAttributes(ctx, attribute.String("langfuse.trace.metadata."+key, value))
}

func SetTraceInput(ctx context.Context, value string) {
	value = TruncateText(value, maxTraceTextLen)
	if value == "" {
		return
	}
	SetAttributes(ctx, attribute.String("langfuse.trace.input", value))
}

func SetTraceOutput(ctx context.Context, value string) {
	value = TruncateText(value, maxTraceTextLen)
	if value == "" {
		return
	}
	SetAttributes(ctx, attribute.String("langfuse.trace.output", value))
}

func SetObservationMetadata(ctx context.Context, key, value string) {
	if value == "" {
		return
	}
	SetAttributes(ctx, attribute.String("langfuse.observation.metadata."+key, value))
}

func RecordLLMUsage(
	ctx context.Context,
	model string,
	promptTokens int,
	completionTokens int,
	toolCallCount int,
	contentLen int,
) {
	if model != "" {
		SetAttributes(ctx,
			attribute.String("gen_ai.request.model", model),
			attribute.String("gen_ai.response.model", model),
		)
	}
	if promptTokens > 0 {
		SetAttributes(ctx,
			attribute.Int("gen_ai.usage.input_tokens", promptTokens),
			attribute.Int("gen_ai.usage.prompt_tokens", promptTokens),
		)
	}
	if completionTokens > 0 {
		SetAttributes(ctx,
			attribute.Int("gen_ai.usage.output_tokens", completionTokens),
			attribute.Int("gen_ai.usage.completion_tokens", completionTokens),
		)
	}
	if promptTokens > 0 || completionTokens > 0 {
		SetAttributes(ctx, attribute.Int("gen_ai.usage.total_tokens", promptTokens+completionTokens))
	}
	SetObservationMetadata(ctx, "tool_call_count", formatInt(toolCallCount))
	SetObservationMetadata(ctx, "content_length", formatInt(contentLen))
}

func formatInt(value int) string {
	if value == 0 {
		return ""
	}
	return strconv.Itoa(value)
}
