package embedded

import "embed"

//go:embed statusline.sh
var StatuslineScript []byte

//go:embed all:skills
var SkillsFS embed.FS

//go:embed all:output-styles
var OutputStylesFS embed.FS
