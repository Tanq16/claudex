package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/convo"
	u "github.com/tanq16/claudex/utils"
)

var switchFlags struct {
	account string
	session string
}

// NoArgs so "switch <id>" typed for "switch --session <id>" errors instead of
// silently moving every session.
var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Move the current project's sessions to another account",
	Args:  cobra.NoArgs,
	Run:   runSwitch,
}

func runSwitch(cmd *cobra.Command, args []string) {
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

	var target string
	if switchFlags.account != "" {
		target = resolveTargetAccount(switchFlags.account, accounts)
		if len(sessionsIn(sessions, target)) == len(sessions) {
			u.PrintSuccess(fmt.Sprintf("Project already in %s; nothing to switch", u.AbbreviatePath(target)))
			return
		}
	} else if u.GlobalForAIFlag {
		u.PrintFatal("switch needs -A/--account in --for-ai mode", nil)
	}

	source := sessions[0].configDir
	inSource := sessionsIn(sessions, source)

	var moving []sessionEntry
	singleSession := false
	switch {
	case switchFlags.session != "":
		idx := slices.IndexFunc(sessions, func(s sessionEntry) bool { return s.sessionID == switchFlags.session })
		if idx < 0 {
			u.PrintFatal("session not found: "+switchFlags.session, nil)
		}
		moving = []sessionEntry{sessions[idx]}
		singleSession = true
	case u.GlobalForAIFlag || len(sessions) == 1:
		moving = inSource
	default:
		labels := make([]string, 0, len(sessions)+1)
		offset := 0
		if len(inSource) > 1 {
			labels = append(labels, fmt.Sprintf("All %d sessions in %s", len(inSource), u.AbbreviatePath(source)))
			offset = 1
		}
		for _, s := range sessions {
			labels = append(labels, sessionLabel(s, true))
		}
		idx, err := u.PromptSelect("Move Sessions", labels)
		if err != nil {
			u.PrintFatal("TUI error", err)
		}
		if idx < 0 {
			return
		}
		if idx < offset {
			moving = inSource
		} else {
			moving = []sessionEntry{sessions[idx-offset]}
			singleSession = true
		}
	}

	current := moving[0].configDir
	if target == current {
		u.PrintSuccess(fmt.Sprintf("Already in %s; nothing to switch", u.AbbreviatePath(current)))
		return
	}
	if target == "" {
		others := slices.DeleteFunc(slices.Clone(accounts), func(a string) bool { return a == current })
		if len(others) == 0 {
			u.PrintFatal("no other account to switch this project into", nil)
		}
		if len(others) == 1 {
			target = others[0]
		} else {
			labels := make([]string, len(others))
			for i, a := range others {
				labels[i] = u.AbbreviatePath(a)
			}
			idx, err := u.PromptSelect("Move to account", labels)
			if err != nil {
				u.PrintFatal("TUI error", err)
			}
			if idx < 0 {
				return
			}
			target = others[idx]
		}
	}

	ids := make([]string, len(moving))
	for i, s := range moving {
		ids[i] = s.sessionID
		srcDir := convo.ProjectDir(s.configDir, s.projectPath)
		dstDir := convo.ProjectDir(target, s.projectPath)
		if err := convo.MoveSession(s.sessionID, srcDir, dstDir); err != nil {
			u.PrintFatal("Failed to move session files", err)
		}
	}

	srcEntries, err := convo.ReadRawHistory(current)
	if err != nil {
		u.PrintWarn("Could not read source history", err)
	} else {
		var matching []convo.RawHistoryEntry
		rest := srcEntries
		for _, id := range ids {
			m, r := convo.FilterBySession(rest, id)
			matching = append(matching, m...)
			rest = r
		}
		if len(matching) > 0 {
			if err := convo.AppendRawHistory(target, matching); err != nil {
				u.PrintWarn("Could not append to target history", err)
			} else if err := convo.WriteRawHistory(current, rest); err != nil {
				u.PrintWarn("Could not update source history", err)
			}
		}
	}

	moved := fmt.Sprintf("project %s (%d session(s))", moving[0].project, len(ids))
	if singleSession {
		moved = fmt.Sprintf("session %s in %s", moving[0].sessionID, moving[0].project)
	}
	u.PrintSuccess(fmt.Sprintf("Switched %s from %s to %s",
		moved, u.AbbreviatePath(current), u.AbbreviatePath(target)))
}

func resolveTargetAccount(flag string, accounts []string) string {
	expanded := filepath.Clean(u.ExpandPath(flag))
	base := filepath.Base(flag)
	for _, a := range accounts {
		if filepath.Clean(a) == expanded || filepath.Base(a) == base {
			return a
		}
	}
	u.PrintFatal(fmt.Sprintf("account %q matches no discovered account", flag), nil)
	return ""
}

func init() {
	switchCmd.Flags().StringVarP(&switchFlags.account, "account", "A", "",
		"Account to switch this project into (source is auto-detected from the current directory)")
	switchCmd.Flags().StringVar(&switchFlags.session, "session", "",
		"Move only this session by id (skips the session picker)")
}
