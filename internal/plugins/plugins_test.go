package plugins

import (
	"encoding/base64"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func TestRepoDirName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"https with .git", "https://github.com/owner/repo.git", "github.com-owner-repo"},
		{"https without .git", "https://github.com/owner/repo", "github.com-owner-repo"},
		{"https trailing slash", "https://github.com/owner/repo/", "github.com-owner-repo"},
		{"scp style", "git@github.com:owner/repo.git", "github.com-owner-repo"},
		{"ssh scheme with userinfo", "ssh://git@github.com/owner/repo.git", "github.com-owner-repo"},
		{"nested groups", "https://gitlab.com/group/sub/repo.git", "gitlab.com-group-sub-repo"},
		{"surrounding whitespace", "  https://github.com/owner/repo.git  ", "github.com-owner-repo"},
		{"host only", "https://github.com/", "github.com"},
		{"empty", "", "plugin"},
		{"unsafe chars sanitized", "https://github.com/o wner/re:po", "github.com-o-wner-re-po"},
		{"dotdot component dropped", "https://github.com/..", "github.com"},
		{"dot component dropped", "https://github.com/.", "github.com"},
		{"scp dotdot dropped", "git@github.com:..", "github.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repoDirName(tt.in)
			if got != tt.want {
				t.Fatalf("repoDirName(%q) = %q, want %q", tt.in, got, tt.want)
			}
			if got == "." || got == ".." {
				t.Fatalf("repoDirName(%q) returned a path-traversal token %q", tt.in, got)
			}
		})
	}
}

func TestRepoDirNameDistinguishesHosts(t *testing.T) {
	gh := repoDirName("https://github.com/foo/bar")
	gl := repoDirName("https://gitlab.com/foo/bar")
	if gh == gl {
		t.Fatalf("repoDirName collapsed different hosts to the same name: %q", gh)
	}
}

func TestClassify(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name        string
		in          string
		wantLocal   bool
		wantName    string
		wantURLName string
	}{
		{"https url is remote", "https://github.com/owner/repo.git", false, "", "github.com-owner-repo"},
		{"scp url is remote", "git@github.com:owner/repo.git", false, "", "github.com-owner-repo"},
		{"existing dir is local", dir, true, filepath.Base(dir), ""},
		{"relative path is local", "./some/plugin", true, "plugin", ""},
		{"bare name is local not shorthand", "owner/repo", true, "repo", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.in)
			if got.IsLocal != tt.wantLocal {
				t.Fatalf("Classify(%q).IsLocal = %v, want %v", tt.in, got.IsLocal, tt.wantLocal)
			}
			if tt.wantLocal && got.Name != tt.wantName {
				t.Fatalf("Classify(%q).Name = %q, want %q", tt.in, got.Name, tt.wantName)
			}
			if !tt.wantLocal {
				if got.URL != tt.in {
					t.Fatalf("Classify(%q).URL = %q, want %q", tt.in, got.URL, tt.in)
				}
				if got.Name != tt.wantURLName {
					t.Fatalf("Classify(%q).Name = %q, want %q", tt.in, got.Name, tt.wantURLName)
				}
			}
		})
	}
}

func TestClassifyExpandsHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	got := Classify("~/plugins/mine")
	want := filepath.Join(home, "plugins", "mine")
	if !got.IsLocal || got.Path != want {
		t.Fatalf("Classify(~/plugins/mine).Path = %q, want %q", got.Path, want)
	}
}

func TestGitAuth(t *testing.T) {
	envFor := func(token string) []string {
		value := "GIT_CONFIG_VALUE_0=AUTHORIZATION: basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:"+token))
		return []string{"GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=http.extraheader", value}
	}

	tests := []struct {
		name        string
		ghToken     string
		githubToken string
		wantEnv     []string
	}{
		{"GH_TOKEN used", "tokA", "", envFor("tokA")},
		{"GITHUB_TOKEN fallback", "", "tokB", envFor("tokB")},
		{"GH_TOKEN takes precedence", "tokA", "tokB", envFor("tokA")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GH_TOKEN", tt.ghToken)
			t.Setenv("GITHUB_TOKEN", tt.githubToken)
			args, env := gitAuth()
			if args != nil {
				t.Fatalf("gitAuth() args = %v, want nil so the token stays out of argv", args)
			}
			if !slices.Equal(env, tt.wantEnv) {
				t.Fatalf("gitAuth() env = %v, want %v", env, tt.wantEnv)
			}
		})
	}
}

