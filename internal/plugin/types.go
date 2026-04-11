package plugin

import (
	"encoding/json"
	"strings"
)

type InstalledPluginsFile struct {
	Version int                       `json:"version"`
	Plugins map[string][]PluginInstall `json:"plugins"`
}

type PluginInstall struct {
	Scope        string `json:"scope"`
	ProjectPath  string `json:"projectPath"`
	InstallPath  string `json:"installPath"`
	Version      string `json:"version"`
	InstalledAt  string `json:"installedAt"`
	LastUpdated  string `json:"lastUpdated"`
	GitCommitSha string `json:"gitCommitSha,omitempty"`
}

type KnownMarketplacesFile map[string]MarketplaceEntry

type MarketplaceEntry struct {
	Source          json.RawMessage `json:"source"`
	InstallLocation string          `json:"installLocation"`
	LastUpdated     string          `json:"lastUpdated"`
}

type MarketplaceJSON struct {
	Name     string `json:"name"`
	Metadata struct {
		Version    string `json:"version"`
		PluginRoot string `json:"pluginRoot"`
	} `json:"metadata"`
	Plugins []MarketplacePluginEntry `json:"plugins"`
}

type MarketplacePluginEntry struct {
	Name        string          `json:"name"`
	Source      json.RawMessage `json:"source"`
	Description string          `json:"description"`
}

func (e MarketplacePluginEntry) SourcePath() string {
	var s string
	if err := json.Unmarshal(e.Source, &s); err == nil {
		return s
	}
	var obj map[string]string
	if err := json.Unmarshal(e.Source, &obj); err == nil {
		if p, ok := obj["path"]; ok {
			return p
		}
	}
	return ""
}

func (e MarketplacePluginEntry) GitHubRepo() (string, bool) {
	var obj map[string]string
	if err := json.Unmarshal(e.Source, &obj); err != nil {
		return "", false
	}
	if obj["source"] != "github" {
		return "", false
	}
	repo, ok := obj["repo"]
	return repo, ok && repo != ""
}

type PluginJSON struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type SettingsLocalJSON struct {
	EnabledPlugins map[string]bool            `json:"enabledPlugins,omitempty"`
	Rest           map[string]json.RawMessage `json:"-"`
}

type CachedVersion struct {
	Version string
	Path    string
}

type PluginSummary struct {
	Key              string
	PluginName       string
	MarketplaceName  string
	Description      string
	InstalledVersion string
	LatestVersion    string
	CachedVersions []CachedVersion
	MktEntry       MarketplacePluginEntry
	MktJSON          MarketplaceJSON
}

func PluginKey(pluginName, marketplaceName string) string {
	return pluginName + "@" + marketplaceName
}

func SplitPluginKey(key string) (pluginName, marketplaceName string) {
	parts := strings.SplitN(key, "@", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return key, ""
}

type ReconcileResult struct {
	Action  string
	Version string
	Message string
}
