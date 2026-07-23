package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const gateHeading = "## Start here — required reading"

var (
	ownRefRe = regexp.MustCompile(`\./references/([\w.-]+\.md)`)
	sibRefRe = regexp.MustCompile(`\.\./([\w.-]+)/references/([\w.-]+\.md)`)
)

var skillRoots = []string{
	filepath.Join("internal", "embedded", "default-skills"),
	"skills",
}

func TestSkillReferenceGates(t *testing.T) {
	for _, root := range skillRoots {
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatalf("read skill root %s: %v", root, err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			skillDir := filepath.Join(root, e.Name())
			data, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
			if err != nil {
				continue
			}
			t.Run(skillDir, func(t *testing.T) {
				checkSkillGate(t, skillDir, string(data))
			})
		}
	}
}

func checkSkillGate(t *testing.T, skillDir, body string) {
	for _, m := range ownRefRe.FindAllStringSubmatch(body, -1) {
		p := filepath.Join(skillDir, "references", m[1])
		if _, err := os.Stat(p); err != nil {
			t.Errorf("cites %s but %s is missing", m[0], p)
		}
	}
	for _, m := range sibRefRe.FindAllStringSubmatch(body, -1) {
		p := filepath.Join(skillDir, "..", m[1], "references", m[2])
		if _, err := os.Stat(p); err != nil {
			t.Errorf("cites %s but %s is missing", m[0], p)
		}
	}

	refs, err := os.ReadDir(filepath.Join(skillDir, "references"))
	if err != nil {
		return
	}
	var refFiles []string
	for _, r := range refs {
		if !r.IsDir() && strings.HasSuffix(r.Name(), ".md") {
			refFiles = append(refFiles, r.Name())
		}
	}
	if len(refFiles) == 0 {
		return
	}

	gate := gateBlock(body)
	if gate == "" {
		t.Fatalf("has %d reference file(s) but no %q gate", len(refFiles), gateHeading)
	}
	for _, name := range refFiles {
		if cite := "./references/" + name; !strings.Contains(gate, cite) {
			t.Errorf("reference %s is not listed in the required-reading gate", cite)
		}
	}
}

func gateBlock(body string) string {
	i := strings.Index(body, gateHeading)
	if i < 0 {
		return ""
	}
	var b strings.Builder
	for _, ln := range strings.Split(body[i+len(gateHeading):], "\n") {
		if strings.HasPrefix(ln, "# ") || strings.HasPrefix(ln, "## ") || strings.HasPrefix(ln, "### ") {
			break
		}
		b.WriteString(ln)
		b.WriteByte('\n')
	}
	return b.String()
}
