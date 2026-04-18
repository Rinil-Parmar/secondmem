package cmd

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rinil-Parmar/secondmem/config"
	"github.com/spf13/cobra"
)

// SkillTemplate is set by main.go with the embedded skill.md template.
var SkillTemplate embed.FS

var initPath string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the secondmem knowledge base",
	Long:  "Creates the ~/.secondmem/ directory structure with config, skill.md, knowledge directory, and database.",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().StringVar(&initPath, "path", "", "custom base path (default: ~/.secondmem)")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	basePath := initPath
	if basePath == "" {
		basePath = config.DefaultBasePath()
	}

	fmt.Printf("Initializing secondmem at %s\n", basePath)

	// Create directory structure
	dirs := []string{
		basePath,
		filepath.Join(basePath, "knowledge"),
		filepath.Join(basePath, "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Write config.toml if it doesn't exist
	configPath := filepath.Join(basePath, "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configContent := `[model]
provider = "openai"

[openai]
api_key = ""
model = "gpt-4o"

[knowledge_base]
path = "` + filepath.Join(basePath, "knowledge") + `"
max_file_lines = 1116
auto_rebalance = true

[graph]
enabled = true
db_path = "` + filepath.Join(basePath, "secondmem.db") + `"

[git]
auto_commit = false
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			return fmt.Errorf("failed to write config.toml: %w", err)
		}
		fmt.Println("  Created config.toml")
	} else {
		fmt.Println("  config.toml already exists, skipping")
	}

	// Write skill.md if it doesn't exist
	skillPath := filepath.Join(basePath, "skill.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		skillContent, err := SkillTemplate.ReadFile("templates/skill.md")
		if err != nil {
			return fmt.Errorf("failed to read skill template: %w", err)
		}
		if err := os.WriteFile(skillPath, skillContent, 0644); err != nil {
			return fmt.Errorf("failed to write skill.md: %w", err)
		}
		fmt.Println("  Created skill.md")
	} else {
		fmt.Println("  skill.md already exists, skipping")
	}

	// Write root hierarchy.md if it doesn't exist
	hierarchyPath := filepath.Join(basePath, "knowledge", "hierarchy.md")
	if _, err := os.Stat(hierarchyPath); os.IsNotExist(err) {
		hierarchyContent := `# Knowledge Base

## Topics

_No topics yet. Use 'secondmem ingest' to add knowledge._
`
		if err := os.WriteFile(hierarchyPath, []byte(hierarchyContent), 0644); err != nil {
			return fmt.Errorf("failed to write hierarchy.md: %w", err)
		}
		fmt.Println("  Created knowledge/hierarchy.md")
	} else {
		fmt.Println("  hierarchy.md already exists, skipping")
	}

	fmt.Println("\nsecondmem initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Set your OpenAI API key:")
	fmt.Println("     secondmem config set openai.api_key sk-...")
	fmt.Println("  2. Start ingesting knowledge:")
	fmt.Println("     secondmem ingest \"Your text here\"")

	return nil
}
