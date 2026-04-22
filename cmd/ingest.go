package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Rinil-Parmar/secondmem/agent"
	"github.com/Rinil-Parmar/secondmem/config"
	"github.com/Rinil-Parmar/secondmem/graph"
	"github.com/Rinil-Parmar/secondmem/parsers"
	"github.com/spf13/cobra"
)

var ingestForce bool

var ingestCmd = &cobra.Command{
	Use:   "ingest [text or file path]",
	Short: "Ingest content into the knowledge base",
	Long:  "Ingest text, a file, or piped input into the knowledge base. The AI classifies and organizes it automatically.",
	RunE:  runIngest,
}

func init() {
	ingestCmd.Flags().BoolVar(&ingestForce, "force", false, "skip deduplication checks")
	rootCmd.AddCommand(ingestCmd)
}

func runIngest(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine content source
	var content string

	if len(args) > 0 {
		input := strings.Join(args, " ")

		// Check if it's a file path
		if info, err := os.Stat(input); err == nil && !info.IsDir() {
			fmt.Printf("Reading from file: %s\n", input)
			if strings.HasSuffix(strings.ToLower(input), ".pdf") {
				content, err = parsers.ParsePDFFile(input)
				if err != nil {
					return fmt.Errorf("failed to parse PDF: %w", err)
				}
			} else {
				data, err := os.ReadFile(input)
				if err != nil {
					return fmt.Errorf("failed to read file %s: %w", input, err)
				}
				content = string(data)
			}
		} else {
			content = input
		}
	} else {
		// Try stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
			content = string(data)
		}
	}

	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("no content provided. Usage: secondmem ingest \"your text\" or secondmem ingest file.txt")
	}

	// Initialize provider
	provider, err := newProvider(cfg)
	if err != nil {
		return err
	}

	// Open graph
	var g *graph.Graph
	if cfg.Graph.Enabled {
		g, err = graph.Open(cfg.Graph.DBPath)
		if err != nil {
			fmt.Printf("Warning: could not open graph database: %v\n", err)
		} else {
			defer g.Close()
		}
	}

	return agent.Ingest(cfg, provider, g, content, agent.IngestOptions{
		Force: ingestForce,
	})
}
