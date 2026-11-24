package convo

import (
	"bytes"
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

var lenient = json.JoinOptions(jsontext.AllowInvalidUTF8(true), jsontext.AllowDuplicateNames(true))

type Message struct {
	Type               string         `json:"type"`
	UUID               string         `json:"uuid"`
	SessionID          string         `json:"session_id"`
	Timestamp          string         `json:"timestamp,omitzero"`
	ParentToolUseID    *string        `json:"parent_tool_use_id"`
	Message            jsontext.Value `json:"message"`
	ToolUseResult      jsontext.Value `json:"tool_use_result,omitzero"`
	ThinkingDurationMs int64          `json:"thinking_duration_ms,omitzero"`
	IsCompactSummary   bool           `json:"isCompactSummary,omitzero"`
}

type Conversation struct {
	SessionID string    `json:"sessionId"`
	Messages  []Message `json:"messages"`
}

type transcriptEntry struct {
	Type               string         `json:"type"`
	UUID               string         `json:"uuid"`
	ParentUUID         string         `json:"parentUuid"`
	SessionID          string         `json:"sessionId"`
	Timestamp          string         `json:"timestamp"`
	IsSidechain        bool           `json:"isSidechain"`
	IsMeta             bool           `json:"isMeta"`
	IsCompactSummary   bool           `json:"isCompactSummary"`
	Message            jsontext.Value `json:"message"`
	ToolUseResult      jsontext.Value `json:"toolUseResult"`
	ThinkingDurationMs int64          `json:"thinkingDurationMs"`
}

func History(configDir, projectPath, before string) (Conversation, error) {
	conv := Conversation{Messages: []Message{}}
	files, err := transcripts(ProjectDir(configDir, projectPath))
	if err != nil || len(files) == 0 {
		return conv, err
	}
	path, data, err := holding(files, before)
	if err != nil || path == "" {
		return conv, err
	}
	conv.SessionID = strings.TrimSuffix(filepath.Base(path), ".jsonl")
	chain := mainChain(data)
	if before != "" {
		i := slices.IndexFunc(chain, func(e transcriptEntry) bool { return e.UUID == before })
		if i < 0 {
			return conv, nil
		}
		chain = chain[:i]
	}
	for _, e := range chain {
		if (e.Type != "user" && e.Type != "assistant") || e.IsMeta {
			continue
		}
		conv.Messages = append(conv.Messages, Message{
			Type:               e.Type,
			UUID:               e.UUID,
			SessionID:          cmp.Or(e.SessionID, conv.SessionID),
			Timestamp:          e.Timestamp,
			Message:            e.Message,
			ToolUseResult:      e.ToolUseResult,
			ThinkingDurationMs: e.ThinkingDurationMs,
			IsCompactSummary:   e.IsCompactSummary,
		})
	}
	return conv, nil
}

func holding(files []string, anchor string) (string, []byte, error) {
	if anchor == "" {
		data, err := os.ReadFile(files[0])
		return files[0], data, err
	}
	for _, f := range files {
		if data, err := os.ReadFile(f); err == nil && bytes.Contains(data, []byte(anchor)) {
			return f, data, nil
		}
	}
	return "", nil, nil
}

func transcripts(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	type file struct {
		path string
		mod  time.Time
	}
	var files []file
	for _, e := range entries {
		if !e.Type().IsRegular() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		if info, err := e.Info(); err == nil {
			files = append(files, file{filepath.Join(dir, e.Name()), info.ModTime()})
		}
	}
	slices.SortFunc(files, func(a, b file) int { return b.mod.Compare(a.mod) })
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.path
	}
	return paths, nil
}

func mainChain(data []byte) []transcriptEntry {
	var entries []transcriptEntry
	index := map[string]int{}
	for line := range bytes.SplitSeq(data, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var e transcriptEntry
		if json.Unmarshal(line, &e, lenient) != nil || e.UUID == "" || e.IsSidechain {
			continue
		}
		index[e.UUID] = len(entries)
		entries = append(entries, e)
	}
	var chain []transcriptEntry
	for i, seen := len(entries)-1, 0; i >= 0 && seen <= len(entries); seen++ {
		chain = append(chain, entries[i])
		next, ok := index[entries[i].ParentUUID]
		if !ok {
			break
		}
		i = next
	}
	slices.Reverse(chain)
	return chain
}
