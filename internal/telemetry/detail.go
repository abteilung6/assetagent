package telemetry

import (
	"encoding/json"
	"strings"
)

type TraceDetail string

const (
	TraceDetailMetadata  TraceDetail = "metadata_only"
	TraceDetailAggregates TraceDetail = "aggregates"
	TraceDetailFullDebug TraceDetail = "full_debug"
)

func ParseTraceDetail(raw string) TraceDetail {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(TraceDetailAggregates):
		return TraceDetailAggregates
	case string(TraceDetailFullDebug):
		return TraceDetailFullDebug
	default:
		return TraceDetailMetadata
	}
}

func RedactToolInput(detail TraceDetail, input json.RawMessage) string {
	switch detail {
	case TraceDetailFullDebug:
		return string(input)
	case TraceDetailAggregates:
		return redactToolArgs(input)
	default:
		return ""
	}
}

func RedactToolResult(detail TraceDetail, result json.RawMessage) string {
	switch detail {
	case TraceDetailFullDebug:
		return string(result)
	case TraceDetailAggregates:
		return redactToolResult(result)
	default:
		return ""
	}
}

func redactToolArgs(input json.RawMessage) string {
	var payload map[string]any
	if err := json.Unmarshal(input, &payload); err != nil {
		return ""
	}

	out := make(map[string]any, len(payload))
	for _, key := range []string{"from", "to", "limit", "q"} {
		if value, ok := payload[key]; ok {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return ""
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func redactToolResult(result json.RawMessage) string {
	var payload map[string]any
	if err := json.Unmarshal(result, &payload); err != nil {
		return ""
	}

	allowed := []string{
		"ok", "error", "from", "to", "income", "expenses", "net", "currency",
		"count", "total", "limit",
	}
	out := make(map[string]any)
	for _, key := range allowed {
		if value, ok := payload[key]; ok {
			out[key] = value
		}
	}

	if rows, ok := payload["counterparties"].([]any); ok {
		out["counterparties_count"] = len(rows)
		if top := summarizeCounterparties(rows, 3); len(top) > 0 {
			out["top_counterparties"] = top
		}
	}
	if rows, ok := payload["transactions"].([]any); ok {
		out["transactions_count"] = len(rows)
		if samples := summarizeTransactions(rows, 3); len(samples) > 0 {
			out["transaction_samples"] = samples
		}
	}
	if rows, ok := payload["items"].([]any); ok {
		out["items_count"] = len(rows)
	}

	if len(out) == 0 {
		return ""
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func summarizeCounterparties(rows []any, limit int) []map[string]any {
	if limit <= 0 {
		return nil
	}

	out := make([]map[string]any, 0, limit)
	for i := 0; i < len(rows) && len(out) < limit; i++ {
		row, ok := rows[i].(map[string]any)
		if !ok {
			continue
		}
		entry := make(map[string]any, 4)
		for _, key := range []string{"counterparty", "total_spent", "transaction_count", "currency"} {
			if value, ok := row[key]; ok {
				entry[key] = value
			}
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func summarizeTransactions(rows []any, limit int) []map[string]any {
	if limit <= 0 {
		return nil
	}

	out := make([]map[string]any, 0, limit)
	for i := 0; i < len(rows) && len(out) < limit; i++ {
		row, ok := rows[i].(map[string]any)
		if !ok {
			continue
		}
		entry := make(map[string]any, 4)
		for _, key := range []string{"booking_date", "counterparty", "amount", "currency"} {
			if value, ok := row[key]; ok {
				entry[key] = value
			}
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}
