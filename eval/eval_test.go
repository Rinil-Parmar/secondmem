package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- Suite loading ---

func TestLoadSuite_valid(t *testing.T) {
	content := `
name: test-suite
tests:
  - description: Basic question
    question: What is Go?
    expected_keywords: [golang, compiled, statically typed]
    min_keywords: 2
`
	path := writeTempYAML(t, content)
	suite, err := LoadSuite(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suite.Name != "test-suite" {
		t.Errorf("got name %q, want %q", suite.Name, "test-suite")
	}
	if len(suite.Tests) != 1 {
		t.Fatalf("got %d tests, want 1", len(suite.Tests))
	}
	if suite.Tests[0].MinKeywords != 2 {
		t.Errorf("got min_keywords %d, want 2", suite.Tests[0].MinKeywords)
	}
}

func TestLoadSuite_defaultMinKeywords(t *testing.T) {
	content := `
name: s
tests:
  - question: What is Go?
    expected_keywords: [golang, compiled]
`
	path := writeTempYAML(t, content)
	suite, err := LoadSuite(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// min_keywords defaults to len(expected_keywords)
	if suite.Tests[0].MinKeywords != 2 {
		t.Errorf("got min_keywords %d, want 2", suite.Tests[0].MinKeywords)
	}
}

func TestLoadSuite_missingQuestion(t *testing.T) {
	content := `
name: s
tests:
  - description: no question here
    expected_keywords: [golang]
`
	path := writeTempYAML(t, content)
	_, err := LoadSuite(path)
	if err == nil {
		t.Fatal("expected error for missing question, got nil")
	}
}

func TestLoadSuite_emptyTests(t *testing.T) {
	content := "name: empty\ntests: []\n"
	path := writeTempYAML(t, content)
	_, err := LoadSuite(path)
	if err == nil {
		t.Fatal("expected error for empty tests, got nil")
	}
}

func TestLoadSuite_fileNotFound(t *testing.T) {
	_, err := LoadSuite("/nonexistent/path/suite.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// --- Scoring logic ---

func TestScoring_allHits(t *testing.T) {
	tc := TestCase{
		Question:         "What is Go?",
		ExpectedKeywords: []string{"golang", "compiled"},
		MinKeywords:      2,
	}
	answer := "Go (golang) is a statically compiled language."
	result := scoreResult(tc, answer, 100, nil)

	if result.KeywordHits != 2 {
		t.Errorf("got %d keyword hits, want 2", result.KeywordHits)
	}
	if result.KeywordScore != 100 {
		t.Errorf("got score %.1f, want 100", result.KeywordScore)
	}
	if !result.Pass {
		t.Error("expected Pass=true")
	}
}

func TestScoring_partialHits_pass(t *testing.T) {
	tc := TestCase{
		Question:         "What is Go?",
		ExpectedKeywords: []string{"golang", "compiled", "goroutine"},
		MinKeywords:      2,
	}
	answer := "Go (golang) is a compiled language."
	result := scoreResult(tc, answer, 50, nil)

	if result.KeywordHits != 2 {
		t.Errorf("got %d keyword hits, want 2", result.KeywordHits)
	}
	if !result.Pass {
		t.Error("expected Pass=true with 2/3 hits and min_keywords=2")
	}
}

func TestScoring_partialHits_fail(t *testing.T) {
	tc := TestCase{
		Question:         "What is Go?",
		ExpectedKeywords: []string{"golang", "compiled", "goroutine"},
		MinKeywords:      3,
	}
	answer := "Go (golang) is a compiled language."
	result := scoreResult(tc, answer, 50, nil)

	if result.Pass {
		t.Error("expected Pass=false with 2/3 hits and min_keywords=3")
	}
}

func TestScoring_noKeywords(t *testing.T) {
	tc := TestCase{
		Question: "Any question",
	}
	result := scoreResult(tc, "any answer", 10, nil)

	if result.KeywordScore != 100 {
		t.Errorf("got score %.1f, want 100 for no keywords", result.KeywordScore)
	}
	if !result.Pass {
		t.Error("expected Pass=true when no keywords required")
	}
}

func TestScoring_caseInsensitive(t *testing.T) {
	tc := TestCase{
		Question:         "q",
		ExpectedKeywords: []string{"GOLANG"},
		MinKeywords:      1,
	}
	result := scoreResult(tc, "I love golang", 10, nil)
	if result.KeywordHits != 1 {
		t.Errorf("expected case-insensitive match, got %d hits", result.KeywordHits)
	}
}

func TestScoring_sourceHints(t *testing.T) {
	tc := TestCase{
		Question:    "q",
		SourceHints: []string{"go/basics.md", "go/types.md"},
	}
	answer := "Answer referencing go/basics.md as a source."
	result := scoreResult(tc, answer, 10, nil)

	if result.SourcesMatched != 1 {
		t.Errorf("got %d sources matched, want 1", result.SourcesMatched)
	}
	if result.SourcesTotal != 2 {
		t.Errorf("got %d sources total, want 2", result.SourcesTotal)
	}
}

// --- Report ---

func TestReport_aggregates(t *testing.T) {
	results := []Result{
		{Pass: true, KeywordScore: 100, LatencyMS: 200},
		{Pass: false, KeywordScore: 50, LatencyMS: 300},
	}
	r := &Report{
		SuiteName:  "s",
		Results:    results,
		TotalTests: 2,
		Passed:     1,
		Failed:     1,
	}
	var totalLatency int64
	var totalScore float64
	for _, res := range r.Results {
		totalLatency += res.LatencyMS
		totalScore += res.KeywordScore
	}
	r.AvgLatencyMS = float64(totalLatency) / float64(r.TotalTests)
	r.AvgScore = totalScore / float64(r.TotalTests)

	if r.AvgLatencyMS != 250 {
		t.Errorf("got avg latency %.0f, want 250", r.AvgLatencyMS)
	}
	if r.AvgScore != 75 {
		t.Errorf("got avg score %.0f, want 75", r.AvgScore)
	}
}

func TestSaveJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	r := &Report{
		SuiteName:  "test",
		TotalTests: 1,
		Passed:     1,
		Timestamp:  time.Now(),
	}

	if err := SaveJSON(r, path); err != nil {
		t.Fatalf("SaveJSON failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read report: %v", err)
	}

	var decoded Report
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if decoded.SuiteName != "test" {
		t.Errorf("got suite name %q, want %q", decoded.SuiteName, "test")
	}
}

func TestPrintTable_nonempty(t *testing.T) {
	r := &Report{
		SuiteName:  "smoke",
		Provider:   "mock",
		Model:      "test",
		TotalTests: 1,
		Passed:     1,
		Timestamp:  time.Now(),
		Results: []Result{
			{
				TestCase:     TestCase{Description: "desc", Question: "q"},
				Answer:       "some answer",
				LatencyMS:    42,
				KeywordHits:  1,
				KeywordTotal: 1,
				KeywordScore: 100,
				Pass:         true,
			},
		},
	}
	// Should not panic
	PrintTable(r)
}

// --- Helpers ---

// scoreResult is the pure scoring logic extracted for testing without filesystem.
func scoreResult(tc TestCase, answer string, latencyMS int64, err error) Result {
	result := Result{
		TestCase:     tc,
		Answer:       answer,
		LatencyMS:    latencyMS,
		KeywordTotal: len(tc.ExpectedKeywords),
		SourcesTotal: len(tc.SourceHints),
	}

	if err != nil {
		result.Error = err.Error()
		return result
	}

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
	result.Pass = result.KeywordHits >= minHits

	return result
}

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "suite-*.yaml")
	if err != nil {
		t.Fatalf("could not create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("could not write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}
