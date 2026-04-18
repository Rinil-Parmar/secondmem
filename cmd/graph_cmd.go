package cmd

import (
	"fmt"
	"strings"

	"github.com/Rinil-Parmar/secondmem/config"
	"github.com/Rinil-Parmar/secondmem/graph"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Manage the LORE-GRAPH knowledge graph",
}

var graphStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show graph statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return err
		}
		g, err := graph.Open(cfg.Graph.DBPath)
		if err != nil {
			return fmt.Errorf("failed to open graph: %w", err)
		}
		defer g.Close()

		nodes, edges, fts, _ := g.Stats()
		fmt.Println("LORE-GRAPH Statistics")
		fmt.Println("=====================")
		fmt.Printf("  Nodes:       %d\n", nodes)
		fmt.Printf("  Edges:       %d\n", edges)
		fmt.Printf("  FTS entries: %d\n", fts)
		return nil
	},
}

var graphSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search the knowledge graph",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return err
		}
		g, err := graph.Open(cfg.Graph.DBPath)
		if err != nil {
			return fmt.Errorf("failed to open graph: %w", err)
		}
		defer g.Close()

		nodes, err := g.Search(query, 10)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		if len(nodes) == 0 {
			fmt.Println("No results found.")
			return nil
		}

		fmt.Printf("Found %d result(s):\n\n", len(nodes))
		for _, n := range nodes {
			fmt.Printf("  %s\n", n.FilePath)
			fmt.Printf("    Title:    %s\n", n.Title)
			fmt.Printf("    Keywords: %s\n", n.Keywords)
			fmt.Println()
		}
		return nil
	},
}

var graphRebuildCmd = &cobra.Command{
	Use:   "rebuild",
	Short: "Rebuild the graph from markdown files",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Graph rebuild not yet implemented.")
		return nil
	},
}

var graphValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate graph against filesystem",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Graph validation not yet implemented.")
		return nil
	},
}

func init() {
	graphCmd.AddCommand(graphStatsCmd)
	graphCmd.AddCommand(graphSearchCmd)
	graphCmd.AddCommand(graphRebuildCmd)
	graphCmd.AddCommand(graphValidateCmd)
	rootCmd.AddCommand(graphCmd)
}
