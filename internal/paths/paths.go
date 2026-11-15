package paths

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func Expand(home, path string) string {
	if len(path) > 0 && path[0] == '~' {
		return filepath.Join(home, path[1:])
	}
	return path
}

func Abbreviate(home, path string) string {
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func DefaultAccount(home string) string {
	return filepath.Join(home, ".claude")
}

func DiscoverAccounts(home string) []string {
	entries, err := os.ReadDir(home)
	if err != nil {
		return []string{DefaultAccount(home)}
	}

	var found []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() {
			continue
		}
		if name == ".claude" || numberedAccount(name) {
			found = append(found, filepath.Join(home, name))
		}
	}

	if len(found) == 0 {
		return []string{DefaultAccount(home)}
	}

	slices.Sort(found)
	return found
}

func numberedAccount(name string) bool {
	suffix, ok := strings.CutPrefix(name, ".claude")
	if !ok || suffix == "" {
		return false
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func ConfigDir(home string) string {
	return filepath.Join(home, ".config", "claudex")
}

func GlobalPluginDir(home string) string {
	return filepath.Join(ConfigDir(home), "global")
}

func PresetsDir(home string) string {
	return filepath.Join(ConfigDir(home), "presets")
}

func RemotePresetsDir(home string) string {
	return filepath.Join(ConfigDir(home), "remote-presets")
}

func StateFile(home string) string {
	return filepath.Join(ConfigDir(home), "state.json")
}

func MCPLibraryFile(home string) string {
	return filepath.Join(ConfigDir(home), "mcp-library.json")
}

func DefaultsFile(home string) string {
	return filepath.Join(ConfigDir(home), "defaults.json")
}

func SettingsSchemaFile(home string) string {
	return filepath.Join(ConfigDir(home), "settings-schema.json")
}

func ProjectsDir(home string) string {
	return filepath.Join(ConfigDir(home), "projects")
}

func ResumeStash(home, projectDir string) string {
	sum := sha256.Sum256([]byte(projectDir))
	return filepath.Join(ConfigDir(home), "resume-stash", filepath.Base(projectDir)+"-"+hex.EncodeToString(sum[:6])+".md")
}
