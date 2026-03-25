package convosCmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/convo"
	"github.com/tanq16/claudex/internal/parser"
	u "github.com/tanq16/claudex/utils"
)

var listFlags struct {
	accounts   []string
	limit      int
	jsonOutput bool
}

var switchFlags struct {
	id   string
	from string
	to   string
}

var reprojectFlags struct {
	id        string
	configDir string
	project   string
}

var findFlags struct {
	id         string
	keyword    string
	accounts   []string
	jsonOutput bool
}

var ConvosCmd = &cobra.Command{
	Use:   "convos",
	Short: "Manage Claude Code conversations (list, switch, reproject, find)",
}

// --- list ---

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent conversations across accounts",
	Run:   runList,
}

func runList(cmd *cobra.Command, args []string) {
	accountPaths := u.ResolveAccountPaths(listFlags.accounts)
	for _, p := range accountPaths {
		acct, _ := parser.ParseAccount(p)
		convos, err := parser.ParseConversations(p)
		if err != nil {
			u.PrintWarn(fmt.Sprintf("skipping %s", p), err)
			continue
		}

		email := acct.Email
		if email == "" {
			email = p
		}

		if listFlags.limit > 0 && len(convos) > listFlags.limit {
			convos = convos[:listFlags.limit]
		}

		if listFlags.jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(map[string]any{
				"account":       email,
				"conversations": convos,
			})
			continue
		}

		u.PrintGeneric("")
		u.PrintInfo(email)

		if len(convos) == 0 {
			u.PrintGeneric("  No conversations found.")
			continue
		}

		rows := make([][]string, len(convos))
		for i, c := range convos {
			snippet := truncate(c.FirstMessage, 40)
			lastActive := time.UnixMilli(c.LastActivity).Local().Format("Jan 02 3:04pm")
			rows[i] = []string{
				c.SessionID,
				fmt.Sprintf("%d", c.MessageCount),
				c.Project,
				snippet,
				lastActive,
			}
		}
		u.PrintTable([]string{"Session", "Msgs", "Project", "First Message", "Last Active"}, rows)
	}
	u.PrintGeneric("")
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

// --- switch ---

var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Move a conversation from one account to another",
	Run:   runSwitch,
}

func runSwitch(cmd *cobra.Command, args []string) {
	if switchFlags.id == "" {
		u.PrintFatal("--id is required", nil)
	}
	if switchFlags.to == "" {
		u.PrintFatal("--to is required", nil)
	}

	fromDir := u.ResolveConfigDir(switchFlags.from)
	toDir := u.ExpandPath(switchFlags.to)

	sf, err := convo.FindSession(fromDir, switchFlags.id)
	if err != nil {
		u.PrintFatal("Error searching source account", err)
	}
	if sf == nil {
		u.PrintFatal(fmt.Sprintf("Session %s not found in %s", switchFlags.id, fromDir), nil)
	}

	projectPath := sf.ProjectPath
	dstProjectDir := convo.ProjectDir(toDir, projectPath)

	if err := convo.MoveSession(switchFlags.id, sf.ProjectDir, dstProjectDir); err != nil {
		u.PrintFatal("Failed to move session files", err)
	}

	srcEntries, err := convo.ReadRawHistory(fromDir)
	if err != nil {
		u.PrintWarn("Could not read source history", err)
	} else {
		matching, rest := convo.FilterBySession(srcEntries, switchFlags.id)
		if len(matching) > 0 {
			if err := convo.AppendRawHistory(toDir, matching); err != nil {
				u.PrintWarn("Could not append to target history", err)
			}
			if err := convo.WriteRawHistory(fromDir, rest); err != nil {
				u.PrintWarn("Could not update source history", err)
			}
		}
	}

	u.PrintSuccess(fmt.Sprintf("Switched session %s from %s to %s", switchFlags.id, fromDir, toDir))
}

// --- reproject ---

var reprojectCmd = &cobra.Command{
	Use:   "reproject",
	Short: "Change which project a conversation is associated with",
	Run:   runReproject,
}

func runReproject(cmd *cobra.Command, args []string) {
	if reprojectFlags.id == "" {
		u.PrintFatal("--id is required", nil)
	}

	configDir := u.ResolveConfigDir(reprojectFlags.configDir)

	targetProject := reprojectFlags.project
	if targetProject == "" {
		cwd, err := os.Getwd()
		if err != nil {
			u.PrintFatal("Cannot determine current directory", err)
		}
		targetProject = cwd
	}

	sf, err := convo.FindSession(configDir, reprojectFlags.id)
	if err != nil {
		u.PrintFatal("Error searching for session", err)
	}
	if sf == nil {
		u.PrintFatal(fmt.Sprintf("Session %s not found in %s", reprojectFlags.id, configDir), nil)
	}

	newProjectDir := convo.ProjectDir(configDir, targetProject)

	if sf.ProjectDir == newProjectDir {
		u.PrintInfo("Session is already associated with this project.")
		return
	}

	if err := convo.MoveSession(reprojectFlags.id, sf.ProjectDir, newProjectDir); err != nil {
		u.PrintFatal("Failed to move session files", err)
	}

	if err := convo.UpdateProjectInHistory(configDir, reprojectFlags.id, targetProject); err != nil {
		u.PrintWarn("Could not update history entries", err)
	}

	u.PrintSuccess(fmt.Sprintf("Reproject session %s to %s", reprojectFlags.id, targetProject))
}

