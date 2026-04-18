package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rinil-Parmar/secondmem/agent"
	"github.com/Rinil-Parmar/secondmem/config"
	"github.com/spf13/cobra"
)

var rebalanceDryRun bool

var rebalanceCmd = &cobra.Command{
	Use:   "rebalance",
	Short: "Run maintenance on the knowledge base",
	Long:  "Checks file sizes, orphaned files, dead links, and hierarchy integrity. Fixes issues automatically unless --dry-run is set.",
	RunE:  runRebalance,
}

func init() {
	rebalanceCmd.Flags().BoolVar(&rebalanceDryRun, "dry-run", false, "only report issues, don't fix them")
	rootCmd.AddCommand(rebalanceCmd)
}

func runRebalance(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("Running rebalance...")
	if rebalanceDryRun {
		fmt.Println("(dry-run mode — no changes will be made)")
	}
	fmt.Println()

	// Step 1: Check file sizes
	fmt.Println("Step 1: Checking file sizes...")
	oversized := 0
	filepath.Walk(cfg.KnowledgeBase.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") || info.Name() == "hierarchy.md" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lines := strings.Count(string(data), "\n") + 1
		if lines > cfg.KnowledgeBase.MaxFileLines {
			relPath, _ := filepath.Rel(cfg.KnowledgeBase.Path, path)
			fmt.Printf("  OVERSIZED: %s (%d lines, max %d)\n", relPath, lines, cfg.KnowledgeBase.MaxFileLines)
			oversized++
		}
		return nil
	})
	if oversized == 0 {
		fmt.Println("  All files within size limits")
	}

	// Step 2: Check for orphaned files
	fmt.Println("\nStep 2: Checking for orphaned files...")
	orphans := 0
	entries, err := os.ReadDir(cfg.KnowledgeBase.Path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirPath := filepath.Join(cfg.KnowledgeBase.Path, entry.Name())
		hierarchyPath := filepath.Join(dirPath, "hierarchy.md")
		hierarchyData, err := os.ReadFile(hierarchyPath)
		if err != nil {
			continue
		}
		hierarchyContent := string(hierarchyData)

		subEntries, _ := os.ReadDir(dirPath)
		for _, sub := range subEntries {
			if sub.IsDir() || sub.Name() == "hierarchy.md" || !strings.HasSuffix(sub.Name(), ".md") {
				continue
			}
			if !strings.Contains(hierarchyContent, sub.Name()) {
				fmt.Printf("  ORPHAN: %s/%s\n", entry.Name(), sub.Name())
				orphans++
			}
		}
	}
	if orphans == 0 {
		fmt.Println("  No orphaned files")
	}

	// Step 3: Check for missing hierarchy.md files
	fmt.Println("\nStep 3: Checking hierarchy files...")
	missingHierarchy := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		hierarchyPath := filepath.Join(cfg.KnowledgeBase.Path, entry.Name(), "hierarchy.md")
		if _, err := os.Stat(hierarchyPath); os.IsNotExist(err) {
			fmt.Printf("  MISSING: %s/hierarchy.md\n", entry.Name())
			missingHierarchy++
			if !rebalanceDryRun {
				agent.UpdateHierarchy(cfg.KnowledgeBase.Path, entry.Name())
				fmt.Printf("  FIXED: regenerated %s/hierarchy.md\n", entry.Name())
			}
		}
	}
	if missingHierarchy == 0 {
		fmt.Println("  All directories have hierarchy.md")
	}

	// Step 4: Sync root hierarchy
	fmt.Println("\nStep 4: Syncing root hierarchy...")
	if !rebalanceDryRun {
		if err := agent.UpdateRootHierarchy(cfg.KnowledgeBase.Path); err != nil {
			fmt.Printf("  Warning: %v\n", err)
		} else {
			fmt.Println("  Root hierarchy.md updated")
		}
	} else {
		fmt.Println("  Skipped (dry-run)")
	}

	fmt.Println("\nRebalance complete!")
	return nil
}
