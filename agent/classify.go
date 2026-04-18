package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Rinil-Parmar/secondmem/providers"
)

// ClassificationResult holds the LLM's routing decision for ingested content.
type ClassificationResult struct {
	Directory     string   `json:"directory"`
	Filename      string   `json:"filename"`
	Summary       string   `json:"summary"`
	Keywords      []string `json:"keywords"`
	RelatedTopics []string `json:"related_topics"`
}

const classifyPrompt = `You are a knowledge classifier. Analyze the provided text and classify it.

Return ONLY a JSON object with these exact fields:
{
  "directory": "kebab-case topic directory (e.g. ai-ml, engineering, startups)",
  "filename": "kebab-case filename without .md extension (max 5 words)",
  "summary": "2-4 sentence summary of the core insight",
  "keywords": ["keyword1", "keyword2", "keyword3", "keyword4", "keyword5"],
  "related_topics": ["related-topic-1", "related-topic-2"]
}

Rules:
- directory must be one of: ai-ml, engineering, startups, productivity, research, business, personal, mental-models, people-insights. Or suggest a new one if none fit.
- filename should be specific and descriptive, not generic
- keywords should be searchable terms
- related_topics should be existing directory names that connect to this content
- Return ONLY valid JSON, no markdown fences, no explanation`

// Classify sends content to the LLM for classification and routing.
func Classify(provider providers.LLMProvider, skillPrompt, content string) (*ClassificationResult, error) {
	systemPrompt := skillPrompt + "\n\n" + classifyPrompt

	response, err := provider.Complete(systemPrompt, content)
	if err != nil {
		return nil, fmt.Errorf("classification failed: %w", err)
	}

	// Clean response - remove markdown fences if present
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var result ClassificationResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse classification JSON: %w\nResponse: %s", err, response)
	}

	// Validate required fields
	if result.Directory == "" || result.Filename == "" || result.Summary == "" {
		return nil, fmt.Errorf("classification missing required fields: directory=%q, filename=%q, summary=%q",
			result.Directory, result.Filename, result.Summary)
	}

	return &result, nil
}
