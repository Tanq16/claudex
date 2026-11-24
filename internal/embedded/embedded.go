package embedded

import "embed"

//go:embed statusline.sh
var StatuslineScript []byte

//go:embed agents/AGENTS.base.md
var AgentsBase []byte

//go:embed all:default-skills
var DefaultSkillsFS embed.FS
