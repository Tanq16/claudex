package workspace

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

const (
	ManifestName = "preset.yaml"
	PartialName  = "AGENTS.partial.md"
	skillFile    = "SKILL.md"
)

type Preset struct {
	Name        string
	Description string
	Dir         string
	Skills      []string
}

type manifest struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Skills      []string `yaml:"skills"`
}

func EnsurePresets(srcFS fs.FS, root, dir string) error {
	entries, err := fs.ReadDir(srcFS, root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := installTree(srcFS, root+"/"+e.Name(), filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func ListPresets(dir string) []Preset {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var found []Preset
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p, err := loadPreset(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		found = append(found, *p)
	}
	slices.SortFunc(found, func(a, b Preset) int { return strings.Compare(a.Name, b.Name) })
	return found
}

func FindPreset(dir, name string) (*Preset, error) {
	if !validName(name) {
		return nil, fmt.Errorf("%q is not a valid preset name", name)
	}
	return loadPreset(filepath.Join(dir, name))
}

func loadPreset(dir string) (*Preset, error) {
	data, err := os.ReadFile(filepath.Join(dir, ManifestName))
	if err != nil {
		return nil, err
	}
	var m manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("%s is not valid YAML: %w", filepath.Join(dir, ManifestName), err)
	}

	skillsDir := filepath.Join(dir, SkillsDir)
	skills := m.Skills
	if len(skills) == 0 {
		skills = skillNames(skillsDir)
	} else {
		for _, s := range skills {
			if _, err := os.Stat(filepath.Join(skillsDir, s, skillFile)); err != nil {
				return nil, fmt.Errorf("preset lists skill %q but %s has no such skill", s, skillsDir)
			}
		}
	}

	name := m.Name
	if name == "" {
		name = filepath.Base(dir)
	}
	return &Preset{Name: name, Description: m.Description, Dir: dir, Skills: skills}, nil
}

func (p Preset) SkillsDir() string { return filepath.Join(p.Dir, SkillsDir) }

func (p Preset) Partial() string {
	data, err := os.ReadFile(filepath.Join(p.Dir, PartialName))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func ScaffoldPreset(dir, name string) (string, error) {
	if !validName(name) {
		return "", fmt.Errorf("%q is not a valid preset name; use lowercase letters, digits, and single hyphens", name)
	}
	target := filepath.Join(dir, name)
	if _, err := os.Stat(target); err == nil {
		return "", fmt.Errorf("%s already exists", target)
	}
	if err := os.MkdirAll(filepath.Join(target, SkillsDir), 0o755); err != nil {
		return "", err
	}
	files := map[string]string{
		ManifestName: fmt.Sprintf("name: %s\ndescription: What this preset is for, shown in the picker\n\n# skills: []  # optional, defaults to every skill under skills/\n", name),
		PartialName:  fmt.Sprintf("## %s\n\nRules this preset adds to AGENTS.md. Keep it short and give every rule a reason.\n", name),
	}
	for base, body := range files {
		if err := os.WriteFile(filepath.Join(target, base), []byte(body), 0o644); err != nil {
			return "", err
		}
	}
	return target, nil
}

// Keeps a preset name a safe path component.
func validName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") || strings.Contains(name, "--") {
		return false
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

func skillNames(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if _, err := os.Stat(filepath.Join(dir, e.Name(), skillFile)); err == nil {
			names = append(names, e.Name())
		}
	}
	slices.Sort(names)
	return names
}

func installSkills(srcFS fs.FS, root, dest string) ([]string, error) {
	entries, err := fs.ReadDir(srcFS, root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := installTree(srcFS, root+"/"+e.Name(), filepath.Join(dest, e.Name())); err != nil {
			return nil, err
		}
		names = append(names, e.Name())
	}
	slices.Sort(names)
	return names, nil
}

// Staged beside dest and swapped in with a rename so an interrupted install never leaves half a skill in place.
func installTree(srcFS fs.FS, root, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
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
		if walkErr != nil || d.IsDir() {
			return walkErr
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
