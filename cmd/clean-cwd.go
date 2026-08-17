package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/workspace"
	u "github.com/tanq16/claudex/utils"
)

var cleanCwdCmd = &cobra.Command{
	Use:   "clean-cwd",
	Short: "Remove the claudex layout from the current directory",
	Args:  cobra.NoArgs,
	Run:   runCleanCwd,
}

func runCleanCwd(cmd *cobra.Command, args []string) {
	root := currentDir()
	if err := workspace.Clean(root); err != nil {
		fatal("failed to remove the layout", err)
	}
	u.PrintSuccess("Removed the claudex layout from " + u.AbbreviatePath(root))
	u.PrintGeneric("  removed: .agents/, CLAUDE.md, .claude/skills, and the claudex sections of AGENTS.md")
}
