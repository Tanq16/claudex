package workspace

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	AgentsFile = "AGENTS.md"
	ClaudeFile = "CLAUDE.md"
	AgentsDir  = ".agents"
	ClaudeDir  = ".claude"
	SkillsDir  = "skills"

	openPrefix   = "<!-- claudex:"
	closePrefix  = "<!-- /claudex:"
	excludeBegin = "# claudex:begin"
	excludeEnd   = "# claudex:end"
)

func SkillsPath(root string) string { return filepath.Join(root, AgentsDir, SkillsDir) }

func Applied(root string) bool {
	info, err := os.Stat(SkillsPath(root))
	return err == nil && info.IsDir()
}

func ApplyBase(root string, base []byte, skillsFS fs.FS, skillsRoot string) ([]string, error) {
	names, err := installSkills(skillsFS, skillsRoot, SkillsPath(root))
	if err != nil {
		return nil, err
	}
	if err := UpsertSection(root, "", string(base)); err != nil {
		return nil, err
	}
	if err := ensureLink(filepath.Join(root, ClaudeFile), AgentsFile); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(root, ClaudeDir), 0o755); err != nil {
		return nil, err
	}
	if err := ensureLink(filepath.Join(root, ClaudeDir, SkillsDir), filepath.Join("..", AgentsDir, SkillsDir)); err != nil {
		return nil, err
	}
	return names, WriteGitExclude(root)
}

func LinkSkills(root, srcDir string, names []string) error {
	if err := os.MkdirAll(SkillsPath(root), 0o755); err != nil {
		return err
	}
	for _, name := range names {
		target, err := filepath.Abs(filepath.Join(srcDir, name))
		if err != nil {
			return err
		}
		if err := ensureLink(filepath.Join(SkillsPath(root), name), target); err != nil {
			return err
		}
	}
	return nil
}

// An empty name addresses the base block; anything else addresses a preset's.
func UpsertSection(root, name, content string) error {
	path := filepath.Join(root, AgentsFile)
	body, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	open, closing := markers(name)
	return writeFileAtomic(path, []byte(upsert(string(body), open, closing, content)), 0o644)
}

func Clean(root string) error {
	if err := os.RemoveAll(filepath.Join(root, AgentsDir)); err != nil {
		return err
	}
	if err := removeIfLink(filepath.Join(root, ClaudeFile)); err != nil {
		return err
	}
	if err := removeIfLink(filepath.Join(root, ClaudeDir, SkillsDir)); err != nil {
		return err
	}
	os.Remove(filepath.Join(root, ClaudeDir))
	if err := cleanAgentsFile(root); err != nil {
		return err
	}
	return StripGitExclude(root)
}

func cleanAgentsFile(root string) error {
	path := filepath.Join(root, AgentsFile)
	body, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	rest := stripSections(string(body))
	if strings.TrimSpace(rest) == "" {
		return os.Remove(path)
	}
	return writeFileAtomic(path, []byte(rest), 0o644)
}

func markers(name string) (open, closing string) {
	if name == "" {
		return openPrefix + "base -->", closePrefix + "base -->"
	}
	return openPrefix + "preset:" + name + " -->", closePrefix + "preset -->"
}

func upsert(body, open, closing, content string) string {
	block := open + "\n\n" + strings.TrimSpace(content) + "\n\n" + closing + "\n"

	head, rest, found := strings.Cut(body, open)
	if !found {
		if body == "" {
			return block
		}
		return strings.TrimRight(body, "\n") + "\n\n" + block
	}
	_, tail, closed := strings.Cut(rest, closing)
	if !closed {
		// A block whose terminator was deleted has no recoverable tail, so replacing to the end is the only way back to a well-formed pair.
		return head + block
	}
	if tail = strings.TrimLeft(tail, "\n"); tail == "" {
		return head + block
	}
	return head + block + "\n" + tail
}

func stripSections(body string) string {
	for {
		head, rest, found := strings.Cut(body, openPrefix)
		if !found {
			return body
		}
		_, closing, closed := strings.Cut(rest, closePrefix)
		if !closed {
			return strings.TrimRight(head, "\n")
		}
		_, tail, ended := strings.Cut(closing, "-->")
		if !ended {
			return strings.TrimRight(head, "\n")
		}
		head = strings.TrimRight(head, "\n")
		tail = strings.TrimLeft(tail, "\n")
		switch {
		case head == "":
			body = tail
		case tail == "":
			body = head + "\n"
		default:
			body = head + "\n\n" + tail
		}
	}
}

func ensureLink(link, target string) error {
	current, err := os.Readlink(link)
	switch {
	case err == nil && current == target:
		return nil
	case err == nil:
		if err := os.Remove(link); err != nil {
			return err
		}
	default:
		if _, err := os.Lstat(link); err == nil {
			return fmt.Errorf("%s exists and is not a claudex symlink; move it aside first", link)
		}
	}
	return os.Symlink(target, link)
}

func removeIfLink(path string) error {
	if _, err := os.Readlink(path); err != nil {
		return nil
	}
	return os.Remove(path)
}

func ExcludeFile(root string) (string, bool) {
	path, _, ok := gitExcludePath(root)
	return path, ok
}

func WriteGitExclude(root string) error {
	path, prefix, ok := gitExcludePath(root)
	if !ok {
		return nil
	}
	block := excludeBegin + "\n" + strings.Join([]string{
		prefix + AgentsFile,
		prefix + ClaudeFile,
		prefix + AgentsDir + "/",
		prefix + ClaudeDir + "/" + SkillsDir,
	}, "\n") + "\n" + excludeEnd + "\n"

	body := stripExcludeBlock(readFile(path))
	if body != "" {
		body = strings.TrimRight(body, "\n") + "\n"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeFileAtomic(path, []byte(body+block), 0o644)
}

func StripGitExclude(root string) error {
	path, _, ok := gitExcludePath(root)
	if !ok {
		return nil
	}
	body := readFile(path)
	stripped := stripExcludeBlock(body)
	if stripped == body {
		return nil
	}
	return writeFileAtomic(path, []byte(stripped), 0o644)
}

func stripExcludeBlock(body string) string {
	head, rest, found := strings.Cut(body, excludeBegin)
	if !found {
		return head
	}
	_, tail, closed := strings.Cut(rest, excludeEnd)
	if !closed {
		return head
	}
	return head + strings.TrimLeft(tail, "\n")
}

// Exclude patterns are anchored at the worktree root, so applying inside a subdirectory has to carry it in every entry; both paths are resolved first because git reports the root with symlinks resolved and the working directory may not be.
func gitExcludePath(root string) (path, prefix string, ok bool) {
	out, err := exec.Command("git", "-C", root, "rev-parse", "--absolute-git-dir", "--show-toplevel").Output()
	if err != nil {
		return "", "", false
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 {
		return "", "", false
	}
	rel, err := filepath.Rel(resolve(lines[1]), resolve(root))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", false
	}
	prefix = "/"
	if rel != "." {
		prefix += filepath.ToSlash(rel) + "/"
	}
	return filepath.Join(lines[0], "info", "exclude"), prefix, true
}

func resolve(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return path
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
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
