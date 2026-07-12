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
	Short: "Install the embedded skill set into the current project",
	Run:   runApplySkills,
}

var applySkillsFlags struct {
	fullWipe      bool
	preserveLocal bool
	dir           string
}

func init() {
	applySkillsCmd.Flags().BoolVar(&applySkillsFlags.fullWipe, "full-wipe", false,
		"Before installing, clear the project's .claude skills and settings for a clean slate (refused in the home directory, which holds the live ~/.claude config)")
	applySkillsCmd.Flags().BoolVar(&applySkillsFlags.preserveLocal, "preserve-local", false,
		"Keep any skill that already exists in the project; install only the ones not already present")
	applySkillsCmd.Flags().StringVar(&applySkillsFlags.dir, "dir", "",
		"Install skills from this directory instead of claudex's embedded set")
	applySkillsCmd.MarkFlagsMutuallyExclusive("full-wipe", "preserve-local")
}

func runApplySkills(cmd *cobra.Command, args []string) {
	cwd, err := os.Getwd()
	if err != nil {
		u.PrintFatal("failed to resolve current directory", err)
	}

	var srcFS fs.FS = embedded.SkillsFS
	srcRoot := "skills"
	sourceLabel := "claudex's embedded set"
	if applySkillsFlags.dir != "" {
		dir := u.ExpandPath(applySkillsFlags.dir)
		info, statErr := os.Stat(dir)
		if statErr != nil || !info.IsDir() {
			u.PrintFatal(fmt.Sprintf("--dir %s is not a directory", applySkillsFlags.dir), statErr)
		}
		srcFS, srcRoot, sourceLabel = os.DirFS(dir), ".", u.AbbreviatePath(dir)
	}

	claudeDir := filepath.Join(cwd, ".claude")
	if applySkillsFlags.fullWipe {
		fullWipeProjectClaude(claudeDir)
	}
	targetRoot := filepath.Join(claudeDir, "skills")

	entries, err := fs.ReadDir(srcFS, srcRoot)
	if err != nil {
		u.PrintFatal("failed to read skills source", err)
	}

	installed, preserved, fileCount := 0, 0, 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		dest := filepath.Join(targetRoot, name)
		if applySkillsFlags.preserveLocal {
			if _, err := os.Stat(dest); err == nil {
				preserved++
				continue
			}
		}
		// Replace any same-named skill wholesale so renamed/removed files never linger.
		if err := os.RemoveAll(dest); err != nil {
			u.PrintFatal(fmt.Sprintf("failed to replace existing skill %q", name), err)
		}
		n, err := writeSkillTree(srcFS, srcRoot, name, dest)
		if err != nil {
			u.PrintFatal(fmt.Sprintf("failed to install skill %q", name), err)
		}
		installed++
		fileCount += n
	}

	msg := fmt.Sprintf("Installed %d skills (%d files) from %s into %s", installed, fileCount, sourceLabel, u.AbbreviatePath(targetRoot))
	if applySkillsFlags.preserveLocal {
		msg += fmt.Sprintf("; preserved %d existing", preserved)
	}
	u.PrintSuccess(msg)
}

func fullWipeProjectClaude(claudeDir string) {
	home, err := os.UserHomeDir()
	if err != nil {
		u.PrintFatal("--full-wipe: cannot resolve home to confirm the wipe is safe", err)
	}
	// Never wipe the live ~/.claude config, even when reached via a symlinked cwd or .claude.
	if samePath(claudeDir, filepath.Join(home, ".claude")) {
		u.PrintWarn("--full-wipe skipped: this .claude is your live ~/.claude config, not a project config", nil)
		return
	}

	targets := []string{
		filepath.Join(claudeDir, "skills"),
		filepath.Join(claudeDir, "settings.json"),
		filepath.Join(claudeDir, "settings.local.json"),
	}
	removed := 0
	for _, t := range targets {
		if _, err := os.Lstat(t); os.IsNotExist(err) {
			continue
		}
		if err := os.RemoveAll(t); err != nil {
			u.PrintFatal(fmt.Sprintf("failed to wipe %q", u.AbbreviatePath(t)), err)
		}
		removed++
	}
	u.PrintSuccess(fmt.Sprintf("--full-wipe: cleared %d existing item(s) under %s", removed, u.AbbreviatePath(claudeDir)))
}

func samePath(a, b string) bool {
	if ai, err := os.Stat(a); err == nil {
		if bi, err := os.Stat(b); err == nil {
			return os.SameFile(ai, bi)
		}
	}
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	if err1 == nil && err2 == nil {
		return filepath.Clean(ra) == filepath.Clean(rb)
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func writeSkillTree(srcFS fs.FS, srcRoot, name, dest string) (int, error) {
	root := name
	if srcRoot != "." {
		root = srcRoot + "/" + name
	}
	count := 0
	err := fs.WalkDir(srcFS, root, func(path string, d fs.DirEntry, walkErr error) error {
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
		data, err := fs.ReadFile(srcFS, path)
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
