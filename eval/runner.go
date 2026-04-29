package eval

import (
	"strings"
	"time"

	"github.com/Rinil-Parmar/secondmem/agent"
	"github.com/Rinil-Parmar/secondmem/config"
	"github.com/Rinil-Parmar/secondmem/graph"
	"github.com/Rinil-Parmar/secondmem/providers"
)

// Result holds the outcome of a single test case.
type Result struct {
	TestCase       TestCase
	Answer         string
	LatencyMS      int64
	KeywordHits    int
	KeywordTotal   int
	KeywordScore   float64 // 0–100
	SourcesMatched int
	SourcesTotal   int
	Pass           bool
	Error          string
}

// Report is the full output of an eval run.
type Report struct {
	SuiteName    string    `json:"suite_name"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	Results      []Result  `json:"results"`
	TotalTests   int       `json:"total_tests"`
	Passed       int       `json:"passed"`
	Failed       int       `json:"failed"`
	AvgLatencyMS float64   `json:"avg_latency_ms"`
	AvgScore     float64   `json:"avg_keyword_score"`
	Timestamp    time.Time `json:"timestamp"`
}

// Run executes every test case in the suite and returns a Report.
func Run(suite *TestSuite, cfg *config.Config, provider providers.LLMProvider, embedder providers.Embedder, g *graph.Graph) *Report {
	report := &Report{
		SuiteName: suite.Name,
		Provider:  cfg.Model.Provider,
		Model:     modelName(cfg),
		Timestamp: time.Now(),
	}

	for _, tc := range suite.Tests {
		result := runOne(tc, cfg, provider, embedder, g)
		report.Results = append(report.Results, result)
		if result.Pass {
			report.Passed++
		} else {
			report.Failed++
		}
	}

	report.TotalTests = len(suite.Tests)

	var totalLatency int64
	var totalScore float64
	for _, r := range report.Results {
		totalLatency += r.LatencyMS
		totalScore += r.KeywordScore
	}
	if report.TotalTests > 0 {
		report.AvgLatencyMS = float64(totalLatency) / float64(report.TotalTests)
		report.AvgScore = totalScore / float64(report.TotalTests)
	}

	return report
}

func runOne(tc TestCase, cfg *config.Config, provider providers.LLMProvider, embedder providers.Embedder, g *graph.Graph) Result {
	result := Result{TestCase: tc, KeywordTotal: len(tc.ExpectedKeywords), SourcesTotal: len(tc.SourceHints)}

	start := time.Now()
	answer, err := agent.Ask(cfg, provider, embedder, g, tc.Question, len(tc.SourceHints) > 0)
	result.LatencyMS = time.Since(start).Milliseconds()

	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Answer = answer

	lowerAnswer := strings.ToLower(answer)

	for _, kw := range tc.ExpectedKeywords {
		if strings.Contains(lowerAnswer, strings.ToLower(kw)) {
			result.KeywordHits++
		}
	}
	if result.KeywordTotal > 0 {
		result.KeywordScore = float64(result.KeywordHits) / float64(result.KeywordTotal) * 100
	} else {
		result.KeywordScore = 100
	}

	for _, hint := range tc.SourceHints {
		if strings.Contains(lowerAnswer, strings.ToLower(hint)) {
			result.SourcesMatched++
		}
	}

	minHits := tc.MinKeywords
	if minHits == 0 {
		minHits = result.KeywordTotal
	}
	result.Pass = result.KeywordHits >= minHits && result.Error == ""

	return result
}

func modelName(cfg *config.Config) string {
	switch cfg.Model.Provider {
	case "ollama":
		return cfg.Ollama.Model
	case "openai":
		return cfg.OpenAI.Model
	case "copilot":
		return cfg.Copilot.Model
	default:
		return "unknown"
	}
}
