package cmd

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/workspace"
	u "github.com/tanq16/claudex/utils"
)

var applyPresetFlags struct {
	skills bool
	agents bool
}

var applyPresetCmd = &cobra.Command{
	Use:   "apply-preset [name...]",
	Short: "Add a preset's skills and its AGENTS.md section on top of the base layout",
	Args:  cobra.ArbitraryArgs,
	Run:   runApplyPreset,
}

func runApplyPreset(cmd *cobra.Command, args []string) {
	root := currentDir()
	if !workspace.Applied(root) {
		u.PrintFatal("no claudex layout here; run claudex apply first", nil)
	}
	skills, agents := presetParts(applyPresetFlags.skills, applyPresetFlags.agents)

	dir := presetsDir()
	available := workspace.ListPresets(dir)
	if len(available) == 0 {
		u.PrintFatal("no presets found in "+u.AbbreviatePath(dir), nil)
	}

	selected := args
	if len(selected) == 0 {
		if selected = choosePresets(available); len(selected) == 0 {
			return
		}
	}

	presets := make([]*workspace.Preset, 0, len(selected))
	var conflicts []workspace.Conflict
	if agents {
		conflicts = append(conflicts, workspace.PreflightAgentsFile(root)...)
	}
	for _, name := range selected {
		p, err := workspace.FindPreset(dir, name)
		if err != nil {
			fatal("preset not found: "+name, err)
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
				fatal("failed to link the skills of preset "+p.Name, err)
			}
		}
		partial := ""
		if agents {
			if partial = p.Partial(); partial != "" {
				if err := workspace.UpsertSection(root, p.Name, partial); err != nil {
					fatal("failed to write the AGENTS.md section of preset "+p.Name, err)
				}
			}
		}

		u.PrintSuccess("Applied preset: " + p.Name)
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
	}
}

// Neither flag applies the whole preset; either one narrows the run to that half.
func presetParts(skillsFlag, agentsFlag bool) (skills, agents bool) {
	if !skillsFlag && !agentsFlag {
		return true, true
	}
	return skillsFlag, agentsFlag
}

func choosePresets(available []workspace.Preset) []string {
	labels := make([]string, len(available))
	for i, p := range available {
		labels[i] = p.Name
		if p.Description != "" {
			labels[i] += " — " + u.Truncate(p.Description, 70)
		}
	}

	picked, err := u.PromptMultiSelect("Presets", labels)
	if err != nil {
		u.PrintFatal("TUI error", err)
	}

	// Deselecting leaves a false entry behind, and map order is random.
	var indices []int
	for i, on := range picked {
		if on {
			indices = append(indices, i)
		}
	}
	slices.Sort(indices)

	names := make([]string, len(indices))
	for i, idx := range indices {
		names[i] = available[idx].Name
	}
	return names
}

func init() {
	applyPresetCmd.Flags().BoolVarP(&applyPresetFlags.skills, "skills", "s", false, "Link only the preset's skills, leaving AGENTS.md alone")
	applyPresetCmd.Flags().BoolVarP(&applyPresetFlags.agents, "agents", "a", false, "Write only the preset's AGENTS.md section, linking no skills")
}
