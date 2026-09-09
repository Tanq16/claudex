package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type PullConfig struct {
	Repo string
	Path string
	Dir  string
}

type PullResult struct {
	Slug    string
	Dir     string
	Presets []string
}

type presetSource struct {
	dir  string
	name string
}

func EnsureRemotePresets(dir string) error {
	return os.MkdirAll(dir, configModes.dir)
}

func PullPresets(cfg PullConfig) (PullResult, error) {
	cloneURL, owner, repo, err := parseRepo(cfg.Repo)
	if err != nil {
		return PullResult{}, err
	}
	slug := slugify(owner) + "-" + slugify(repo)
	if !validName(slug) {
		return PullResult{}, fmt.Errorf("%q does not reduce to a usable slug", cfg.Repo)
	}

	tmp, err := os.MkdirTemp("", "claudex-preset-")
	if err != nil {
		return PullResult{}, err
	}
	defer os.RemoveAll(tmp)

	clone := filepath.Join(tmp, slugify(repo))
	if err := gitClone(cloneURL, clone); err != nil {
		return PullResult{}, err
	}
	if err := os.RemoveAll(filepath.Join(clone, ".git")); err != nil {
		return PullResult{}, err
	}

	slugDir := filepath.Join(cfg.Dir, slug)
	var names []string
	if cfg.Path != "" {
		names, err = pullOne(clone, cfg.Path, slugDir)
	} else {
		names, err = pullAll(clone, slugDir)
	}
	if err != nil {
		return PullResult{}, err
	}
	return PullResult{Slug: slug, Dir: slugDir, Presets: names}, nil
}

func pullAll(clone, slugDir string) ([]string, error) {
	sources, err := findPresets(clone)
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("no preset found at the repository root or one level under it")
	}

	staging := slugDir + ".staging"
	if err := os.RemoveAll(staging); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(staging, configModes.dir); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(sources))
	for _, src := range sources {
		if err := copyTree(os.DirFS(src.dir), ".", filepath.Join(staging, src.name), configModes); err != nil {
			os.RemoveAll(staging)
			return nil, err
		}
		names = append(names, src.name)
	}
	if err := os.RemoveAll(slugDir); err != nil {
		os.RemoveAll(staging)
		return nil, err
	}
	if err := os.Rename(staging, slugDir); err != nil {
		os.RemoveAll(staging)
		return nil, err
	}
	return names, nil
}

func pullOne(clone, path, slugDir string) ([]string, error) {
	dir, err := resolveInRepo(clone, path)
	if err != nil {
		return nil, err
	}
	src, err := presetAt(dir)
	if err != nil {
		return nil, fmt.Errorf("%s is not a preset: %w", path, err)
	}
	if err := os.MkdirAll(slugDir, configModes.dir); err != nil {
		return nil, err
	}
	if err := installTree(os.DirFS(src.dir), ".", filepath.Join(slugDir, src.name), configModes); err != nil {
		return nil, err
	}
	return []string{src.name}, nil
}

func findPresets(clone string) ([]presetSource, error) {
	if src, err := presetAt(clone); err == nil {
		return []presetSource{src}, nil
	}
	var found []presetSource
	seen := make(map[string]string)
	for _, entry := range subdirs(clone) {
		if strings.HasPrefix(entry, ".") {
			continue
		}
		src, err := presetAt(filepath.Join(clone, entry))
		if err != nil {
			continue
		}
		if prev, ok := seen[src.name]; ok {
			return nil, fmt.Errorf("%s and %s both define a preset named %q", prev, entry, src.name)
		}
		seen[src.name] = entry
		found = append(found, src)
	}
	slices.SortFunc(found, func(a, b presetSource) int { return strings.Compare(a.name, b.name) })
	return found, nil
}

func presetAt(dir string) (presetSource, error) {
	p, err := loadPreset(dir)
	if err != nil {
		return presetSource{}, err
	}
	if !validName(p.Name) {
		return presetSource{}, fmt.Errorf("%q is not a valid preset name; use lowercase letters, digits, and single hyphens", p.Name)
	}
	return presetSource{dir: dir, name: p.Name}, nil
}

func resolveInRepo(clone, path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("--path resolves from the repository root, so it cannot be absolute")
	}
	dir := filepath.Join(clone, filepath.Clean("/"+path))
	info, err := os.Stat(dir)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", path)
	}
	return dir, nil
}

func gitClone(url, dest string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", "--quiet", url, dest)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			return fmt.Errorf("%s: %w", detail, err)
		}
		return err
	}
	return nil
}

func parseRepo(repo string) (cloneURL, owner, name string, err error) {
	trimmed := strings.TrimSpace(repo)
	path := strings.TrimSuffix(trimmed, ".git")
	cloneURL = trimmed

	switch {
	case strings.HasPrefix(path, "git@"):
		if _, after, ok := strings.Cut(path, ":"); ok {
			path = after
		}
	case strings.Contains(path, "://"):
		_, afterScheme, _ := strings.Cut(path, "://")
		if _, afterHost, ok := strings.Cut(afterScheme, "/"); ok {
			path = afterHost
		}
	default:
		cloneURL = "https://github.com/" + strings.Trim(path, "/") + ".git"
	}

	var segments []string
	for segment := range strings.SplitSeq(strings.Trim(path, "/"), "/") {
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	if len(segments) < 2 {
		return "", "", "", fmt.Errorf("%q is not an <owner>/<repo> reference or a git URL ending in one", repo)
	}
	return cloneURL, segments[len(segments)-2], segments[len(segments)-1], nil
}

func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		if current := b.String(); current != "" && !strings.HasSuffix(current, "-") {
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
