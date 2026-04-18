package providers

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements LLMProvider using the OpenAI API.
type OpenAIProvider struct {
	client *openai.Client
	model  string
}

// NewOpenAIProvider creates a new OpenAI provider with the given API key and model.
func NewOpenAIProvider(apiKey, model string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required. Set it with: secondmem config set openai.api_key sk-...")
	}
	if model == "" {
		model = "gpt-4o"
	}
	client := openai.NewClient(apiKey)
	return &OpenAIProvider{client: client, model: model}, nil
}

// Complete sends a system prompt and user prompt to the OpenAI API.
func (p *OpenAIProvider) Complete(systemPrompt string, userPrompt string) (string, error) {
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
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("OpenAI returned no choices")
	}

	return resp.Choices[0].Message.Content, nil
}
