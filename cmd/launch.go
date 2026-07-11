package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/embedded"
	"github.com/tanq16/claudex/internal/flavors"
	"github.com/tanq16/claudex/internal/parser"
	"github.com/tanq16/claudex/internal/plugins"
	u "github.com/tanq16/claudex/utils"
)

type sessionEntry struct {
	sessionID    string
	project      string
	firstMessage string
	lastActivity int64
	configDir    string
}

var launchFlags struct {
	plugins []string
}

var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch a Claude Code session with interactive config selection",
	Run:   runLaunch,
}

func init() {
	launchCmd.Flags().StringSliceVar(&launchFlags.plugins, "plugins", nil,
		"Local plugin directories or git repo URLs to load via --plugin-dir (repeatable or comma-separated)")
}

func runLaunch(cmd *cobra.Command, args []string) {
	if u.GlobalForAIFlag {
		u.PrintFatal("launch requires an interactive terminal", nil)
	}

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		u.PrintFatal("claude not found in PATH", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		u.PrintFatal("failed to resolve current directory", err)
	}

	accounts := u.DiscoverAccountPaths()
	sessions := discoverSessions(accounts, cwd)
	multiAccount := len(accounts) > 1

	var resumeMode bool
	if len(sessions) > 0 {
		idx, err := u.PromptSelect("Launch", []string{"New session", "Resume"})
		if err != nil {
			u.PrintFatal("TUI error", err)
		}
		if idx < 0 {
			return
		}
		resumeMode = idx == 1
	}

	var account string
	var cliArgs []string
	var summary []string

	if resumeMode {
		labels := make([]string, len(sessions))
		for i, s := range sessions {
			msg := padRight(u.Truncate(strings.Join(strings.Fields(s.firstMessage), " "), 60), 60)
			t := time.UnixMilli(s.lastActivity).Local().Format("Jan 02 3:04pm")
			labels[i] = fmt.Sprintf("%s  %s", msg, t)
			if multiAccount {
				labels[i] += "  " + u.AbbreviatePath(s.configDir)
			}
		}

		idx, err := u.PromptSelect("Resume Session", labels)
		if err != nil {
			u.PrintFatal("TUI error", err)
		}
		if idx < 0 {
			return
		}

		s := sessions[idx]
		account = s.configDir
		cliArgs = []string{"claude", "--resume", s.sessionID}
		summary = append(summary, "resume", s.project, u.AbbreviatePath(s.configDir))
	} else {
		if multiAccount {
			acctLabels := make([]string, len(accounts))
			for i, a := range accounts {
				acctLabels[i] = u.AbbreviatePath(a)
			}
			idx, err := u.PromptSelect("Account", acctLabels)
			if err != nil {
				u.PrintFatal("TUI error", err)
			}
			if idx < 0 {
				return
			}
			account = accounts[idx]
		} else {
			account = accounts[0]
		}
		summary = append(summary, u.AbbreviatePath(account))

		mcpIdx, err := u.PromptSelect("MCP + Connectors", []string{
			"MCPs only",
			"MCPs + Connectors",
			"None",
		})
		if err != nil {
			u.PrintFatal("TUI error", err)
		}
		if mcpIdx < 0 {
			return
		}

		cliArgs = []string{"claude"}
		switch mcpIdx {
		case 0:
			summary = append(summary, "mcp: on")
		case 1:
			settingsJSON, _ := json.Marshal(map[string]any{
				"env": map[string]string{
					"ENABLE_CLAUDEAI_MCP_SERVERS": "true",
				},
			})
			cliArgs = append(cliArgs, "--settings", string(settingsJSON))
			summary = append(summary, "mcp: on", "connectors: on")
		case 2:
			cliArgs = append(cliArgs, "--strict-mcp-config")
			summary = append(summary, "mcp: off")
		}
	}

	if flavor, ok := selectFlavor(); ok {
		if flavor == nil {
			return
		}
		cliArgs = append(cliArgs, "--append-system-prompt", flavor.Body)
		summary = append(summary, "flavor: "+flavor.Name)
	}

	for _, dir := range resolvePluginDirs() {
		cliArgs = append(cliArgs, "--plugin-dir", dir)
	}
	if len(launchFlags.plugins) > 0 {
		summary = append(summary, fmt.Sprintf("plugins: %d", len(launchFlags.plugins)))
	}

	cliArgs = append(cliArgs, "--dangerously-skip-permissions")

	// strip any inherited CLAUDE_CONFIG_DIR so it can't override the chosen account
	env := os.Environ()
	home, _ := os.UserHomeDir()
	defaultDir := filepath.Join(home, ".claude")
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, "CLAUDE_CONFIG_DIR=") {
			filtered = append(filtered, e)
		}
	}
	env = filtered
	if account != defaultDir {
		env = append(env, "CLAUDE_CONFIG_DIR="+account)
	}

	u.PrintInfo("Launching: " + strings.Join(summary, " · "))

	if err := syscall.Exec(claudePath, cliArgs, env); err != nil {
		u.PrintFatal("Failed to exec claude", err)
	}
}

func discoverSessions(accounts []string, cwd string) []sessionEntry {
	target := filepath.Clean(cwd)
	var all []sessionEntry
	for _, configDir := range accounts {
		convos, err := parser.ParseConversations(configDir)
		if err != nil {
			continue
		}
		for _, c := range convos {
			if filepath.Clean(c.ProjectPath) != target {
				continue
			}
			all = append(all, sessionEntry{
				sessionID:    c.SessionID,
				project:      c.Project,
				firstMessage: c.FirstMessage,
				lastActivity: c.LastActivity,
				configDir:    configDir,
			})
		}
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].lastActivity > all[j].lastActivity
	})
	if len(all) > 10 {
		all = all[:10]
	}
	return all
}

func padRight(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(runes))
}

// (nil, false) means proceed with no flavor; (f, true) apply f; (nil, true) the user cancelled.
func selectFlavor() (*flavors.Flavor, bool) {
	opts, err := flavors.Load(u.FlavorsDir())
	if err != nil {
		u.PrintWarn("could not read flavors", err)
		return nil, false
	}
	if opts.Auto != nil {
		return opts.Auto, true
	}
	if len(opts.Choices) == 0 {
		return nil, false
	}

	labels := make([]string, len(opts.Choices)+1)
	for i, f := range opts.Choices {
		labels[i] = f.Name
	}
	labels[len(opts.Choices)] = "None"

	idx, err := u.PromptSelect("Flavor", labels)
	if err != nil {
		u.PrintFatal("TUI error", err)
	}
	if idx < 0 {
		return nil, true
	}
	if idx == len(opts.Choices) {
		return nil, false
	}
	return &opts.Choices[idx], true
}

func resolvePluginDirs() []string {
	var dirs []string

	globalDir := u.GlobalPluginDir()
	if err := plugins.BuildGlobalPlugin(globalDir, embedded.SkillsFS, embedded.OutputStylesFS, false); err != nil {
		u.PrintWarn("could not prepare the global plugin", err)
	} else {
		dirs = append(dirs, globalDir)
	}

	for _, spec := range launchFlags.plugins {
		src := plugins.Classify(spec)
		if !src.IsLocal {
			u.PrintRunning("Fetching plugin " + src.Name)
		}
		dir, err := plugins.Fetch(src, u.PluginsDir())
		if !src.IsLocal {
			u.ClearLines(1)
		}
		if err != nil {
			u.PrintWarn("Plugin "+spec+" skipped", err)
			continue
		}
		if !src.IsLocal {
			u.PrintSuccess("Plugin ready: " + src.Name)
		}
		dirs = append(dirs, dir)
	}

	return dirs
}
