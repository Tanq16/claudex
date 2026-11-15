package accounts

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tanq16/claudex/internal/fsutil"
)

type Paths struct {
	Statusline string
	Settings   string
}

func Configure(dir, label string, statusline []byte) (Paths, error) {
	var paths Paths

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return paths, fmt.Errorf("account config dir not found: %s", dir)
	}

	paths.Statusline = filepath.Join(dir, "statusline.sh")
	if err := os.WriteFile(paths.Statusline, statusline, 0o755); err != nil {
		return paths, fmt.Errorf("write statusline script: %w", err)
	}

	paths.Settings = filepath.Join(dir, "settings.json")
	settings := map[string]any{}
	if data, err := os.ReadFile(paths.Settings); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return paths, fmt.Errorf("existing %s is not valid JSON; refusing to overwrite: %w", paths.Settings, err)
		}
	} else if !os.IsNotExist(err) {
		return paths, fmt.Errorf("read settings.json: %w", err)
	}

	for key, value := range defaults {
		setPath(settings, key, value)
	}
	command := paths.Statusline
	if label != "" {
		command += " " + shellQuote(label)
	}
	setPath(settings, statuslineKey, command)

	out, err := json.Marshal(settings, jsontext.WithIndent("  "))
	if err != nil {
		return paths, fmt.Errorf("encode settings.json: %w", err)
	}
	out = append(out, '\n')
	if err := fsutil.WriteFileAtomic(paths.Settings, out, 0o644); err != nil {
		return paths, fmt.Errorf("write settings.json: %w", err)
	}
	return paths, nil
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n\"'\\$`&|;<>()*?[]{}~#!") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func Env(dir, defaultDir string) []string {
	base := os.Environ()
	// An inherited CLAUDE_CONFIG_DIR would override the chosen account, so it is stripped.
	env := make([]string, 0, len(base)+1)
	for _, e := range base {
		if !strings.HasPrefix(e, "CLAUDE_CONFIG_DIR=") {
			env = append(env, e)
		}
	}
	if dir != defaultDir {
		env = append(env, "CLAUDE_CONFIG_DIR="+dir)
	}
	return env
}
