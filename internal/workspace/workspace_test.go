package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
