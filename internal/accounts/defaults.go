package accounts

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tanq16/claudex/internal/fsutil"
)

const (
	statuslineKey = "statusLine.command"
	ownedPrefix   = "statusLine"
)

var builtin = map[string]any{
	"attribution.commit":                "",
	"attribution.pr":                    "",
	"effortLevel":                       "xhigh",
	"tui":                               "fullscreen",
	"outputStyle":                       "Concise",
	"autoMemoryEnabled":                 false,
	"autoCompactEnabled":                false,
	"idleCompaction":                    false,
	"skipDangerousModePermissionPrompt": true,
	"enableAllProjectMcpServers":        true,
	"enableArtifact":                    false,
	"feedbackDrafts":                    "off",
	"feedbackSurveyRate":                0,
	"disableBundledSkills":              true,
	"syncClaudeAiSkills":                false,
	"syncClaudeAiPlugins":               false,
	"includeGitInstructions":            false,
	"spinnerTipsEnabled":                false,
	"promptSuggestionEnabled":           false,
	"awaySummaryEnabled":                false,
	"env.DISABLE_AUTOUPDATER":           "1",
	"env.ENABLE_CLAUDEAI_MCP_SERVERS":   "false",
}

var owned = map[string]any{
	"statusLine.type":    "command",
	"statusLine.padding": 0,
}

type Defaults struct {
	Settings map[string]any      `json:"settings"`
	Written  map[string][]string `json:"written,omitzero"`
}

func LoadDefaults(path string) (*Defaults, error) {
	d := &Defaults{}
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		d.Settings = maps.Clone(builtin)
	case err != nil:
		return nil, err
	default:
		if err := json.Unmarshal(data, d); err != nil {
			return nil, fmt.Errorf("%s is not valid JSON: %w", path, err)
		}
	}
	if d.Settings == nil {
		d.Settings = map[string]any{}
	}
	if d.Written == nil {
		d.Written = map[string][]string{}
	}
	return d, nil
}

func (d *Defaults) Save(path string) error {
	data, err := json.Marshal(d, jsontext.WithIndent("  "), json.Deterministic(true))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, append(data, '\n'), 0o600)
}

func Owned(key string) bool {
	return key == ownedPrefix || strings.HasPrefix(key, ownedPrefix+".")
}

func OwnedSettings(dir string) map[string]any {
	m := maps.Clone(owned)
	m[statuslineKey] = filepath.Join(dir, "statusline.sh")
	return m
}

func (d *Defaults) Set(key string, value any) error {
	if key == "" || slices.Contains(strings.Split(key, "."), "") {
		return fmt.Errorf("%q is not a settings key", key)
	}
	if Owned(key) {
		return fmt.Errorf("%s is managed by claudex", key)
	}
	if value == nil {
		return fmt.Errorf("%s needs a value", key)
	}
	for other := range d.Settings {
		if other != key && (strings.HasPrefix(other, key+".") || strings.HasPrefix(key, other+".")) {
			return fmt.Errorf("%s overlaps the active key %s; reset that one first", key, other)
		}
	}
	d.Settings[key] = value
	return nil
}

func (d *Defaults) Reset(key string) bool {
	_, ok := d.Settings[key]
	delete(d.Settings, key)
	return ok
}

func (d *Defaults) stale(dir string) []string {
	var keys []string
	for _, key := range d.Written[dir] {
		if _, active := d.Settings[key]; !active && !Owned(key) {
			keys = append(keys, key)
		}
	}
	return keys
}

func (d *Defaults) Manages(key string) bool {
	if Owned(key) {
		return true
	}
	for active := range d.Settings {
		if key == active || strings.HasPrefix(key, active+".") {
			return true
		}
	}
	return false
}

func (d *Defaults) Document() map[string]any {
	doc := map[string]any{}
	for key, value := range d.Settings {
		setPath(doc, key, value)
	}
	return doc
}

