package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/workspace"
	u "github.com/tanq16/claudex/utils"
)

var createPresetCmd = &cobra.Command{
	Use:   "create-preset <name>",
	Short: "Scaffold a new preset under ~/.config/claudex/presets",
	Args:  cobra.ExactArgs(1),
	Run:   runCreatePreset,
}

func runCreatePreset(cmd *cobra.Command, args []string) {
	name := args[0]

	dir, err := workspace.ScaffoldPreset(presetsDir(), name)
	if err != nil {
		fatal("failed to scaffold the preset", err)
	}

	u.PrintSuccess("Created preset: " + name)
	u.PrintGeneric("  manifest: " + u.AbbreviatePath(filepath.Join(dir, workspace.ManifestName)))
	u.PrintGeneric("  agents:   " + u.AbbreviatePath(filepath.Join(dir, workspace.PartialName)))
	u.PrintGeneric("  skills:   " + u.AbbreviatePath(filepath.Join(dir, workspace.SkillsDir)))
	u.PrintGeneric("  apply it: claudex apply-preset " + name)
}
