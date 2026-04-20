package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rinil-Parmar/secondmem/config"
	"github.com/Rinil-Parmar/secondmem/graph"
	"github.com/Rinil-Parmar/secondmem/providers"
)

// SplitPlan describes how to divide an oversized file.
type SplitPlan struct {
	Files []SplitFile `json:"files"`
}

type SplitFile struct {
	Filename string   `json:"filename"` // kebab-case, no .md
	Theme    string   `json:"theme"`    // short description
	Lines    []int    `json:"lines"`    // 1-based line numbers to include
}

const splitPrompt = `You are a knowledge organizer. A markdown knowledge file has grown too large and must be split into thematic sub-files.

The file has %d lines. Split it into 2-4 smaller files grouped by theme. Each resulting file should be roughly equal in size.

Return ONLY a JSON object with this structure:
{
  "files": [
    {
      "filename": "kebab-case-name-no-extension",
      "theme": "short theme description",
      "lines": [1, 2, 3, ...]
    }
  ]
}

Rules:
- Every line number from 1 to %d must appear in exactly one file
- The first file keeps line 1 (the # Title header)
- Each file should get its own coherent theme
- filenames must be kebab-case, no .md extension
- Return ONLY valid JSON, no markdown fences`

// SplitOversizedFile splits a knowledge file exceeding maxLines into themed sub-files.
// Returns paths of new files created and removes the original.
func SplitOversizedFile(cfg *config.Config, provider providers.LLMProvider, g *graph.Graph, filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	lineCount := len(lines)

	if lineCount <= cfg.KnowledgeBase.MaxFileLines {
		return nil, fmt.Errorf("file has %d lines, under limit of %d", lineCount, cfg.KnowledgeBase.MaxFileLines)
	}

	skillPrompt, err := LoadSkill(cfg)
	if err != nil {
		skillPrompt = "You are a knowledge management agent."
	}

	prompt := fmt.Sprintf(splitPrompt, lineCount, lineCount)
	systemPrompt := skillPrompt + "\n\n" + prompt

	response, err := provider.Complete(systemPrompt, string(data))
	if err != nil {
		return nil, fmt.Errorf("split planning failed: %w", err)
	}

	response = cleanJSON(response)
	var plan SplitPlan
	if err := json.Unmarshal([]byte(response), &plan); err != nil {
		return nil, fmt.Errorf("parse split plan: %w\nResponse: %s", err, response)
	}

	if len(plan.Files) < 2 {
		return nil, fmt.Errorf("split plan returned fewer than 2 files")
	}

	dir := filepath.Dir(filePath)
	relDir, _ := filepath.Rel(cfg.KnowledgeBase.Path, dir)
	var created []string

	for i, sf := range plan.Files {
		if len(sf.Lines) == 0 {
			continue
		}

		sf.Filename = strings.ToLower(strings.ReplaceAll(sf.Filename, " ", "-"))
		sf.Filename = strings.TrimSuffix(sf.Filename, ".md")

		var content strings.Builder
		if i == 0 {
			content.WriteString(fmt.Sprintf("# %s\n\n", formatTitle(sf.Filename)))
		} else {
			content.WriteString(fmt.Sprintf("# %s\n\n", formatTitle(sf.Filename)))
		}

		for _, lineNum := range sf.Lines {
			if lineNum < 1 || lineNum > lineCount {
				continue
			}
			// Skip original title line for non-first files, use their own title
			if i > 0 && lineNum == 1 {
				continue
			}
			content.WriteString(lines[lineNum-1])
			content.WriteString("\n")
		}

		newPath := filepath.Join(dir, sf.Filename+".md")
		if err := os.WriteFile(newPath, []byte(content.String()), 0644); err != nil {
			return nil, fmt.Errorf("write split file %s: %w", sf.Filename, err)
		}
		created = append(created, newPath)

		// Update graph
		if g != nil {
			relPath := filepath.Join(relDir, sf.Filename+".md")
			newLineCount := strings.Count(content.String(), "\n") + 1
			g.UpsertNode(graph.Node{
				FilePath:  relPath,
				Directory: relDir,
				Title:     formatTitle(sf.Filename),
				Summary:   sf.Theme,
				Keywords:  "",
				NodeType:  "file",
				LineCount: newLineCount,
			})
		}
	}

	// Remove original file and its graph node
	relPath, _ := filepath.Rel(cfg.KnowledgeBase.Path, filePath)
	if g != nil {
		g.DeleteNode(relPath)
	}
	os.Remove(filePath)

	return created, nil
}
