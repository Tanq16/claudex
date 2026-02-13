package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tanq16/claude-usage/internal/model"
	"github.com/tanq16/claude-usage/internal/tracker"
	u "github.com/tanq16/claude-usage/internal/utils"
)

var statusFlags struct {
	jsonOutput bool
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cba6f7"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
	valueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f9e2af"))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
	barBgStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a"))
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current usage for all accounts",
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

		if statusFlags.jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(map[string]any{"accounts": accounts})
			return
		}

		for _, acct := range accounts {
			email := acct.Account.Email
			if email == "" {
				email = acct.Account.ConfigDir
			}
			org := acct.Account.Organization
			header := titleStyle.Render(email)
			if org != "" {
				header += dimStyle.Render(fmt.Sprintf(" (%s)", org))
			}
			u.PrintGeneric("\n" + header)

			if acct.TokenExpired {
				u.PrintWarn("OAuth token expired - launch Claude Code on this account to refresh", nil)
				continue
			}

			// 5-hour window
			if acct.FiveHour != nil {
				w := acct.FiveHour
				resetStr := formatResetTime(w.ResetsAt)
				u.PrintGeneric(fmt.Sprintf("  5h Session: %s %s",
					renderBar(w.Utilization),
					dimStyle.Render(fmt.Sprintf("(%s)", resetStr))))
			}

			// 7-day window
			if acct.SevenDay != nil {
				w := acct.SevenDay
				resetStr := formatResetTime(w.ResetsAt)
				u.PrintGeneric(fmt.Sprintf("  7d All:     %s %s",
					renderBar(w.Utilization),
					dimStyle.Render(fmt.Sprintf("(%s)", resetStr))))
			}

			// Sonnet-specific
			if acct.SevenDaySonnet != nil {
				w := acct.SevenDaySonnet
				resetStr := formatResetTime(w.ResetsAt)
				u.PrintGeneric(fmt.Sprintf("  7d Sonnet:  %s %s",
					renderBar(w.Utilization),
					dimStyle.Render(fmt.Sprintf("(%s)", resetStr))))
			}

			// Recommendation
			if acct.FiveHour != nil && acct.FiveHour.Utilization >= 80 {
				u.PrintGeneric("")
				u.PrintWarn("Approaching 5h limit. Consider switching accounts.", nil)
			} else if acct.FiveHour != nil && acct.FiveHour.Utilization < 10 {
				u.PrintGeneric("")
				u.PrintSuccess("Plenty of capacity available.")
			}
		}
		u.PrintGeneric("")
	},
}

func formatResetTime(t time.Time) string {
	if t.IsZero() {
		return "no reset"
	}
	d := time.Until(t)
	if d <= 0 {
		return "resetting"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 24 {
		return fmt.Sprintf("resets %s", t.Local().Format("Mon 3pm"))
	}
	if h > 0 {
		return fmt.Sprintf("resets in %dh%dm", h, m)
	}
	return fmt.Sprintf("resets in %dm", m)
}

func renderBar(pct float64) string {
	if pct > 100 {
		pct = 100
	}
	if pct < 0 {
		pct = 0
	}
	filled := int(pct / 5)
	empty := 20 - filled

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	bg := ""
	for i := 0; i < empty; i++ {
		bg += "░"
	}

	var colorStyle lipgloss.Style
	if pct >= 90 {
		colorStyle = redStyle
	} else if pct >= 70 {
		colorStyle = yellowStyle
	} else {
		colorStyle = greenStyle
	}

	return colorStyle.Render(bar) + barBgStyle.Render(bg) + valueStyle.Render(fmt.Sprintf(" %3.0f%%", pct))
}

func init() {
	statusCmd.Flags().BoolVar(&statusFlags.jsonOutput, "json", false, "Output as JSON")
}
