package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/tanq16/claude-usage/internal/model"
	"github.com/tanq16/claude-usage/internal/planner"
	"github.com/tanq16/claude-usage/internal/store"
	"github.com/tanq16/claude-usage/internal/tracker"
	u "github.com/tanq16/claude-usage/internal/utils"
)

var planFlags struct {
	jsonOutput bool
}

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Plan task assignment across accounts based on available capacity",
	Run: func(cmd *cobra.Command, args []string) {
		var accounts []model.AccountUsage
		for _, p := range AccountPaths {
			usage, err := tracker.ComputeAccountUsage(p)
			if err != nil {
				u.PrintWarn(fmt.Sprintf("skipping %s", p), err)
				continue
			}
			accounts = append(accounts, usage)
		}

		if len(accounts) == 0 {
			u.PrintFatal("No accounts found", nil)
		}

		s, err := store.Open(store.DefaultPath())
		if err != nil {
			u.PrintFatal("Failed to open task store", err)
		}
		defer s.Close()

		tasks, err := s.ListTasks()
		if err != nil {
			u.PrintFatal("Failed to list tasks", err)
		}

		suggestions := planner.Plan(accounts, tasks)

		if planFlags.jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(suggestions)
			return
		}

		acctStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13"))

		for _, sg := range suggestions {
			u.PrintGeneric("\n" + acctStyle.Render(sg.AccountEmail))
			u.PrintGeneric(fmt.Sprintf("  Available: ~%.0f%% of 5h window", sg.RemainingPct))

			if len(sg.Tasks) == 0 {
				u.PrintGeneric("  " + dimStyle.Render("No tasks assigned"))
			} else {
				u.PrintGeneric("  Suggested tasks:")
				for _, t := range sg.Tasks {
					est := model.SizeEstimates[t.Size]
					u.PrintGeneric(fmt.Sprintf("    - %s [%s] (~%d turns, ~%.0f%% capacity)", t.Name, t.Size, est.Messages, est.Percent5H))
				}
			}

			if sg.Warning != "" {
				u.PrintWarn(sg.Warning, nil)
			}
		}
		u.PrintGeneric("")
	},
}

func init() {
	planCmd.Flags().BoolVar(&planFlags.jsonOutput, "json", false, "Output as JSON")
}
