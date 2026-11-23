package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/workspace"
	u "github.com/tanq16/claudex/utils"
)

var applyPresetFlags struct {
	skills bool
	agents bool
}

var applyPresetCmd = &cobra.Command{
	Use:   "apply-preset <name>...",
	Short: "Add a preset's skills, its AGENTS.md section, and its MCP servers on top of the base layout",
	Args:  cobra.MinimumNArgs(1),
	Run:   runApplyPreset,
}

func runApplyPreset(cmd *cobra.Command, args []string) {
	root := currentDir()
	if !workspace.Applied(root) {
		u.PrintFatal("no claudex layout here; run claudex apply first", nil)
	}
	skills, agents := presetParts(applyPresetFlags.skills, applyPresetFlags.agents)

	res, err := workspace.AddPresets(workspace.AddConfig{
		Root:          root,
		LocalPresets:  presetsDir(),
		RemotePresets: remotePresetsDir(),
		Refs:          args,
		Skills:        skills,
		Agents:        agents,
	})
	if len(res.Pruned) > 0 {
		u.PrintInfo("Removed dead skill links: " + strings.Join(res.Pruned, ", "))
	}
	if c, ok := errors.AsType[*workspace.ConflictError](err); ok {
		refuse("cannot apply to "+u.AbbreviatePath(root), c.Conflicts)
	}
	for _, p := range res.Presets {
		u.PrintSuccess("Applied preset: " + p.Ref)
		if skills {
			u.PrintGeneric(fmt.Sprintf("  skills: %d linked into .agents/skills", p.Skills))
		}
		if agents {
			if p.Section {
				u.PrintGeneric("  agents: section written to AGENTS.md")
			} else {
				u.PrintGeneric("  agents: this preset carries no section")
			}
		}
		if len(p.Servers) > 0 {
			u.PrintGeneric("  mcp:    " + strings.Join(p.Servers, ", ") + " merged into " + workspace.MCPFile)
		}
	}
	if p, ok := errors.AsType[*workspace.PresetError](err); ok {
		if errors.Is(p.Err, fs.ErrNotExist) {
			u.PrintFatal("preset not found: "+p.Ref, err)
		}
		u.PrintFatal("failed to apply preset "+p.Ref, err)
	}
	if err != nil {
		u.PrintFatal("failed to apply the presets", err)
	}
}

func presetParts(skillsFlag, agentsFlag bool) (skills, agents bool) {
	if !skillsFlag && !agentsFlag {
		return true, true
	}
	return skillsFlag, agentsFlag
}

func init() {
	applyPresetCmd.Flags().BoolVar(&applyPresetFlags.skills, "skills", false, "Link only the preset's skills, leaving AGENTS.md alone")
	applyPresetCmd.Flags().BoolVar(&applyPresetFlags.agents, "agents", false, "Write only the preset's AGENTS.md section, linking no skills")
}
