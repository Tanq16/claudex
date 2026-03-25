package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func SaveInstalledPlugins(configDir string, file InstalledPluginsFile) error {
	p := filepath.Join(configDir, "plugins", "installed_plugins.json")
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

func SaveSettingsLocal(projectDir string, plugins map[string]bool) error {
	dir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	existing, err := LoadSettingsLocal(projectDir)
	if err != nil {
		return err
	}

	for k, v := range plugins {
		existing.EnabledPlugins[k] = v
	}

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

func RemoveInstallEntries(file *InstalledPluginsFile, key string) {
	delete(file.Plugins, key)
}

func PruneStaleInstallEntries(file *InstalledPluginsFile) int {
	pruned := 0
	for key, installs := range file.Plugins {
		var valid []PluginInstall
		for _, e := range installs {
			if _, err := os.Stat(e.InstallPath); err == nil {
				valid = append(valid, e)
			} else {
				pruned++
			}
		}
		if len(valid) == 0 {
			delete(file.Plugins, key)
		} else {
			file.Plugins[key] = valid
		}
	}
	return pruned
}

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