func testGlobalFS() (skills, styles fs.FS) {
	return fstest.MapFS{
			"skills/cross-ai/SKILL.md":    {Data: []byte("cross-ai v1")},
			"skills/ai-docs/SKILL.md":     {Data: []byte("ai-docs v1")},
			"skills/ai-docs/refs/note.md": {Data: []byte("nested v1")},
			"skills/go-cli/SKILL.md":      {Data: []byte("not curated")},
		}, fstest.MapFS{
			"output-styles/caveman.md": {Data: []byte("caveman v1")},
			"output-styles/other.md":   {Data: []byte("not curated")},
		}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestBuildGlobalPluginCurates(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "global")
	skills, styles := testGlobalFS()

	if err := BuildGlobalPlugin(dir, skills, styles, false); err != nil {
		t.Fatalf("BuildGlobalPlugin() error = %v", err)
	}

	manifestPath := filepath.Join(dir, ".claude-plugin", "plugin.json")
	var manifest map[string]any
	if err := json.Unmarshal([]byte(read(t, manifestPath)), &manifest); err != nil {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}
	if manifest["name"] != "claudex" {
		t.Fatalf("manifest name = %v, want claudex", manifest["name"])
	}

	if got := read(t, filepath.Join(dir, "skills", "cross-ai", "SKILL.md")); got != "cross-ai v1" {
		t.Fatalf("cross-ai skill = %q", got)
	}
	if got := read(t, filepath.Join(dir, "skills", "ai-docs", "refs", "note.md")); got != "nested v1" {
		t.Fatalf("ai-docs nested file = %q", got)
	}
	if got := read(t, filepath.Join(dir, "output-styles", "caveman.md")); got != "caveman v1" {
		t.Fatalf("caveman = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "skills", "go-cli")); err == nil {
		t.Fatal("non-curated skill go-cli was copied into the plugin")
	}
}

func TestBuildGlobalPluginRefreshVsWriteIfMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "global")
	skills, styles := testGlobalFS()
	if err := BuildGlobalPlugin(dir, skills, styles, false); err != nil {
		t.Fatalf("initial build error = %v", err)
	}

	skillFile := filepath.Join(dir, "skills", "cross-ai", "SKILL.md")
	styleFile := filepath.Join(dir, "output-styles", "caveman.md")
	if err := os.WriteFile(skillFile, []byte("user edit"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(styleFile, []byte("user edit"), 0o644); err != nil {
		t.Fatal(err)
	}

	newSkills := fstest.MapFS{
		"skills/cross-ai/SKILL.md": {Data: []byte("cross-ai v2")},
		"skills/ai-docs/SKILL.md":  {Data: []byte("ai-docs v2")},
	}
	newStyles := fstest.MapFS{"output-styles/caveman.md": {Data: []byte("caveman v2")}}

	if err := BuildGlobalPlugin(dir, newSkills, newStyles, false); err != nil {
		t.Fatalf("write-if-missing build error = %v", err)
	}
	if got := read(t, skillFile); got != "user edit" {
		t.Fatalf("write-if-missing clobbered an existing skill: %q", got)
	}
	if got := read(t, styleFile); got != "user edit" {
		t.Fatalf("write-if-missing clobbered an existing style: %q", got)
	}

	if err := BuildGlobalPlugin(dir, newSkills, newStyles, true); err != nil {
		t.Fatalf("refresh build error = %v", err)
	}
	if got := read(t, skillFile); got != "cross-ai v2" {
		t.Fatalf("refresh did not replace the skill: %q", got)
	}
	if got := read(t, styleFile); got != "caveman v2" {
		t.Fatalf("refresh did not replace the style: %q", got)
	}
}

func TestBuildGlobalPluginRefreshIsCleanSwap(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "global")
	skills, styles := testGlobalFS()
	if err := BuildGlobalPlugin(dir, skills, styles, true); err != nil {
		t.Fatalf("initial build error = %v", err)
	}

	newSkills := fstest.MapFS{
		"skills/cross-ai/SKILL.md": {Data: []byte("cross-ai v2")},
		"skills/ai-docs/SKILL.md":  {Data: []byte("ai-docs v2")},
	}
	if err := BuildGlobalPlugin(dir, newSkills, styles, true); err != nil {
		t.Fatalf("refresh build error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "skills", "ai-docs", "refs", "note.md")); err == nil {
		t.Fatal("refresh left a file the new source dropped; the tree was not replaced wholesale")
	}
	assertNoResidue(t, dir)
}

func assertNoResidue(t *testing.T, root string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(d.Name(), ".staging") || strings.HasSuffix(d.Name(), ".tmp") {
			t.Fatalf("build left staging residue: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
}
