package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/embedded"
	u "github.com/tanq16/claudex/utils"
)

var configureFlags struct {
	account string
	label   string
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Apply claudex's preferred Claude Code settings and statusline to an account",
	Run:   runConfigure,
}

func runConfigure(cmd *cobra.Command, args []string) {
	accountDir := u.ResolveConfigDir(configureFlags.account)

	info, err := os.Stat(accountDir)
	if err != nil || !info.IsDir() {
		u.PrintFatal(fmt.Sprintf("account config dir not found: %s", accountDir), err)
	}

	scriptPath := filepath.Join(accountDir, "statusline.sh")
	if err := os.WriteFile(scriptPath, embedded.StatuslineScript, 0o755); err != nil {
		u.PrintFatal("failed to write statusline script", err)
	}

	settingsPath := filepath.Join(accountDir, "settings.json")
	settings := map[string]any{}
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			u.PrintFatal(fmt.Sprintf("existing %s is not valid JSON; refusing to overwrite", settingsPath), err)
		}
	} else if !os.IsNotExist(err) {
		u.PrintFatal("failed to read settings.json", err)
	}

	applyPreferredSettings(settings)

	command := scriptPath
	if configureFlags.label != "" {
		command += " " + shellQuote(configureFlags.label)
	}
	settings["statusLine"] = map[string]any{
		"type":    "command",
		"command": command,
		"padding": 0,
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		u.PrintFatal("failed to encode settings.json", err)
	}
	out = append(out, '\n')
	if err := writeFileAtomic(settingsPath, out, 0o644); err != nil {
		u.PrintFatal("failed to write settings.json", err)
	}

	labelDesc := configureFlags.label
	if labelDesc == "" {
		labelDesc = "(auto)"
	}
	u.PrintSuccess(fmt.Sprintf("Configured %s (label: %s)", u.AbbreviatePath(accountDir), labelDesc))
	u.PrintGeneric("  statusline: " + scriptPath)
	u.PrintGeneric("  settings:   " + settingsPath)
}

func applyPreferredSettings(settings map[string]any) {
	settings["attribution"] = map[string]any{"commit": ""}
	settings["effortLevel"] = "xhigh"
	settings["tui"] = "fullscreen"
	settings["autoMemoryEnabled"] = false
	settings["skipDangerousModePermissionPrompt"] = true
	settings["CLAUDE_AFK_TIMEOUT_MS"] = "10000000"

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
	configureCmd.Flags().StringVarP(&configureFlags.label, "label", "l", "", "Override the account label shown in the statusline (default: word derived from dir name)")
}
