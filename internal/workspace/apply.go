package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const BasePreset = "base"

type ApplyConfig struct {
	Root          string
	Base          []byte
	BaseSkills    string
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

type PresetError struct {
	Ref string
	Err error
}

func (e *PresetError) Error() string {
	return "preset " + e.Ref + ": " + e.Err.Error()
}

func (e *PresetError) Unwrap() error {
	return e.Err
}

type LayoutConfig struct {
	Root       string
	Base       []byte
	BaseSkills string
	Foreign    string
	Stash      string
}

type LayoutResult struct {
	Skills   []string
	OldFiles string
	Exclude  string
}

func ApplyLayout(cfg LayoutConfig) (LayoutResult, error) {
	var res LayoutResult
	if cfg.Stash != "" {
		if _, err := os.Stat(cfg.Stash); err != nil {
			return res, fmt.Errorf("%w: %w", ErrNoStash, err)
		}
	}
	if found := Foreign(cfg.Root); len(found) > 0 {
		if cfg.Foreign != "preserve" && cfg.Foreign != "clean" {
			return res, &ForeignError{Found: found}
		}
		var err error
		if res.OldFiles, err = ClearForeign(cfg.Root, found, cfg.Foreign == "preserve"); err != nil {
			return res, err
		}
	}
	if conflicts := PreflightBase(cfg.Root); len(conflicts) > 0 {
		return res, &ConflictError{Conflicts: conflicts}
	}
	var err error
	if res.Skills, err = ApplyBase(cfg.Root, cfg.Base, cfg.BaseSkills); err != nil {
		return res, err
	}
	path, ok, err := WriteGitExclude(cfg.Root)
	if err != nil {
		return res, err
	}
	if ok {
		res.Exclude = path
	}
	if cfg.Stash != "" {
		return res, RestoreResume(cfg.Root, cfg.Stash)
	}
	return res, nil
}

type AddConfig struct {
	Root          string
	LocalPresets  string
	RemotePresets string
	Refs          []string
	Skills        bool
	Agents        bool
}

type AddResult struct {
	Pruned  []string
	Presets []AddedPreset
}

type AddedPreset struct {
	Ref     string
	Skills  int
	Section bool
	Servers []string
}

func AddPresets(cfg AddConfig) (AddResult, error) {
	var res AddResult
	var err error
	if res.Pruned, err = PruneDeadSkillLinks(cfg.Root); err != nil {
		return res, err
	}

	presets := make([]*Preset, 0, len(cfg.Refs))
	var conflicts []Conflict
	if cfg.Agents {
		conflicts = PreflightAgentsFile(cfg.Root)
	}
	for _, ref := range cfg.Refs {
		p, err := FindPreset(cfg.LocalPresets, cfg.RemotePresets, ref)
		if err != nil {
			return res, &PresetError{Ref: ref, Err: err}
		}
		presets = append(presets, p)
		if cfg.Skills {
			conflicts = append(conflicts, PreflightPresetSkills(cfg.Root, p.Skills)...)
		}
	}
	if len(conflicts) > 0 {
		return res, &ConflictError{Conflicts: conflicts}
	}

	for _, p := range presets {
		added, err := addPreset(cfg, p)
		if err != nil {
			return res, &PresetError{Ref: p.Ref, Err: err}
		}
		res.Presets = append(res.Presets, added)
	}
	return res, nil
}

func addPreset(cfg AddConfig, p *Preset) (AddedPreset, error) {
	added := AddedPreset{Ref: p.Ref}
	if cfg.Skills {
		if err := LinkSkills(cfg.Root, p.SkillsDir(), p.Skills); err != nil {
			return added, err
		}
		added.Skills = len(p.Skills)
	}
	if cfg.Agents {
		if partial := p.Partial(); partial != "" {
			if err := UpsertSection(cfg.Root, p.Ref, partial); err != nil {
				return added, err
			}
			added.Section = true
		}
	}
	if cfg.Skills && cfg.Agents {
		var err error
		if added.Servers, err = MergePresetMCP(cfg.Root, p); err != nil {
			return added, err
		}
	}
	return added, nil
}

func Apply(cfg ApplyConfig) error {
	var selected []*Preset
	for _, ref := range cfg.Presets {
		if ref == BasePreset || slices.ContainsFunc(selected, func(p *Preset) bool { return p.Ref == ref }) {
			continue
		}
		p, err := FindPreset(cfg.LocalPresets, cfg.RemotePresets, ref)
		if err != nil {
			return &PresetError{Ref: ref, Err: err}
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

	if _, err := ApplyBase(cfg.Root, cfg.Base, cfg.BaseSkills); err != nil {
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
			return &PresetError{Ref: p.Ref, Err: err}
		}
		if err := UpsertSection(cfg.Root, p.Ref, p.Partial()); err != nil {
			return &PresetError{Ref: p.Ref, Err: err}
		}
		if _, err := MergePresetMCP(cfg.Root, p); err != nil {
			return &PresetError{Ref: p.Ref, Err: err}
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

	p, err := FindPreset(cfg.LocalPresets, cfg.RemotePresets, ref)
	if err != nil {
		return nil
	}
	if err := unlinkSkills(cfg.Root, p); err != nil {
		return err
	}
	return removePresetMCP(cfg.Root, p)
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
