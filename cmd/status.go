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
	u "github.com/tanq16/claudex/internal/utils"
)

var statusFlags struct {
	accounts   []string
	separate   bool
	jsonOutput bool
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	valueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	barBgStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current usage for all accounts",
	Run: func(cmd *cobra.Command, args []string) {
		accountPaths := ResolveAccountPaths(statusFlags.accounts)
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

		if statusFlags.separate {
			if statusFlags.jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.Encode(map[string]any{"accounts": accounts})
				return
			}
			renderSeparate(accounts)
		} else {
			if statusFlags.jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.Encode(buildCombinedJSON(accounts))
				return
			}
			renderCombined(accounts)
		}
	},
}

func renderSeparate(accounts []model.AccountUsage) {
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

		renderWindows(acct.FiveHour, acct.SevenDay, acct.SevenDaySonnet)

		if acct.FiveHour != nil && acct.FiveHour.Utilization >= 80 {
			u.PrintGeneric("")
			u.PrintWarn("Approaching 5h limit. Consider switching accounts.", nil)
		} else if acct.FiveHour != nil && acct.FiveHour.Utilization < 10 {
			u.PrintGeneric("")
			u.PrintSuccess("Plenty of capacity available.")
		}
	}
	u.PrintGeneric("")
}

func renderCombined(accounts []model.AccountUsage) {
	var active []model.AccountUsage
	for _, acct := range accounts {
		if acct.TokenExpired {
			email := acct.Account.Email
			if email == "" {
				email = acct.Account.ConfigDir
			}
			u.PrintWarn(fmt.Sprintf("excluding %s (token expired)", email), nil)
			continue
		}
		active = append(active, acct)
	}

	if len(active) == 0 {
		u.PrintFatal("No accounts with valid tokens", nil)
	}

	fiveHour := averageWindow(active, func(a model.AccountUsage) *model.UsageWindow { return a.FiveHour })
	sevenDay := averageWindow(active, func(a model.AccountUsage) *model.UsageWindow { return a.SevenDay })
	sevenDaySonnet := averageWindow(active, func(a model.AccountUsage) *model.UsageWindow { return a.SevenDaySonnet })

	header := titleStyle.Render(fmt.Sprintf("Combined (%d accounts)", len(active)))
	u.PrintGeneric("\n" + header)

	renderWindows(fiveHour, sevenDay, sevenDaySonnet)

	if fiveHour != nil && fiveHour.Utilization >= 80 {
		u.PrintGeneric("")
		u.PrintWarn("Approaching 5h limit. Consider switching accounts.", nil)
	} else if fiveHour != nil && fiveHour.Utilization < 10 {
		u.PrintGeneric("")
		u.PrintSuccess("Plenty of capacity available.")
	}
	u.PrintGeneric("")
}

func renderWindows(fiveHour, sevenDay, sevenDaySonnet *model.UsageWindow) {
	if fiveHour != nil {
		resetStr := formatResetTime(fiveHour.ResetsAt)
		u.PrintGeneric(fmt.Sprintf("  5h Session: %s %s",
			renderBar(fiveHour.Utilization),
			dimStyle.Render(fmt.Sprintf("(%s)", resetStr))))
	}
	if sevenDay != nil {
		resetStr := formatResetTime(sevenDay.ResetsAt)
		u.PrintGeneric(fmt.Sprintf("  7d All:     %s %s",
			renderBar(sevenDay.Utilization),
			dimStyle.Render(fmt.Sprintf("(%s)", resetStr))))
	}
	if sevenDaySonnet != nil {
		resetStr := formatResetTime(sevenDaySonnet.ResetsAt)
		u.PrintGeneric(fmt.Sprintf("  7d Sonnet:  %s %s",
			renderBar(sevenDaySonnet.Utilization),
			dimStyle.Render(fmt.Sprintf("(%s)", resetStr))))
	}
}

func averageWindow(accounts []model.AccountUsage, getter func(model.AccountUsage) *model.UsageWindow) *model.UsageWindow {
	var sum float64
	var count int
	var earliest time.Time

	for _, acct := range accounts {
		w := getter(acct)
		if w == nil {
			continue
		}
		sum += w.Utilization
		count++
		if !w.ResetsAt.IsZero() && (earliest.IsZero() || w.ResetsAt.Before(earliest)) {
			earliest = w.ResetsAt
		}
	}

	if count == 0 {
		return nil
	}

	return &model.UsageWindow{
		Utilization: sum / float64(count),
		ResetsAt:    earliest,
	}
}

func buildCombinedJSON(accounts []model.AccountUsage) map[string]any {
	var active []model.AccountUsage
	for _, acct := range accounts {
		if !acct.TokenExpired {
			active = append(active, acct)
		}
	}

	result := map[string]any{"accountCount": len(active)}

	fiveHour := averageWindow(active, func(a model.AccountUsage) *model.UsageWindow { return a.FiveHour })
	sevenDay := averageWindow(active, func(a model.AccountUsage) *model.UsageWindow { return a.SevenDay })
	sevenDaySonnet := averageWindow(active, func(a model.AccountUsage) *model.UsageWindow { return a.SevenDaySonnet })

	combined := map[string]any{}
	if fiveHour != nil {
		combined["fiveHour"] = fiveHour
	}
	if sevenDay != nil {
		combined["sevenDay"] = sevenDay
	}
	if sevenDaySonnet != nil {
		combined["sevenDaySonnet"] = sevenDaySonnet
	}
	result["combined"] = combined

	return result
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
	statusCmd.Flags().StringSliceVarP(&statusFlags.accounts, "accounts", "a", []string{}, "Additional Claude config directories to monitor")
	statusCmd.Flags().BoolVarP(&statusFlags.separate, "separate", "s", false, "Show each account separately")
	statusCmd.Flags().BoolVarP(&statusFlags.jsonOutput, "json", "j", false, "Output as JSON")
}
