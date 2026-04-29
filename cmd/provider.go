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
	case "copilot":
		return providers.NewCopilotProvider(cfg.Copilot.GithubToken, cfg.Copilot.Model)
	default:
		return nil, fmt.Errorf("unknown provider %q. Supported: ollama, openai, copilot", cfg.Model.Provider)
	}
}

// newEmbedder creates the embedder based on embed.provider config.
// Defaults to ollama (nomic-embed-text). Returns nil without error if unavailable
// so callers can degrade gracefully to FTS search.
func newEmbedder(cfg *config.Config) providers.Embedder {
	switch cfg.Embed.Provider {
	case "openai":
		p, err := providers.NewOpenAIProvider(cfg.OpenAI.APIKey, cfg.OpenAI.Model)
		if err != nil {
			return nil
		}
		return p
	case "copilot":
		p, err := providers.NewCopilotProvider(cfg.Copilot.GithubToken, cfg.Copilot.Model)
		if err != nil {
			return nil
		}
		return p
	default: // "ollama" or anything else
		return providers.NewOllamaProvider(cfg.Ollama.URL, cfg.Ollama.Model)
	}
}
