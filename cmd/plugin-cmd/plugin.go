package pluginCmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/plugin"
	u "github.com/tanq16/claudex/utils"
)

type pluginResult struct {
	key     string
	action  string
	message string
	err     error
}

var instateFlags struct {
	configDir string
	plugins   string
	all       bool
	update    bool
}

var cleanupFlags struct {
	configDir   string
	plugins     string
	orphansOnly bool
	all         bool
}

var PluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage Claude Code plugins (instate, cleanup)",
}

var instateCmd = &cobra.Command{
	Use:   "instate",
	Short: "Instantiate plugins for a local project with version reconciliation",
	Run:   runInstate,
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up orphaned or stale plugin cache entries",
	Run:   runCleanup,
}

func runInstate(cmd *cobra.Command, args []string) {
	configDir := u.ResolveConfigDir(instateFlags.configDir)
	projectDir, err := os.Getwd()
	if err != nil {
		u.PrintFatal("Cannot determine current directory", err)
	}

	if instateFlags.update {
		known, err := plugin.LoadKnownMarketplaces(configDir)
		if err != nil {
			u.PrintFatal("Failed to load known_marketplaces.json", err)
		}

		toPull := known
		if instateFlags.plugins != "" {
			relevantMkts := make(map[string]bool)
			for _, k := range strings.Split(instateFlags.plugins, ",") {
				_, mktName := plugin.SplitPluginKey(strings.TrimSpace(k))
				if mktName != "" {
					relevantMkts[mktName] = true
				}
			}
			if len(relevantMkts) > 0 {
				toPull = make(plugin.KnownMarketplacesFile)
				for name, entry := range known {
					if relevantMkts[name] {
						toPull[name] = entry
					}
				}
			}
		}

		u.PrintRunning("(Running) Pulling marketplace repos")
		var pullLineCount int
		var pullErrors []pluginResult
		for name, entry := range toPull {
			if err := plugin.GitPull(entry.InstallLocation); err != nil {
				u.PrintIndentedError(name, err)
				pullErrors = append(pullErrors, pluginResult{key: name, err: err})
			} else {
				u.PrintIndentedSuccess(name)
			}
			pullLineCount++
		}
		u.ClearLines(pullLineCount + 1)
		if len(pullErrors) > 0 {
			u.PrintError("Pulling marketplaces: partially completed with errors", nil)
			for _, e := range pullErrors {
				u.PrintIndentedError(e.key, e.err)
			}
		} else {
			u.PrintInfo(fmt.Sprintf("Pulled %d marketplace(s)", len(toPull)))
		}
	}

	summaries, err := plugin.BuildPluginSummaries(configDir)
	if err != nil {
		u.PrintFatal("Failed to build plugin summaries", err)
	}
	if len(summaries) == 0 {
		u.PrintInfo("No plugins found in any marketplace.")
		return
	}

	var selected []plugin.PluginSummary
	if instateFlags.all {
		selected = summaries
	} else if instateFlags.plugins != "" {
		keys := strings.Split(instateFlags.plugins, ",")
		keySet := make(map[string]bool)
		for _, k := range keys {
			keySet[strings.TrimSpace(k)] = true
		}
		for _, s := range summaries {
			if keySet[s.Key] {
				selected = append(selected, s)
			}
		}
		if len(selected) == 0 {
			u.PrintFatal("None of the specified plugins were found in marketplaces", nil)
		}
	} else {
		selected = interactiveSelect(summaries)
	}

	if len(selected) == 0 {
		u.PrintInfo("No plugins selected.")
		return
	}

	installed, _ := plugin.LoadInstalledPlugins(configDir)
	enabledPlugins := make(map[string]bool)

	u.PrintRunning("(Running) Reconciling plugins")
	var lineCount int
	var reconcileErrors []pluginResult

	for _, s := range selected {
		result, err := plugin.ReconcilePlugin(configDir, s.MktEntry, s.MktJSON, s.MarketplaceName)
		if err != nil {
			u.PrintIndentedError(fmt.Sprintf("[%s] reconcile failed", s.Key), err)
			reconcileErrors = append(reconcileErrors, pluginResult{key: s.Key, err: err})
			lineCount++
			continue
		}

		switch result.Action {
		case "skipped":
			u.PrintIndentedWarn(fmt.Sprintf("[%s] %s", s.Key, result.Message), nil)
		case "up-to-date":
			u.PrintIndentedSuccess(fmt.Sprintf("[%s] %s", s.Key, result.Message))
		default:
			u.PrintIndentedSuccess(fmt.Sprintf("[%s] %s — %s", s.Key, result.Action, result.Message))
		}
		lineCount++

		if result.Action == "skipped" {
			continue
		}

		version := result.Version
		installPath := filepath.Join(configDir, "plugins", "cache", s.MarketplaceName, s.PluginName, version)

		known, _ := plugin.LoadKnownMarketplaces(configDir)
		gitSha := ""
		if mktInfo, ok := known[s.MarketplaceName]; ok {
			gitSha = plugin.GetGitCommitSha(mktInfo.InstallLocation)
		}

		now := time.Now().UTC().Format(time.RFC3339)
		entry := plugin.PluginInstall{
			Scope:        "local",
			ProjectPath:  projectDir,
			InstallPath:  installPath,
			Version:      version,
			InstalledAt:  now,
			LastUpdated:  now,
			GitCommitSha: gitSha,
		}

		plugin.AddInstallEntry(&installed, s.Key, entry)
		enabledPlugins[s.Key] = true
	}

	u.ClearLines(lineCount + 1)
	if len(reconcileErrors) > 0 {
		u.PrintError("Reconciliation partially completed with errors", nil)
		for _, e := range reconcileErrors {
			u.PrintIndentedError(e.key, e.err)
		}
	} else {
		u.PrintInfo(fmt.Sprintf("Reconciled %d plugin(s): %d enabled", len(selected), len(enabledPlugins)))
	}

	if err := plugin.SaveInstalledPlugins(configDir, installed); err != nil {
		u.PrintFatal("Failed to save installed_plugins.json", err)
	}

	if len(enabledPlugins) > 0 {
		if err := plugin.SaveSettingsLocal(projectDir, enabledPlugins, true); err != nil {
			u.PrintFatal("Failed to save settings.local.json", err)
		}
	}

	u.PrintSuccess(fmt.Sprintf("Instated %d plugin(s) for project %s", len(enabledPlugins), projectDir))
}

