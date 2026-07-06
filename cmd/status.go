package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/model"
	"github.com/tanq16/claudex/internal/tracker"
	u "github.com/tanq16/claudex/utils"
)

var statusFlags struct {
	account    string
	jsonOutput bool
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(13))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	valueStyle  = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(15))
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(10))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(11))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(9))
	barBgStyle  = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current usage for all accounts",
	Run: func(cmd *cobra.Command, args []string) {
		accountPaths := u.ResolveAccountPaths(statusFlags.account)
		var accounts []model.AccountUsage
		for _, p := range accountPaths {
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
		renderStatus(accounts)
	},
}

func renderStatus(accounts []model.AccountUsage) {
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

		if acct.APIError != "" {
			u.PrintWarn(fmt.Sprintf("API error: %s", acct.APIError), nil)
			continue
		}

		renderWindows(acct.Windows)

		if s := sessionWindow(acct.Windows); s != nil && s.Utilization >= 80 {
			u.PrintGeneric("")
			u.PrintWarn("Approaching 5h limit. Consider switching accounts.", nil)
		} else if s != nil && s.Utilization < 10 {
			u.PrintGeneric("")
			u.PrintSuccess("Plenty of capacity available.")
		}
	}
	u.PrintGeneric("")
}

func renderWindows(windows []model.UsageWindow) {
	labels := make([]string, len(windows))
	width := 0
	for i, w := range windows {
		labels[i] = windowLabel(w)
		width = max(width, len(labels[i]))
	}
	for i, w := range windows {
		resetStr := formatResetTime(w.ResetsAt)
		u.PrintGeneric(fmt.Sprintf("  %-*s %s %s",
			width+1, labels[i]+":",
			renderBar(w.Utilization),
			dimStyle.Render(fmt.Sprintf("(%s)", resetStr))))
	}
}

func windowLabel(w model.UsageWindow) string {
	switch w.Kind {
	case "session":
		return "5h Session"
	case "weekly_all":
		return "7d All"
	case "weekly_scoped":
		if w.Scope != "" {
			return "7d " + w.Scope
		}
		return "7d Scoped"
	default:
		return w.Kind
	}
}

func sessionWindow(windows []model.UsageWindow) *model.UsageWindow {
	for i := range windows {
		if windows[i].Kind == "session" {
			return &windows[i]
		}
	}
	return nil
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
	statusCmd.Flags().StringVarP(&statusFlags.account, "account", "A", "", "Use only this specific account directory (default: all discovered accounts)")
	statusCmd.Flags().BoolVarP(&statusFlags.jsonOutput, "json", "j", false, "Output as JSON")
}
