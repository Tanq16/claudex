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
	cfg := workspace.LayoutConfig{
		Root:       root,
		Base:       embedded.AgentsBase,
		BaseSkills: baseSkillsDir(),
		Foreign:    string(applyFlags.foreign),
	}
	if applyFlags.restoreResume {
		cfg.Stash = u.ResumeStash(root)
	}

	res, err := workspace.ApplyLayout(cfg)
	if f, ok := errors.AsType[*workspace.ForeignError](err); ok {
		u.PrintError("cannot apply to "+u.AbbreviatePath(root)+": it holds AI files claudex did not create; nothing was written", nil)
		for _, path := range f.Found {
			u.PrintIndentedError(path, nil)
		}
		u.PrintGeneric("  pass --foreign preserve to zip them into .agents/" + workspace.OldFilesZip + ", or --foreign clean to delete them")
		os.Exit(1)
	}
	if c, ok := errors.AsType[*workspace.ConflictError](err); ok {
		refuse("cannot apply to "+u.AbbreviatePath(root), c.Conflicts)
	}
	switch {
	case errors.Is(err, workspace.ErrNoStash):
		u.PrintFatal("no stashed resume file for this directory", err)
	case errors.Is(err, workspace.ErrResumeExists):
		u.PrintFatal("failed to restore the resume file", err)
	case err != nil:
		msg := "failed to apply the layout"
		if res.OldFiles != "" {
			msg += "; the old files are zipped in " + res.OldFiles
		}
		u.PrintFatal(msg, err)
	}

	u.PrintSuccess("Applied to " + u.AbbreviatePath(root))
	u.PrintGeneric("  agents:  AGENTS.md, CLAUDE.md -> AGENTS.md")
	u.PrintGeneric("  skills:  .agents/skills (" + strings.Join(res.Skills, ", ") + " -> " + u.AbbreviatePath(cfg.BaseSkills) + "), .claude/skills -> ../.agents/skills")
	if res.OldFiles != "" {
		u.PrintGeneric("  old:     " + res.OldFiles)
	}
	if res.Exclude != "" {
		u.PrintGeneric("  ignored: " + u.AbbreviatePath(res.Exclude))
	}
	if applyFlags.restoreResume {
		u.PrintGeneric("  resume:  " + workspace.ResumeFile)
	}
	u.PrintGeneric("  presets: claudex apply-preset")
}

func presetsDir() string {
	baseSkillsDir()
	return u.PresetsDir()
}

func baseSkillsDir() string {
	dir := u.BaseSkillsDir()
	if err := workspace.Extract(embedded.DefaultSkillsFS, "default-skills", dir); err != nil {
		u.PrintFatal("failed to lay down the base skills", err)
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
