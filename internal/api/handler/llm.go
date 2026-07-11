package handler

import (
	"net/http"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/llm"
)

func (h *Handler) GetLLMModels(w http.ResponseWriter, r *http.Request) {
	if h.llmRegistry == nil {
		writeInternalError(w, "llm registry is not configured")
		return
	}

	catalog := h.llmRegistry.ModelCatalog()
	writeJSON(w, http.StatusOK, toAPIModelCatalog(catalog))
}

func toAPIModelCatalog(catalog llm.ModelCatalog) gen.LLMModelCatalog {
	options := make([]gen.LLMModelOption, len(catalog.Options))
	for i, option := range catalog.Options {
		apiOption := gen.LLMModelOption{
			Provider: gen.LLMModelOptionProvider(option.Provider),
			Model:    option.Model,
			Label:    option.Label,
		}
		if option.Group != "" {
			group := option.Group
			apiOption.Group = &group
		}
		options[i] = apiOption
	}

	return gen.LLMModelCatalog{
		Default: gen.LLMModelSelection{
			Provider: gen.LLMModelSelectionProvider(catalog.Default.Provider),
			Model:    catalog.Default.Model,
		},
		Options: options,
	}
}
