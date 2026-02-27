package plugin

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RemoveCacheDirectory removes the entire cache directory for a plugin.
func RemoveCacheDirectory(configDir, marketplace, pluginName string) error {
	dir := filepath.Join(configDir, "plugins", "cache", marketplace, pluginName)
	return os.RemoveAll(dir)
}

// RemoveOrphanedVersions removes only version dirs that have an .orphaned_at marker.
func RemoveOrphanedVersions(configDir, marketplace, pluginName string) (int, error) {
	versions, err := ListCachedVersions(configDir, marketplace, pluginName)
	if err != nil {
		return 0, err
	}

	removed := 0
	for _, v := range versions {
		if v.Orphaned {
			if err := os.RemoveAll(v.Path); err != nil {
				return removed, fmt.Errorf("removing %s: %w", v.Path, err)
			}
			removed++
		}
	}
	return removed, nil
}

// RemoveAllOrphans scans all cached plugins and removes every orphaned version directory.
func RemoveAllOrphans(configDir string) (int, error) {
	cacheRoot := filepath.Join(configDir, "plugins", "cache")
	marketplaces, err := os.ReadDir(cacheRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	total := 0
	for _, mkt := range marketplaces {
		if !mkt.IsDir() {
			continue
		}
		plugins, err := os.ReadDir(filepath.Join(cacheRoot, mkt.Name()))
		if err != nil {
			continue
		}
		for _, pl := range plugins {
			if !pl.IsDir() {
				continue
			}
			n, err := RemoveOrphanedVersions(configDir, mkt.Name(), pl.Name())
			if err != nil {
				return total, err
			}
			total += n
		}
	}
	return total, nil
}

// copyDir recursively copies src to dst, skipping .git/ directories.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.Name() == ".git" {
			continue
		}

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// GetGitCommitSha returns the HEAD commit SHA of a git repo.
func GetGitCommitSha(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// GitPull runs git pull --ff-only on a marketplace directory.
func GitPull(marketplaceDir string) error {
	cmd := exec.Command("git", "-C", marketplaceDir, "pull", "--ff-only")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
