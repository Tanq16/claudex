package embedded

import (
	"bytes"
	"errors"
	"io/fs"
	"path"
	"testing"

	"github.com/goccy/go-yaml"
)

type skillFrontmatter struct {
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
	UserInvocable bool   `yaml:"user-invocable"`
}

func TestEmbeddedSkillFrontmatter(t *testing.T) {
	sources := map[string]fs.FS{
		"default-skills": DefaultSkillsFS,
		"presets":        PresetsFS,
	}

	found := 0
	for source, fsys := range sources {
		err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || d.Name() != "SKILL.md" {
				return nil
			}
			found++
			t.Run(p, func(t *testing.T) {
				data, err := fs.ReadFile(fsys, p)
				if err != nil {
					t.Fatalf("reading %s: %v", p, err)
				}
				block, err := frontmatterBlock(data)
				if err != nil {
					t.Fatalf("%s: %v", p, err)
				}
				var meta skillFrontmatter
				if err := yaml.Unmarshal(block, &meta); err != nil {
					t.Fatalf("%s: frontmatter does not parse: %v", p, err)
				}
				if dir := path.Base(path.Dir(p)); meta.Name != dir {
					t.Errorf("%s: name is %q, want the directory name %q", p, meta.Name, dir)
				}
				if meta.Description == "" {
					t.Errorf("%s: description is empty", p)
				}
				if len(meta.Description) > 1024 {
					t.Errorf("%s: description is %d chars, over the 1024 limit", p, len(meta.Description))
				}
			})
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", source, err)
		}
	}

	if found == 0 {
		t.Fatal("no SKILL.md found in the embedded filesystems")
	}
}

func TestFrontmatterBlock(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"empty file", "", "", true},
		{"no frontmatter", "# Title\n", "", true},
		{"unterminated", "---\nname: x\n", "", true},
		{"body holds a rule", "---\nname: x\n---\n\ntext\n\n---\n\nmore\n", "name: x", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := frontmatterBlock([]byte(tt.in))
			if (err != nil) != tt.wantErr {
				t.Fatalf("frontmatterBlock(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			}
			if err == nil && string(got) != tt.want {
				t.Errorf("frontmatterBlock(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func frontmatterBlock(data []byte) ([]byte, error) {
	rest, ok := bytes.CutPrefix(data, []byte("---\n"))
	if !ok {
		return nil, errors.New("no frontmatter block")
	}
	block, _, ok := bytes.Cut(rest, []byte("\n---"))
	if !ok {
		return nil, errors.New("unterminated frontmatter block")
	}
	return block, nil
}
