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

	applyPreferred(settings)

	command := paths.Statusline
	if label != "" {
		command += " " + shellQuote(label)
	}
	settings["statusLine"] = map[string]any{
		"type":    "command",
		"command": command,
		"padding": 0,
	}

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

func applyPreferred(settings map[string]any) {
	settings["attribution"] = map[string]any{"commit": ""}
	settings["effortLevel"] = "xhigh"
	settings["tui"] = "fullscreen"
	settings["autoMemoryEnabled"] = false
	settings["skipDangerousModePermissionPrompt"] = true
	settings["outputStyle"] = "Concise"

	env, ok := settings["env"].(map[string]any)
	if !ok {
		env = map[string]any{}
	}
	env["DISABLE_AUTOUPDATER"] = "1"
	env["ENABLE_CLAUDEAI_MCP_SERVERS"] = "false"
	settings["env"] = env
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
