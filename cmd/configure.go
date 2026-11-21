package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/accounts"
	"github.com/tanq16/claudex/internal/embedded"
	"github.com/tanq16/claudex/internal/plugins"
	"github.com/tanq16/claudex/internal/schema"
	u "github.com/tanq16/claudex/utils"
)

var configureFlags struct {
	account string
	label   string
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Provision all accounts (settings + statusline) and lay down the global defaults: language servers and presets",
	Args:  cobra.NoArgs,
	Run:   runConfigure,
}

func runConfigure(cmd *cobra.Command, args []string) {
	if configureFlags.label != "" && configureFlags.account == "" {
		u.PrintFatal("--label only applies with -A; without it, labels are auto-derived per account", nil)
	}

	defaultsFile := u.DefaultsFile()
	defaults, err := accounts.LoadDefaults(defaultsFile)
	if err != nil {
		u.PrintFatal("failed to read the defaults", err)
	}
	catalog, _ := schema.Load(u.SettingsSchemaFile())
	var schemaKeys []string
	for _, e := range catalog.Entries {
		schemaKeys = append(schemaKeys, e.Key)
	}

	if configureFlags.account != "" {
		dir := u.ResolveConfigDir(configureFlags.account)
		paths, err := accounts.Configure(dir, configureFlags.label, embedded.StatuslineScript, defaults)
		if err != nil {
			u.PrintFatal("failed to configure account", err)
		}
		reportAccount(dir, configureFlags.label, paths)
		reportUnmanaged(dir, defaults, schemaKeys)
	} else {
		configured := 0
		for _, dir := range u.DiscoverAccountPaths() {
			paths, err := accounts.Configure(dir, "", embedded.StatuslineScript, defaults)
			if err != nil {
				u.PrintWarn("Skipped "+u.AbbreviatePath(dir), err)
				continue
			}
			reportAccount(dir, "", paths)
			reportUnmanaged(dir, defaults, schemaKeys)
			configured++
		}
		if configured == 0 {
			u.PrintWarn("No accounts configured; laying down global defaults only", nil)
		}
	}

	if err := defaults.Save(defaultsFile); err != nil {
		u.PrintFatal("failed to save "+u.AbbreviatePath(defaultsFile), err)
	}
	u.PrintSuccess(fmt.Sprintf("Applied %d default keys from %s", len(defaults.Settings), u.AbbreviatePath(defaultsFile)))

	applyGlobalDefaults()
}

func reportAccount(dir, label string, paths accounts.Paths) {
	if label == "" {
		label = "(auto)"
	}
	u.PrintSuccess(fmt.Sprintf("Configured %s (label: %s)", u.AbbreviatePath(dir), label))
	u.PrintGeneric("  statusline: " + paths.Statusline)
	u.PrintGeneric("  settings:   " + paths.Settings)
}

func reportUnmanaged(dir string, defaults *accounts.Defaults, schemaKeys []string) {
	unmanaged := accounts.Audit(dir, defaults, schemaKeys).Unmanaged
	if len(unmanaged) == 0 {
		return
	}
	keys := make([]string, len(unmanaged))
	for i, k := range unmanaged {
		keys[i] = k.Key
	}
	u.PrintIndentedWarn(fmt.Sprintf("%d keys not managed by claudex: %s", len(keys), strings.Join(keys, ", ")), nil)
}

func applyGlobalDefaults() {
	globalDir := u.GlobalPluginDir()
	if err := plugins.BuildGlobalPlugin(globalDir); err != nil {
		u.PrintFatal("failed to build the global plugin", err)
	}
	plugins.PruneGlobal(globalDir, embedded.DefaultSkillsFS, "default-skills")

	u.PrintSuccess("Refreshed global defaults")
	u.PrintGeneric("  plugin:  " + u.AbbreviatePath(globalDir) + " (language servers)")
	u.PrintGeneric("  presets: " + u.AbbreviatePath(presetsDir()))
}

func init() {
	configureCmd.Flags().StringVarP(&configureFlags.account, "account", "A", "", "Account config dir to configure (default ~/.claude)")
	configureCmd.Flags().StringVarP(&configureFlags.label, "label", "l", "", "Override the account label shown in the statusline; requires -A (errors without a single-account target)")
}
