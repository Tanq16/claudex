package plugin

import (
	"encoding/json"
	"strings"
)

// InstalledPluginsFile maps to ~/.claude{N}/plugins/installed_plugins.json
type InstalledPluginsFile struct {
	Version int                       `json:"version"`
	Plugins map[string][]PluginInstall `json:"plugins"`
}

// PluginInstall is a single install entry keyed by "plugin@marketplace"
type PluginInstall struct {
	Scope        string `json:"scope"`
	ProjectPath  string `json:"projectPath"`
	InstallPath  string `json:"installPath"`
	Version      string `json:"version"`
	InstalledAt  string `json:"installedAt"`
	LastUpdated  string `json:"lastUpdated"`
	GitCommitSha string `json:"gitCommitSha,omitempty"`
}

// KnownMarketplacesFile is the top-level known_marketplaces.json — a flat map keyed by name.
// Parsed manually (not via struct tags) since the file is a plain JSON object.
type KnownMarketplacesFile map[string]MarketplaceEntry

// MarketplaceEntry is a single marketplace in known_marketplaces.json
type MarketplaceEntry struct {
	Source          json.RawMessage `json:"source"`
	InstallLocation string          `json:"installLocation"`
	LastUpdated     string          `json:"lastUpdated"`
}

// MarketplaceJSON maps to .claude-plugin/marketplace.json inside a marketplace repo
type MarketplaceJSON struct {
	Name     string `json:"name"`
	Metadata struct {
		Version    string `json:"version"`
		PluginRoot string `json:"pluginRoot"`
	} `json:"metadata"`
	Plugins []MarketplacePluginEntry `json:"plugins"`
}

// MarketplacePluginEntry is a single plugin listed in marketplace.json
type MarketplacePluginEntry struct {
	Name        string          `json:"name"`
	Source      json.RawMessage `json:"source"`
	Description string          `json:"description"`
}

// SourcePath returns the local relative path from the source field.
// Source can be a string ("./plugins/core") or an object ({"source":"github","repo":"..."}).
// Returns empty string for non-local sources.
func (e MarketplacePluginEntry) SourcePath() string {
	// Try as plain string first
	var s string
	if err := json.Unmarshal(e.Source, &s); err == nil {
		return s
	}
	// Try as object with a "path" or "source" field
	var obj map[string]string
	if err := json.Unmarshal(e.Source, &obj); err == nil {
		if p, ok := obj["path"]; ok {
			return p
		}
	}
	return ""
}

// PluginJSON maps to .claude-plugin/plugin.json inside a cached plugin directory
type PluginJSON struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// SettingsLocalJSON maps to .claude/settings.local.json in a project directory
type SettingsLocalJSON struct {
	EnabledPlugins map[string]bool            `json:"enabledPlugins,omitempty"`
	Rest           map[string]json.RawMessage `json:"-"`
}

// CachedVersion describes a version directory in the plugin cache
type CachedVersion struct {
	Version  string
	Path     string
	Orphaned bool
}

// PluginSummary is a display-friendly summary for interactive selection
type PluginSummary struct {
	Key             string // "plugin@marketplace" (e.g. "core@ai-brain")
	PluginName      string
	MarketplaceName string
	Description     string
	InstalledVersion string
	LatestVersion    string
	CachedVersions  []CachedVersion
	OrphanCount     int
	MktEntry        MarketplacePluginEntry
	MktJSON         MarketplaceJSON
}

// PluginKey builds the standard "plugin@marketplace" key
func PluginKey(pluginName, marketplaceName string) string {
	return pluginName + "@" + marketplaceName
}

// SplitPluginKey splits "plugin@marketplace" into its parts
func SplitPluginKey(key string) (pluginName, marketplaceName string) {
	parts := strings.SplitN(key, "@", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return key, ""
}

// ReconcileResult describes what action was taken during version reconciliation
type ReconcileResult struct {
	Action  string // "up-to-date", "un-orphaned", "copied-from-marketplace", "skipped"
	Version string
	Message string
}
