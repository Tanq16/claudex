package convo

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"
)

func SessionIDs(configDir, projectPath string) ([]string, error) {
	entries, err := os.ReadDir(ProjectDir(configDir, projectPath))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if id, ok := strings.CutSuffix(e.Name(), ".jsonl"); ok && e.Type().IsRegular() {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func LastActivity(configDir, projectPath string) time.Time {
	entries, err := os.ReadDir(ProjectDir(configDir, projectPath))
	if err != nil {
		return time.Time{}
	}
	var latest time.Time
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		if info, err := e.Info(); err == nil && info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	return latest
}

func MoveProject(projectPath, srcConfigDir, dstConfigDir string) (int, error) {
	ids, err := SessionIDs(srcConfigDir, projectPath)
	if err != nil || len(ids) == 0 {
		return 0, err
	}
	srcDir := ProjectDir(srcConfigDir, projectPath)
	dstDir := ProjectDir(dstConfigDir, projectPath)
	for _, id := range ids {
		if err := MoveSession(id, srcDir, dstDir); err != nil {
			return 0, err
		}
	}

	entries, err := ReadRawHistory(srcConfigDir)
	if err != nil {
		return len(ids), fmt.Errorf("reading source history: %w", err)
	}
	var moving, rest []RawHistoryEntry
	for _, e := range entries {
		if slices.Contains(ids, e.Parsed.SessionID) {
			moving = append(moving, e)
		} else {
			rest = append(rest, e)
		}
	}
	if len(moving) == 0 {
		return len(ids), nil
	}
	if err := AppendRawHistory(dstConfigDir, moving); err != nil {
		return len(ids), err
	}
	if err := WriteRawHistory(srcConfigDir, rest); err != nil {
		return len(ids), err
	}
	return len(ids), nil
}
