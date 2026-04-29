package eval

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// TestCase is a single eval question with expected outputs.
type TestCase struct {
	Description      string   `yaml:"description"`
	Question         string   `yaml:"question"`
	ExpectedKeywords []string `yaml:"expected_keywords"`
	SourceHints      []string `yaml:"source_hints"`
	// MinKeywords is the minimum number of expected_keywords that must appear.
	// Defaults to len(ExpectedKeywords) if 0.
	MinKeywords int `yaml:"min_keywords"`
}

// TestSuite is a named collection of test cases loaded from a YAML file.
type TestSuite struct {
	Name  string     `yaml:"name"`
	Tests []TestCase `yaml:"tests"`
}

// LoadSuite reads and parses a YAML test suite file.
func LoadSuite(path string) (*TestSuite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read suite file %q: %w", path, err)
	}

	var suite TestSuite
	if err := yaml.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("invalid YAML in %q: %w", path, err)
	}

	if len(suite.Tests) == 0 {
		return nil, fmt.Errorf("suite %q has no tests", path)
	}

	for i, tc := range suite.Tests {
		if tc.Question == "" {
			return nil, fmt.Errorf("test %d in suite %q is missing 'question'", i+1, suite.Name)
		}
		if suite.Tests[i].MinKeywords == 0 {
			suite.Tests[i].MinKeywords = len(tc.ExpectedKeywords)
		}
	}

	return &suite, nil
}
