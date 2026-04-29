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
func Ingest(cfg *config.Config, provider providers.LLMProvider, embedder providers.Embedder, g *graph.Graph, content string, opts IngestOptions) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content is empty")
	}

	// Step 1: Exact dedup via SHA256
	if !opts.Force {
		dup, err := CheckExactDuplicate(cfg.KnowledgeBase.Path, content)
		if err == nil && dup.IsDuplicate {
			fmt.Printf("Duplicate detected (exact match): %s\nSkipping. Use --force to override.\n", dup.ExistingFile)
			return nil
		}
	}

	// Load skill.md
	skillPrompt, err := LoadSkill(cfg)
	if err != nil {
		skillPrompt = "You are a knowledge management agent."
	}

	// Step 2: Classify content
	fmt.Println("Classifying content...")
	result, err := Classify(provider, skillPrompt, content)
	if err != nil {
		return fmt.Errorf("classification failed: %w", err)
	}

	fmt.Printf("  Topic:    %s\n", result.Directory)
	fmt.Printf("  File:     %s.md\n", result.Filename)
	fmt.Printf("  Keywords: %s\n", strings.Join(result.Keywords, ", "))

	// Step 3: Semantic dedup — check graph FTS for similar content
	if !opts.Force && g != nil {
		candidates, err := buildSemanticCandidates(cfg.KnowledgeBase.Path, g, strings.Join(result.Keywords, " "))
		if err == nil && len(candidates) > 0 {
			semDup, err := CheckSemanticDuplicate(provider, content, candidates)
			if err == nil && semDup.IsDuplicate {
				fmt.Printf("Duplicate detected (semantic ~%d%% overlap). Skipping. Use --force to override.\n", semDup.SimilarityPct)
				return nil
			}
		}
	}

	// Step 4: Write knowledge file
	dirPath := filepath.Join(cfg.KnowledgeBase.Path, result.Directory)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	filePath := filepath.Join(dirPath, result.Filename+".md")
	if err := writeKnowledgeFile(filePath, result, content); err != nil {
		return fmt.Errorf("failed to write knowledge file: %w", err)
	}
	fmt.Printf("  Written:  %s\n", filePath)

	// Step 5: Update hierarchy files
	if err := UpdateHierarchy(cfg.KnowledgeBase.Path, result.Directory); err != nil {
		fmt.Printf("  Warning: failed to update hierarchy: %v\n", err)
	}
	if err := UpdateRootHierarchy(cfg.KnowledgeBase.Path); err != nil {
		fmt.Printf("  Warning: failed to update root hierarchy: %v\n", err)
	}

	// Step 6: Update graph
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
			for _, related := range result.RelatedTopics {
				nodes, err := g.Search(related, 1)
				if err == nil && len(nodes) > 0 {
					g.AddEdge(nodeID, nodes[0].ID, "related", 0.7)
				}
			}
			fmt.Println("  Graph updated")

			// Step 7: Embed and store chunks for semantic search
			if embedder != nil {
				embedChunks(g, embedder, nodeID, content)
			}

			// Step 8: Bidirectional cross-references (was Step 7)
			relPath, _ := filepath.Rel(cfg.KnowledgeBase.Path, filePath)
			if err := UpdateCrossRefs(cfg, provider, g, relPath); err != nil {
				fmt.Printf("  Warning: cross-ref update failed: %v\n", err)
			}
		}
	}

	fmt.Println("\nIngestion complete!")
	return nil
}

// buildSemanticCandidates reads top FTS matches and returns their file contents.
func buildSemanticCandidates(knowledgePath string, g *graph.Graph, keywords string) ([]string, error) {
	nodes, err := g.Search(keywords, 3)
	if err != nil || len(nodes) == 0 {
		return nil, err
	}
	var candidates []string
	for _, n := range nodes {
		data, err := os.ReadFile(filepath.Join(knowledgePath, n.FilePath))
		if err == nil {
			candidates = append(candidates, string(data))
		}
	}
	return candidates, nil
}

// writeKnowledgeFile creates or appends to a knowledge markdown file.
// Embeds SHA256 hash of original content for exact dedup checks.
func writeKnowledgeFile(filePath string, result *ClassificationResult, originalContent string) error {
	timestamp := time.Now().Format("2006-01-02 15:04")

	source := originalContent
	if len(source) > 200 {
		source = source[:200] + "..."
	}

	entry := fmt.Sprintf("\n## Entry — %s\n\n%s\n\n**Source:** %s\n**Keywords:** %s\n**Hash:** %s\n",
		timestamp,
		result.Summary,
		source,
		strings.Join(result.Keywords, ", "),
		SHA256HashForFile(originalContent),
	)

	// Append to existing file
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

// embedChunks splits content into overlapping chunks, embeds each, and stores them.
func embedChunks(g *graph.Graph, embedder providers.Embedder, nodeID int64, content string) {
	chunks := chunkText(content, 400, 50)
	g.DeleteChunksByNode(nodeID)
	for i, chunk := range chunks {
		vec, err := embedder.Embed(chunk)
		if err != nil {
			fmt.Printf("  Warning: embed chunk %d failed: %v\n", i, err)
			continue
		}
		if err := g.SaveChunk(nodeID, i, chunk, vec); err != nil {
			fmt.Printf("  Warning: save chunk %d failed: %v\n", i, err)
		}
	}
	fmt.Printf("  Embedded %d chunk(s)\n", len(chunks))
}

// chunkText splits text into overlapping windows by word count.
func chunkText(text string, size, overlap int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var chunks []string
	for i := 0; i < len(words); i += size - overlap {
		end := i + size
		if end > len(words) {
			end = len(words)
		}
		chunks = append(chunks, strings.Join(words[i:end], " "))
		if end == len(words) {
			break
		}
	}
	return chunks
}
