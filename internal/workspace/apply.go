package workspace

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const BasePreset = "base"

type ApplyConfig struct {
	Root          string
	Base          []byte
	BaseSkills    fs.FS
	BaseSkillsDir string
	LocalPresets  string
	RemotePresets string
	Presets       []string
}

type ConflictError struct {
	Conflicts []Conflict
}

func (e *ConflictError) Error() string {
	parts := make([]string, len(e.Conflicts))
	for i, c := range e.Conflicts {
		parts[i] = c.Path + " " + c.Why
	}
	return strings.Join(parts, "; ")
}

func Apply(cfg ApplyConfig) error {
	var selected []*Preset
	for _, ref := range cfg.Presets {
		if ref == BasePreset || slices.ContainsFunc(selected, func(p *Preset) bool { return p.Ref == ref }) {
			continue
		}
		p, err := FindPreset(cfg.LocalPresets, cfg.RemotePresets, ref)
		if err != nil {
			return fmt.Errorf("preset %s: %w", ref, err)
		}
		selected = append(selected, p)
	}

	conflicts := PreflightBase(cfg.Root)
	for _, p := range selected {
		for _, c := range PreflightPresetSkills(cfg.Root, p.Skills) {
			if !slices.Contains(conflicts, c) {
				conflicts = append(conflicts, c)
			}
		}
	}
	if len(conflicts) > 0 {
		return &ConflictError{Conflicts: conflicts}
	}

	if _, err := ApplyBase(cfg.Root, cfg.Base, cfg.BaseSkills, cfg.BaseSkillsDir); err != nil {
		return err
	}
	if _, _, err := WriteGitExclude(cfg.Root); err != nil {
		return err
	}
	if _, err := PruneDeadSkillLinks(cfg.Root); err != nil {
		return err
	}
	for _, ref := range AppliedPresets(cfg.Root) {
		if !slices.ContainsFunc(selected, func(p *Preset) bool { return p.Ref == ref }) {
			if err := removePreset(cfg, ref); err != nil {
				return err
			}
		}
	}
	for _, p := range selected {
		if err := LinkSkills(cfg.Root, p.SkillsDir(), p.Skills); err != nil {
			return fmt.Errorf("preset %s: %w", p.Ref, err)
		}
		if err := UpsertSection(cfg.Root, p.Ref, p.Partial()); err != nil {
			return fmt.Errorf("preset %s: %w", p.Ref, err)
		}
	}
	return nil
}

func Presets(root string) []string {
	if !Applied(root) {
		return nil
	}
	return append([]string{BasePreset}, AppliedPresets(root)...)
}

func AppliedPresets(root string) []string {
	body := readFile(filepath.Join(root, AgentsFile))
	var refs []string
	for {
		_, rest, found := strings.Cut(body, openPrefix+"preset:")
		if !found {
			return refs
		}
		ref, after, ended := strings.Cut(rest, " -->")
		if !ended {
			return refs
		}
		refs = append(refs, ref)
		body = after
	}
}

func removePreset(cfg ApplyConfig, ref string) error {
	path := filepath.Join(cfg.Root, AgentsFile)
	body := readFile(path)
	open, closing := markers(ref)
	if stripped := removeSection(body, open, closing); stripped != body {
		if err := writeFileAtomic(path, []byte(stripped), projectModes.file); err != nil {
			return err
		}
	}

	if p, err := FindPreset(cfg.LocalPresets, cfg.RemotePresets, ref); err == nil {
		return unlinkSkills(cfg.Root, p)
	}
	return nil
}

func unlinkSkills(root string, p *Preset) error {
	for _, name := range p.Skills {
		target, err := filepath.Abs(filepath.Join(p.SkillsDir(), name))
		if err != nil {
			return err
		}
		link := filepath.Join(SkillsPath(root), name)
		if current, err := os.Readlink(link); err == nil && current == target {
			if err := os.Remove(link); err != nil {
				return err
			}
		}
	}
	return nil
}

func removeSection(body, open, closing string) string {
	head, rest, found := strings.Cut(body, open)
	if !found {
		return body
	}
	head = strings.TrimRight(head, "\n")
	_, tail, closed := strings.Cut(rest, closing)
	if !closed {
		return head + "\n"
	}
	tail = strings.TrimLeft(tail, "\n")
	switch {
	case head == "":
		return tail
	case tail == "":
		return head + "\n"
	default:
		return head + "\n\n" + tail
	}
}
