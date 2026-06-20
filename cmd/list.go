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

var listFlags struct {
	account    string
	limit      int
	jsonOutput bool
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent conversations across accounts",
	Run:   runList,
}

func runList(cmd *cobra.Command, args []string) {
	accountPaths := u.ResolveAccountPaths(listFlags.account)
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
			snippet := u.Truncate(c.FirstMessage, 40)
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

func init() {
	listCmd.Flags().StringVarP(&listFlags.account, "account", "A", "", "Use only this specific account directory (default: all discovered accounts)")
	listCmd.Flags().IntVarP(&listFlags.limit, "limit", "n", 10, "Number of conversations to show")
	listCmd.Flags().BoolVarP(&listFlags.jsonOutput, "json", "j", false, "Output as JSON")
}
