package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/parser"
	u "github.com/tanq16/claudex/utils"
)

var convosFlags struct {
	accounts   []string
	limit      int
	jsonOutput bool
}

var convosCmd = &cobra.Command{
	Use:     "conversations",
	Aliases: []string{"convos"},
	Short:   "List recent conversations across accounts",
	Run: func(cmd *cobra.Command, args []string) {
		accountPaths := ResolveAccountPaths(convosFlags.accounts)
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

			if convosFlags.limit > 0 && len(convos) > convosFlags.limit {
				convos = convos[:convosFlags.limit]
			}

			if convosFlags.jsonOutput {
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
	},
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

func init() {
	convosCmd.Flags().StringSliceVarP(&convosFlags.accounts, "accounts", "a", []string{}, "Additional Claude config directories to monitor")
	convosCmd.Flags().IntVarP(&convosFlags.limit, "limit", "n", 10, "Number of conversations to show")
	convosCmd.Flags().BoolVarP(&convosFlags.jsonOutput, "json", "j", false, "Output as JSON")
}
