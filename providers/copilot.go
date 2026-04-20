package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const (
	copilotAPIBaseURL   = "https://api.githubcopilot.com"
	defaultCopilotModel = "gpt-4o-mini"
)

// copilotTransport injects VSCode-like headers on every request.
type copilotTransport struct{ inner http.RoundTripper }

func (t *copilotTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", "GitHubCopilot/1.200.0")
	req.Header.Set("Editor-Version", "vscode/1.96.0")
	req.Header.Set("Editor-Plugin-Version", "copilot-chat/0.22.0")
	req.Header.Set("Openai-Intent", "conversation-panel")
	return t.inner.RoundTrip(req)
}

// CopilotProvider implements LLMProvider using GitHub Copilot.
type CopilotProvider struct {
	token  string
	model  string
	client *openai.Client
}

type copilotConfig struct {
	CopilotTokens map[string]string `json:"copilotTokens"`
}

// copilotTokenFromCLI reads the token from ~/.copilot/config.json automatically.
func copilotTokenFromCLI() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(home, ".copilot", "config.json"))
	if err != nil {
		return "", fmt.Errorf("Copilot CLI config not found — run: copilot auth login")
	}
	var cfg copilotConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("failed to parse Copilot CLI config: %w", err)
	}
	for _, token := range cfg.CopilotTokens {
		if token != "" {
			return token, nil
		}
	}
	return "", fmt.Errorf("no token found in Copilot CLI config — run: copilot auth login")
}

// NewCopilotProvider creates a Copilot provider.
// If token is empty, reads automatically from ~/.copilot/config.json.
func NewCopilotProvider(token, model string) (*CopilotProvider, error) {
	if token == "" {
		var err error
		token, err = copilotTokenFromCLI()
		if err != nil {
			return nil, err
		}
	}
	if model == "" {
		model = defaultCopilotModel
	}

	cfg := openai.DefaultConfig(token)
	cfg.BaseURL = copilotAPIBaseURL
	cfg.HTTPClient = &http.Client{
		Transport: &copilotTransport{inner: http.DefaultTransport},
	}

	return &CopilotProvider{
		token:  token,
		model:  model,
		client: openai.NewClientWithConfig(cfg),
	}, nil
}

// Complete sends prompts to the Copilot API.
func (p *CopilotProvider) Complete(systemPrompt, userPrompt string) (string, error) {
	resp, err := p.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: p.model,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
				{Role: openai.ChatMessageRoleUser, Content: userPrompt},
			},
			Temperature: 0.3,
		},
	)
	if err != nil {
		if isAuthError(err) {
			return "", fmt.Errorf("Copilot auth failed — re-run: copilot auth login\n%w", err)
		}
		return "", fmt.Errorf("Copilot completion failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("Copilot returned no choices")
	}
	return resp.Choices[0].Message.Content, nil
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, kw := range []string{"401", "403", "unauthorized", "expired", "invalid token"} {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}
