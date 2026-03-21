package convo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/tanq16/claudex/internal/model"
)

// RawHistoryEntry preserves the original JSONL line for lossless write-back.
type RawHistoryEntry struct {
	Parsed model.HistoryEntry
	Raw    []byte
}

// SessionFiles describes all files belonging to a conversation session.
type SessionFiles struct {
	ConfigDir    string
	ProjectDir   string
	ProjectPath  string
	SessionJSONL string
	SubAgentDir  string
}

// SearchResult holds a keyword search match aggregated by session.
type SearchResult struct {
	SessionID  string `json:"sessionId"`
	Project    string `json:"project"`
	ConfigDir  string `json:"configDir"`
	MatchCount int    `json:"matchCount"`
	Sample     string `json:"sample"`
}

// EncodeProjectPath converts an absolute path to the directory name used by Claude.
// e.g., "/Users/foo/bar" -> "-Users-foo-bar"
func EncodeProjectPath(absPath string) string {
	return strings.ReplaceAll(absPath, "/", "-")
}

// DecodeProjectPath converts an encoded directory name back to an absolute path.
// e.g., "-Users-foo-bar" -> "/Users/foo/bar"
func DecodeProjectPath(encoded string) string {
	return strings.ReplaceAll(encoded, "-", "/")
}

// ProjectDir returns the full path to a project's conversation directory.
func ProjectDir(configDir, projectPath string) string {
	return filepath.Join(configDir, "projects", EncodeProjectPath(projectPath))
}

// --- History JSONL operations ---

// ReadRawHistory reads all entries from history.jsonl, preserving raw bytes.
func ReadRawHistory(configDir string) ([]RawHistoryEntry, error) {
	path := filepath.Join(configDir, "history.jsonl")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []RawHistoryEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		raw := make([]byte, len(scanner.Bytes()))
		copy(raw, scanner.Bytes())

		var entry model.HistoryEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue
		}
		entries = append(entries, RawHistoryEntry{Parsed: entry, Raw: raw})
	}

	return entries, scanner.Err()
}

// WriteRawHistory atomically overwrites history.jsonl.
func WriteRawHistory(configDir string, entries []RawHistoryEntry) error {
	path := filepath.Join(configDir, "history.jsonl")
	tmp := path + ".tmp"

	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(f)
	for _, e := range entries {
		w.Write(e.Raw)
		w.WriteByte('\n')
	}

	if err := w.Flush(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}

	return os.Rename(tmp, path)
}

// AppendRawHistory appends entries to history.jsonl.
func AppendRawHistory(configDir string, entries []RawHistoryEntry) error {
	path := filepath.Join(configDir, "history.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, e := range entries {
		w.Write(e.Raw)
		w.WriteByte('\n')
	}
	return w.Flush()
}

// FilterBySession partitions entries into matching and non-matching for a session ID.
func FilterBySession(entries []RawHistoryEntry, sessionID string) (matching, rest []RawHistoryEntry) {
	for _, e := range entries {
		if e.Parsed.SessionID == sessionID {
			matching = append(matching, e)
		} else {
			rest = append(rest, e)
		}
	}
	return
}

// --- Session file discovery ---

// FindSession searches for a session's files in a specific config dir.
func FindSession(configDir, sessionID string) (*SessionFiles, error) {
	projectsDir := filepath.Join(configDir, "projects")
	dirEntries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	jsonlName := sessionID + ".jsonl"

	for _, de := range dirEntries {
		if !de.IsDir() {
			continue
		}
		projDir := filepath.Join(projectsDir, de.Name())

		sf := &SessionFiles{
			ConfigDir:   configDir,
			ProjectDir:  projDir,
			ProjectPath: DecodeProjectPath(de.Name()),
		}

		jsonlPath := filepath.Join(projDir, jsonlName)
		if _, err := os.Stat(jsonlPath); err == nil {
			sf.SessionJSONL = jsonlPath
		}

		subDir := filepath.Join(projDir, sessionID)
		if info, err := os.Stat(subDir); err == nil && info.IsDir() {
			sf.SubAgentDir = subDir
		}

		if sf.SessionJSONL != "" || sf.SubAgentDir != "" {
			return sf, nil
		}
	}

	return nil, nil
}

// FindSessionAllAccounts searches across multiple config dirs.
func FindSessionAllAccounts(configDirs []string) func(sessionID string) (*SessionFiles, error) {
	return func(sessionID string) (*SessionFiles, error) {
		for _, dir := range configDirs {
			sf, err := FindSession(dir, sessionID)
			if err != nil {
				return nil, err
			}
			if sf != nil {
				return sf, nil
			}
		}
		return nil, nil
	}
}

