package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/convo"
	u "github.com/tanq16/claudex/utils"
)

var switchFlags struct {
	id   string
	from string
	to   string
}

var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Move a conversation from one account to another",
	Run:   runSwitch,
}

func runSwitch(cmd *cobra.Command, args []string) {
	if switchFlags.id != "" {
		runSwitchExplicit()
		return
	}
	if u.GlobalForAIFlag {
		u.PrintFatal("switch needs --id and --to in --for-ai mode", nil)
	}
	runSwitchInteractive()
}

func runSwitchExplicit() {
	if switchFlags.to == "" {
		u.PrintFatal("--to is required with --id", nil)
	}
	fromDir := u.ResolveConfigDir(switchFlags.from)
	toDir := u.ExpandPath(switchFlags.to)
	doSwitch(switchFlags.id, fromDir, toDir)
}

func runSwitchInteractive() {
	cwd, err := os.Getwd()
	if err != nil {
		u.PrintFatal("failed to resolve current directory", err)
	}

	accounts := u.DiscoverAccountPaths()
	if len(accounts) < 2 {
		u.PrintFatal("switch needs at least two accounts; only one was found", nil)
	}

	sessions := discoverSessions(accounts, cwd)
	if len(sessions) == 0 {
		u.PrintFatal("no sessions for this project were found in any account", nil)
	}

	labels := make([]string, len(sessions))
	for i, s := range sessions {
		msg := padRight(u.Truncate(strings.Join(strings.Fields(s.firstMessage), " "), 50), 50)
		t := time.UnixMilli(s.lastActivity).Local().Format("Jan 02 3:04pm")
		labels[i] = fmt.Sprintf("%s  %s  %s", msg, t, u.AbbreviatePath(s.configDir))
	}
	idx, err := u.PromptSelect("Session to move", labels)
	if err != nil {
		u.PrintFatal("TUI error", err)
	}
	if idx < 0 {
		return
	}
	sel := sessions[idx]

	var others []string
	for _, a := range accounts {
		if a != sel.configDir {
			others = append(others, a)
		}
	}

	toDir := others[0]
	if len(others) > 1 {
		acctLabels := make([]string, len(others))
		for i, a := range others {
			acctLabels[i] = u.AbbreviatePath(a)
		}
		aidx, err := u.PromptSelect("Move to account", acctLabels)
		if err != nil {
			u.PrintFatal("TUI error", err)
		}
		if aidx < 0 {
			return
		}
		toDir = others[aidx]
	}

	doSwitch(sel.sessionID, sel.configDir, toDir)
}

func doSwitch(id, fromDir, toDir string) {
	sf, err := convo.FindSession(fromDir, id)
	if err != nil {
		u.PrintFatal("Error searching source account", err)
	}
	if sf == nil {
		u.PrintFatal(fmt.Sprintf("Session %s not found in %s", id, u.AbbreviatePath(fromDir)), nil)
	}

	dstProjectDir := convo.ProjectDir(toDir, sf.ProjectPath)
	if err := convo.MoveSession(id, sf.ProjectDir, dstProjectDir); err != nil {
		u.PrintFatal("Failed to move session files", err)
	}

	srcEntries, err := convo.ReadRawHistory(fromDir)
	if err != nil {
		u.PrintWarn("Could not read source history", err)
	} else {
		matching, rest := convo.FilterBySession(srcEntries, id)
		if len(matching) > 0 {
			if err := convo.AppendRawHistory(toDir, matching); err != nil {
				u.PrintWarn("Could not append to target history", err)
			} else if err := convo.WriteRawHistory(fromDir, rest); err != nil {
				u.PrintWarn("Could not update source history", err)
			}
		}
	}

	u.PrintSuccess(fmt.Sprintf("Switched session %s from %s to %s", id, u.AbbreviatePath(fromDir), u.AbbreviatePath(toDir)))
}

func init() {
	switchCmd.Flags().StringVar(&switchFlags.id, "id", "", "Session UUID to switch (non-interactive; skips the selector)")
	switchCmd.Flags().StringVar(&switchFlags.from, "from", "", "Source config directory (default ~/.claude)")
	switchCmd.Flags().StringVar(&switchFlags.to, "to", "", "Target config directory (required with --id)")
}
