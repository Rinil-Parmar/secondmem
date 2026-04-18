package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rinil-Parmar/secondmem/config"
	"github.com/Rinil-Parmar/secondmem/graph"
	"github.com/Rinil-Parmar/secondmem/providers"
)

// IngestOptions configures the ingestion behavior.
type IngestOptions struct {
	Force bool // Skip deduplication checks
}

// Ingest processes content and stores it in the knowledge base.
func Ingest(cfg *config.Config, provider providers.LLMProvider, g *graph.Graph, content string, opts IngestOptions) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content is empty")
	}

	// Load skill.md
	skillPrompt, err := LoadSkill(cfg)
	if err != nil {
		fmt.Println("Warning: could not load skill.md, using defaults")
		skillPrompt = "You are a knowledge management agent."
	}

	// Classify content
	fmt.Println("Classifying content...")
	result, err := Classify(provider, skillPrompt, content)
	if err != nil {
		return fmt.Errorf("classification failed: %w", err)
	}

	fmt.Printf("  Topic: %s\n", result.Directory)
	fmt.Printf("  File:  %s.md\n", result.Filename)
	fmt.Printf("  Keywords: %s\n", strings.Join(result.Keywords, ", "))

	// Ensure directory exists
	dirPath := filepath.Join(cfg.KnowledgeBase.Path, result.Directory)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	// Write knowledge file
	filePath := filepath.Join(dirPath, result.Filename+".md")
	if err := writeKnowledgeFile(filePath, result, content); err != nil {
		return fmt.Errorf("failed to write knowledge file: %w", err)
	}
	fmt.Printf("  Written to: %s\n", filePath)

	// Update hierarchy files
	if err := UpdateHierarchy(cfg.KnowledgeBase.Path, result.Directory); err != nil {
		fmt.Printf("  Warning: failed to update directory hierarchy: %v\n", err)
	}
	if err := UpdateRootHierarchy(cfg.KnowledgeBase.Path); err != nil {
		fmt.Printf("  Warning: failed to update root hierarchy: %v\n", err)
	}

	// Update graph
	if g != nil {
		relPath, _ := filepath.Rel(cfg.KnowledgeBase.Path, filePath)
		lineCount := strings.Count(readFileContent(filePath), "\n") + 1

		nodeID, err := g.UpsertNode(graph.Node{
			FilePath:  relPath,
			Directory: result.Directory,
			Title:     formatTitle(result.Filename),
			Summary:   result.Summary,
			Keywords:  strings.Join(result.Keywords, ", "),
			NodeType:  "file",
			LineCount: lineCount,
		})
		if err != nil {
			fmt.Printf("  Warning: failed to update graph: %v\n", err)
		} else {
			// Create edges to related topics
			for _, related := range result.RelatedTopics {
				nodes, err := g.Search(related, 1)
				if err == nil && len(nodes) > 0 {
					g.AddEdge(nodeID, nodes[0].ID, "related", 0.7)
				}
			}
			fmt.Println("  Graph updated")
		}
	}

	fmt.Println("\nIngestion complete!")
	return nil
}

// writeKnowledgeFile creates or appends to a knowledge markdown file.
func writeKnowledgeFile(filePath string, result *ClassificationResult, originalContent string) error {
	timestamp := time.Now().Format("2006-01-02 15:04")

	// Truncate original content for source reference
	source := originalContent
	if len(source) > 200 {
		source = source[:200] + "..."
	}

	entry := fmt.Sprintf("\n## Entry — %s\n\n%s\n\n**Source:** %s\n**Keywords:** %s\n",
		timestamp,
		result.Summary,
		source,
		strings.Join(result.Keywords, ", "),
	)

	// If file exists, append
	if _, err := os.Stat(filePath); err == nil {
		f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.WriteString(entry)
		return err
	}

	// New file
	title := formatTitle(strings.TrimSuffix(filepath.Base(filePath), ".md"))
	content := fmt.Sprintf("# %s\n%s", title, entry)
	return os.WriteFile(filePath, []byte(content), 0644)
}

// readFileContent reads a file and returns its content as a string.
func readFileContent(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
