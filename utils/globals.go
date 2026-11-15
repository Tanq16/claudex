package utils

import (
	"os"

	"github.com/charmbracelet/x/term"
)

var GlobalDebugFlag bool

var StdoutIsTerminal = term.IsTerminal(os.Stdout.Fd())
