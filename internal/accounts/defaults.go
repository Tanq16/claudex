package accounts

import (
	"encoding/json/v2"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const statuslineKey = "statusLine.command"

var defaults = map[string]any{
	"attribution.commit":                "",
	"effortLevel":                       "xhigh",
	"tui":                               "fullscreen",
	"autoMemoryEnabled":                 false,
	"skipDangerousModePermissionPrompt": true,
	"outputStyle":                       "Concise",
	"enableAllProjectMcpServers":        true,
	"env.DISABLE_AUTOUPDATER":           "1",
	"env.ENABLE_CLAUDEAI_MCP_SERVERS":   "false",
	"statusLine.type":                   "command",
	"statusLine.padding":                0,
}

func Defaults() map[string]any {
	return maps.Clone(defaults)
}

type Drift struct {
	Key      string `json:"key"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
}

func Compliance(dir string) (bool, []Drift) {
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	settings := map[string]any{}
	if err == nil {
		err = json.Unmarshal(data, &settings)
	}
	if err != nil {
		return true, []Drift{{Key: "settings.json", Expected: "readable JSON", Actual: err.Error()}}
	}

	var drift []Drift
	for _, key := range slices.Sorted(maps.Keys(defaults)) {
		want := defaults[key]
		got, ok := getPath(settings, key)
		if !ok {
			drift = append(drift, Drift{Key: key, Expected: display(want), Actual: "missing"})
			continue
		}
		if display(got) != display(want) {
			drift = append(drift, Drift{Key: key, Expected: display(want), Actual: display(got)})
		}
	}

	script := filepath.Join(dir, "statusline.sh")
	command, ok := getPath(settings, statuslineKey)
	switch s, isString := command.(string); {
	case !ok:
		drift = append(drift, Drift{Key: statuslineKey, Expected: script, Actual: "missing"})
	case !isString || (s != script && !strings.HasPrefix(s, script+" ")):
		drift = append(drift, Drift{Key: statuslineKey, Expected: script, Actual: display(command)})
	}
	return true, drift
}

func display(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	data, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		return "unreadable"
	}
	return string(data)
}

func setPath(m map[string]any, key string, value any) {
	head, rest, nested := strings.Cut(key, ".")
	if !nested {
		m[key] = value
		return
	}
	child, ok := m[head].(map[string]any)
	if !ok {
		child = map[string]any{}
		m[head] = child
	}
	setPath(child, rest, value)
}

func getPath(m map[string]any, key string) (any, bool) {
	head, rest, nested := strings.Cut(key, ".")
	v, ok := m[head]
	if !ok || !nested {
		return v, ok
	}
	child, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	return getPath(child, rest)
}
