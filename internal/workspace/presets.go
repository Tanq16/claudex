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

type fileModes struct {
	dir  os.FileMode
	file os.FileMode
}

var (
	configModes  = fileModes{dir: 0o700, file: 0o600}
	projectModes = fileModes{dir: 0o755, file: 0o644}
)

type Preset struct {
	Ref         string
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
	if err := os.MkdirAll(dir, configModes.dir); err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := installTree(srcFS, root+"/"+e.Name(), filepath.Join(dir, e.Name()), configModes); err != nil {
			return err
		}
	}
	return nil
}

func ListPresets(localDir, remoteDir string) []Preset {
	found := presetsIn(localDir, "")
	for _, slug := range subdirs(remoteDir) {
		found = append(found, presetsIn(filepath.Join(remoteDir, slug), slug)...)
	}
	slices.SortFunc(found, func(a, b Preset) int { return strings.Compare(a.Ref, b.Ref) })
	return found
}

func FindPreset(localDir, remoteDir, ref string) (*Preset, error) {
	slug, name, remote := strings.Cut(ref, "/")
	dir := localDir
	if !remote {
		slug, name = "", slug
	} else {
		if !validName(slug) {
			return nil, fmt.Errorf("%q is not a valid repository slug", slug)
		}
		dir = filepath.Join(remoteDir, slug)
	}
	if !validName(name) {
		return nil, fmt.Errorf("%q is not a valid preset name", name)
	}
	p, err := loadPreset(filepath.Join(dir, name))
	if err != nil {
		return nil, err
	}
	p.Ref = presetRef(slug, name)
	return p, nil
}

func presetsIn(dir, slug string) []Preset {
	var found []Preset
	for _, name := range subdirs(dir) {
		p, err := loadPreset(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		p.Ref = presetRef(slug, name)
		found = append(found, *p)
	}
	return found
}

func presetRef(slug, name string) string {
	if slug == "" {
		return name
	}
	return slug + "/" + name
}

func subdirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
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
	if err := os.MkdirAll(filepath.Join(target, SkillsDir), configModes.dir); err != nil {
		return "", err
	}
	files := map[string]string{
		ManifestName: fmt.Sprintf("name: %s\ndescription: What this preset is for, shown in the picker\n\n# skills: []  # optional, defaults to every skill under skills/\n", name),
		PartialName:  "",
	}
	for base, body := range files {
		if err := os.WriteFile(filepath.Join(target, base), []byte(body), configModes.file); err != nil {
			return "", err
		}
	}
	return target, nil
}

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
	if err := os.MkdirAll(dest, projectModes.dir); err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := installTree(srcFS, root+"/"+e.Name(), filepath.Join(dest, e.Name()), projectModes); err != nil {
			return nil, err
		}
		names = append(names, e.Name())
	}
	slices.Sort(names)
	return names, nil
}

func installTree(srcFS fs.FS, root, dest string, m fileModes) error {
	if err := os.MkdirAll(filepath.Dir(dest), m.dir); err != nil {
		return err
	}
	// Staged and swapped in with a rename so an interrupted install never leaves half a skill in place.
	staging := dest + ".staging"
	if err := os.RemoveAll(staging); err != nil {
		return err
	}
	if err := copyTree(srcFS, root, staging, m); err != nil {
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

func copyTree(srcFS fs.FS, root, dest string, m fileModes) error {
	return fs.WalkDir(srcFS, root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		out := filepath.Join(dest, strings.TrimPrefix(path, root+"/"))
		if err := os.MkdirAll(filepath.Dir(out), m.dir); err != nil {
			return err
		}
		data, err := fs.ReadFile(srcFS, path)
		if err != nil {
			return err
		}
		return os.WriteFile(out, data, m.file)
	})
}
