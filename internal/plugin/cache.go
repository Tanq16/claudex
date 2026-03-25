package plugin

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RemoveCacheDirectory(configDir, marketplace, pluginName string) error {
	dir := filepath.Join(configDir, "plugins", "cache", marketplace, pluginName)
	return os.RemoveAll(dir)
}

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

func GetGitCommitSha(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func GitPull(marketplaceDir string) error {
	cmd := exec.Command("git", "-C", marketplaceDir, "pull", "--ff-only")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func CloneGitHubRepo(repo string) (tmpDir string, err error) {
	tmpDir, err = os.MkdirTemp("", "claudex-plugin-*")
	if err != nil {
		return "", err
	}
	url := "https://github.com/" + repo + ".git"
	cmd := exec.Command("git", "clone", "--depth", "1", url, tmpDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("git clone %s: %s", repo, strings.TrimSpace(string(out)))
	}
	return tmpDir, nil
}
