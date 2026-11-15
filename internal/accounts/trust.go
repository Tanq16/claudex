package accounts

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tanq16/claudex/internal/fsutil"
)

func ClaudeJSON(dir, defaultDir string) string {
	if filepath.Clean(dir) == filepath.Clean(defaultDir) {
		return filepath.Clean(dir) + ".json"
	}
	return filepath.Join(dir, ".claude.json")
}

func Trust(dir, defaultDir, projectDir string) error {
	path := ClaudeJSON(dir, defaultDir)
	mode := os.FileMode(0o600)
	doc := map[string]jsontext.Value{}

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("%s is not valid JSON: %w", path, err)
		}
		if info, err := os.Stat(path); err == nil {
			mode = info.Mode().Perm()
		}
	case !errors.Is(err, os.ErrNotExist):
		return err
	}

	projects := map[string]map[string]jsontext.Value{}
	if raw, ok := doc["projects"]; ok {
		if err := json.Unmarshal(raw, &projects); err != nil {
			return fmt.Errorf("projects in %s is not an object of objects: %w", path, err)
		}
	}
	entry := projects[projectDir]
	if entry == nil {
		entry = map[string]jsontext.Value{}
	}
	entry["hasTrustDialogAccepted"] = jsontext.Value("true")
	projects[projectDir] = entry

	encoded, err := json.Marshal(projects, json.Deterministic(true))
	if err != nil {
		return err
	}
	doc["projects"] = encoded

	out, err := json.Marshal(doc, jsontext.WithIndent("  "), json.Deterministic(true))
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, append(out, '\n'), mode)
}
