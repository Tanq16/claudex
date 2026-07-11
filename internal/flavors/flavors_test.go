package flavors

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string
		wantAuto    string
		wantChoices []string
	}{
		{"empty dir", nil, "", nil},
		{"no md files", map[string]string{"notes.txt": "hi", "README": "x"}, "", nil},
		{"only default", map[string]string{"default.md": "be terse"}, "default", nil},
		{
			"default plus others",
			map[string]string{"default.md": "d", "security.md": "s", "learning.md": "l"},
			"",
			[]string{"default", "learning", "security"},
		},
		{
			"others no default",
			map[string]string{"security.md": "s", "dev.md": "d"},
			"",
			[]string{"dev", "security"},
		},
		{
			"blank default is skipped",
			map[string]string{"default.md": "   \n", "dev.md": "d"},
			"",
			[]string{"dev"},
		},
		{"all blank yields nothing", map[string]string{"default.md": "\n\t "}, "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for n, b := range tt.files {
				write(t, dir, n, b)
			}
			got, err := Load(dir)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if tt.wantAuto == "" {
				if got.Auto != nil {
					t.Fatalf("Auto = %q, want nil", got.Auto.Name)
				}
			} else {
				if got.Auto == nil || got.Auto.Name != tt.wantAuto {
					t.Fatalf("Auto = %v, want %q", got.Auto, tt.wantAuto)
				}
				if got.Auto.Body == "" {
					t.Fatal("Auto.Body is empty, want file contents")
				}
			}
			if len(got.Choices) != len(tt.wantChoices) {
				t.Fatalf("Choices = %v, want %v", names(got.Choices), tt.wantChoices)
			}
			for i, c := range got.Choices {
				if c.Name != tt.wantChoices[i] {
					t.Fatalf("Choices[%d] = %q, want %q", i, c.Name, tt.wantChoices[i])
				}
				if c.Body == "" {
					t.Fatalf("Choices[%d] %q has empty body", i, c.Name)
				}
			}
		})
	}
}

func TestLoadMissingDir(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("Load() on missing dir error = %v, want nil", err)
	}
	if got.Auto != nil || len(got.Choices) != 0 {
		t.Fatalf("Load() on missing dir = %+v, want empty Options", got)
	}
}

func names(fs []Flavor) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.Name
	}
	return out
}
