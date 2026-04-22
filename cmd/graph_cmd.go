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
	Short: "Rebuild the graph by scanning all knowledge files",
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

		added := 0
		err = filepath.Walk(cfg.KnowledgeBase.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") || info.Name() == "hierarchy.md" {
				return nil
			}
			relPath, _ := filepath.Rel(cfg.KnowledgeBase.Path, path)
			dir := filepath.Dir(relPath)

			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			content := string(data)

			// Extract title from first # heading
			title := strings.TrimSuffix(info.Name(), ".md")
			for _, line := range strings.SplitN(content, "\n", 5) {
				if strings.HasPrefix(line, "# ") {
					title = strings.TrimPrefix(line, "# ")
					break
				}
			}

			// Extract keywords from **Keywords:** line
			keywords := ""
			for _, line := range strings.Split(content, "\n") {
				if strings.HasPrefix(line, "**Keywords:**") {
					keywords = strings.TrimPrefix(line, "**Keywords:** ")
					break
				}
			}

			lineCount := strings.Count(content, "\n") + 1
			_, upsertErr := g.UpsertNode(graph.Node{
				FilePath:  relPath,
				Directory: dir,
				Title:     title,
				Keywords:  keywords,
				NodeType:  "file",
				LineCount: lineCount,
			})
			if upsertErr == nil {
				added++
			}
			return nil
		})
		if err != nil {
			return err
		}

		fmt.Printf("Graph rebuilt: %d node(s) indexed.\n", added)
		return nil
	},
}

var graphValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate graph nodes against filesystem",
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

		nodes, err := g.AllNodes()
		if err != nil {
			return fmt.Errorf("failed to read nodes: %w", err)
		}

		stale := 0
		for _, n := range nodes {
			absPath := filepath.Join(cfg.KnowledgeBase.Path, n.FilePath)
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				fmt.Printf("  STALE NODE: %s (file missing)\n", n.FilePath)
				stale++
			}
		}

		// Check for files not in graph
		missing := 0
		filepath.Walk(cfg.KnowledgeBase.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") || info.Name() == "hierarchy.md" {
				return nil
			}
			relPath, _ := filepath.Rel(cfg.KnowledgeBase.Path, path)
			for _, n := range nodes {
				if n.FilePath == relPath {
					return nil
				}
			}
			fmt.Printf("  MISSING NODE: %s (file exists, not in graph)\n", relPath)
			missing++
			return nil
		})

		if stale == 0 && missing == 0 {
			fmt.Printf("Graph valid: %d node(s), no issues.\n", len(nodes))
		} else {
			fmt.Printf("\n%d stale, %d missing. Run 'secondmem graph rebuild' to fix.\n", stale, missing)
		}
		return nil
	},
}

var graphConnectionsCmd = &cobra.Command{
	Use:   "connections [file-path]",
	Short: "Show connections for a knowledge file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return err
		}
		g, err := graph.Open(cfg.Graph.DBPath)
		if err != nil {
			return fmt.Errorf("failed to open graph: %w", err)
		}
		defer g.Close()

		// Accept partial path or full relative path
		var node *graph.Node
		node, err = g.GetNodeByPath(query)
		if err != nil || node == nil {
			// Fall back to FTS search
			nodes, searchErr := g.Search(query, 1)
			if searchErr != nil || len(nodes) == 0 {
				return fmt.Errorf("no node found for %q", query)
			}
			node = &nodes[0]
		}

		fmt.Printf("Connections: %s\n", node.FilePath)
		fmt.Printf("  Title:    %s\n", node.Title)
		fmt.Printf("  Keywords: %s\n\n", node.Keywords)

		related, err := g.GetRelated(node.ID)
		if err != nil {
			return fmt.Errorf("failed to get connections: %w", err)
		}

		if len(related) == 0 {
			fmt.Println("  No connections found.")
			return nil
		}

		fmt.Printf("  %d connection(s):\n", len(related))
		for _, r := range related {
			fmt.Printf("    %-45s  %s\n", r.FilePath, r.Keywords)
		}
		return nil
	},
}

func init() {
	graphCmd.AddCommand(graphStatsCmd)
	graphCmd.AddCommand(graphSearchCmd)
	graphCmd.AddCommand(graphRebuildCmd)
	graphCmd.AddCommand(graphValidateCmd)
	graphCmd.AddCommand(graphConnectionsCmd)
	rootCmd.AddCommand(graphCmd)
}
