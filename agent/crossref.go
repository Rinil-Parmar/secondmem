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

// CrossRefResult holds the related files identified for a piece of content.
type CrossRefResult struct {
	RelatedFiles []string // relative paths
}

const crossRefPrompt = `Given the NEW FILE content and a list of EXISTING FILES, identify up to 5 existing files that are closely related.

Return ONLY a JSON array of relative file paths (the exact paths provided), e.g.:
["ai-ml/transformers.md", "engineering/attention-mechanism.md"]

If no files are closely related, return: []

NEW FILE: %s

EXISTING FILES:
%s

Return ONLY valid JSON array, no explanation.`

// FindRelatedFiles asks the LLM to identify related files for cross-referencing.
func FindRelatedFiles(provider providers.LLMProvider, newFileContent string, candidates []graph.Node) (*CrossRefResult, error) {
	if len(candidates) == 0 {
		return &CrossRefResult{}, nil
	}

	var fileList strings.Builder
	for _, n := range candidates {
		fileList.WriteString(fmt.Sprintf("- %s: %s\n", n.FilePath, n.Summary))
	}

	prompt := fmt.Sprintf(crossRefPrompt, newFileContent, fileList.String())
	response, err := provider.Complete("You are a knowledge graph builder. Identify related content.", prompt)
	if err != nil {
		return nil, fmt.Errorf("cross-ref identification failed: %w", err)
	}

	response = cleanJSON(response)
	var paths []string
	if err := parseJSON(response, &paths); err != nil {
		return &CrossRefResult{}, nil
	}

	return &CrossRefResult{RelatedFiles: paths}, nil
}

// AddCrossRef adds a "## Related" section to a file, appending to existing ones.
// Bidirectional: adds the link to both files.
func AddCrossRef(cfg *config.Config, sourceRelPath, targetRelPath string) error {
	sourcePath := filepath.Join(cfg.KnowledgeBase.Path, sourceRelPath)
	targetPath := filepath.Join(cfg.KnowledgeBase.Path, targetRelPath)

	if err := addRelatedLink(sourcePath, targetRelPath); err != nil {
		return fmt.Errorf("failed to add link in %s: %w", sourceRelPath, err)
	}
	if err := addRelatedLink(targetPath, sourceRelPath); err != nil {
		return fmt.Errorf("failed to add link in %s: %w", targetRelPath, err)
	}
	return nil
}

// addRelatedLink appends a link to the "## Related" section of a file.
func addRelatedLink(filePath, linkTarget string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	content := string(data)
	linkLine := fmt.Sprintf("- [%s](%s)", formatTitle(strings.TrimSuffix(filepath.Base(linkTarget), ".md")), linkTarget)

	// Already linked?
	if strings.Contains(content, linkTarget) {
		return nil
	}

	// Append to existing "## Related" section or add new one
	if strings.Contains(content, "\n## Related\n") {
		content = strings.Replace(content, "\n## Related\n", "\n## Related\n"+linkLine+"\n", 1)
	} else {
		content += "\n## Related\n" + linkLine + "\n"
	}

	return os.WriteFile(filePath, []byte(content), 0644)
}

// UpdateCrossRefs runs cross-reference linking for a newly ingested file.
func UpdateCrossRefs(cfg *config.Config, provider providers.LLMProvider, g *graph.Graph, newFileRelPath string) error {
	newFilePath := filepath.Join(cfg.KnowledgeBase.Path, newFileRelPath)
	newContent := readFileContent(newFilePath)
	if newContent == "" {
		return nil
	}

	// Search graph for related nodes (exclude self)
	newNode, err := g.GetNodeByPath(newFileRelPath)
	if err != nil || newNode == nil {
		return nil
	}

	candidates, err := g.Search(newNode.Keywords, 10)
	if err != nil {
		return nil
	}

	// Filter out self
	var filtered []graph.Node
	for _, n := range candidates {
		if n.FilePath != newFileRelPath {
			filtered = append(filtered, n)
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	// Ask LLM for related files
	result, err := FindRelatedFiles(provider, newContent, filtered)
	if err != nil || len(result.RelatedFiles) == 0 {
		return nil
	}

	// Add bidirectional links
	for _, relPath := range result.RelatedFiles {
		targetPath := filepath.Join(cfg.KnowledgeBase.Path, relPath)
		if _, err := os.Stat(targetPath); err != nil {
			continue // skip if file doesn't exist
		}
		if err := AddCrossRef(cfg, newFileRelPath, relPath); err != nil {
			fmt.Printf("  Warning: cross-ref failed (%s ↔ %s): %v\n", newFileRelPath, relPath, err)
		} else {
			// Add graph edge
			targetNode, err := g.GetNodeByPath(relPath)
			if err == nil && targetNode != nil {
				g.AddEdge(newNode.ID, targetNode.ID, "references", 0.9)
			}
		}
	}

	return nil
}
