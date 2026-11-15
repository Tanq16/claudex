package model

type AccountInfo struct {
	Email        string `json:"email"`
	Organization string `json:"organization"`
	DisplayName  string `json:"displayName"`
	ConfigDir    string `json:"configDir"`
}

type HistoryEntry struct {
	Display   string `json:"display"`
	Timestamp int64  `json:"timestamp"`
	Project   string `json:"project"`
	SessionID string `json:"sessionId"`
}
