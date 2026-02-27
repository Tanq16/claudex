package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SaveInstalledPlugins writes the InstalledPluginsFile back to disk.
func SaveInstalledPlugins(configDir string, file InstalledPluginsFile) error {
	p := filepath.Join(configDir, "plugins", "installed_plugins.json")
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// SaveSettingsLocal creates .claude/ if needed and merges enabledPlugins into
// the existing settings.local.json, preserving other keys.
func SaveSettingsLocal(projectDir string, plugins map[string]bool) error {
	dir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	existing, err := LoadSettingsLocal(projectDir)
	if err != nil {
		return err
	}

	// Merge new plugins into existing
	for k, v := range plugins {
		existing.EnabledPlugins[k] = v
	}

	// Rebuild full JSON preserving unknown fields
	out := make(map[string]any)
	for k, v := range existing.Rest {
		var parsed any
		if err := json.Unmarshal(v, &parsed); err == nil {
			out[k] = parsed
		}
	}
	out["enabledPlugins"] = existing.EnabledPlugins

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}

	p := filepath.Join(dir, "settings.local.json")
	return os.WriteFile(p, data, 0644)
}

// AddInstallEntry adds or updates an install entry in the InstalledPluginsFile.
// It deduplicates by projectPath — if an entry with the same projectPath already exists, it's replaced.
func AddInstallEntry(file *InstalledPluginsFile, key string, install PluginInstall) {
	existing := file.Plugins[key]
	var updated []PluginInstall
	replaced := false
	for _, e := range existing {
		if e.ProjectPath == install.ProjectPath {
			updated = append(updated, install)
			replaced = true
		} else {
			updated = append(updated, e)
		}
	}
	if !replaced {
		updated = append(updated, install)
	}
	file.Plugins[key] = updated
}

// RemoveInstallEntries removes all entries for a plugin key.
func RemoveInstallEntries(file *InstalledPluginsFile, key string) {
	delete(file.Plugins, key)
}

// RemoveInstallEntryByProject removes the entry matching a specific projectPath.
func RemoveInstallEntryByProject(file *InstalledPluginsFile, key, projectPath string) {
	existing := file.Plugins[key]
	var updated []PluginInstall
	for _, e := range existing {
		if e.ProjectPath != projectPath {
			updated = append(updated, e)
		}
	}
	if len(updated) == 0 {
		delete(file.Plugins, key)
	} else {
		file.Plugins[key] = updated
	}
}
