package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func TestUpsert(t *testing.T) {
	open, closing := markers("dev")

	tests := []struct {
		name    string
		body    string
		content string
		want    string
	}{
		{"empty file", "", "rules", open + "\n\nrules\n\n" + closing + "\n"},
		{"appends after existing prose", "intro\n", "rules", "intro\n\n" + open + "\n\nrules\n\n" + closing + "\n"},
		{
			"replaces in place, keeping both sides",
			"intro\n\n" + open + "\n\nold\n\n" + closing + "\n\ntrailing\n",
			"new",
			"intro\n\n" + open + "\n\nnew\n\n" + closing + "\n\ntrailing\n",
		},
		{
			"a block whose terminator was deleted is replaced to the end",
			"intro\n\n" + open + "\n\nold\nstray\n",
			"new",
			"intro\n\n" + open + "\n\nnew\n\n" + closing + "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := upsert(tt.body, open, closing, tt.content)
			if got != tt.want {
				t.Fatalf("upsert() = %q, want %q", got, tt.want)
			}
			if again := upsert(got, open, closing, tt.content); again != got {
				t.Fatalf("upsert is not idempotent: %q then %q", got, again)
			}
		})
	}
}

func TestStripSections(t *testing.T) {
	baseOpen, baseClose := markers("")
	devOpen, devClose := markers("dev")

	body := "mine\n\n" + baseOpen + "\n\nbase\n\n" + baseClose + "\n\n" + devOpen + "\n\ndev\n\n" + devClose + "\n\nalso mine\n"
	if got := stripSections(body); got != "mine\n\nalso mine\n" {
		t.Fatalf("stripSections() = %q, want the user's prose only", got)
	}
	if got := strings.TrimSpace(stripSections(baseOpen + "\n\nbase\n\n" + baseClose + "\n")); got != "" {
		t.Fatalf("stripSections() left %q for a file claudex wrote entirely", got)
	}
}

func TestPreflightBase(t *testing.T) {
	claudeSkills := filepath.Join(ClaudeDir, SkillsDir)

	tests := []struct {
		name  string
		setup func(t *testing.T, root string)
		want  []string
	}{
		{"an empty directory takes the whole layout", func(*testing.T, string) {}, nil},
		{"prose already in AGENTS.md", func(t *testing.T, root string) {
			mustWrite(t, filepath.Join(root, AgentsFile), "my rules\n")
		}, nil},
		{"a hand-written CLAUDE.md", func(t *testing.T, root string) {
			mustWrite(t, filepath.Join(root, ClaudeFile), "my rules\n")
		}, []string{ClaudeFile}},
		{"an AGENTS.md symlink of the user's own", func(t *testing.T, root string) {
			mustWrite(t, filepath.Join(root, "shared.md"), "shared\n")
			mustLink(t, filepath.Join(root, AgentsFile), "shared.md")
		}, []string{AgentsFile}},
		{"a real .claude/skills directory", func(t *testing.T, root string) {
			mustDir(t, filepath.Join(root, claudeSkills))
		}, []string{claudeSkills}},
		{".claude/skills pointing somewhere else", func(t *testing.T, root string) {
			mustLink(t, filepath.Join(root, claudeSkills), "/elsewhere")
		}, []string{claudeSkills}},
		{".agents taken by a regular file", func(t *testing.T, root string) {
			mustWrite(t, filepath.Join(root, AgentsDir), "not a directory\n")
		}, []string{AgentsDir}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)

			var paths []string
			for _, c := range PreflightBase(root) {
				paths = append(paths, c.Path)
			}
			if !slices.Equal(paths, tt.want) {
				t.Fatalf("PreflightBase() = %v, want %v", paths, tt.want)
			}
		})
	}
}

func TestPreflightBaseRefusesWhatApplyBaseCannotFinish(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ClaudeFile), "my rules\n")

	conflicts := PreflightBase(root)
	if len(conflicts) != 1 || conflicts[0].Path != ClaudeFile {
		t.Fatalf("PreflightBase() = %v, want the CLAUDE.md conflict alone", conflicts)
	}
	if _, err := ApplyBase(root, []byte("base"), demoSkills(), "src"); err == nil {
		t.Fatal("ApplyBase() succeeded where preflight found a conflict")
	}
}

func TestPreflightBaseAcceptsAnAlreadyAppliedDirectory(t *testing.T) {
	root := t.TempDir()
	if _, err := ApplyBase(root, []byte("base"), demoSkills(), "src"); err != nil {
		t.Fatalf("ApplyBase() error = %v", err)
	}
	if got := PreflightBase(root); got != nil {
		t.Fatalf("PreflightBase() after apply = %v, want none", got)
	}
}

func TestPreflightPreset(t *testing.T) {
	root := t.TempDir()
	mustDir(t, SkillsPath(root))
	mustLink(t, filepath.Join(SkillsPath(root), "stale"), "/gone/stale")
	mustDir(t, filepath.Join(SkillsPath(root), "handwritten"))

	got := PreflightPreset(root, []string{"absent", "stale", "handwritten"})
	want := filepath.Join(AgentsDir, SkillsDir, "handwritten")
	if len(got) != 1 || got[0].Path != want {
		t.Fatalf("PreflightPreset() = %v, want %s alone", got, want)
	}
}

func demoSkills() fstest.MapFS {
	return fstest.MapFS{"src/demo/SKILL.md": &fstest.MapFile{Data: []byte("demo\n")}}
}

func mustDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	mustDir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustLink(t *testing.T, path, target string) {
	t.Helper()
	mustDir(t, filepath.Dir(path))
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}

func TestGitExcludeIsAnchoredAtTheWorktreeRoot(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("git init failed: %v: %s", err, out)
	}
	sub := filepath.Join(repo, "nested", "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	excludePath := filepath.Join(repo, ".git", "info", "exclude")
	if err := os.WriteFile(excludePath, []byte("*.local\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteGitExclude(sub); err != nil {
		t.Fatalf("WriteGitExclude() error = %v", err)
	}
	body := readFile(excludePath)
	for _, want := range []string{"*.local", "/nested/pkg/AGENTS.md", "/nested/pkg/.agents/", "/nested/pkg/.claude/skills"} {
		if !strings.Contains(body, want+"\n") {
			t.Fatalf("exclude file missing %q:\n%s", want, body)
		}
	}

	if err := WriteGitExclude(sub); err != nil {
		t.Fatalf("second WriteGitExclude() error = %v", err)
	}
	if got := strings.Count(readFile(excludePath), excludeBegin); got != 1 {
		t.Fatalf("re-write left %d claudex blocks, want 1", got)
	}

	if err := StripGitExclude(sub); err != nil {
		t.Fatalf("StripGitExclude() error = %v", err)
	}
	if after := readFile(excludePath); after != "*.local\n" {
		t.Fatalf("StripGitExclude() = %q, want the pre-existing entry alone", after)
	}
}
