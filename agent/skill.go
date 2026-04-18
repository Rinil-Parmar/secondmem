package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Rinil-Parmar/secondmem/config"
)

var (
	cachedSkill string
	skillOnce   sync.Once
	skillErr    error
)

// LoadSkill reads the skill.md file and caches it for subsequent calls.
func LoadSkill(cfg *config.Config) (string, error) {
	basePath := filepath.Dir(cfg.KnowledgeBase.Path)

	skillOnce.Do(func() {
		skillPath := filepath.Join(basePath, "skill.md")
		data, err := os.ReadFile(skillPath)
		if err != nil {
			skillErr = fmt.Errorf("failed to read skill.md at %s: %w", skillPath, err)
			return
		}
		cachedSkill = string(data)
	})

	return cachedSkill, skillErr
}
