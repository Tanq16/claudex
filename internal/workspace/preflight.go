package workspace

import (
	"os"
	"path/filepath"
)

type Conflict struct {
	Path string
	Why  string
}

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

func PreflightPresetSkills(root string, names []string) []Conflict {
	c := check{root: root}
	c.dir(AgentsDir)
	c.dir(filepath.Join(AgentsDir, SkillsDir))
	for _, name := range names {
		c.ownedLink(filepath.Join(AgentsDir, SkillsDir, name))
	}
	return c.found
}

func PreflightAgentsFile(root string) []Conflict {
	c := check{root: root}
	c.file(AgentsFile)
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
