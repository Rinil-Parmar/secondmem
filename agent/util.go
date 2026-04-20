package agent

import (
	"encoding/json"
	"strings"
)

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func parseJSON(s string, v any) error {
	return json.Unmarshal([]byte(s), v)
}
