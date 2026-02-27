package plugin

import (
	"fmt"
	"os"
	"path/filepath"
)

// ReconcilePlugin ensures the latest version of a plugin is cached and not orphaned.
//
// Steps:
//  1. Read plugin.json from the marketplace source dir to get latest version
//     (falls back to marketplace metadata.version)
//  2. List cached versions for this plugin
//  3. If latest exists in cache and not orphaned → up-to-date
//  4. If latest exists but orphaned → remove .orphaned_at → un-orphaned
//  5. If latest not in cache and marketplace has local source → copyDir → copied
//  6. If external repo marketplace → skip (can't copy without clone)
func ReconcilePlugin(configDir string, mktEntry MarketplacePluginEntry, mktJSON MarketplaceJSON, marketplaceName string) (ReconcileResult, error) {
	// Resolve marketplace location
	known, err := LoadKnownMarketplaces(configDir)
	if err != nil {
		return ReconcileResult{Action: "skipped", Message: "cannot read known_marketplaces.json"}, err
	}
	mktInfo, ok := known[marketplaceName]
	if !ok {
		return ReconcileResult{Action: "skipped", Message: fmt.Sprintf("marketplace %q not found", marketplaceName)}, nil
	}
	mktDir := mktInfo.InstallLocation

	// Step 1: resolve latest version
	latestVersion := mktJSON.Metadata.Version
	srcPath := mktEntry.SourcePath()
	absSource := ""
	if srcPath != "" {
		absSource = srcPath
		if !filepath.IsAbs(srcPath) {
			absSource = filepath.Join(mktDir, srcPath)
		}
		if pj, err := LoadPluginJSON(absSource); err == nil && pj.Version != "" {
			latestVersion = pj.Version
		}
	}

	if latestVersion == "" {
		return ReconcileResult{Action: "skipped", Message: "cannot determine latest version"}, nil
	}

	// Step 2: list cached versions
	cached, err := ListCachedVersions(configDir, marketplaceName, mktEntry.Name)
	if err != nil {
		return ReconcileResult{Action: "skipped", Message: "cannot list cached versions"}, err
	}

	// Step 3 & 4: check if latest is cached
	for _, cv := range cached {
		if cv.Version == latestVersion {
			if !cv.Orphaned {
				// Step 3: already up-to-date
				return ReconcileResult{
					Action:  "up-to-date",
					Version: latestVersion,
					Message: fmt.Sprintf("version %s already cached and active", latestVersion),
				}, nil
			}
			// Step 4: un-orphan it
			orphanFile := filepath.Join(cv.Path, ".orphaned_at")
			if err := os.Remove(orphanFile); err != nil {
				return ReconcileResult{Action: "skipped", Message: "failed to remove .orphaned_at"}, err
			}
			return ReconcileResult{
				Action:  "un-orphaned",
				Version: latestVersion,
				Message: fmt.Sprintf("version %s un-orphaned", latestVersion),
			}, nil
		}
	}

	// Step 5: copy from marketplace source if local
	if absSource != "" {
		destDir := filepath.Join(configDir, "plugins", "cache", marketplaceName, mktEntry.Name, latestVersion)
		if err := copyDir(absSource, destDir); err != nil {
			return ReconcileResult{Action: "skipped", Message: "failed to copy from marketplace"}, err
		}
		return ReconcileResult{
			Action:  "copied-from-marketplace",
			Version: latestVersion,
			Message: fmt.Sprintf("version %s copied from marketplace source", latestVersion),
		}, nil
	}

	// Step 6: external source, can't auto-copy
	return ReconcileResult{
		Action:  "skipped",
		Version: latestVersion,
		Message: "external source — manual install required",
	}, nil
}
