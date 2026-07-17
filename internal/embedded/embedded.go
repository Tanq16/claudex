package embedded

import "embed"

//go:embed statusline.sh
var StatuslineScript []byte

//go:embed all:default-skills
var DefaultSkillsFS embed.FS

//go:embed all:output-styles
var OutputStylesFS embed.FS
