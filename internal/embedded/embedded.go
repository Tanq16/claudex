package embedded

import _ "embed"

//go:embed statusline.sh
var StatuslineScript []byte
