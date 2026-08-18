package workspace

import (
	"os"
	"path/filepath"
)

// A path the layout needs that already holds something claudex did not put there.
type Conflict struct {
	Path string
	Why  string
}

// Everything ApplyBase writes, checked before it writes any of it, so a directory that cannot take the whole layout keeps the state it had instead of a partial one.
func PreflightBase(root string) []Conflict {
	c := check{root: root}
	c.dir(AgentsDir)
	c.dir(filepath.Join(AgentsDir, SkillsDir))
	c.file(AgentsFile)
	c.link(ClaudeFile, AgentsFile)
	c.dir(ClaudeDir)
	c.link(filepath.Join(ClaudeDir, SkillsDir), filepath.Join("..", AgentsDir, SkillsDir))
	return c.found
}

// The same check for what LinkSkills and a preset's AGENTS.md section write.
func PreflightPreset(root string, names []string) []Conflict {
	c := check{root: root}
	c.dir(AgentsDir)
	c.dir(filepath.Join(AgentsDir, SkillsDir))
	c.file(AgentsFile)
	for _, name := range names {
		c.ownedLink(filepath.Join(AgentsDir, SkillsDir, name))
	}
	return c.found
}

type check struct {
	root  string
	found []Conflict
}

func (c *check) dir(rel string) {
	path := filepath.Join(c.root, rel)
	if _, err := os.Lstat(path); err != nil {
		return
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return
	}
	c.add(rel, "exists and is not a directory")
}

func (c *check) file(rel string) {
	info, err := os.Lstat(filepath.Join(c.root, rel))
	if err != nil || info.Mode().IsRegular() {
		return
	}
	c.add(rel, "exists and is not a regular file")
}

func (c *check) link(rel, target string) {
	path := filepath.Join(c.root, rel)
	if current, err := os.Readlink(path); err == nil {
		if current != target {
			c.add(rel, "is a symlink to "+current+", not to "+target)
		}
		return
	}
	if _, err := os.Lstat(path); err == nil {
		c.add(rel, "exists and is not a claudex symlink")
	}
}

// A symlink under .agents/skills is claudex's own and gets repointed; anything else there was put in by hand.
func (c *check) ownedLink(rel string) {
	path := filepath.Join(c.root, rel)
	if _, err := os.Readlink(path); err == nil {
		return
	}
	if _, err := os.Lstat(path); err == nil {
		c.add(rel, "exists and is not a claudex symlink")
	}
}

func (c *check) add(rel, why string) {
	c.found = append(c.found, Conflict{Path: rel, Why: why})
}
