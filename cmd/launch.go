package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/plugins"
	u "github.com/tanq16/claudex/utils"
)

const resumeListLimit = 10

var launchFlags struct {
	account    string
	mcp        string
	newSession bool
	resume     bool
	session    string
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
	if len(sessions) > resumeListLimit {
		sessions = sessions[:resumeListLimit]
	}
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
		case len(sessions) == 1:
			s = sessions[0]
		default:
			labels := make([]string, len(sessions))
			for i, sess := range sessions {
				labels[i] = sessionLabel(sess, multiAccount)
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

	if dir := globalPluginDir(); dir != "" {
		cliArgs = append(cliArgs, "--plugin-dir", dir)
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

func globalPluginDir() string {
	dir := u.GlobalPluginDir()
	if err := plugins.BuildGlobalPlugin(dir); err != nil {
		u.PrintWarn("could not prepare the global plugin", err)
		return ""
	}
	return dir
}
