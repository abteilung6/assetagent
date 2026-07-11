package telemetry

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/abteilung6/assetagent"

type Config struct {
	Enabled        bool
	PublicKey      string
	SecretKey      string
	OTLPEndpoint   string
	TraceDetail    TraceDetail
	ServiceName    string
	ExportTimeout  time.Duration
}

func Tracer() trace.Tracer {
	return otel.Tracer(tracerName)
}

func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if !cfg.Enabled {
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
		return func(context.Context) error { return nil }, nil
	}
	if cfg.PublicKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("langfuse tracing enabled but LANGFUSE_PUBLIC_KEY or LANGFUSE_SECRET_KEY is missing")
	}

	endpoint, urlPath, insecure, err := parseOTLPEndpoint(cfg.OTLPEndpoint)
	if err != nil {
		return nil, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(cfg.PublicKey + ":" + cfg.SecretKey))
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithURLPath(urlPath),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization":                "Basic " + auth,
			"x-langfuse-ingestion-version": "4",
		}),
	}
	if insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("otlp exporter: %w", err)
	}

	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "assetagent"
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	timeout := cfg.ExportTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithExportTimeout(timeout)),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)

	return provider.Shutdown, nil
}

func TraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, func()) {
	ctx, span := Tracer().Start(ctx, name)
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	return ctx, func() { span.End() }
}

func RecordError(ctx context.Context, err error) {
	if err == nil {
		return
	}
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	trace.SpanFromContext(ctx).SetAttributes(attrs...)
}

func parseOTLPEndpoint(raw string) (endpoint string, urlPath string, insecure bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "localhost:3000", "/api/public/otel/v1/traces", true, nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", false, fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT: %w", err)
	}

	switch parsed.Scheme {
	case "http":
		insecure = true
	case "https":
		insecure = false
	case "":
		insecure = true
	default:
		return "", "", false, fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT: unsupported scheme %q", parsed.Scheme)
	}

	host := parsed.Host
	if host == "" {
		host = parsed.Path
	}
	if host == "" {
		return "", "", false, fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT: host is required")
	}

	path := strings.TrimSuffix(parsed.Path, "/")
	if path == "" || path == "/api/public/otel" {
		urlPath = "/api/public/otel/v1/traces"
	} else {
		urlPath = path
	}

	return host, urlPath, insecure, nil
}
