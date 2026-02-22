package model

import "time"

type AccountInfo struct {
	Email        string `json:"email"`
	Organization string `json:"organization"`
	DisplayName  string `json:"displayName"`
	ConfigDir    string `json:"configDir"`
}

type UsageWindow struct {
	Utilization float64   `json:"utilization"`
	ResetsAt    time.Time `json:"resetsAt"`
}

type AccountUsage struct {
	Account          AccountInfo `json:"account"`
	FiveHour         *UsageWindow `json:"fiveHour,omitempty"`
	SevenDay         *UsageWindow `json:"sevenDay,omitempty"`
	SevenDaySonnet   *UsageWindow `json:"sevenDaySonnet,omitempty"`
	TokenExpired     bool         `json:"tokenExpired,omitempty"`
}

type DailyActivity struct {
	Date          string `json:"date"`
	MessageCount  int    `json:"messageCount"`
	SessionCount  int    `json:"sessionCount"`
	ToolCallCount int    `json:"toolCallCount"`
}

type DailyModelTokens struct {
	Date          string         `json:"date"`
	TokensByModel map[string]int `json:"tokensByModel"`
}

type StatsCache struct {
	Version          int                `json:"version"`
	LastComputedDate string             `json:"lastComputedDate"`
	DailyActivity    []DailyActivity    `json:"dailyActivity"`
	DailyModelTokens []DailyModelTokens `json:"dailyModelTokens"`
}

type TaskSize string

const (
	TaskSizeS  TaskSize = "S"
	TaskSizeM  TaskSize = "M"
	TaskSizeL  TaskSize = "L"
	TaskSizeXL TaskSize = "XL"
)

type TaskSizeEstimate struct {
	Messages int `json:"messages"`
	Percent5H float64 `json:"percent5h"`
}

// Size estimates as approximate percentage of 5h window
var SizeEstimates = map[TaskSize]TaskSizeEstimate{
	TaskSizeS:  {Messages: 10, Percent5H: 5},
	TaskSizeM:  {Messages: 30, Percent5H: 15},
	TaskSizeL:  {Messages: 75, Percent5H: 35},
	TaskSizeXL: {Messages: 150, Percent5H: 70},
}

type Task struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Size      TaskSize  `json:"size"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"createdAt"`
}

type PlanSuggestion struct {
	AccountEmail   string  `json:"accountEmail"`
	Tasks          []Task  `json:"tasks"`
	RemainingPct   float64 `json:"remainingPct"`
	Warning        string  `json:"warning,omitempty"`
}

type HistoryEntry struct {
	Display   string `json:"display"`
	Timestamp int64  `json:"timestamp"`
	Project   string `json:"project"`
	SessionID string `json:"sessionId"`
}

type Conversation struct {
	SessionID    string `json:"sessionId"`
	MessageCount int    `json:"messageCount"`
	Project      string `json:"project"`
	FirstMessage string `json:"firstMessage"`
	LastActivity int64  `json:"lastActivity"`
}
