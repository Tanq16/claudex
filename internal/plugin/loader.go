package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadInstalledPlugins reads and parses installed_plugins.json from the config dir.
func LoadInstalledPlugins(configDir string) (InstalledPluginsFile, error) {
	p := filepath.Join(configDir, "plugins", "installed_plugins.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return InstalledPluginsFile{Plugins: make(map[string][]PluginInstall)}, err
	}

	var f InstalledPluginsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return InstalledPluginsFile{Plugins: make(map[string][]PluginInstall)}, err
	}
	if f.Plugins == nil {
		f.Plugins = make(map[string][]PluginInstall)
	}
	return f, nil
}

// LoadKnownMarketplaces reads and parses known_marketplaces.json from the config dir.
// The file is a flat JSON object keyed by marketplace name.
func LoadKnownMarketplaces(configDir string) (KnownMarketplacesFile, error) {
	p := filepath.Join(configDir, "plugins", "known_marketplaces.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	var f KnownMarketplacesFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f, nil
}

// LoadMarketplaceJSON reads and parses .claude-plugin/marketplace.json inside a marketplace dir.
func LoadMarketplaceJSON(marketplaceDir string) (MarketplaceJSON, error) {
	p := filepath.Join(marketplaceDir, ".claude-plugin", "marketplace.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return MarketplaceJSON{}, err
	}

	var m MarketplaceJSON
	if err := json.Unmarshal(data, &m); err != nil {
		return MarketplaceJSON{}, err
	}
	return m, nil
}

// LoadPluginJSON reads and parses .claude-plugin/plugin.json inside a plugin dir.
func LoadPluginJSON(pluginDir string) (PluginJSON, error) {
	p := filepath.Join(pluginDir, ".claude-plugin", "plugin.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return PluginJSON{}, err
	}

	var pj PluginJSON
	if err := json.Unmarshal(data, &pj); err != nil {
		return PluginJSON{}, err
	}
	return pj, nil
}

// LoadSettingsLocal reads and parses .claude/settings.local.json from a project dir.
// Returns an empty struct (not error) if the file doesn't exist yet.
func LoadSettingsLocal(projectDir string) (SettingsLocalJSON, error) {
	p := filepath.Join(projectDir, ".claude", "settings.local.json")
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return SettingsLocalJSON{EnabledPlugins: make(map[string]bool)}, nil
		}
		return SettingsLocalJSON{}, err
	}

	// Parse into a generic map first to preserve unknown fields
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return SettingsLocalJSON{}, err
	}

	result := SettingsLocalJSON{
		EnabledPlugins: make(map[string]bool),
		Rest:           make(map[string]json.RawMessage),
	}

	for k, v := range raw {
		if k == "enabledPlugins" {
			if err := json.Unmarshal(v, &result.EnabledPlugins); err != nil {
				return SettingsLocalJSON{}, err
			}
		} else {
			result.Rest[k] = v
		}
	}

	return result, nil
}

// ListCachedVersions lists version directories for a plugin in the cache.
// Cache layout: {configDir}/plugins/cache/{marketplace}/{plugin}/{version}/
func ListCachedVersions(configDir, marketplace, pluginName string) ([]CachedVersion, error) {
	cacheDir := filepath.Join(configDir, "plugins", "cache", marketplace, pluginName)
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var versions []CachedVersion
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		vDir := filepath.Join(cacheDir, e.Name())
		_, orphanErr := os.Stat(filepath.Join(vDir, ".orphaned_at"))
		versions = append(versions, CachedVersion{
			Version:  e.Name(),
			Path:     vDir,
			Orphaned: orphanErr == nil,
		})
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version // newest first
	})
	return versions, nil
}

// BuildPluginSummaries combines installed plugins, marketplace data, and cache info
// into display-friendly summaries.
func BuildPluginSummaries(configDir string) ([]PluginSummary, error) {
	installed, _ := LoadInstalledPlugins(configDir)
	known, err := LoadKnownMarketplaces(configDir)
	if err != nil {
		return nil, err
	}

	var summaries []PluginSummary

	for mktName, mktEntry := range known {
		mktDir := mktEntry.InstallLocation
		mktJSON, err := LoadMarketplaceJSON(mktDir)
		if err != nil {
			continue
		}

		for _, pe := range mktJSON.Plugins {
			key := PluginKey(pe.Name, mktName)

			// Get latest version from plugin source dir, fall back to marketplace metadata
			latestVersion := mktJSON.Metadata.Version
			srcPath := pe.SourcePath()
			if srcPath != "" {
				absSource := srcPath
				if !filepath.IsAbs(srcPath) {
					absSource = filepath.Join(mktDir, srcPath)
				}
				if pj, err := LoadPluginJSON(absSource); err == nil && pj.Version != "" {
					latestVersion = pj.Version
				}
			}

			// Installed version
			installedVersion := ""
			if installs, ok := installed.Plugins[key]; ok && len(installs) > 0 {
				installedVersion = installs[0].Version
			}

			// Cached versions
			cached, _ := ListCachedVersions(configDir, mktName, pe.Name)
			orphanCount := 0
			for _, cv := range cached {
				if cv.Orphaned {
					orphanCount++
				}
			}

			desc := pe.Description
			if desc == "" {
				desc = strings.TrimSpace(pe.Name)
			}

			summaries = append(summaries, PluginSummary{
				Key:              key,
				PluginName:       pe.Name,
				MarketplaceName:  mktName,
				Description:      desc,
				InstalledVersion: installedVersion,
				LatestVersion:    latestVersion,
				CachedVersions:   cached,
				OrphanCount:      orphanCount,
				MktEntry:         pe,
				MktJSON:          mktJSON,
			})
		}
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Key < summaries[j].Key
	})
	return summaries, nil
}
