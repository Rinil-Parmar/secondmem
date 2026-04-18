package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rinil-Parmar/secondmem/config"
	"github.com/Rinil-Parmar/secondmem/graph"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show knowledge base statistics",
	RunE:  runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)
}

func runStats(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Count directories and files
	topicCount := 0
	fileCount := 0
	totalLines := 0

	entries, err := os.ReadDir(cfg.KnowledgeBase.Path)
	if err != nil {
		return fmt.Errorf("knowledge base not found at %s. Run 'secondmem init' first.", cfg.KnowledgeBase.Path)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		topicCount++
		subEntries, err := os.ReadDir(filepath.Join(cfg.KnowledgeBase.Path, entry.Name()))
		if err != nil {
			continue
		}
		for _, sub := range subEntries {
			if sub.IsDir() || !strings.HasSuffix(sub.Name(), ".md") || sub.Name() == "hierarchy.md" {
				continue
			}
			fileCount++
			data, err := os.ReadFile(filepath.Join(cfg.KnowledgeBase.Path, entry.Name(), sub.Name()))
			if err == nil {
				totalLines += strings.Count(string(data), "\n") + 1
			}
		}
	}

	fmt.Println("Knowledge Base Statistics")
	fmt.Println("========================")
	fmt.Printf("  Topics:      %d\n", topicCount)
	fmt.Printf("  Files:       %d\n", fileCount)
	fmt.Printf("  Total lines: %d\n", totalLines)
	fmt.Printf("  Path:        %s\n", cfg.KnowledgeBase.Path)

	// Graph stats
	if cfg.Graph.Enabled {
		g, err := graph.Open(cfg.Graph.DBPath)
		if err == nil {
			defer g.Close()
			nodes, edges, fts, _ := g.Stats()
			fmt.Printf("\nGraph (LORE-GRAPH)\n")
			fmt.Printf("  Nodes:       %d\n", nodes)
			fmt.Printf("  Edges:       %d\n", edges)
			fmt.Printf("  FTS entries: %d\n", fts)
		}
	}

	return nil
}
