package plugins

import (
	"bytes"
	"cmp"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Source struct {
	Raw     string
	IsLocal bool
	Path    string
	URL     string
	Name    string
}

func Classify(spec string) Source {
	spec = strings.TrimSpace(spec)
	if strings.Contains(spec, "://") || strings.HasPrefix(spec, "git@") {
		return Source{Raw: spec, URL: spec, Name: repoDirName(spec)}
	}
	path := expandHome(spec)
	return Source{Raw: spec, IsLocal: true, Path: path, Name: filepath.Base(path)}
}

func Fetch(src Source, pluginsBase string) (string, error) {
	if src.IsLocal {
		info, err := os.Stat(src.Path)
		if err != nil {
			return "", err
		}
		if !info.IsDir() {
			return "", fmt.Errorf("%s is not a directory", src.Path)
		}
		return src.Path, nil
	}

	if _, err := exec.LookPath("git"); err != nil {
		return "", fmt.Errorf("git is required to fetch plugin repos but was not found in PATH")
	}
	if err := os.MkdirAll(pluginsBase, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(pluginsBase, src.Name)
	if _, err := os.Stat(filepath.Join(dest, ".git")); err == nil {
		if err := update(dest); err != nil {
			return "", err
		}
		return dest, nil
	}
	// git clone refuses a non-empty dir, so clear any partial checkout first.
	if err := os.RemoveAll(dest); err != nil {
		return "", err
	}
	if err := clone(src.URL, dest); err != nil {
		return "", err
	}
	return dest, nil
}

func BuildGlobalPlugin(dir string, skillsFS, outputStylesFS fs.FS, refresh bool) error {
	if err := writeGlobalManifest(dir); err != nil {
		return err
	}
	if err := writeGlobalLSP(dir); err != nil {
		return err
	}

	skills, err := fs.ReadDir(skillsFS, "default-skills")
	if err != nil {
		return err
	}
	for _, e := range skills {
		if !e.IsDir() {
			continue
		}
		dest := filepath.Join(dir, "skills", e.Name())
		if err := installTree(skillsFS, "default-skills/"+e.Name(), dest, refresh); err != nil {
			return err
		}
	}

	styles, err := fs.ReadDir(outputStylesFS, "output-styles")
	if err != nil {
		return err
	}
	for _, e := range styles {
		if e.IsDir() {
			continue
		}
		dest := filepath.Join(dir, "output-styles", e.Name())
		if err := installFile(outputStylesFS, "output-styles/"+e.Name(), dest, refresh); err != nil {
			return err
		}
	}

	return nil
}

// Always rewritten (not write-if-missing) so a manifest from an older plugin name migrates to "claudex".
func writeGlobalManifest(dir string) error {
	manifest := filepath.Join(dir, ".claude-plugin", "plugin.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(map[string]any{
		"name":        "claudex",
		"description": "claudex's curated skills, output styles, and language servers, auto-loaded across every account",
		"version":     "0.0.1",
	}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(manifest, data, 0o644)
}

// Rewritten every build (not write-if-missing like skills/styles) so an added server or schema change reaches existing installs. A server whose binary is absent is skipped by Claude Code, so shipping all three by default is safe.
func writeGlobalLSP(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(map[string]any{
		"go": map[string]any{
			"command":             "gopls",
			"args":                []string{"serve"},
			"extensionToLanguage": map[string]string{".go": "go"},
		},
		"python": map[string]any{
			"command":             "pyright-langserver",
			"args":                []string{"--stdio"},
			"extensionToLanguage": map[string]string{".py": "python", ".pyi": "python"},
		},
		"typescript": map[string]any{
			"command": "typescript-language-server",
			"args":    []string{"--stdio"},
			"extensionToLanguage": map[string]string{
				".ts": "typescript", ".mts": "typescript", ".cts": "typescript",
				".tsx": "typescriptreact",
				".js":  "javascript", ".mjs": "javascript", ".cjs": "javascript",
				".jsx": "javascriptreact",
			},
		},
	}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(filepath.Join(dir, ".lsp.json"), data, 0o644)
}

func installTree(srcFS fs.FS, root, dest string, refresh bool) error {
	if _, err := os.Stat(dest); err == nil && !refresh {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	// Stage the whole tree beside dest, then swap it in with a rename so an interrupted build never leaves a half-written skill in place.
	staging := dest + ".staging"
	if err := os.RemoveAll(staging); err != nil {
		return err
	}
	if err := copyTree(srcFS, root, staging); err != nil {
		os.RemoveAll(staging)
		return err
	}
	if err := os.RemoveAll(dest); err != nil {
		os.RemoveAll(staging)
		return err
	}
	if err := os.Rename(staging, dest); err != nil {
		os.RemoveAll(staging)
		return err
	}
	return nil
}

func copyTree(srcFS fs.FS, root, dest string) error {
	return fs.WalkDir(srcFS, root, func(path string, d fs.DirEntry, walkErr error) error {
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
		return os.WriteFile(out, data, 0o644)
	})
}

func installFile(srcFS fs.FS, srcPath, dest string, refresh bool) error {
	if _, err := os.Stat(dest); err == nil && !refresh {
		return nil
	}
	data, err := fs.ReadFile(srcFS, srcPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return writeFileAtomic(dest, data, 0o644)
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func clone(url, dest string) error {
	args, env := gitAuth()
	args = append(args, "clone", "--depth", "1", url, dest)
	return runGit(env, args...)
}

func update(dest string) error {
	args, env := gitAuth()
	fetchArgs := append([]string{"-C", dest}, args...)
	fetchArgs = append(fetchArgs, "fetch", "--depth", "1", "origin")
	if err := runGit(env, fetchArgs...); err != nil {
		return err
	}
	return runGit(nil, "-C", dest, "reset", "--hard", "FETCH_HEAD")
}

func gitAuth() (args, env []string) {
	if token := cmp.Or(os.Getenv("GH_TOKEN"), os.Getenv("GITHUB_TOKEN")); token != "" {
		basic := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token))
		// Pass the token via env, not a -c arg, so it never lands in the process table.
		return nil, []string{
			"GIT_CONFIG_COUNT=1",
			"GIT_CONFIG_KEY_0=http.extraheader",
			"GIT_CONFIG_VALUE_0=AUTHORIZATION: basic " + basic,
		}
	}
	if _, err := exec.LookPath("gh"); err == nil {
		return []string{"-c", "credential.https://github.com.helper=!gh auth git-credential"}, nil
	}
	return nil, nil
}

func runGit(extraEnv []string, args ...string) error {
	cmd := exec.Command("git", args...)
	// Fail fast instead of hanging on git's interactive credential prompt.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	cmd.Env = append(cmd.Env, extraEnv...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}

func repoDirName(raw string) string {
	s := strings.TrimSuffix(strings.TrimSpace(raw), ".git")
	switch {
	case strings.Contains(s, "://"):
		s = s[strings.Index(s, "://")+3:]
		if i := strings.IndexByte(s, '@'); i >= 0 {
			s = s[i+1:]
		}
	case strings.HasPrefix(s, "git@"):
		s = strings.Replace(strings.TrimPrefix(s, "git@"), ":", "/", 1)
	}
	// Host-qualified to avoid cross-host collisions; "." / ".." dropped so the name can't escape the base.
	var parts []string
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return r == '/' }) {
		if p = sanitizeName(p); p != "" && p != "." && p != ".." {
			parts = append(parts, p)
		}
	}
	name := strings.Join(parts, "-")
	if name == "" {
		return "plugin"
	}
	return name
}

func sanitizeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[1:])
	}
	return p
}