// --- find ---

var findCmd = &cobra.Command{
	Use:   "find",
	Short: "Find conversations by ID or keyword",
	Run:   runFind,
}

func runFind(cmd *cobra.Command, args []string) {
	if findFlags.id == "" && findFlags.keyword == "" {
		u.PrintFatal("Either --id or --keyword is required", nil)
	}

	accountPaths := u.ResolveAccountPaths(findFlags.accounts)

	if findFlags.id != "" {
		runFindByID(accountPaths)
	} else {
		runFindByKeyword(accountPaths)
	}
}

func runFindByID(accountPaths []string) {
	finder := convo.FindSessionAllAccounts(accountPaths)
	sf, err := finder(findFlags.id)
	if err != nil {
		u.PrintFatal("Error searching for session", err)
	}
	if sf == nil {
		u.PrintFatal(fmt.Sprintf("Session %s not found in any account", findFlags.id), nil)
	}

	if findFlags.jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]any{
			"sessionId":  findFlags.id,
			"configDir":  sf.ConfigDir,
			"projectDir": sf.ProjectDir,
			"project":    sf.ProjectPath,
			"hasJSONL":   sf.SessionJSONL != "",
			"hasSubDir":  sf.SubAgentDir != "",
		})
		return
	}

	u.PrintGeneric("")
	u.PrintSuccess(fmt.Sprintf("Found session %s", findFlags.id))
	u.PrintGeneric(fmt.Sprintf("  Account:  %s", sf.ConfigDir))
	u.PrintGeneric(fmt.Sprintf("  Project:  %s", sf.ProjectPath))
	u.PrintGeneric(fmt.Sprintf("  Dir:      %s", sf.ProjectDir))
	if sf.SessionJSONL != "" {
		u.PrintGeneric(fmt.Sprintf("  JSONL:    %s", sf.SessionJSONL))
	}
	if sf.SubAgentDir != "" {
		u.PrintGeneric(fmt.Sprintf("  Agents:   %s", sf.SubAgentDir))
	}
	u.PrintGeneric("")
}

func runFindByKeyword(accountPaths []string) {
	results, err := convo.SearchHistory(accountPaths, findFlags.keyword)
	if err != nil {
		u.PrintFatal("Search failed", err)
	}

	if len(results) == 0 {
		u.PrintInfo("No conversations matched.")
		return
	}

	limit := 5
	if len(results) < limit {
		limit = len(results)
	}
	results = results[:limit]

	if findFlags.jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]any{"results": results})
		return
	}

	u.PrintGeneric("")
	u.PrintInfo(fmt.Sprintf("Top %d results for \"%s\":", limit, findFlags.keyword))

	rows := make([][]string, len(results))
	for i, r := range results {
		rows[i] = []string{
			r.SessionID,
			r.Project,
			r.ConfigDir,
			fmt.Sprintf("%d", r.MatchCount),
			truncate(r.Sample, 40),
		}
	}
	u.PrintTable([]string{"Session", "Project", "Account", "Matches", "Sample"}, rows)
	u.PrintGeneric("")
}

// --- init ---

func init() {
	listCmd.Flags().StringSliceVarP(&listFlags.accounts, "accounts", "a", []string{}, "Additional Claude config directories")
	listCmd.Flags().IntVarP(&listFlags.limit, "limit", "n", 10, "Number of conversations to show")
	listCmd.Flags().BoolVarP(&listFlags.jsonOutput, "json", "j", false, "Output as JSON")

	switchCmd.Flags().StringVar(&switchFlags.id, "id", "", "Session UUID to switch")
	switchCmd.Flags().StringVar(&switchFlags.from, "from", "", "Source config directory (default ~/.claude)")
	switchCmd.Flags().StringVar(&switchFlags.to, "to", "", "Target config directory")
	switchCmd.MarkFlagRequired("id")
	switchCmd.MarkFlagRequired("to")

	reprojectCmd.Flags().StringVar(&reprojectFlags.id, "id", "", "Session UUID to reproject")
	reprojectCmd.Flags().StringVarP(&reprojectFlags.configDir, "config-dir", "c", "", "Config directory (default ~/.claude)")
	reprojectCmd.Flags().StringVar(&reprojectFlags.project, "project", "", "Target project path (default: current directory)")
	reprojectCmd.MarkFlagRequired("id")

	findCmd.Flags().StringVar(&findFlags.id, "id", "", "Session UUID to find")
	findCmd.Flags().StringVarP(&findFlags.keyword, "keyword", "k", "", "Regex keyword to search")
	findCmd.Flags().StringSliceVarP(&findFlags.accounts, "accounts", "a", []string{}, "Additional Claude config directories")
	findCmd.Flags().BoolVarP(&findFlags.jsonOutput, "json", "j", false, "Output as JSON")

	ConvosCmd.AddCommand(listCmd, switchCmd, reprojectCmd, findCmd)
}
