package providers

// LLMProvider defines the interface for language model backends.
type LLMProvider interface {
	// Complete sends a system prompt and user prompt to the LLM and returns the response.
	Complete(systemPrompt string, userPrompt string) (string, error)
}
