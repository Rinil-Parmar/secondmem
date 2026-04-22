package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rinil-Parmar/secondmem/config"
	"github.com/Rinil-Parmar/secondmem/graph"
	"github.com/Rinil-Parmar/secondmem/providers"
)

const askPrompt = `You are a knowledge retrieval agent. Answer the user's question based ONLY on the provided context files.

Rules:
- Only use information from the provided context
- If the context doesn't contain relevant information, say so
- Cite the source file paths when referencing information
- Be concise and direct
- If multiple sources agree, synthesize them into a single answer`

// Ask queries the knowledge base and returns an AI-synthesized answer.
func Ask(cfg *config.Config, provider providers.LLMProvider, g *graph.Graph, question string, cite bool) (string, error) {
	if strings.TrimSpace(question) == "" {
		return "", fmt.Errorf("question is empty")
	}

	// Load skill.md
	skillPrompt, err := LoadSkill(cfg)
	if err != nil {
		skillPrompt = "You are a knowledge management agent."
	}

	// Rewrite natural language question into search keywords
	searchQuery := extractSearchKeywords(provider, skillPrompt, question)

	// Search graph for relevant nodes
	var contextFiles []string
	var filePaths []string

	if g != nil {
		nodes, err := g.Search(searchQuery, 5)
		if err == nil && len(nodes) > 0 {
			// Expand with related nodes
			seen := make(map[int64]bool)
			var allNodes []graph.Node
			for _, n := range nodes {
				if !seen[n.ID] {
					seen[n.ID] = true
					allNodes = append(allNodes, n)
				}
				related, err := g.GetRelated(n.ID)
				if err == nil {
					for _, r := range related {
						if !seen[r.ID] {
							seen[r.ID] = true
							allNodes = append(allNodes, r)
						}
					}
				}
			}

			// Read file contents (limit to top 7)
			limit := 7
			if len(allNodes) < limit {
				limit = len(allNodes)
			}
			for _, n := range allNodes[:limit] {
				fullPath := filepath.Join(cfg.KnowledgeBase.Path, n.FilePath)
				content, err := os.ReadFile(fullPath)
				if err != nil {
					continue
				}
				contextFiles = append(contextFiles, fmt.Sprintf("--- File: %s ---\n%s", n.FilePath, string(content)))
				filePaths = append(filePaths, n.FilePath)
			}
		}
	}

	// Fallback: if no graph results, scan hierarchy
	if len(contextFiles) == 0 {
		contextFiles, filePaths = fallbackHierarchySearch(cfg, provider, skillPrompt, question)
	}

	if len(contextFiles) == 0 {
		return "No relevant knowledge found in your knowledge base. Try ingesting some content first.", nil
	}

	// Build the prompt
	systemPrompt := skillPrompt + "\n\n" + askPrompt
	userPrompt := fmt.Sprintf("Context:\n%s\n\nQuestion: %s", strings.Join(contextFiles, "\n\n"), question)

	if cite {
		userPrompt += "\n\nPlease cite the source file paths in your answer."
	}

	answer, err := provider.Complete(systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate answer: %w", err)
	}

	if cite && len(filePaths) > 0 {
		answer += "\n\n---\nSources:\n"
		for _, p := range filePaths {
			answer += fmt.Sprintf("  - %s\n", p)
		}
	}

	return answer, nil
}

// extractSearchKeywords rewrites a natural language question into FTS-friendly keywords.
// Falls back to the original question if LLM call fails.
func extractSearchKeywords(provider providers.LLMProvider, skillPrompt, question string) string {
	prompt := `Extract 5-8 search keywords from this question. Return ONLY the keywords as a comma-separated list, no explanation.
Question: ` + question
	result, err := provider.Complete(skillPrompt, prompt)
	if err != nil || strings.TrimSpace(result) == "" {
		return question
	}
	return strings.TrimSpace(result)
}

// fallbackHierarchySearch reads hierarchy files to find relevant content.
func fallbackHierarchySearch(cfg *config.Config, provider providers.LLMProvider, skillPrompt, question string) ([]string, []string) {
	// Read root hierarchy
	rootHierarchy := filepath.Join(cfg.KnowledgeBase.Path, "hierarchy.md")
	data, err := os.ReadFile(rootHierarchy)
	if err != nil {
		return nil, nil
	}

	// Ask LLM which directories are relevant
	dirPrompt := fmt.Sprintf("Given this knowledge base structure:\n%s\n\nWhich directories are most likely to contain information relevant to this question: %q\n\nReturn ONLY the directory names, one per line, no explanation. Max 3 directories.", string(data), question)
	dirResponse, err := provider.Complete(skillPrompt, dirPrompt)
	if err != nil {
		return nil, nil
	}

	var contextFiles []string
	var filePaths []string

	dirs := strings.Split(strings.TrimSpace(dirResponse), "\n")
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		dir = strings.Trim(dir, "- []/()")
		if dir == "" {
			continue
		}

		// Read all .md files in the directory
		dirPath := filepath.Join(cfg.KnowledgeBase.Path, dir)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || entry.Name() == "hierarchy.md" || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			fullPath := filepath.Join(dirPath, entry.Name())
			content, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}
			relPath := filepath.Join(dir, entry.Name())
			contextFiles = append(contextFiles, fmt.Sprintf("--- File: %s ---\n%s", relPath, string(content)))
			filePaths = append(filePaths, relPath)
		}
	}

	return contextFiles, filePaths
}