func runCleanup(cmd *cobra.Command, args []string) {
	configDir := u.ResolveConfigDir(cleanupFlags.configDir)

	if cleanupFlags.orphansOnly && cleanupFlags.all {
		u.PrintRunning("Removing all orphans")
		n, err := plugin.RemoveAllOrphans(configDir)
		u.ClearLines(1)
		if err != nil {
			u.PrintFatal("Failed to remove orphans", err)
		}
		installed, _ := plugin.LoadInstalledPlugins(configDir)
		plugin.PruneStaleInstallEntries(&installed)
		if err := plugin.SaveInstalledPlugins(configDir, installed); err != nil {
			u.PrintFatal("Failed to save installed_plugins.json", err)
		}
		u.PrintSuccess(fmt.Sprintf("Removed %d orphaned version(s)", n))
		return
	}

	summaries, err := plugin.BuildPluginSummaries(configDir)
	if err != nil {
		u.PrintFatal("Failed to build plugin summaries", err)
	}

	var selected []plugin.PluginSummary
	if cleanupFlags.all {
		selected = summaries
	} else if cleanupFlags.plugins != "" {
		keys := strings.Split(cleanupFlags.plugins, ",")
		keySet := make(map[string]bool)
		for _, k := range keys {
			keySet[strings.TrimSpace(k)] = true
		}
		for _, s := range summaries {
			if keySet[s.Key] {
				selected = append(selected, s)
			}
		}
	} else {
		selected = interactiveSelect(summaries)
	}

	if len(selected) == 0 {
		u.PrintInfo("No plugins selected.")
		return
	}

	installed, _ := plugin.LoadInstalledPlugins(configDir)
	totalRemoved := 0

	u.PrintRunning("(Running) Cleaning up plugins")
	var lineCount int
	var cleanupErrors []pluginResult

	for _, s := range selected {
		if cleanupFlags.orphansOnly {
			n, err := plugin.RemoveOrphanedVersions(configDir, s.MarketplaceName, s.PluginName)
			if err != nil {
				u.PrintIndentedError(fmt.Sprintf("[%s] orphan cleanup failed", s.Key), err)
				cleanupErrors = append(cleanupErrors, pluginResult{key: s.Key, err: err})
			} else {
				totalRemoved += n
				u.PrintIndentedSuccess(fmt.Sprintf("[%s] removed %d orphan(s)", s.Key, n))
			}
		} else {
			if err := plugin.RemoveCacheDirectory(configDir, s.MarketplaceName, s.PluginName); err != nil {
				u.PrintIndentedError(fmt.Sprintf("[%s] cache removal failed", s.Key), err)
				cleanupErrors = append(cleanupErrors, pluginResult{key: s.Key, err: err})
			} else {
				plugin.RemoveInstallEntries(&installed, s.Key)
				totalRemoved++
				u.PrintIndentedSuccess(fmt.Sprintf("[%s] cache and install entries removed", s.Key))
			}
		}
		lineCount++
	}

	u.ClearLines(lineCount + 1)
	if len(cleanupErrors) > 0 {
		u.PrintError("Cleanup partially completed with errors", nil)
		for _, e := range cleanupErrors {
			u.PrintIndentedError(e.key, e.err)
		}
	} else {
		u.PrintInfo(fmt.Sprintf("Cleaned %d plugin(s)", len(selected)))
	}

	if cleanupFlags.orphansOnly {
		plugin.PruneStaleInstallEntries(&installed)
	}
	if err := plugin.SaveInstalledPlugins(configDir, installed); err != nil {
		u.PrintFatal("Failed to save installed_plugins.json", err)
	}

	u.PrintSuccess(fmt.Sprintf("Cleanup complete — %d item(s) removed", totalRemoved))
}

