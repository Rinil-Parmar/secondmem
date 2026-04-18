package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Rinil-Parmar/secondmem/config"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Check integrity of the knowledge base",
	RunE:  runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("Validating knowledge base...")
	issues := 0

	// Check each directory has hierarchy.md
	entries, err := os.ReadDir(cfg.KnowledgeBase.Path)
	if err != nil {
		return fmt.Errorf("cannot read knowledge base: %w", err)
	}

	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(cfg.KnowledgeBase.Path, entry.Name())
		hierarchyPath := filepath.Join(dirPath, "hierarchy.md")

		if _, err := os.Stat(hierarchyPath); os.IsNotExist(err) {
			fmt.Printf("  MISSING: %s/hierarchy.md\n", entry.Name())
			issues++
			continue
		}

		// Check links in hierarchy.md
		data, err := os.ReadFile(hierarchyPath)
		if err != nil {
			continue
		}

		matches := linkRegex.FindAllStringSubmatch(string(data), -1)
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}
			linkTarget := match[2]
			if strings.HasPrefix(linkTarget, "http") {
				continue
			}
			targetPath := filepath.Join(dirPath, linkTarget)
			if _, err := os.Stat(targetPath); os.IsNotExist(err) {
				fmt.Printf("  DEAD LINK: %s/hierarchy.md -> %s\n", entry.Name(), linkTarget)
				issues++
			}
		}

		// Check for files not listed in hierarchy
		subEntries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		hierarchyContent := string(data)
		for _, sub := range subEntries {
			if sub.IsDir() || sub.Name() == "hierarchy.md" || !strings.HasSuffix(sub.Name(), ".md") {
				continue
			}
			if !strings.Contains(hierarchyContent, sub.Name()) {
				fmt.Printf("  ORPHAN: %s/%s not in hierarchy.md\n", entry.Name(), sub.Name())
				issues++
			}
		}
	}

	// Check root hierarchy
	rootHierarchy := filepath.Join(cfg.KnowledgeBase.Path, "hierarchy.md")
	if _, err := os.Stat(rootHierarchy); os.IsNotExist(err) {
		fmt.Println("  MISSING: root hierarchy.md")
		issues++
	}

	if issues == 0 {
		fmt.Println("All checks passed!")
	} else {
		fmt.Printf("\nFound %d issue(s)\n", issues)
	}

	return nil
}
