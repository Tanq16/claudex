package model

import "time"

type AccountInfo struct {
	Email        string `json:"email"`
	Organization string `json:"organization"`
	DisplayName  string `json:"displayName"`
	ConfigDir    string `json:"configDir"`
}

type UsageWindow struct {
	Kind        string    `json:"kind"`
	Scope       string    `json:"scope,omitempty"`
	Utilization float64   `json:"utilization"`
	ResetsAt    time.Time `json:"resetsAt"`
}

type AccountUsage struct {
	Account      AccountInfo   `json:"account"`
	Windows      []UsageWindow `json:"windows,omitempty"`
	TokenExpired bool          `json:"tokenExpired,omitempty"`
	APIError     string        `json:"apiError,omitempty"`
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
	ProjectPath  string `json:"projectPath"`
	FirstMessage string `json:"firstMessage"`
	LastActivity int64  `json:"lastActivity"`
}
