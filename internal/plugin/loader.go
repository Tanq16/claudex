package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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

func LoadSettingsLocal(projectDir string) (SettingsLocalJSON, error) {
	p := filepath.Join(projectDir, ".claude", "settings.local.json")
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return SettingsLocalJSON{EnabledPlugins: make(map[string]bool)}, nil
		}
		return SettingsLocalJSON{}, err
	}

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
		versions = append(versions, CachedVersion{
			Version: e.Name(),
			Path:    vDir,
		})
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

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
			if latestVersion == "" {
				if _, ok := pe.GitHubRepo(); ok {
					latestVersion = "external"
				}
			}

			installedVersion := ""
			if installs, ok := installed.Plugins[key]; ok && len(installs) > 0 {
				installedVersion = installs[0].Version
			}

			cached, _ := ListCachedVersions(configDir, mktName, pe.Name)

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
				MktEntry:         pe,
				MktJSON:          mktJSON,
			})
		}
	}

	// Discover cached plugins not listed in any marketplace (delisted plugins)
	knownKeys := make(map[string]bool)
	for _, s := range summaries {
		knownKeys[s.Key] = true
	}
	cacheRoot := filepath.Join(configDir, "plugins", "cache")
	if mktDirs, err := os.ReadDir(cacheRoot); err == nil {
		for _, mktDir := range mktDirs {
			if !mktDir.IsDir() {
				continue
			}
			mktName := mktDir.Name()
			pluginDirs, err := os.ReadDir(filepath.Join(cacheRoot, mktName))
			if err != nil {
				continue
			}
			for _, plDir := range pluginDirs {
				if !plDir.IsDir() {
					continue
				}
				key := PluginKey(plDir.Name(), mktName)
				if knownKeys[key] {
					continue
				}
				cached, _ := ListCachedVersions(configDir, mktName, plDir.Name())
				installedVersion := ""
				if installs, ok := installed.Plugins[key]; ok && len(installs) > 0 {
					installedVersion = installs[0].Version
				}
				summaries = append(summaries, PluginSummary{
					Key:              key,
					PluginName:       plDir.Name(),
					MarketplaceName:  mktName,
					Description:      "(delisted from marketplace)",
					InstalledVersion: installedVersion,
					LatestVersion:    "",
					CachedVersions:   cached,
				})
			}
		}
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Key < summaries[j].Key
	})
	return summaries, nil
}
