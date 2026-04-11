package plugin

import (
	"fmt"
	"os"
	"path/filepath"
)

func ReconcilePlugin(configDir string, mktEntry MarketplacePluginEntry, mktJSON MarketplaceJSON, marketplaceName string, update bool) (ReconcileResult, error) {
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

	if repo, ok := mktEntry.GitHubRepo(); ok && absSource == "" {
		return reconcileFromGitHub(configDir, mktEntry.Name, marketplaceName, repo, latestVersion, update)
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
			if update {
				removeOldCachedVersions(configDir, marketplaceName, mktEntry.Name, latestVersion)
			}
			return ReconcileResult{
				Action:  "up-to-date",
				Version: latestVersion,
				Message: fmt.Sprintf("version %s already cached", latestVersion),
			}, nil
		}
	}

	if absSource != "" {
		destDir := filepath.Join(configDir, "plugins", "cache", marketplaceName, mktEntry.Name, latestVersion)
		if err := copyDir(absSource, destDir); err != nil {
			return ReconcileResult{Action: "skipped", Message: "failed to copy from marketplace"}, err
		}
		if update {
			removeOldCachedVersions(configDir, marketplaceName, mktEntry.Name, latestVersion)
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

func reconcileFromGitHub(configDir, pluginName, marketplaceName, repo, fallbackVersion string, update bool) (ReconcileResult, error) {
	tmpDir, err := CloneGitHubRepo(repo)
	if err != nil {
		return ReconcileResult{Action: "skipped", Message: fmt.Sprintf("clone failed: %v", err)}, nil
	}
	defer os.RemoveAll(tmpDir)

	latestVersion := fallbackVersion
	if pj, err := LoadPluginJSON(tmpDir); err == nil && pj.Version != "" {
		latestVersion = pj.Version
	}
	if latestVersion == "" {
		sha := GetGitCommitSha(tmpDir)
		if sha == "" {
			return ReconcileResult{Action: "skipped", Message: "cannot determine version from cloned repo"}, nil
		}
		latestVersion = sha[:12]
	}

	cached, err := ListCachedVersions(configDir, marketplaceName, pluginName)
	if err != nil {
		return ReconcileResult{Action: "skipped", Message: "cannot list cached versions"}, err
	}

	for _, cv := range cached {
		if cv.Version == latestVersion {
			if update {
				removeOldCachedVersions(configDir, marketplaceName, pluginName, latestVersion)
			}
			return ReconcileResult{
				Action:  "up-to-date",
				Version: latestVersion,
				Message: fmt.Sprintf("version %s already cached", latestVersion),
			}, nil
		}
	}

	destDir := filepath.Join(configDir, "plugins", "cache", marketplaceName, pluginName, latestVersion)
	if err := copyDir(tmpDir, destDir); err != nil {
		return ReconcileResult{Action: "skipped", Message: "failed to copy cloned repo to cache"}, err
	}
	if update {
		removeOldCachedVersions(configDir, marketplaceName, pluginName, latestVersion)
	}
	return ReconcileResult{
		Action:  "cloned-from-github",
		Version: latestVersion,
		Message: fmt.Sprintf("version %s cloned from github.com/%s", latestVersion, repo),
	}, nil
}

func removeOldCachedVersions(configDir, marketplace, pluginName, currentVersion string) {
	cached, err := ListCachedVersions(configDir, marketplace, pluginName)
	if err != nil {
		return
	}
	for _, cv := range cached {
		if cv.Version == currentVersion {
			continue
		}
		_ = os.RemoveAll(cv.Path)
	}
}
