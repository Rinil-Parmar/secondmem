package cmd

import (
	"fmt"

	"github.com/Rinil-Parmar/secondmem/config"
	"github.com/Rinil-Parmar/secondmem/eval"
	"github.com/Rinil-Parmar/secondmem/graph"
	"github.com/spf13/cobra"
)

var (
	evalSuitePath  string
	evalOutputPath string
)

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Run an evaluation suite against your knowledge base",
	Long: `Runs a YAML test suite against the ask pipeline and scores each result.

Each test case specifies a question, expected keywords, and optional source hints.
Results are printed as a table and optionally saved as JSON.

Example:
  secondmem eval --suite examples/eval-suite.yaml --output report.json`,
	RunE: runEval,
}

func init() {
	evalCmd.Flags().StringVar(&evalSuitePath, "suite", "", "path to YAML eval suite file (required)")
	evalCmd.Flags().StringVar(&evalOutputPath, "output", "", "path to write JSON report (optional)")
	_ = evalCmd.MarkFlagRequired("suite")
	rootCmd.AddCommand(evalCmd)
}

func runEval(cmd *cobra.Command, args []string) error {
	suite, err := eval.LoadSuite(evalSuitePath)
	if err != nil {
		return fmt.Errorf("failed to load suite: %w", err)
	}

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

	fmt.Printf("Running %d test(s) from %q...\n", len(suite.Tests), evalSuitePath)

	report := eval.Run(suite, cfg, provider, embedder, g)
	eval.PrintTable(report)

	if evalOutputPath != "" {
		if err := eval.SaveJSON(report, evalOutputPath); err != nil {
			return err
		}
		fmt.Printf("Report saved to %s\n", evalOutputPath)
	}

	if report.Failed > 0 {
		return fmt.Errorf("%d test(s) failed", report.Failed)
	}
	return nil
}
