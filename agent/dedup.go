package agent

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rinil-Parmar/secondmem/providers"
)

// DuplicateResult describes the outcome of a dedup check.
type DuplicateResult struct {
	IsDuplicate    bool
	ExistingFile   string // relative path to existing file
	Reason         string // "exact" or "semantic"
	SimilarityPct  int    // for semantic matches
}

// CheckExactDuplicate computes SHA256 of content and checks if any existing
// file has the same hash stored in its metadata footer.
func CheckExactDuplicate(knowledgePath, content string) (*DuplicateResult, error) {
	hash := sha256Hash(content)

	var found string
	err := filepath.Walk(knowledgePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if strings.Contains(string(data), "sha256:"+hash) {
			rel, _ := filepath.Rel(knowledgePath, path)
			found = rel
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if found != "" {
		return &DuplicateResult{
			IsDuplicate:  true,
			ExistingFile: found,
			Reason:       "exact",
		}, nil
	}
	return &DuplicateResult{IsDuplicate: false}, nil
}

// CheckSemanticDuplicate asks the LLM if content overlaps >70% with candidates.
// candidates are the top FTS search results for the content.
func CheckSemanticDuplicate(provider providers.LLMProvider, content string, candidates []string) (*DuplicateResult, error) {
	if len(candidates) == 0 {
		return &DuplicateResult{IsDuplicate: false}, nil
	}

	candidateBlock := strings.Join(candidates, "\n\n---\n\n")
	prompt := fmt.Sprintf(`Compare the NEW CONTENT against each EXISTING ENTRY below.
Return a JSON object: {"is_duplicate": bool, "similarity_pct": int, "most_similar_index": int}
- is_duplicate: true if similarity > 70%%
- similarity_pct: 0-100 estimated overlap
- most_similar_index: 0-based index of most similar entry (-1 if none)

NEW CONTENT:
%s

EXISTING ENTRIES:
%s

Return ONLY valid JSON, no explanation.`, content, candidateBlock)

	response, err := provider.Complete("You are a content deduplication checker.", prompt)
	if err != nil {
		return nil, fmt.Errorf("semantic dedup check failed: %w", err)
	}

	response = cleanJSON(response)

	var result struct {
		IsDuplicate        bool `json:"is_duplicate"`
		SimilarityPct      int  `json:"similarity_pct"`
		MostSimilarIndex   int  `json:"most_similar_index"`
	}
	if err := parseJSON(response, &result); err != nil {
		// If parsing fails, assume not a duplicate to avoid blocking ingest
		return &DuplicateResult{IsDuplicate: false}, nil
	}

	if result.IsDuplicate {
		return &DuplicateResult{
			IsDuplicate:   true,
			Reason:        "semantic",
			SimilarityPct: result.SimilarityPct,
		}, nil
	}
	return &DuplicateResult{IsDuplicate: false, SimilarityPct: result.SimilarityPct}, nil
}

// SHA256HashForFile computes the hash to embed in knowledge file metadata.
func SHA256HashForFile(content string) string {
	return "sha256:" + sha256Hash(content)
}

func sha256Hash(content string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return fmt.Sprintf("%x", sum)
}
