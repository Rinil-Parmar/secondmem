package parsers

import (
	"fmt"
	"os"
	"strings"
)

// ParseTextFile reads a text or markdown file and returns its content.
func ParseTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return "", fmt.Errorf("file %s is empty", path)
	}

	return content, nil
}
