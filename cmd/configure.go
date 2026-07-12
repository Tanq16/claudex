package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/embedded"
	"github.com/tanq16/claudex/internal/plugins"
	u "github.com/tanq16/claudex/utils"
)

var configureFlags struct {
	account string
	label   string
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Provision all accounts (settings + statusline) and lay down the global default plugin and flavors",
	Run:   runConfigure,
}

func runConfigure(cmd *cobra.Command, args []string) {
	if configureFlags.label != "" && configureFlags.account == "" {
		u.PrintFatal("--label only applies with -A; without it, labels are auto-derived per account", nil)
	}

	if configureFlags.account != "" {
		if err := configureAccount(u.ResolveConfigDir(configureFlags.account), configureFlags.label); err != nil {
			u.PrintFatal("failed to configure account", err)
		}
	} else {
		configured := 0
		for _, accountDir := range u.DiscoverAccountPaths() {
			if err := configureAccount(accountDir, ""); err != nil {
				u.PrintWarn("Skipped "+u.AbbreviatePath(accountDir), err)
				continue
			}
			configured++
		}
		if configured == 0 {
			u.PrintWarn("No accounts configured; laying down global defaults only", nil)
		}
	}

	applyGlobalDefaults()
}

func configureAccount(accountDir, label string) error {
	info, err := os.Stat(accountDir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("account config dir not found: %s", accountDir)
	}

	scriptPath := filepath.Join(accountDir, "statusline.sh")
	if err := os.WriteFile(scriptPath, embedded.StatuslineScript, 0o755); err != nil {
		return fmt.Errorf("write statusline script: %w", err)
	}

	settingsPath := filepath.Join(accountDir, "settings.json")
	settings := map[string]any{}
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("existing %s is not valid JSON; refusing to overwrite: %w", settingsPath, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read settings.json: %w", err)
	}

	applyPreferredSettings(settings)

	command := scriptPath
	if label != "" {
		command += " " + shellQuote(label)
	}
	settings["statusLine"] = map[string]any{
		"type":    "command",
		"command": command,
		"padding": 0,
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings.json: %w", err)
	}
	out = append(out, '\n')
	if err := writeFileAtomic(settingsPath, out, 0o644); err != nil {
		return fmt.Errorf("write settings.json: %w", err)
	}

	labelDesc := label
	if labelDesc == "" {
		labelDesc = "(auto)"
	}
	u.PrintSuccess(fmt.Sprintf("Configured %s (label: %s)", u.AbbreviatePath(accountDir), labelDesc))
	u.PrintGeneric("  statusline: " + scriptPath)
	u.PrintGeneric("  settings:   " + settingsPath)
	return nil
}

func applyGlobalDefaults() {
	globalDir := u.GlobalPluginDir()
	if err := plugins.BuildGlobalPlugin(globalDir, embedded.DefaultSkillsFS, embedded.OutputStylesFS, true); err != nil {
		u.PrintFatal("failed to build the global plugin", err)
	}
	flavorsDir := u.FlavorsDir()
	if err := os.MkdirAll(flavorsDir, 0o755); err != nil {
		u.PrintFatal("failed to create the flavors directory", err)
	}
	u.PrintSuccess("Refreshed global defaults")
	u.PrintGeneric("  plugin:  " + u.AbbreviatePath(globalDir))
	u.PrintGeneric("  flavors: " + u.AbbreviatePath(flavorsDir))
}

func applyPreferredSettings(settings map[string]any) {
	settings["attribution"] = map[string]any{"commit": ""}
	settings["effortLevel"] = "xhigh"
	settings["tui"] = "fullscreen"
	settings["autoMemoryEnabled"] = false
	settings["skipDangerousModePermissionPrompt"] = true

	env, ok := settings["env"].(map[string]any)
	if !ok {
		env = map[string]any{}
	}
	env["DISABLE_AUTOUPDATER"] = "1"
	env["ENABLE_CLAUDEAI_MCP_SERVERS"] = "false"
	settings["env"] = env
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n\"'\\$`&|;<>()*?[]{}~#!") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func init() {
	configureCmd.Flags().StringVarP(&configureFlags.account, "account", "A", "", "Account config dir to configure (default ~/.claude)")
	configureCmd.Flags().StringVarP(&configureFlags.label, "label", "l", "", "Override the account label shown in the statusline; requires -A (errors without a single-account target)")
}
