package mcp

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/tanq16/claudex/internal/fsutil"
)

const projectFile = ".mcp.json"

var validName = regexp.MustCompile(`^[\w.-]+$`)

type Entry struct {
	Name      string            `json:"name"`
	Transport string            `json:"transport"`
	URL       string            `json:"url,omitzero"`
	Headers   map[string]string `json:"headers,omitzero"`
	Command   string            `json:"command,omitzero"`
	Args      []string          `json:"args,omitzero"`
	Env       map[string]string `json:"env,omitzero"`
}

func (e *Entry) Validate() error {
	e.Name = strings.TrimSpace(e.Name)
	if !validName.MatchString(e.Name) {
		return fmt.Errorf("%q is not a valid server name; use letters, digits, dots, dashes, or underscores", e.Name)
	}
	switch e.Transport {
	case "http":
		if !strings.HasPrefix(e.URL, "http://") && !strings.HasPrefix(e.URL, "https://") {
			return fmt.Errorf("%s needs an http or https URL", e.Name)
		}
	case "stdio":
		if strings.TrimSpace(e.Command) == "" {
			return fmt.Errorf("%s needs a command", e.Name)
		}
	default:
		return fmt.Errorf("%s has transport %q; use http or stdio", e.Name, e.Transport)
	}
	e.normalize()
	return nil
}

func (e *Entry) normalize() {
	if e.Transport == "http" {
		e.Command, e.Args, e.Env = "", nil, nil
		if e.Headers == nil {
			e.Headers = map[string]string{}
		}
		return
	}
	e.URL, e.Headers = "", nil
	if e.Args == nil {
		e.Args = []string{}
	}
	if e.Env == nil {
		e.Env = map[string]string{}
	}
}

func (e Entry) config() map[string]any {
	if e.Transport == "http" {
		c := map[string]any{"type": "http", "url": e.URL}
		if len(e.Headers) > 0 {
			c["headers"] = e.Headers
		}
		return c
	}
	c := map[string]any{"type": "stdio", "command": e.Command, "args": e.Args}
	if len(e.Env) > 0 {
		c["env"] = e.Env
	}
	return c
}

func (e Entry) Config() (jsontext.Value, error) {
	return json.Marshal(e.config(), json.Deterministic(true))
}

type library struct {
	Servers []Entry `json:"servers"`
}

func LoadLibrary(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var lib library
	if err := json.Unmarshal(data, &lib); err != nil {
		return nil, fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	for i := range lib.Servers {
		lib.Servers[i].normalize()
	}
	return lib.Servers, nil
}

func SaveLibrary(path string, entries []Entry) error {
	data, err := json.Marshal(library{Servers: entries}, jsontext.WithIndent("  "))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, append(data, '\n'), 0o600)
}

func ProjectServers(dir string) (map[string]jsontext.Value, error) {
	doc, err := readProject(dir)
	if err != nil {
		return nil, err
	}
	servers := map[string]jsontext.Value{}
	if raw, ok := doc["mcpServers"]; ok {
		if err := json.Unmarshal(raw, &servers); err != nil {
			return nil, fmt.Errorf("mcpServers in %s is not an object: %w", projectFile, err)
		}
	}
	return servers, nil
}

func ProjectNames(dir string) []string {
	servers, err := ProjectServers(dir)
	if err != nil {
		return nil
	}
	return slices.Sorted(maps.Keys(servers))
}

func WriteProject(dir string, servers map[string]jsontext.Value) error {
	doc, err := readProject(dir)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(servers, json.Deterministic(true))
	if err != nil {
		return err
	}
	doc["mcpServers"] = encoded
	data, err := json.Marshal(doc, jsontext.WithIndent("  "), json.Deterministic(true))
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(filepath.Join(dir, projectFile), append(data, '\n'), 0o600)
}

func readProject(dir string) (map[string]jsontext.Value, error) {
	doc := map[string]jsontext.Value{}
	data, err := os.ReadFile(filepath.Join(dir, projectFile))
	if errors.Is(err, os.ErrNotExist) {
		return doc, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s is not valid JSON: %w", projectFile, err)
	}
	return doc, nil
}
