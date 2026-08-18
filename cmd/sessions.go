package cmd

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/tanq16/claudex/internal/parser"
	u "github.com/tanq16/claudex/utils"
)

type sessionEntry struct {
	sessionID    string
	project      string
	firstMessage string
	lastActivity int64
	configDir    string
	projectPath  string
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
				projectPath:  c.ProjectPath,
			})
		}
	}
	slices.SortFunc(all, func(a, b sessionEntry) int {
		return cmp.Compare(b.lastActivity, a.lastActivity)
	})
	return all
}

func sessionsIn(sessions []sessionEntry, configDir string) []sessionEntry {
	var out []sessionEntry
	for _, s := range sessions {
		if s.configDir == configDir {
			out = append(out, s)
		}
	}
	return out
}

func sessionLabel(s sessionEntry, withAccount bool) string {
	id := shortSessionID(s.sessionID)
	msg := padRight(u.Truncate(strings.Join(strings.Fields(s.firstMessage), " "), 60), 60)
	t := time.UnixMilli(s.lastActivity).Local().Format("Jan 02 3:04pm")
	label := fmt.Sprintf("%s  %s  %s", id, msg, t)
	if withAccount {
		label += "  " + u.AbbreviatePath(s.configDir)
	}
	return label
}

func shortSessionID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func padRight(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(runes))
}
