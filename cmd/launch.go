package cmd

import (
	"os/exec"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/accounts"
	"github.com/tanq16/claudex/internal/plugins"
	"github.com/tanq16/claudex/internal/session"
	u "github.com/tanq16/claudex/utils"
)

var launchFlags struct {
	account       string
	connectors    bool
	continueLast  bool
	remoteControl string
}

var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Exec claude in the current directory under one account",
	Args:  cobra.NoArgs,
	Run:   runLaunch,
}

func init() {
	launchCmd.Flags().StringVarP(&launchFlags.account, "account", "A", "", "Account to launch under (default ~/.claude)")
	launchCmd.Flags().BoolVar(&launchFlags.connectors, "connectors", false, "Load the claude.ai connectors as MCP servers")
	launchCmd.Flags().BoolVar(&launchFlags.continueLast, "continue", false, "Continue the most recent conversation in this directory")
	launchCmd.Flags().StringVarP(&launchFlags.remoteControl, "remote-control", "r", "", "Start with Remote Control under this session name")
}

func runLaunch(cmd *cobra.Command, args []string) {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		u.PrintFatal("claude not found in PATH", err)
	}
	cwd := currentDir()
	defaultDir := u.ResolveConfigDir("")
	account := defaultDir
	if launchFlags.account != "" {
		account = resolveAccount(launchFlags.account)
	}

	if err := accounts.Trust(account, defaultDir, cwd); err != nil {
		u.PrintWarn("could not mark this directory as trusted; claude will ask", err)
	}

	x := session.Launch{
		RCName:     launchFlags.remoteControl,
		Connectors: launchFlags.connectors,
		Continue:   launchFlags.continueLast,
	}
	argv := append([]string{"claude"}, session.Args(x, globalPluginDir())...)

	summary := []string{u.AbbreviatePath(account)}
	if x.Continue {
		summary = append(summary, "continue")
	}
	if x.RCName != "" {
		summary = append(summary, "remote control: "+x.RCName)
	}
	if x.Connectors {
		summary = append(summary, "connectors: on")
	}
	u.PrintInfo("Launching: " + strings.Join(summary, " · "))

	if err := syscall.Exec(claudePath, argv, accounts.Env(account, defaultDir)); err != nil {
		u.PrintFatal("Failed to exec claude", err)
	}
}

func globalPluginDir() string {
	dir := u.GlobalPluginDir()
	if err := plugins.BuildGlobalPlugin(dir); err != nil {
		u.PrintWarn("could not prepare the global plugin", err)
		return ""
	}
	return dir
}
