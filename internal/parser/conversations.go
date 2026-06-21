package parser

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/tanq16/claudex/internal/model"
)

func ParseConversations(configDir string) ([]model.Conversation, error) {
	f, err := os.Open(filepath.Join(configDir, "history.jsonl"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	type sessionAgg struct {
		messageCount int
		project      string
		projectPath  string
		firstMessage string
		firstTS      int64
		lastTS       int64
	}

	sessions := make(map[string]*sessionAgg)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var entry model.HistoryEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if entry.SessionID == "" {
			continue
		}

		agg, ok := sessions[entry.SessionID]
		if !ok {
			agg = &sessionAgg{
				firstTS:      entry.Timestamp,
				firstMessage: entry.Display,
				project:      filepath.Base(entry.Project),
				projectPath:  entry.Project,
			}
			sessions[entry.SessionID] = agg
		}

		agg.messageCount++

		if entry.Timestamp < agg.firstTS {
			agg.firstTS = entry.Timestamp
			agg.firstMessage = entry.Display
		}
		if entry.Timestamp > agg.lastTS {
			agg.lastTS = entry.Timestamp
			if entry.Project != "" {
				agg.project = filepath.Base(entry.Project)
				agg.projectPath = entry.Project
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	convos := make([]model.Conversation, 0, len(sessions))
	for sid, agg := range sessions {
		convos = append(convos, model.Conversation{
			SessionID:    sid,
			MessageCount: agg.messageCount,
			Project:      agg.project,
			ProjectPath:  agg.projectPath,
			FirstMessage: agg.firstMessage,
			LastActivity: agg.lastTS,
		})
	}

	sort.Slice(convos, func(i, j int) bool {
		return convos[i].LastActivity > convos[j].LastActivity
	})

	return convos, nil
}
