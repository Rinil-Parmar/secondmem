package cmd

import (
	"fmt"

	"github.com/Rinil-Parmar/secondmem/config"
	"github.com/Rinil-Parmar/secondmem/providers"
)

// newProvider creates the appropriate LLM provider based on config.
func newProvider(cfg *config.Config) (providers.LLMProvider, error) {
	switch cfg.Model.Provider {
	case "ollama":
		return providers.NewOllamaProvider(cfg.Ollama.URL, cfg.Ollama.Model), nil
	case "openai":
		return providers.NewOpenAIProvider(cfg.OpenAI.APIKey, cfg.OpenAI.Model)
	default:
		return nil, fmt.Errorf("unknown provider %q. Supported: ollama, openai", cfg.Model.Provider)
	}
}
