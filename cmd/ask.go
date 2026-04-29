package cmd

import (
	"fmt"
	"strings"

	"github.com/Rinil-Parmar/secondmem/agent"
	"github.com/Rinil-Parmar/secondmem/config"
	"github.com/Rinil-Parmar/secondmem/graph"
	"github.com/spf13/cobra"
)

var askCite bool

var askCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Query your knowledge base",
	Long:  "Ask a question and get an answer synthesized from your stored knowledge.",
	RunE:  runAsk,
}

func init() {
	askCmd.Flags().BoolVar(&askCite, "cite", false, "include source file citations")
	rootCmd.AddCommand(askCmd)
}

func runAsk(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("please provide a question. Usage: secondmem ask \"your question\"")
	}

	question := strings.Join(args, " ")

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	provider, err := newProvider(cfg)
	if err != nil {
		return err
	}

	embedder := newEmbedder(cfg)

	var g *graph.Graph
	if cfg.Graph.Enabled {
		g, err = graph.Open(cfg.Graph.DBPath)
		if err != nil {
			fmt.Printf("Warning: could not open graph database: %v\n", err)
		} else {
			defer g.Close()
		}
	}

	answer, err := agent.Ask(cfg, provider, embedder, g, question, askCite)
	if err != nil {
		return err
	}

	fmt.Println(answer)
	return nil
}