func interactiveSelect(summaries []plugin.PluginSummary) []plugin.PluginSummary {
	headers := []string{"#", "Plugin", "Marketplace", "Installed", "Latest", "Orphans"}
	rows := make([][]string, len(summaries))
	for i, s := range summaries {
		rows[i] = []string{
			strconv.Itoa(i + 1),
			s.PluginName,
			s.MarketplaceName,
			orDash(s.InstalledVersion),
			orDash(s.LatestVersion),
			strconv.Itoa(s.OrphanCount),
		}
	}
	u.PrintTable(headers, rows)

	input, err := u.PromptInput("\nSelect plugins (comma-separated numbers, or 'all')", "")
	if err != nil {
		u.PrintWarn("Failed to read input", err)
		return nil
	}
	if input == "" {
		return nil
	}

	if strings.EqualFold(input, "all") {
		return summaries
	}

	var selected []plugin.PluginSummary
	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > len(summaries) {
			u.PrintWarn(fmt.Sprintf("Skipping invalid selection: %s", part), nil)
			continue
		}
		selected = append(selected, summaries[n-1])
	}
	return selected
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func init() {
	instateCmd.Flags().StringVarP(&instateFlags.configDir, "config-dir", "c", "", "Claude config directory (default ~/.claude)")
	instateCmd.Flags().StringVarP(&instateFlags.plugins, "plugins", "P", "", "Comma-separated plugin keys (e.g. core@ai-brain,praetorian@ai-brain)")
	instateCmd.Flags().BoolVarP(&instateFlags.all, "all", "A", false, "Instate all available plugins")
	instateCmd.Flags().BoolVarP(&instateFlags.update, "update", "u", false, "Git pull marketplace repos before reconciling")

	cleanupCmd.Flags().StringVarP(&cleanupFlags.configDir, "config-dir", "c", "", "Claude config directory (default ~/.claude)")
	cleanupCmd.Flags().StringVarP(&cleanupFlags.plugins, "plugins", "P", "", "Comma-separated plugin keys")
	cleanupCmd.Flags().BoolVarP(&cleanupFlags.orphansOnly, "orphans-only", "o", false, "Only remove orphaned version dirs")
	cleanupCmd.Flags().BoolVarP(&cleanupFlags.all, "all", "A", false, "Target all plugins")

	PluginCmd.AddCommand(instateCmd, cleanupCmd)
}
