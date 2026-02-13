package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tanq16/claude-usage/internal/model"
	"github.com/tanq16/claude-usage/internal/parser"
	u "github.com/tanq16/claude-usage/internal/utils"
)

var historyFlags struct {
	days       int
	jsonOutput bool
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show daily usage history from stats-cache",
	Run: func(cmd *cobra.Command, args []string) {
		for _, p := range AccountPaths {
			acct, _ := parser.ParseAccount(p)
			stats, err := parser.ParseStatsCache(p)
			if err != nil {
				u.PrintWarn(fmt.Sprintf("skipping %s", p), err)
				continue
			}

			email := acct.Email
			if email == "" {
				email = p
			}

			activity := tailActivity(stats.DailyActivity, historyFlags.days)
			tokens := tailTokens(stats.DailyModelTokens, historyFlags.days)

			if historyFlags.jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.Encode(map[string]any{
					"account":  email,
					"activity": activity,
					"tokens":   tokens,
				})
				continue
			}

			u.PrintGeneric("")
			u.PrintInfo(email)

			rows := make([][]string, len(activity))
			for i, d := range activity {
				rows[i] = []string{
					d.Date,
					fmt.Sprintf("%d", d.MessageCount),
					fmt.Sprintf("%d", d.SessionCount),
					fmt.Sprintf("%d", d.ToolCallCount),
				}
			}
			u.PrintTable([]string{"Date", "Messages", "Sessions", "Tools"}, rows)

			var tokenRows [][]string
			for _, d := range tokens {
				for m, t := range d.TokensByModel {
					tokenRows = append(tokenRows, []string{d.Date, m, fmt.Sprintf("%d", t)})
				}
			}
			if len(tokenRows) > 0 {
				u.PrintGeneric("")
				u.PrintTable([]string{"Date", "Model", "Tokens"}, tokenRows)
			}
		}
		u.PrintGeneric("")
	},
}

func tailActivity(a []model.DailyActivity, n int) []model.DailyActivity {
	if len(a) <= n {
		return a
	}
	return a[len(a)-n:]
}

func tailTokens(a []model.DailyModelTokens, n int) []model.DailyModelTokens {
	if len(a) <= n {
		return a
	}
	return a[len(a)-n:]
}

func init() {
	historyCmd.Flags().IntVarP(&historyFlags.days, "days", "d", 7, "Number of days to show")
	historyCmd.Flags().BoolVar(&historyFlags.jsonOutput, "json", false, "Output as JSON")
}
