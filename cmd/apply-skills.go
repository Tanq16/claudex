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

var applySkillsCmd = &cobra.Command{
	Use:   "apply-skills",
	Short: "Install the embedded skills (and the caveman output style) into the current project",
	Run:   runApplySkills,
}

var applySkillsFlags struct {
	fullWipe bool
}

func init() {
	applySkillsCmd.Flags().BoolVar(&applySkillsFlags.fullWipe, "full-wipe", false,
		"Before installing, clear the project's .claude skills, output-styles, and settings for a clean slate (refused in the home directory, which holds the live ~/.claude config)")
}

func runApplySkills(cmd *cobra.Command, args []string) {
	cwd, err := os.Getwd()
	if err != nil {
		u.PrintFatal("failed to resolve current directory", err)
	}

	claudeDir := filepath.Join(cwd, ".claude")

	if applySkillsFlags.fullWipe {
		fullWipeProjectClaude(cwd, claudeDir)
	}

	targetRoot := filepath.Join(claudeDir, "skills")

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

	// Output styles are a Claude Code feature; install them into .claude/output-styles alongside the
	// skills. Existing files there are left untouched — only same-named styles are overwritten.
	styleDest := filepath.Join(claudeDir, "output-styles")
	styleCount, err := writeOutputStyles(styleDest)
	if err != nil {
		u.PrintFatal("failed to install output styles", err)
	}
	u.PrintSuccess(fmt.Sprintf("Installed %d output style(s) into %s (enable with /config)", styleCount, u.AbbreviatePath(styleDest)))
}

// fullWipeProjectClaude clears the project's claudex-managed .claude artifacts — skills,
// output-styles, and the settings files — so the install that follows starts from a clean
// slate. It refuses to run in the home directory, where .claude is the default account's live
// Claude Code config dir rather than a project-local one.
func fullWipeProjectClaude(cwd, claudeDir string) {
	if home, err := os.UserHomeDir(); err == nil && filepath.Clean(cwd) == filepath.Clean(home) {
		u.PrintWarn("--full-wipe skipped: the home directory's .claude is your live Claude Code config, not a project config", nil)
		return
	}

	targets := []string{
		filepath.Join(claudeDir, "skills"),
		filepath.Join(claudeDir, "output-styles"),
		filepath.Join(claudeDir, "settings.json"),
		filepath.Join(claudeDir, "settings.local.json"),
	}
	removed := 0
	for _, t := range targets {
		if _, err := os.Stat(t); os.IsNotExist(err) {
			continue
		}
		if err := os.RemoveAll(t); err != nil {
			u.PrintFatal(fmt.Sprintf("failed to wipe %q", u.AbbreviatePath(t)), err)
		}
		removed++
	}
	u.PrintSuccess(fmt.Sprintf("--full-wipe: cleared %d existing item(s) under %s", removed, u.AbbreviatePath(claudeDir)))
}

func writeOutputStyles(dest string) (int, error) {
	entries, err := fs.ReadDir(embedded.OutputStylesFS, "output-styles")
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return 0, err
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := embedded.OutputStylesFS.ReadFile("output-styles/" + entry.Name())
		if err != nil {
			return count, err
		}
		if err := os.WriteFile(filepath.Join(dest, entry.Name()), data, 0o644); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
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
