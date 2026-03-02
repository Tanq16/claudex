package plugin

import (
	"fmt"
	"os"
	"path/filepath"
)

func ReconcilePlugin(configDir string, mktEntry MarketplacePluginEntry, mktJSON MarketplaceJSON, marketplaceName string) (ReconcileResult, error) {
	known, err := LoadKnownMarketplaces(configDir)
	if err != nil {
		return ReconcileResult{Action: "skipped", Message: "cannot read known_marketplaces.json"}, err
	}
	mktInfo, ok := known[marketplaceName]
	if !ok {
		return ReconcileResult{Action: "skipped", Message: fmt.Sprintf("marketplace %q not found", marketplaceName)}, nil
	}
	mktDir := mktInfo.InstallLocation

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

	cached, err := ListCachedVersions(configDir, marketplaceName, mktEntry.Name)
	if err != nil {
		return ReconcileResult{Action: "skipped", Message: "cannot list cached versions"}, err
	}

	for _, cv := range cached {
		if cv.Version == latestVersion {
			if !cv.Orphaned {
				return ReconcileResult{
					Action:  "up-to-date",
					Version: latestVersion,
					Message: fmt.Sprintf("version %s already cached and active", latestVersion),
				}, nil
			}
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

	return ReconcileResult{
		Action:  "skipped",
		Version: latestVersion,
		Message: "external source — manual install required",
	}, nil
}
