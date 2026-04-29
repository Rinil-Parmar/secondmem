package providers

// LLMProvider defines the interface for language model backends.
type LLMProvider interface {
	Complete(systemPrompt string, userPrompt string) (string, error)
}

// Embedder converts text into a dense float32 vector for semantic search.
type Embedder interface {
	Embed(text string) ([]float32, error)
}
