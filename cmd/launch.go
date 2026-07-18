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
	plugins    []string
	account    string
	mcp        string
	newSession bool
	resume     bool
	session    string
	flavor     string
	noFlavor   bool
}

// NoArgs so a stray positional (e.g. "--resume <id>" typed for "--session <id>")
// errors instead of being silently ignored.
var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch a Claude Code session with interactive config selection",
	Args:  cobra.NoArgs,
	Run:   runLaunch,
}

func init() {
	launchCmd.Flags().StringSliceVarP(&launchFlags.plugins, "plugins", "P", nil,
		"Local plugin directories or git repo URLs to load via --plugin-dir (repeatable or comma-separated)")
	launchCmd.Flags().StringVarP(&launchFlags.account, "account", "A", "",
		"Account to launch under (skips the account picker)")
	launchCmd.Flags().StringVar(&launchFlags.mcp, "mcp", "",
		`MCP mode: "mcps", "connectors", or "none" (skips the MCP picker)`)
	launchCmd.Flags().BoolVar(&launchFlags.newSession, "new", false,
		"Start a new session (skip the new/resume prompt)")
	launchCmd.Flags().BoolVar(&launchFlags.resume, "resume", false,
		"Resume mode: pick the latest session, or list them when there's more than one")
	launchCmd.Flags().StringVar(&launchFlags.session, "session", "",
		"Resume a specific session by id (skips the new/resume prompt)")
	launchCmd.MarkFlagsMutuallyExclusive("new", "resume", "session")
	launchCmd.Flags().StringVar(&launchFlags.flavor, "flavor", "",
		"Select a flavor by name (skips the flavor picker)")
	launchCmd.Flags().BoolVar(&launchFlags.noFlavor, "no-flavor", false,
		"Do not apply any flavor (skips the flavor picker)")
	launchCmd.MarkFlagsMutuallyExclusive("flavor", "no-flavor")
}

func runLaunch(cmd *cobra.Command, args []string) {
	if u.GlobalForAIFlag {
		u.PrintFatal("launch requires an interactive terminal", nil)
	}

	if launchFlags.mcp != "" && launchFlags.mcp != "mcps" && launchFlags.mcp != "connectors" && launchFlags.mcp != "none" {
		u.PrintFatal(`--mcp must be one of "mcps", "connectors", or "none"`, nil)
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

	resumeID := launchFlags.session
	resumeSet := launchFlags.resume || resumeID != ""
	if resumeSet && len(sessions) == 0 {
		u.PrintFatal("no sessions to resume for this project", nil)
	}

	var resumeMode bool
	switch {
	case launchFlags.newSession:
		resumeMode = false
	case resumeSet:
		resumeMode = true
	default:
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
	}

	var account string
	var cliArgs []string
	var summary []string

	if resumeMode {
		var s sessionEntry
		switch {
		case resumeID != "":
			found := false
			for _, cand := range sessions {
				if cand.sessionID == resumeID {
					s = cand
					found = true
					break
				}
			}
			if !found {
				u.PrintFatal("session not found: "+resumeID, nil)
			}
		case resumeSet && len(sessions) == 1:
			s = sessions[0]
		default:
			labels := make([]string, len(sessions))
			for i, sess := range sessions {
				msg := padRight(u.Truncate(strings.Join(strings.Fields(sess.firstMessage), " "), 60), 60)
				t := time.UnixMilli(sess.lastActivity).Local().Format("Jan 02 3:04pm")
				labels[i] = fmt.Sprintf("%s  %s", msg, t)
				if multiAccount {
					labels[i] += "  " + u.AbbreviatePath(sess.configDir)
				}
			}
			idx, err := u.PromptSelect("Resume Session", labels)
			if err != nil {
				u.PrintFatal("TUI error", err)
			}
			if idx < 0 {
				return
			}
			s = sessions[idx]
		}

		account = s.configDir
		cliArgs = []string{"claude", "--resume", s.sessionID}
		summary = append(summary, "resume", s.project, u.AbbreviatePath(s.configDir))
	} else {
		if launchFlags.account != "" {
			account = resolveAccountFlag(launchFlags.account, accounts)
		} else if multiAccount {
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
		cliArgs = []string{"claude"}
	}

	mode := launchFlags.mcp
	if mode == "" {
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
		mode = []string{"mcps", "connectors", "none"}[mcpIdx]
	}
	cliArgs, summary = applyMCPMode(mode, cliArgs, summary)

	if !launchFlags.noFlavor {
		if launchFlags.flavor != "" {
			opts, err := flavors.Load(u.FlavorsDir())
			if err != nil {
				u.PrintFatal("could not read flavors", err)
			}
			flavor := findFlavor(opts, launchFlags.flavor)
			if flavor == nil {
				u.PrintFatal("flavor not found: "+launchFlags.flavor, nil)
			}
			cliArgs = append(cliArgs, "--append-system-prompt", flavor.Body)
			summary = append(summary, "flavor: "+flavor.Name)
		} else if flavor, ok := selectFlavor(); ok {
			if flavor == nil {
				return
			}
			cliArgs = append(cliArgs, "--append-system-prompt", flavor.Body)
			summary = append(summary, "flavor: "+flavor.Name)
		}
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

func applyMCPMode(mode string, cliArgs, summary []string) ([]string, []string) {
	switch mode {
	case "mcps":
		summary = append(summary, "mcp: on")
	case "connectors":
		settingsJSON, _ := json.Marshal(map[string]any{
			"env": map[string]string{
				"ENABLE_CLAUDEAI_MCP_SERVERS": "true",
			},
		})
		cliArgs = append(cliArgs, "--settings", string(settingsJSON))
		summary = append(summary, "mcp: on", "connectors: on")
	case "none":
		cliArgs = append(cliArgs, "--strict-mcp-config")
		summary = append(summary, "mcp: off")
	}
	return cliArgs, summary
}

func resolveAccountFlag(flag string, accounts []string) string {
	expanded := filepath.Clean(u.ExpandPath(flag))
	for _, a := range accounts {
		if filepath.Clean(a) == expanded || filepath.Base(a) == flag {
			return a
		}
	}
	u.PrintFatal("account not found: "+flag, nil)
	return ""
}

func findFlavor(opts *flavors.Options, name string) *flavors.Flavor {
	if opts.Auto != nil && opts.Auto.Name == name {
		return opts.Auto
	}
	for i := range opts.Choices {
		if opts.Choices[i].Name == name {
			return &opts.Choices[i]
		}
	}
	return nil
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
	if err := plugins.BuildGlobalPlugin(globalDir, embedded.DefaultSkillsFS, embedded.OutputStylesFS, false); err != nil {
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
