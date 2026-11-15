package workspace

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/tanq16/claudex/internal/fsutil"
)

var (
	ResumeFile = filepath.Join(AgentsDir, "session-resume", "resume.md")

	ErrNoStash      = errors.New("no stashed resume file")
	ErrResumeExists = errors.New(ResumeFile + " already exists; move it aside first")
)

func StashResume(root, stash string) (bool, error) {
	data, err := os.ReadFile(filepath.Join(root, ResumeFile))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(stash), configModes.dir); err != nil {
		return false, err
	}
	return true, fsutil.WriteFileAtomic(stash, data, configModes.file)
}

func RestoreResume(root, stash string) error {
	data, err := os.ReadFile(stash)
	if errors.Is(err, os.ErrNotExist) {
		return ErrNoStash
	}
	if err != nil {
		return err
	}
	dest := filepath.Join(root, ResumeFile)
	if _, err := os.Lstat(dest); err == nil {
		return ErrResumeExists
	}
	if err := writeFileAtomic(dest, data, projectModes.file); err != nil {
		return err
	}
	return os.Remove(stash)
}