// --- Move operations ---

// MoveSession moves session files from srcProjectDir to dstProjectDir.
func MoveSession(sessionID, srcProjectDir, dstProjectDir string) error {
	if err := os.MkdirAll(dstProjectDir, 0700); err != nil {
		return fmt.Errorf("creating target directory: %w", err)
	}

	jsonlName := sessionID + ".jsonl"
	srcJSONL := filepath.Join(srcProjectDir, jsonlName)
	dstJSONL := filepath.Join(dstProjectDir, jsonlName)

	if _, err := os.Stat(dstJSONL); err == nil {
		return fmt.Errorf("session %s already exists in target directory", sessionID)
	}

	if _, err := os.Stat(srcJSONL); err == nil {
		if err := moveFileOrDir(srcJSONL, dstJSONL); err != nil {
			return fmt.Errorf("moving session JSONL: %w", err)
		}
	}

	srcSubDir := filepath.Join(srcProjectDir, sessionID)
	dstSubDir := filepath.Join(dstProjectDir, sessionID)
	if info, err := os.Stat(srcSubDir); err == nil && info.IsDir() {
		if err := moveFileOrDir(srcSubDir, dstSubDir); err != nil {
			return fmt.Errorf("moving sub-agent directory: %w", err)
		}
	}

	return nil
}

// moveFileOrDir tries os.Rename first, falls back to copy+remove for cross-filesystem.
func moveFileOrDir(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// Fall back to copy for cross-filesystem moves
	if linkErr, ok := err.(*os.LinkError); ok && linkErr.Err.Error() == "cross-device link" {
		return copyAndRemove(src, dst)
	}
	return err
}

func copyAndRemove(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDirAndRemove(src, dst)
	}

	return copyFileAndRemove(src, dst, info.Mode())
}

func copyFileAndRemove(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	in.Close()
	return os.Remove(src)
}

func copyDirAndRemove(src, dst string) error {
	if err := os.MkdirAll(dst, 0700); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())

		info, err := e.Info()
		if err != nil {
			return err
		}

		if e.IsDir() {
			if err := copyDirAndRemove(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFileAndRemove(srcPath, dstPath, info.Mode()); err != nil {
				return err
			}
		}
	}

	return os.Remove(src)
}

// --- Search ---

// SearchHistory searches history.jsonl display fields by regex across accounts.
func SearchHistory(configDirs []string, pattern string) ([]SearchResult, error) {
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}

	type sessionKey struct {
		sessionID string
		configDir string
	}

	type agg struct {
		project    string
		matchCount int
		sample     string
	}

	results := make(map[sessionKey]*agg)

	for _, dir := range configDirs {
		entries, err := ReadRawHistory(dir)
		if err != nil {
			continue
		}

		for _, e := range entries {
			if e.Parsed.Display == "" {
				continue
			}
			if !re.MatchString(e.Parsed.Display) {
				continue
			}

			key := sessionKey{sessionID: e.Parsed.SessionID, configDir: dir}
			a, ok := results[key]
			if !ok {
				a = &agg{
					project: filepath.Base(e.Parsed.Project),
					sample:  e.Parsed.Display,
				}
				results[key] = a
			}
			a.matchCount++
		}
	}

	out := make([]SearchResult, 0, len(results))
	for key, a := range results {
		out = append(out, SearchResult{
			SessionID:  key.sessionID,
			Project:    a.project,
			ConfigDir:  key.configDir,
			MatchCount: a.matchCount,
			Sample:     a.sample,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].MatchCount > out[j].MatchCount
	})

	return out, nil
}

// UpdateProjectInHistory updates the project field for a session's history entries.
func UpdateProjectInHistory(configDir, sessionID, newProject string) error {
	entries, err := ReadRawHistory(configDir)
	if err != nil {
		return err
	}

	var modified []RawHistoryEntry
	for _, e := range entries {
		if e.Parsed.SessionID == sessionID {
			var m map[string]any
			if err := json.Unmarshal(e.Raw, &m); err != nil {
				modified = append(modified, e)
				continue
			}
			m["project"] = newProject
			raw, err := json.Marshal(m)
			if err != nil {
				modified = append(modified, e)
				continue
			}
			e.Raw = raw
			e.Parsed.Project = newProject
		}
		modified = append(modified, e)
	}

	return WriteRawHistory(configDir, modified)
}
