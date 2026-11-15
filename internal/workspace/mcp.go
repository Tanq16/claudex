package workspace

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/tanq16/claudex/internal/mcp"
)

const PresetMCPName = "mcp.json"

func (p Preset) MCPServers() (map[string]jsontext.Value, error) {
	path := filepath.Join(p.Dir, PresetMCPName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var doc struct {
		MCPServers map[string]jsontext.Value `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	for name, v := range doc.MCPServers {
		if v.Kind() != '{' {
			return nil, fmt.Errorf("mcpServers.%s in %s is not an object", name, path)
		}
	}
	return doc.MCPServers, nil
}

func MergePresetMCP(root string, p *Preset) ([]string, error) {
	servers, err := p.MCPServers()
	if err != nil || len(servers) == 0 {
		return nil, err
	}
	project, err := mcp.ProjectServers(root)
	if err != nil {
		return nil, err
	}
	maps.Copy(project, servers)
	if err := mcp.WriteProject(root, project); err != nil {
		return nil, err
	}
	if _, _, err := WriteGitExclude(root); err != nil {
		return nil, err
	}
	return slices.Sorted(maps.Keys(servers)), nil
}

func removePresetMCP(root string, p *Preset) error {
	servers, err := p.MCPServers()
	if err != nil || len(servers) == 0 {
		return err
	}
	project, err := mcp.ProjectServers(root)
	if err != nil {
		return err
	}
	removed := false
	for name, v := range servers {
		if current, ok := project[name]; ok && sameJSON(current, v) {
			delete(project, name)
			removed = true
		}
	}
	if !removed {
		return nil
	}
	return mcp.WriteProject(root, project)
}

func sameJSON(a, b jsontext.Value) bool {
	a, b = a.Clone(), b.Clone()
	if a.Canonicalize() != nil || b.Canonicalize() != nil {
		return false
	}
	return bytes.Equal(a, b)
}
