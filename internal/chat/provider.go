package chat

import (
	"github.com/abteilung6/assetagent/internal/llm"
)

func resolvedProviderID(requested string, parsed llm.ProviderID, registry *llm.Registry) llm.ProviderID {
	if parsed != "" {
		return parsed
	}
	if requested != "" {
		if id, err := llm.ParseProviderID(requested); err == nil {
			return id
		}
	}
	return registry.DefaultProviderID()
}
