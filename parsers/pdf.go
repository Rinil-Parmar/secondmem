package parsers

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
)

const (
	pdfTinyMaxBytes  = 500_000   // <500 KB  — extract all pages
	pdfMediumMaxBytes = 5_000_000 // <5 MB   — extract up to 50 pages
)

// ParsePDFFile extracts text from a PDF using a size-based strategy:
//   - tiny  (<500 KB):  all pages
//   - medium (<5 MB):   first 50 pages
//   - large (≥5 MB):    first 20 + last 5 pages (intro + conclusion)
func ParsePDFFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", path, err)
	}

	size := info.Size()

	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("open PDF %s: %w", path, err)
	}
	defer f.Close()

	totalPages := r.NumPage()
	if totalPages == 0 {
		return "", fmt.Errorf("PDF has no pages: %s", path)
	}

	var pageRange [][2]int // [start, end] pairs (1-based, inclusive)
	strategy := ""

	switch {
	case size < pdfTinyMaxBytes:
		strategy = "full"
		pageRange = [][2]int{{1, totalPages}}
	case size < pdfMediumMaxBytes:
		strategy = "medium"
		end := totalPages
		if end > 50 {
			end = 50
		}
		pageRange = [][2]int{{1, end}}
	default:
		strategy = "large"
		end := 20
		if end > totalPages {
			end = totalPages
		}
		pageRange = [][2]int{{1, end}}
		// Append last 5 pages if the doc is long enough
		if totalPages > 25 {
			tail := totalPages - 4
			pageRange = append(pageRange, [2]int{tail, totalPages})
		}
	}

	var buf bytes.Buffer
	extracted := 0

	for _, rng := range pageRange {
		for p := rng[0]; p <= rng[1]; p++ {
			page := r.Page(p)
			if page.V.IsNull() {
				continue
			}
			text, err := page.GetPlainText(nil)
			if err != nil {
				continue
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			buf.WriteString(text)
			buf.WriteString("\n\n")
			extracted++
		}
	}

	if buf.Len() == 0 {
		return "", fmt.Errorf("no text extracted from PDF (may be image-only): %s", path)
	}

	header := fmt.Sprintf("[PDF: %s | %d pages total | strategy: %s | extracted: %d pages]\n\n",
		info.Name(), totalPages, strategy, extracted)

	return header + buf.String(), nil
}