func (d *Defaults) Flatten(doc map[string]any, schemaKeys []string) (map[string]any, error) {
	flat := map[string]any{}
	var errs []error
	walk(doc, "", d.splitter(schemaKeys), func(key string, value any) {
		switch {
		case Owned(key):
			errs = append(errs, fmt.Errorf("%s is managed by claudex", key))
		case value == nil:
			errs = append(errs, fmt.Errorf("%s needs a value", key))
		default:
			flat[key] = value
		}
	})
	return flat, errors.Join(errs...)
}

func (d *Defaults) splitter(schemaKeys []string) func(string) bool {
	keys := slices.Concat(schemaKeys, slices.Collect(maps.Keys(d.Settings)))
	return func(path string) bool {
		return slices.ContainsFunc(keys, func(k string) bool { return strings.HasPrefix(k, path+".") })
	}
}

func walk(m map[string]any, prefix string, descend func(string) bool, visit func(string, any)) {
	for _, name := range slices.Sorted(maps.Keys(m)) {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		if child, ok := m[name].(map[string]any); ok && len(child) > 0 && descend(path) {
			walk(child, path, descend, visit)
			continue
		}
		visit(path, m[name])
	}
}

type Drift struct {
	Key      string `json:"key"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
}

type Unmanaged struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Report struct {
	Found     bool
	Drift     []Drift
	Unmanaged []Unmanaged
}

func readSettings(dir string) (map[string]any, error) {
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		return nil, err
	}
	settings := map[string]any{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func Audit(dir string, d *Defaults, schemaKeys []string) Report {
	settings, err := readSettings(dir)
	if errors.Is(err, os.ErrNotExist) {
		return Report{}
	}
	if err != nil {
		return Report{Found: true, Drift: []Drift{{Key: "settings.json", Expected: "readable JSON", Actual: err.Error()}}}
	}
	return Report{Found: true, Drift: drift(dir, settings, d), Unmanaged: d.unmanaged(settings, schemaKeys)}
}

func (d *Defaults) unmanaged(settings map[string]any, schemaKeys []string) []Unmanaged {
	split := d.splitter(schemaKeys)
	var found []Unmanaged
	walk(settings, "", func(path string) bool { return !d.Manages(path) && split(path) }, func(key string, value any) {
		if !d.Manages(key) {
			found = append(found, Unmanaged{Key: key, Value: display(value)})
		}
	})
	return found
}

func Lookup(dir, key string) (any, error) {
	settings, err := readSettings(dir)
	if err != nil {
		return nil, err
	}
	value, ok := getPath(settings, key)
	if !ok {
		return nil, fmt.Errorf("%s is not set in %s", key, filepath.Join(dir, "settings.json"))
	}
	return value, nil
}

func RemoveKey(dir, key string) error {
	settings, err := readSettings(dir)
	if err != nil {
		return err
	}
	if _, ok := getPath(settings, key); !ok {
		return fmt.Errorf("%s is not set in %s", key, filepath.Join(dir, "settings.json"))
	}
	deletePath(settings, key)
	out, err := json.Marshal(settings, jsontext.WithIndent("  "))
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(filepath.Join(dir, "settings.json"), append(out, '\n'), 0o644)
}

func drift(dir string, settings map[string]any, d *Defaults) []Drift {
	want := maps.Clone(d.Settings)
	maps.Copy(want, owned)
	var drift []Drift
	for _, key := range slices.Sorted(maps.Keys(want)) {
		got, ok := getPath(settings, key)
		switch {
		case !ok:
			drift = append(drift, Drift{Key: key, Expected: display(want[key]), Actual: "missing"})
		case display(got) != display(want[key]):
			drift = append(drift, Drift{Key: key, Expected: display(want[key]), Actual: display(got)})
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

	for _, key := range d.stale(dir) {
		if got, ok := getPath(settings, key); ok {
			drift = append(drift, Drift{Key: key, Expected: "unset", Actual: display(got)})
		}
	}
	return drift
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

func deletePath(m map[string]any, key string) {
	head, rest, nested := strings.Cut(key, ".")
	if !nested {
		delete(m, key)
		return
	}
	child, ok := m[head].(map[string]any)
	if !ok {
		return
	}
	deletePath(child, rest)
	if len(child) == 0 {
		delete(m, head)
	}
}
