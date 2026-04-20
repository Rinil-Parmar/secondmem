package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const (
	copilotTokenURL     = "https://api.github.com/copilot_internal/v2/token"
	copilotAPIURL       = "https://api.githubcopilot.com"
	defaultCopilotModel = "claude-haiku-4-5"
)

// copilotTransport injects VSCode-like headers to mimic a legitimate Copilot client.
type copilotTransport struct {
	inner http.RoundTripper
}

func (t *copilotTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", "GitHubCopilot/1.200.0")
	req.Header.Set("Editor-Version", "vscode/1.96.0")
	req.Header.Set("Editor-Plugin-Version", "copilot-chat/0.22.0")
	req.Header.Set("Openai-Intent", "conversation-panel")
	return t.inner.RoundTrip(req)
}

// CopilotProvider implements LLMProvider using GitHub Copilot.
type CopilotProvider struct {
	githubToken  string
	sessionToken string
	model        string
	client       *openai.Client
}

type copilotTokenResponse struct {
	Token string `json:"token"`
}

// NewCopilotProvider creates a Copilot provider using a GitHub OAuth token.
// Get token via: gh auth token
func NewCopilotProvider(githubToken, model string) (*CopilotProvider, error) {
	if githubToken == "" {
		return nil, fmt.Errorf("GitHub token required. Run: gh auth token")
	}
	if model == "" {
		model = defaultCopilotModel
	}
	p := &CopilotProvider{
		githubToken: githubToken,
		model:       model,
	}
	if err := p.refreshClient(); err != nil {
		return nil, fmt.Errorf("Copilot auth failed: %w", err)
	}
	return p, nil
}

// refreshClient exchanges GitHub token for a short-lived Copilot session token.
func (p *CopilotProvider) refreshClient() error {
	req, err := http.NewRequest("GET", copilotTokenURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+p.githubToken)
	req.Header.Set("User-Agent", "GitHubCopilot/1.200.0")
	req.Header.Set("Editor-Version", "vscode/1.96.0")
	req.Header.Set("Editor-Plugin-Version", "copilot-chat/0.22.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp copilotTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}
	if tokenResp.Token == "" {
		return fmt.Errorf("empty session token received")
	}

	p.sessionToken = tokenResp.Token

	cfg := openai.DefaultConfig(p.sessionToken)
	cfg.BaseURL = copilotAPIURL
	cfg.HTTPClient = &http.Client{
		Transport: &copilotTransport{inner: http.DefaultTransport},
	}
	p.client = openai.NewClientWithConfig(cfg)
	return nil
}

// Complete sends prompts to Copilot, auto-retries once on auth failure.
func (p *CopilotProvider) Complete(systemPrompt, userPrompt string) (string, error) {
	result, err := p.doCompletion(systemPrompt, userPrompt)
	if err != nil && isAuthError(err) {
		if refreshErr := p.refreshClient(); refreshErr != nil {
			return "", fmt.Errorf("token refresh failed: %w", refreshErr)
		}
		result, err = p.doCompletion(systemPrompt, userPrompt)
	}
	return result, err
}

func (p *CopilotProvider) doCompletion(systemPrompt, userPrompt string) (string, error) {
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
	for _, kw := range []string{"401", "403", "unauthorized", "expired", "invalid token", "authentication"} {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}

// ListCopilotModels returns available Copilot models.
func ListCopilotModels() []string {
	return []string{
		"claude-haiku-4-5",
		"claude-sonnet-4-5",
		"gpt-4o",
		"gpt-4o-mini",
		"o1-mini",
		"o3-mini",
	}
}
