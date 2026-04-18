package main

import (
	"embed"

	"github.com/Rinil-Parmar/secondmem/cmd"
)

//go:embed templates/skill.md
var skillTemplate embed.FS

func main() {
	cmd.SkillTemplate = skillTemplate
	cmd.Execute()
}
