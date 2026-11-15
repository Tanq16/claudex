package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/embedded"
	"github.com/tanq16/claudex/internal/workspace"
	u "github.com/tanq16/claudex/utils"
)

var foreignChoices = []string{"preserve", "clean"}

type foreignChoice string

func (f *foreignChoice) String() string { return string(*f) }

func (f *foreignChoice) Type() string { return strings.Join(foreignChoices, "|") }

func (f *foreignChoice) Set(v string) error {
	if !slices.Contains(foreignChoices, v) {
		return fmt.Errorf("must be one of %s", strings.Join(foreignChoices, ", "))
	}
	*f = foreignChoice(v)
	return nil
}

var applyFlags struct {
	foreign       foreignChoice
	restoreResume bool
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Write the base layout of AGENTS.md and the base skills into the current directory",
	Args:  cobra.NoArgs,
	Run:   runApply,
}

func init() {
	applyCmd.Flags().Var(&applyFlags.foreign, "foreign", "What to do with AI files claudex did not create: zip them into .agents/"+workspace.OldFilesZip+" first, or delete them")
	applyCmd.Flags().BoolVar(&applyFlags.restoreResume, "restore-resume", false, "Restore the resume file that clean-cwd stashed for this directory")
}

func runApply(cmd *cobra.Command, args []string) {
	root := currentDir()
	if applyFlags.restoreResume {
		if _, err := os.Stat(u.ResumeStash(root)); err != nil {
			u.PrintFatal("no stashed resume file for this directory", err)
		}
	}
	oldFiles := clearForeign(root)
	if conflicts := workspace.PreflightBase(root); len(conflicts) > 0 {
		refuse("cannot apply to "+u.AbbreviatePath(root), conflicts)
	}

	names, err := workspace.ApplyBase(root, embedded.AgentsBase, embedded.DefaultSkillsFS, "default-skills")
	if err != nil {
		u.PrintFatal("failed to apply the layout", err)
	}

	u.PrintSuccess("Applied to " + u.AbbreviatePath(root))
	u.PrintGeneric("  agents:  AGENTS.md, CLAUDE.md -> AGENTS.md")
	u.PrintGeneric("  skills:  .agents/skills (" + strings.Join(names, ", ") + "), .claude/skills -> ../.agents/skills")
	if oldFiles != "" {
		u.PrintGeneric("  old:     " + oldFiles)
	}
	path, ok, err := workspace.WriteGitExclude(root)
	switch {
	case err != nil:
		u.PrintWarn("could not write the git exclude entries", err)
	case ok:
		u.PrintGeneric("  ignored: " + u.AbbreviatePath(path))
	}
	if applyFlags.restoreResume {
		restoreResume(root)
	}
	u.PrintGeneric("  presets: claudex apply-preset")
}

func clearForeign(root string) string {
	found := workspace.Foreign(root)
	if len(found) == 0 {
		return ""
	}
	if applyFlags.foreign == "" {
		u.PrintError("cannot apply to "+u.AbbreviatePath(root)+": it holds AI files claudex did not create; nothing was written", nil)
		for _, f := range found {
			u.PrintIndentedError(f, nil)
		}
		u.PrintGeneric("  pass --foreign preserve to zip them into .agents/" + workspace.OldFilesZip + ", or --foreign clean to delete them")
		os.Exit(1)
	}
	oldFiles, err := workspace.ClearForeign(root, found, applyFlags.foreign == "preserve")
	if err != nil {
		msg := "failed to clear the existing AI files"
		if oldFiles != "" {
			msg += "; they are zipped in " + oldFiles
		}
		u.PrintFatal(msg, err)
	}
	return oldFiles
}

func restoreResume(root string) {
	err := workspace.RestoreResume(root, u.ResumeStash(root))
	switch {
	case errors.Is(err, workspace.ErrNoStash):
		u.PrintFatal("no stashed resume file for this directory", nil)
	case err != nil:
		u.PrintFatal("failed to restore the resume file", err)
	}
	u.PrintGeneric("  resume:  " + workspace.ResumeFile)
}

func presetsDir() string {
	dir := u.PresetsDir()
	if err := workspace.EnsurePresets(embedded.PresetsFS, "presets", dir); err != nil {
		u.PrintFatal("failed to lay down the built-in presets", err)
	}
	return dir
}

func remotePresetsDir() string {
	dir := u.RemotePresetsDir()
	if err := workspace.EnsureRemotePresets(dir); err != nil {
		u.PrintFatal("failed to create the remote presets directory", err)
	}
	return dir
}

func currentDir() string {
	cwd, err := os.Getwd()
	if err == nil {
		cwd, err = filepath.EvalSymlinks(cwd)
	}
	if err != nil {
		u.PrintFatal("failed to resolve current directory", err)
	}
	return cwd
}

func refuse(msg string, conflicts []workspace.Conflict) {
	u.PrintError(msg+"; nothing was written", nil)
	for _, c := range conflicts {
		u.PrintIndentedError(c.Path+": "+c.Why, nil)
	}
	u.PrintGeneric("  move each one aside, then run the command again")
	os.Exit(1)
}
