package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/convo"
	"github.com/tanq16/claudex/internal/parser"
	u "github.com/tanq16/claudex/utils"
)

var switchFlags struct {
	account string
}

var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Move the current project's sessions to another account",
	Run:   runSwitch,
}

type projectSession struct {
	sessionID    string
	project      string
	lastActivity int64
	configDir    string
	projectPath  string
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

	sessions := gatherProjectSessions(accounts, cwd)
	if len(sessions) == 0 {
		u.PrintFatal("no sessions for this project were found in any account", nil)
	}

	current := sessions[0].configDir

	var target string
	if switchFlags.account != "" {
		target = resolveTargetAccount(switchFlags.account, accounts)
		if target == current {
			u.PrintSuccess(fmt.Sprintf("Project already in %s; nothing to switch", u.AbbreviatePath(current)))
			return
		}
	} else if u.GlobalForAIFlag {
		u.PrintFatal("switch needs -A/--account in --for-ai mode", nil)
	} else {
		var others []string
		for _, a := range accounts {
			if a != current {
				others = append(others, a)
			}
		}
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

	var ids []string
	projectPath := ""
	for _, s := range sessions {
		if s.configDir == current {
			ids = append(ids, s.sessionID)
			projectPath = s.projectPath
		}
	}

	srcDir := convo.ProjectDir(current, projectPath)
	dstDir := convo.ProjectDir(target, projectPath)
	for _, id := range ids {
		if err := convo.MoveSession(id, srcDir, dstDir); err != nil {
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

	u.PrintSuccess(fmt.Sprintf("Switched project %s (%d session(s)) from %s to %s",
		sessions[0].project, len(ids), u.AbbreviatePath(current), u.AbbreviatePath(target)))
}

func gatherProjectSessions(accounts []string, cwd string) []projectSession {
	target := filepath.Clean(cwd)
	var all []projectSession
	for _, configDir := range accounts {
		convos, err := parser.ParseConversations(configDir)
		if err != nil {
			continue
		}
		for _, c := range convos {
			if filepath.Clean(c.ProjectPath) != target {
				continue
			}
			all = append(all, projectSession{
				sessionID:    c.SessionID,
				project:      c.Project,
				lastActivity: c.LastActivity,
				configDir:    configDir,
				projectPath:  c.ProjectPath,
			})
		}
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].lastActivity > all[j].lastActivity
	})
	return all
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
}
