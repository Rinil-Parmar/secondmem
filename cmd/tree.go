package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rinil-Parmar/secondmem/config"
	"github.com/spf13/cobra"
)

var treeDepth int

var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Display knowledge base structure",
	RunE:  runTree,
}

func init() {
	treeCmd.Flags().IntVar(&treeDepth, "depth", 2, "maximum depth to display")
	rootCmd.AddCommand(treeCmd)
}

func runTree(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println(cfg.KnowledgeBase.Path)
	return printTree(cfg.KnowledgeBase.Path, "", treeDepth)
}

func printTree(path, prefix string, depth int) error {
	if depth <= 0 {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// Filter: only dirs and .md files (skip hierarchy.md)
	var filtered []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			filtered = append(filtered, e)
		} else if strings.HasSuffix(e.Name(), ".md") && e.Name() != "hierarchy.md" {
			filtered = append(filtered, e)
		}
	}

	for i, entry := range filtered {
		isLast := i == len(filtered)-1
		connector := "├── "
		childPrefix := "│   "
		if isLast {
			connector = "└── "
			childPrefix = "    "
		}

		fmt.Printf("%s%s%s\n", prefix, connector, entry.Name())

		if entry.IsDir() {
			printTree(filepath.Join(path, entry.Name()), prefix+childPrefix, depth-1)
		}
	}

	return nil
}
