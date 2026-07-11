package flavors

import (
	"cmp"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const defaultName = "default"

type Flavor struct {
	Name string
	Body string
}

type Options struct {
	Auto    *Flavor
	Choices []Flavor
}

func Load(dir string) (*Options, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return &Options{}, nil
		}
		return nil, err
	}

	var def *Flavor
	var others []Flavor
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(string(body)) == "" {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		f := Flavor{Name: name, Body: string(body)}
		if name == defaultName {
			def = &f
			continue
		}
		others = append(others, f)
	}

	slices.SortFunc(others, func(a, b Flavor) int { return cmp.Compare(a.Name, b.Name) })

	if def != nil && len(others) == 0 {
		return &Options{Auto: def}, nil
	}
	var choices []Flavor
	if def != nil {
		choices = append(choices, *def)
	}
	choices = append(choices, others...)
	return &Options{Choices: choices}, nil
}
