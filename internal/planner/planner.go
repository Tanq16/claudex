package planner

import (
	"sort"

	"github.com/tanq16/claude-usage/internal/model"
)

func Plan(accounts []model.AccountUsage, tasks []model.Task) []model.PlanSuggestion {
	var pending []model.Task
	for _, t := range tasks {
		if !t.Done {
			pending = append(pending, t)
		}
	}

	// Sort smallest-first for greedy bin-pack
	sizeOrder := map[model.TaskSize]int{
		model.TaskSizeS: 0, model.TaskSizeM: 1, model.TaskSizeL: 2, model.TaskSizeXL: 3,
	}
	sort.Slice(pending, func(i, j int) bool {
		return sizeOrder[pending[i].Size] < sizeOrder[pending[j].Size]
	})

	var suggestions []model.PlanSuggestion

	for _, acct := range accounts {
		remainingPct := 100.0
		if acct.FiveHour != nil {
			remainingPct = 100.0 - acct.FiveHour.Utilization
		}
		if remainingPct < 0 {
			remainingPct = 0
		}

		suggestion := model.PlanSuggestion{
			AccountEmail: acct.Account.Email,
			RemainingPct: remainingPct,
		}

		if acct.TokenExpired {
			suggestion.Warning = "OAuth token expired - launch Claude Code on this account to refresh"
			suggestions = append(suggestions, suggestion)
			continue
		}

		pctLeft := remainingPct
		var assigned []model.Task
		remaining := make([]model.Task, len(pending))
		copy(remaining, pending)

		for i := 0; i < len(remaining); {
			est := model.SizeEstimates[remaining[i].Size]
			if est.Percent5H <= pctLeft {
				task := remaining[i]
				if est.Percent5H > pctLeft*0.8 {
					suggestion.Warning = "some tasks are a tight fit for remaining capacity"
				}
				assigned = append(assigned, task)
				pctLeft -= est.Percent5H
				remaining = append(remaining[:i], remaining[i+1:]...)
			} else {
				i++
			}
		}

		suggestion.Tasks = assigned
		suggestions = append(suggestions, suggestion)
		pending = remaining
	}

	return suggestions
}
