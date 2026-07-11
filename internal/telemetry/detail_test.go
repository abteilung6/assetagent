package telemetry

import (
	"testing"
)

func TestRedactToolResult_metadataOnly(t *testing.T) {
	raw := []byte(`{"ok":true,"expenses":"15.98","currency":"EUR","transactions":[{"purpose":"secret"}]}`)
	got := RedactToolResult(TraceDetailMetadata, raw)
	if got != "" {
		t.Fatalf("RedactToolResult() = %q, want empty", got)
	}
}

func TestRedactToolResult_aggregates(t *testing.T) {
	raw := []byte(`{"ok":true,"expenses":"15.98","currency":"EUR","transactions":[{"purpose":"secret"}]}`)
	got := RedactToolResult(TraceDetailAggregates, raw)
	if got == "" {
		t.Fatal("RedactToolResult() = empty, want aggregate fields")
	}
	if contains(got, "secret") {
		t.Fatalf("RedactToolResult() leaked raw transaction: %q", got)
	}
	if !contains(got, "15.98") {
		t.Fatalf("RedactToolResult() = %q, want expenses", got)
	}
}

func TestParseTraceDetail_defaultsToMetadata(t *testing.T) {
	if got := ParseTraceDetail(""); got != TraceDetailMetadata {
		t.Fatalf("ParseTraceDetail() = %q, want metadata_only", got)
	}
}

func TestParseOTLPEndpoint(t *testing.T) {
	endpoint, path, insecure, err := parseOTLPEndpoint("http://localhost:3000/api/public/otel")
	if err != nil {
		t.Fatalf("parseOTLPEndpoint() error = %v", err)
	}
	if endpoint != "localhost:3000" {
		t.Fatalf("endpoint = %q", endpoint)
	}
	if path != "/api/public/otel/v1/traces" {
		t.Fatalf("path = %q", path)
	}
	if !insecure {
		t.Fatal("expected insecure http")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexSubstring(s, sub) >= 0)
}

func indexSubstring(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
