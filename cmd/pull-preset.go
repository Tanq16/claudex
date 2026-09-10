package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/workspace"
	u "github.com/tanq16/claudex/utils"
)

var pullPresetFlags struct {
	path string
}

var pullPresetCmd = &cobra.Command{
	Use:   "pull-preset <repo>",
	Short: "Clone a git repository and install the presets it holds under ~/.config/claudex/remote-presets",
	Args:  cobra.ExactArgs(1),
	Run:   runPullPreset,
}

func runPullPreset(cmd *cobra.Command, args []string) {
	repo := args[0]
	dir := remotePresetsDir()

	u.PrintRunning("cloning " + repo)
	res, err := workspace.PullPresets(workspace.PullConfig{Repo: repo, Path: pullPresetFlags.path, Dir: dir})
	u.ClearLines(1)
	if err != nil {
		u.PrintFatal("failed to pull presets from "+repo, err)
	}

	u.PrintSuccess("Pulled " + res.Slug + " into " + u.AbbreviatePath(res.Dir))
	for _, name := range res.Presets {
		u.PrintGeneric("  preset:  " + res.Slug + "/" + name)
	}
	u.PrintGeneric("  apply it: claudex apply-preset " + res.Slug + "/" + res.Presets[0])
}

func init() {
	pullPresetCmd.Flags().StringVar(&pullPresetFlags.path, "path", "", "Install only the preset at this path, resolved from the repository root")
}
