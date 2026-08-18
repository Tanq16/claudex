package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/embedded"
	"github.com/tanq16/claudex/internal/workspace"
	u "github.com/tanq16/claudex/utils"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Write the base layout — AGENTS.md and the base skills — into the current directory",
	Args:  cobra.NoArgs,
	Run:   runApply,
}

func runApply(cmd *cobra.Command, args []string) {
	root := currentDir()
	if conflicts := workspace.PreflightBase(root); len(conflicts) > 0 {
		refuse("cannot apply to "+u.AbbreviatePath(root), conflicts)
	}

	names, err := workspace.ApplyBase(root, embedded.AgentsBase, embedded.DefaultSkillsFS, "default-skills")
	if err != nil {
		fatal("failed to apply the layout", err)
	}

	u.PrintSuccess("Applied to " + u.AbbreviatePath(root))
	u.PrintGeneric("  agents:  AGENTS.md, CLAUDE.md -> AGENTS.md")
	u.PrintGeneric("  skills:  .agents/skills (" + strings.Join(names, ", ") + "), .claude/skills -> ../.agents/skills")
	if path, ok := workspace.ExcludeFile(root); ok {
		u.PrintGeneric("  ignored: " + u.AbbreviatePath(path))
	}
	u.PrintGeneric("  presets: claudex apply-preset")
}

func presetsDir() string {
	dir := u.PresetsDir()
	if err := workspace.EnsurePresets(embedded.PresetsFS, "presets", dir); err != nil {
		fatal("failed to lay down the built-in presets", err)
	}
	return dir
}

func currentDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		u.PrintFatal("failed to resolve current directory", err)
	}
	return cwd
}

// Every conflict is reported at once, because clearing them one refusal at a time is one run per path.
func refuse(msg string, conflicts []workspace.Conflict) {
	u.PrintError(msg+"; nothing was written", nil)
	for _, c := range conflicts {
		u.PrintIndentedError(c.Path+": "+c.Why, nil)
	}
	u.PrintGeneric("  move each one aside, then run the command again")
	os.Exit(1)
}

// The printers only surface the error argument under --debug, and these are the failures the user has to act on.
func fatal(msg string, err error) {
	if err != nil {
		msg += ": " + err.Error()
	}
	u.PrintFatal(msg, err)
}
