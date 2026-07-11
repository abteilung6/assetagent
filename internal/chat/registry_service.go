package chat

import (
	"context"

	"github.com/abteilung6/assetagent/internal/llm"
	"github.com/abteilung6/assetagent/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

type RegistryService struct {
	registry *llm.Registry
	tools    ToolRunner
	cfg      Config
}

func NewRegistryService(registry *llm.Registry, tools ToolRunner, cfg Config) *RegistryService {
	return &RegistryService{
		registry: registry,
		tools:    tools,
		cfg:      cfg,
	}
}

func (s *RegistryService) Chat(
	ctx context.Context,
	provider string,
	model string,
	messages []Message,
) (Result, error) {
	var providerID llm.ProviderID
	if provider != "" {
		var err error
		providerID, err = llm.ParseProviderID(provider)
		if err != nil {
			return Result{}, err
		}
	}

	resolved, err := s.registry.Resolve(ctx, providerID, model)
	if err != nil {
		return Result{}, err
	}

	telemetry.SetAttributes(ctx,
		attribute.String("chat.provider", string(resolvedProviderID(provider, providerID, s.registry))),
		attribute.String("chat.model", resolved.Model()),
	)
	telemetry.SetTraceMetadata(ctx, "provider", string(resolvedProviderID(provider, providerID, s.registry)))
	telemetry.SetTraceMetadata(ctx, "model", resolved.Model())

	svc := NewService(resolved, s.tools, s.cfg)
	return svc.Chat(ctx, messages)
}

func (s *RegistryService) StreamChat(
	ctx context.Context,
	provider string,
	model string,
	messages []Message,
	write StreamWriter,
) error {
	return s.Stream(ctx, provider, model, messages, write)
}
