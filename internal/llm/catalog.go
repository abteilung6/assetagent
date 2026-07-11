package llm

import "strings"

const (
	GroupCloud = "Cloud"
	GroupLocal = "Local"
)

type ModelSelection struct {
	Provider ProviderID `json:"provider"`
	Model    string     `json:"model"`
}

type ModelOption struct {
	Provider ProviderID `json:"provider"`
	Model    string     `json:"model"`
	Label    string     `json:"label"`
	Group    string     `json:"group,omitempty"`
}

type ModelCatalog struct {
	Default ModelSelection `json:"default"`
	Options []ModelOption  `json:"options"`
}

func (r *Registry) ModelCatalog() ModelCatalog {
	options := make([]ModelOption, 0, 4)
	hasCloud := false
	hasLocal := false

	if r.isEnabled(ProviderOllama) {
		options = append(options, ModelOption{
			Provider: ProviderOllama,
			Model:    r.cfg.OllamaModel,
			Label:    ModelLabel(r.cfg.OllamaModel),
		})
		hasLocal = true
	}

	if r.OpenRouterConfigured() {
		for _, model := range r.openRouterModels() {
			options = append(options, ModelOption{
				Provider: ProviderOpenRouter,
				Model:    model,
				Label:    ModelLabel(model),
			})
		}
		hasCloud = true
	}

	if hasCloud && hasLocal {
		for i := range options {
			switch options[i].Provider {
			case ProviderOpenRouter:
				options[i].Group = GroupCloud
			case ProviderOllama:
				options[i].Group = GroupLocal
			}
		}
	}

	return ModelCatalog{
		Default: r.defaultSelection(),
		Options: options,
	}
}

func (r *Registry) defaultSelection() ModelSelection {
	switch r.cfg.DefaultProvider {
	case ProviderOpenRouter:
		model := r.cfg.OpenRouterDefaultModel
		if model == "" && len(r.cfg.OpenRouterModelAllowlist) > 0 {
			model = r.cfg.OpenRouterModelAllowlist[0]
		}
		return ModelSelection{Provider: ProviderOpenRouter, Model: model}
	default:
		return ModelSelection{Provider: ProviderOllama, Model: r.cfg.OllamaModel}
	}
}

func (r *Registry) openRouterModels() []string {
	if len(r.cfg.OpenRouterModelAllowlist) > 0 {
		return append([]string(nil), r.cfg.OpenRouterModelAllowlist...)
	}
	if r.cfg.OpenRouterDefaultModel != "" {
		return []string{r.cfg.OpenRouterDefaultModel}
	}
	return nil
}

var knownModelLabels = map[string]string{
	"openai/gpt-4o-mini":           "GPT-4o mini",
	"openai/gpt-4o":                "GPT-4o",
	"anthropic/claude-sonnet-4":    "Claude Sonnet 4",
	"anthropic/claude-3.5-sonnet":  "Claude 3.5 Sonnet",
	"google/gemma-3-12b-it":        "Gemma 3 12B",
	"gemma4:12b":                   "Gemma 4 12B",
	"llama3.2":                     "Llama 3.2",
	"qwen2.5:7b":                   "Qwen 2.5 7B",
}

func ModelLabel(model string) string {
	if label, ok := knownModelLabels[model]; ok {
		return label
	}
	return humanizeModelID(model)
}

func humanizeModelID(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return model
	}

	base := model
	if idx := strings.LastIndex(model, "/"); idx >= 0 {
		base = model[idx+1:]
	}

	parts := strings.FieldsFunc(base, func(r rune) bool {
		return r == ':' || r == '-' || r == '_'
	})

	for i, part := range parts {
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		switch lower {
		case "gpt", "llama", "qwen", "gemma", "claude", "it", "b":
			if len(part) <= 4 {
				parts[i] = strings.ToUpper(part)
				continue
			}
		}
		if strings.ContainsAny(part, "0123456789") {
			parts[i] = strings.ToUpper(part)
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}

	return strings.Join(parts, " ")
}
