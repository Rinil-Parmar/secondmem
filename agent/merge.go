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

const mergePrompt = `You are a knowledge consolidator. Two related knowledge files need to be merged into one.

Merge the content into a single coherent knowledge document. Eliminate redundancy but preserve all unique insights.

FILE A:
%s

FILE B:
%s

Return ONLY the merged markdown content. Start with a # Title header. No explanation.`

// MergeFiles merges two knowledge files into one, removes the originals.
// Returns the path of the merged file.
func MergeFiles(cfg *config.Config, provider providers.LLMProvider, g *graph.Graph, pathA, pathB string) (string, error) {
	contentA, err := os.ReadFile(pathA)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", pathA, err)
	}
	contentB, err := os.ReadFile(pathB)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", pathB, err)
	}

	skillPrompt, err := LoadSkill(cfg)
	if err != nil {
		skillPrompt = "You are a knowledge management agent."
	}

	prompt := fmt.Sprintf(mergePrompt, string(contentA), string(contentB))
	merged, err := provider.Complete(skillPrompt, prompt)
	if err != nil {
		return "", fmt.Errorf("merge failed: %w", err)
	}

	merged = strings.TrimSpace(merged)

	// Derive merged filename from the first file's name
	dir := filepath.Dir(pathA)
	baseName := strings.TrimSuffix(filepath.Base(pathA), ".md") + "-merged"
	mergedPath := filepath.Join(dir, baseName+".md")

	// Append a timestamp entry so dedup hashes remain valid
	timestamp := time.Now().Format("2006-01-02 15:04")
	merged += fmt.Sprintf("\n\n*Merged on %s*\n", timestamp)

	if err := os.WriteFile(mergedPath, []byte(merged), 0644); err != nil {
		return "", fmt.Errorf("write merged file: %w", err)
	}

	// Update graph: remove old nodes, upsert merged node
	relDir, _ := filepath.Rel(cfg.KnowledgeBase.Path, dir)
	if g != nil {
		relA, _ := filepath.Rel(cfg.KnowledgeBase.Path, pathA)
		relB, _ := filepath.Rel(cfg.KnowledgeBase.Path, pathB)
		g.DeleteNode(relA)
		g.DeleteNode(relB)

		relMerged := filepath.Join(relDir, baseName+".md")
		lineCount := strings.Count(merged, "\n") + 1
		g.UpsertNode(graph.Node{
			FilePath:  relMerged,
			Directory: relDir,
			Title:     formatTitle(baseName),
			Summary:   "Merged knowledge entry",
			NodeType:  "file",
			LineCount: lineCount,
		})
	}

	// Remove originals
	os.Remove(pathA)
	os.Remove(pathB)

	return mergedPath, nil
}
