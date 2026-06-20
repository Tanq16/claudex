package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/embedded"
	u "github.com/tanq16/claudex/utils"
)

var applySkillsFlags struct {
	agentsio bool
}

var applySkillsCmd = &cobra.Command{
	Use:   "apply-skills",
	Short: "Install the embedded skills into the current project (replaces same-named skills, leaves others untouched)",
	Run:   runApplySkills,
}

func runApplySkills(cmd *cobra.Command, args []string) {
	cwd, err := os.Getwd()
	if err != nil {
		u.PrintFatal("failed to resolve current directory", err)
	}

	targetRoot := filepath.Join(cwd, ".claude", "skills")
	if applySkillsFlags.agentsio {
		targetRoot = filepath.Join(cwd, ".agents", "skills")
	}

	entries, err := fs.ReadDir(embedded.SkillsFS, "skills")
	if err != nil {
		u.PrintFatal("failed to read embedded skills", err)
	}

	skillCount, fileCount := 0, 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		dest := filepath.Join(targetRoot, name)
		// Replace any same-named skill wholesale so renamed/removed files never linger.
		if err := os.RemoveAll(dest); err != nil {
			u.PrintFatal(fmt.Sprintf("failed to replace existing skill %q", name), err)
		}
		n, err := writeSkillTree(name, dest)
		if err != nil {
			u.PrintFatal(fmt.Sprintf("failed to install skill %q", name), err)
		}
		skillCount++
		fileCount += n
	}

	u.PrintSuccess(fmt.Sprintf("Installed %d skills (%d files) into %s", skillCount, fileCount, u.AbbreviatePath(targetRoot)))
}

func writeSkillTree(name, dest string) (int, error) {
	root := "skills/" + name
	count := 0
	err := fs.WalkDir(embedded.SkillsFS, root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		out := filepath.Join(dest, strings.TrimPrefix(path, root+"/"))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		data, err := embedded.SkillsFS.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(out, data, 0o644); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}

func init() {
	applySkillsCmd.Flags().BoolVar(&applySkillsFlags.agentsio, "agentsio", false, "Install into .agents/skills instead of .claude/skills")
}
