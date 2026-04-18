package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// UpdateHierarchy regenerates the hierarchy.md file for a given directory.
func UpdateHierarchy(knowledgePath, directory string) error {
	dirPath := filepath.Join(knowledgePath, directory)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("# %s", formatTitle(directory)))
	lines = append(lines, "")

	fileCount := 0
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "hierarchy.md" {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		title := formatTitle(name)
		lines = append(lines, fmt.Sprintf("- [%s](%s)", title, entry.Name()))
		fileCount++
	}

	if fileCount == 0 {
		lines = append(lines, "_No entries yet._")
	}

	lines = append(lines, "")

	hierarchyPath := filepath.Join(dirPath, "hierarchy.md")
	return os.WriteFile(hierarchyPath, []byte(strings.Join(lines, "\n")), 0644)
}

// UpdateRootHierarchy regenerates the root hierarchy.md from all topic directories.
func UpdateRootHierarchy(knowledgePath string) error {
	entries, err := os.ReadDir(knowledgePath)
	if err != nil {
		return fmt.Errorf("failed to read knowledge directory: %w", err)
	}

	var lines []string
	lines = append(lines, "# Knowledge Base")
	lines = append(lines, "")
	lines = append(lines, "## Topics")
	lines = append(lines, "")

	dirCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Count files in directory
		subEntries, err := os.ReadDir(filepath.Join(knowledgePath, entry.Name()))
		if err != nil {
			continue
		}
		fileCount := 0
		for _, sub := range subEntries {
			if !sub.IsDir() && strings.HasSuffix(sub.Name(), ".md") && sub.Name() != "hierarchy.md" {
				fileCount++
			}
		}

		title := formatTitle(entry.Name())
		lines = append(lines, fmt.Sprintf("- [%s](%s/) — %d files", title, entry.Name(), fileCount))
		dirCount++
	}

	if dirCount == 0 {
		lines = append(lines, "_No topics yet. Use 'secondmem ingest' to add knowledge._")
	}

	lines = append(lines, "")

	hierarchyPath := filepath.Join(knowledgePath, "hierarchy.md")
	return os.WriteFile(hierarchyPath, []byte(strings.Join(lines, "\n")), 0644)
}

// formatTitle converts kebab-case to Title Case.
func formatTitle(name string) string {
	parts := strings.Split(name, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}
