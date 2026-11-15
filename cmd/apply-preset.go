package cmd

import (
	"fmt"
	"os"
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
	Args:  cobra.ArbitraryArgs,
	Run:   runApplyPreset,
}

func runApplyPreset(cmd *cobra.Command, args []string) {
	root := currentDir()
	if !workspace.Applied(root) {
		u.PrintFatal("no claudex layout here; run claudex apply first", nil)
	}
	skills, agents := presetParts(applyPresetFlags.skills, applyPresetFlags.agents)

	local, remote := presetsDir(), remotePresetsDir()
	available := workspace.ListPresets(local, remote)
	if len(available) == 0 {
		u.PrintFatal("no presets found in "+u.AbbreviatePath(local)+" or "+u.AbbreviatePath(remote), nil)
	}

	if len(args) == 0 {
		u.PrintError("name at least one preset", nil)
		for _, p := range available {
			u.PrintGeneric("  " + p.Ref + "  " + u.Truncate(p.Description, 70))
		}
		os.Exit(1)
	}

	pruned, err := workspace.PruneDeadSkillLinks(root)
	if err != nil {
		u.PrintFatal("failed to remove dead skill links", err)
	}
	if len(pruned) > 0 {
		u.PrintInfo("Removed dead skill links: " + strings.Join(pruned, ", "))
	}

	presets := make([]*workspace.Preset, 0, len(args))
	var conflicts []workspace.Conflict
	if agents {
		conflicts = append(conflicts, workspace.PreflightAgentsFile(root)...)
	}
	for _, ref := range args {
		p, err := workspace.FindPreset(local, remote, ref)
		if err != nil {
			u.PrintFatal("preset not found: "+ref, err)
		}
		presets = append(presets, p)
		if skills {
			conflicts = append(conflicts, workspace.PreflightPresetSkills(root, p.Skills)...)
		}
	}
	if len(conflicts) > 0 {
		refuse("cannot apply to "+u.AbbreviatePath(root), conflicts)
	}

	for _, p := range presets {
		if skills {
			if err := workspace.LinkSkills(root, p.SkillsDir(), p.Skills); err != nil {
				u.PrintFatal("failed to link the skills of preset "+p.Ref, err)
			}
		}
		partial := ""
		if agents {
			if partial = p.Partial(); partial != "" {
				if err := workspace.UpsertSection(root, p.Ref, partial); err != nil {
					u.PrintFatal("failed to write the AGENTS.md section of preset "+p.Ref, err)
				}
			}
		}

		var servers []string
		if skills && agents {
			var err error
			if servers, err = workspace.MergePresetMCP(root, p); err != nil {
				u.PrintFatal("failed to merge the MCP servers of preset "+p.Ref, err)
			}
		}

		u.PrintSuccess("Applied preset: " + p.Ref)
		if skills {
			u.PrintGeneric(fmt.Sprintf("  skills: %d linked into .agents/skills", len(p.Skills)))
		}
		if agents {
			if partial != "" {
				u.PrintGeneric("  agents: section written to AGENTS.md")
			} else {
				u.PrintGeneric("  agents: this preset carries no section")
			}
		}
		if len(servers) > 0 {
			u.PrintGeneric("  mcp:    " + strings.Join(servers, ", ") + " merged into " + workspace.MCPFile)
		}
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
