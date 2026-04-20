package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rinil-Parmar/secondmem/agent"
	"github.com/Rinil-Parmar/secondmem/config"
	"github.com/Rinil-Parmar/secondmem/graph"
	"github.com/spf13/cobra"
)

var rebalanceDryRun bool

var rebalanceCmd = &cobra.Command{
	Use:   "rebalance",
	Short: "Run maintenance on the knowledge base",
	Long:  "7-step KB maintenance: split oversized, fix orphans, validate hierarchy, check cross-refs, list merge candidates, sync graph, rebuild root ToC.",
	RunE:  runRebalance,
}

func init() {
	rebalanceCmd.Flags().BoolVar(&rebalanceDryRun, "dry-run", false, "report issues only, make no changes")
	rootCmd.AddCommand(rebalanceCmd)
}

func runRebalance(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var g *graph.Graph
	if cfg.Graph.Enabled {
		g, err = graph.Open(cfg.Graph.DBPath)
		if err != nil {
			fmt.Printf("Warning: graph unavailable: %v\n", err)
		} else {
			defer g.Close()
		}
	}

	var provider = func() interface{ Complete(string, string) (string, error) } { return nil }
	_ = provider // provider loaded lazily below when needed

	fmt.Println("Running rebalance...")
	if rebalanceDryRun {
		fmt.Println("(dry-run mode — no changes will be made)")
	}
	fmt.Println()

	totalIssues := 0

	// ── Step 1: Split oversized files ─────────────────────────────────────
	fmt.Println("Step 1: Checking file sizes...")
	var oversizedPaths []string
	filepath.Walk(cfg.KnowledgeBase.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") || info.Name() == "hierarchy.md" {
			return nil
		}
		data, _ := os.ReadFile(path)
		lines := strings.Count(string(data), "\n") + 1
		if lines > cfg.KnowledgeBase.MaxFileLines {
			relPath, _ := filepath.Rel(cfg.KnowledgeBase.Path, path)
			fmt.Printf("  OVERSIZED: %s (%d lines, max %d)\n", relPath, lines, cfg.KnowledgeBase.MaxFileLines)
			oversizedPaths = append(oversizedPaths, path)
			totalIssues++
		}
		return nil
	})
	if len(oversizedPaths) == 0 {
		fmt.Println("  All files within size limits")
	} else if !rebalanceDryRun {
		p, err := newProvider(cfg)
		if err != nil {
			fmt.Printf("  Warning: no provider for split: %v\n", err)
		} else {
			for _, path := range oversizedPaths {
				created, err := agent.SplitOversizedFile(cfg, p, g, path)
				if err != nil {
					fmt.Printf("  SPLIT FAILED: %s: %v\n", path, err)
				} else {
					fmt.Printf("  SPLIT: %s → %d files\n", path, len(created))
				}
			}
		}
	}

	// ── Step 2: Detect orphaned files (not in hierarchy.md) ───────────────
	fmt.Println("\nStep 2: Checking for orphaned files...")
	orphanCount := 0
	dirEntries, _ := os.ReadDir(cfg.KnowledgeBase.Path)
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			continue
		}
		dirPath := filepath.Join(cfg.KnowledgeBase.Path, entry.Name())
		hierarchyData, err := os.ReadFile(filepath.Join(dirPath, "hierarchy.md"))
		if err != nil {
			continue
		}
		hContent := string(hierarchyData)
		subEntries, _ := os.ReadDir(dirPath)
		for _, sub := range subEntries {
			if sub.IsDir() || sub.Name() == "hierarchy.md" || !strings.HasSuffix(sub.Name(), ".md") {
				continue
			}
			if !strings.Contains(hContent, sub.Name()) {
				fmt.Printf("  ORPHAN: %s/%s\n", entry.Name(), sub.Name())
				orphanCount++
				totalIssues++
				if !rebalanceDryRun {
					agent.UpdateHierarchy(cfg.KnowledgeBase.Path, entry.Name())
					fmt.Printf("  FIXED: regenerated %s/hierarchy.md\n", entry.Name())
					break // hierarchy regenerated, re-check not needed in same pass
				}
			}
		}
	}
	if orphanCount == 0 {
		fmt.Println("  No orphaned files")
	}

	// ── Step 3: Validate hierarchy.md dead links ──────────────────────────
	fmt.Println("\nStep 3: Validating hierarchy links...")
	deadLinks := 0
	filepath.Walk(cfg.KnowledgeBase.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.Name() != "hierarchy.md" {
			return nil
		}
		data, _ := os.ReadFile(path)
		dir := filepath.Dir(path)
		for _, line := range strings.Split(string(data), "\n") {
			if !strings.Contains(line, "](") {
				continue
			}
			start := strings.Index(line, "](") + 2
			end := strings.Index(line[start:], ")")
			if end < 0 {
				continue
			}
			ref := line[start : start+end]
			target := filepath.Join(dir, ref)
			if _, err := os.Stat(target); os.IsNotExist(err) {
				relH, _ := filepath.Rel(cfg.KnowledgeBase.Path, path)
				fmt.Printf("  DEAD LINK: %s → %s\n", relH, ref)
				deadLinks++
				totalIssues++
			}
		}
		return nil
	})
	if deadLinks == 0 {
		fmt.Println("  All hierarchy links valid")
	} else if !rebalanceDryRun {
		// Regenerate all hierarchies to remove dead links
		for _, entry := range dirEntries {
			if entry.IsDir() {
				agent.UpdateHierarchy(cfg.KnowledgeBase.Path, entry.Name())
			}
		}
		fmt.Println("  FIXED: regenerated all hierarchy files")
	}

	// ── Step 4: Check cross-reference integrity ───────────────────────────
	fmt.Println("\nStep 4: Checking cross-reference integrity...")
	brokenRefs := 0
	filepath.Walk(cfg.KnowledgeBase.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") || info.Name() == "hierarchy.md" {
			return nil
		}
		data, _ := os.ReadFile(path)
		content := string(data)
		inRelated := false
		for _, line := range strings.Split(content, "\n") {
			if strings.TrimSpace(line) == "## Related" {
				inRelated = true
				continue
			}
			if inRelated && strings.HasPrefix(line, "## ") {
				inRelated = false
			}
			if !inRelated || !strings.Contains(line, "](") {
				continue
			}
			start := strings.Index(line, "](") + 2
			end := strings.Index(line[start:], ")")
			if end < 0 {
				continue
			}
			ref := line[start : start+end]
			target := filepath.Join(cfg.KnowledgeBase.Path, ref)
			if _, err := os.Stat(target); os.IsNotExist(err) {
				relPath, _ := filepath.Rel(cfg.KnowledgeBase.Path, path)
				fmt.Printf("  BROKEN XREF: %s → %s\n", relPath, ref)
				brokenRefs++
				totalIssues++
			}
		}
		return nil
	})
	if brokenRefs == 0 {
		fmt.Println("  All cross-references valid")
	}

	// ── Step 5: Identify merge candidates (>70% keyword overlap) ──────────
	fmt.Println("\nStep 5: Scanning for merge candidates...")
	if g != nil {
		nodes, err := g.AllNodes()
		if err == nil {
			mergePairs := findMergeCandidates(nodes)
			if len(mergePairs) == 0 {
				fmt.Println("  No merge candidates found")
			}
			for _, pair := range mergePairs {
				fmt.Printf("  MERGE CANDIDATE: %s ↔ %s\n", pair[0], pair[1])
				totalIssues++
				// merge.go provides MergeFiles for manual or automated invocation
			}
		}
	} else {
		fmt.Println("  Skipped (graph unavailable)")
	}

	// ── Step 6: Sync graph nodes with filesystem ──────────────────────────
	fmt.Println("\nStep 6: Syncing graph with filesystem...")
	if g != nil && !rebalanceDryRun {
		nodes, err := g.AllNodes()
		if err == nil {
			removed := 0
			for _, n := range nodes {
				absPath := filepath.Join(cfg.KnowledgeBase.Path, n.FilePath)
				if _, err := os.Stat(absPath); os.IsNotExist(err) {
					g.DeleteNode(n.FilePath)
					fmt.Printf("  REMOVED STALE: %s\n", n.FilePath)
					removed++
				}
			}
			if removed == 0 {
				fmt.Println("  Graph in sync with filesystem")
			}
		}
	} else if rebalanceDryRun {
		fmt.Println("  Skipped (dry-run)")
	} else {
		fmt.Println("  Skipped (graph unavailable)")
	}

	// ── Step 7: Rebuild root hierarchy ────────────────────────────────────
	fmt.Println("\nStep 7: Syncing root hierarchy...")
	if !rebalanceDryRun {
		if err := agent.UpdateRootHierarchy(cfg.KnowledgeBase.Path); err != nil {
			fmt.Printf("  Warning: %v\n", err)
		} else {
			fmt.Println("  Root hierarchy.md updated")
		}
	} else {
		fmt.Println("  Skipped (dry-run)")
	}

	fmt.Printf("\nRebalance complete. %d issue(s) found.\n", totalIssues)
	return nil
}

// findMergeCandidates returns pairs of nodes with >70% keyword overlap.
func findMergeCandidates(nodes []graph.Node) [][2]string {
	var pairs [][2]string
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			a := tokenSet(nodes[i].Keywords)
			b := tokenSet(nodes[j].Keywords)
			if keywordOverlap(a, b) >= 0.7 {
				pairs = append(pairs, [2]string{nodes[i].FilePath, nodes[j].FilePath})
			}
		}
	}
	return pairs
}

func tokenSet(keywords string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, kw := range strings.Split(keywords, ",") {
		t := strings.ToLower(strings.TrimSpace(kw))
		if t != "" {
			set[t] = struct{}{}
		}
	}
	return set
}

func keywordOverlap(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	shared := 0
	for k := range a {
		if _, ok := b[k]; ok {
			shared++
		}
	}
	smaller := len(a)
	if len(b) < smaller {
		smaller = len(b)
	}
	return float64(shared) / float64(smaller)
}
