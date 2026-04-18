package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OllamaProvider implements LLMProvider using a local Ollama instance.
type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

type ollamaRequest struct {
	Model    string           `json:"model"`
	Messages []ollamaMessage  `json:"messages"`
	Stream   bool             `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
}

// NewOllamaProvider creates a new Ollama provider.
// baseURL defaults to http://localhost:11434 if empty.
// model defaults to "llama3.2" if empty.
func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3.2"
	}
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

// Complete sends a system prompt and user prompt to the local Ollama instance.
func (p *OllamaProvider) Complete(systemPrompt string, userPrompt string) (string, error) {
	reqBody := ollamaRequest{
		Model: p.model,
		Messages: []ollamaMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := p.client.Post(p.baseURL+"/api/chat", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("Ollama request failed (is Ollama running at %s?): %w", p.baseURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ollamaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	return result.Message.Content, nil
}
