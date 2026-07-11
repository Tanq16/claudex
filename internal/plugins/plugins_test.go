package plugins

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"slices"
	"testing"
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

func TestEnsureGlobalPlugin(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "global")

	created, err := EnsureGlobalPlugin(dir)
	if err != nil {
		t.Fatalf("EnsureGlobalPlugin() error = %v", err)
	}
	if !created {
		t.Fatal("EnsureGlobalPlugin() created = false on first call, want true")
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude-plugin", "plugin.json")); err != nil {
		t.Fatalf("manifest not written: %v", err)
	}

	created, err = EnsureGlobalPlugin(dir)
	if err != nil {
		t.Fatalf("EnsureGlobalPlugin() second call error = %v", err)
	}
	if created {
		t.Fatal("EnsureGlobalPlugin() created = true on second call, want false")
	}
}
