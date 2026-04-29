package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	colWidth = 38
	passIcon = "PASS"
	failIcon = "FAIL"
)

// PrintTable writes a human-readable summary table to stdout.
func PrintTable(r *Report) {
	fmt.Printf("\nEval: %s  |  Provider: %s/%s  |  %s\n",
		r.SuiteName, r.Provider, r.Model, r.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println(strings.Repeat("─", 90))
	fmt.Printf("%-4s  %-*s  %-6s  %-12s  %-10s  %s\n",
		"#", colWidth, "Description", "Pass?", "KW Score", "Latency", "Keyword Hits")
	fmt.Println(strings.Repeat("─", 90))

	for i, res := range r.Results {
		desc := res.TestCase.Description
		if desc == "" {
			desc = truncate(res.TestCase.Question, colWidth)
		}
		status := passIcon
		if !res.Pass {
			status = failIcon
		}
		kwScore := fmt.Sprintf("%.0f%%", res.KeywordScore)
		latency := fmt.Sprintf("%dms", res.LatencyMS)
		kwHits := fmt.Sprintf("%d/%d", res.KeywordHits, res.KeywordTotal)

		if res.Error != "" {
			fmt.Printf("%-4d  %-*s  %-6s  %-12s  %-10s  ERROR: %s\n",
				i+1, colWidth, truncate(desc, colWidth), failIcon, "—", "—", truncate(res.Error, 30))
		} else {
			fmt.Printf("%-4d  %-*s  %-6s  %-12s  %-10s  %s\n",
				i+1, colWidth, truncate(desc, colWidth), status, kwScore, latency, kwHits)
		}
	}

	fmt.Println(strings.Repeat("─", 90))
	fmt.Printf("Total: %d  |  Passed: %d  |  Failed: %d  |  Avg Score: %.1f%%  |  Avg Latency: %.0fms\n\n",
		r.TotalTests, r.Passed, r.Failed, r.AvgScore, r.AvgLatencyMS)
}

// SaveJSON writes the full report as indented JSON to the given path.
func SaveJSON(r *Report, path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write report to %q: %w", path, err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
