package server

import (
	"cmp"
	"context"
	"encoding/json/v2"
	"net/http"
	"os/exec"
	"path"
	"slices"
	"strings"
	"time"
)

const prTimeout = 10 * time.Second

var passingChecks = []string{"SUCCESS", "NEUTRAL", "SKIPPED"}

type gitStatus struct {
	Branch   string       `json:"branch,omitzero"`
	Detached bool         `json:"detached,omitzero"`
	PR       *pullRequest `json:"pr,omitzero"`
}

type checkCounts struct {
	Total   int `json:"total"`
	Failed  int `json:"failed"`
	Pending int `json:"pending"`
}

type pullRequest struct {
	Number int         `json:"number"`
	Title  string      `json:"title"`
	URL    string      `json:"url"`
	State  string      `json:"state"`
	Draft  bool        `json:"draft"`
	Checks checkCounts `json:"checks"`
}

func (s *Server) handleGit(r *http.Request) (any, error) {
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	dir, _, err := s.tracked(st, r.PathValue("name"))
	if err != nil {
		return nil, err
	}
	var g gitStatus
	out, err := exec.CommandContext(r.Context(), "git", "-C", dir, "branch", "--show-current").Output()
	if err != nil {
		return g, nil
	}
	if g.Branch = strings.TrimSpace(string(out)); g.Branch != "" {
		g.PR = pullRequestOf(r.Context(), dir)
		return g, nil
	}
	if out, err := exec.CommandContext(r.Context(), "git", "-C", dir, "rev-parse", "--short", "HEAD").Output(); err == nil {
		g.Branch, g.Detached = strings.TrimSpace(string(out)), true
	}
	return g, nil
}

var changeRank = map[string]int{"U": 1, "A": 2, "D": 3, "M": 4}

func changeOf(x, y byte) string {
	switch {
	case x == '?':
		return "U"
	case x == 'D' || y == 'D':
		return "D"
	case x == 'A':
		return "A"
	default:
		return "M"
	}
}

func uncommittedChanges(ctx context.Context, dir string) map[string]string {
	prefix, err := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--show-prefix").Output()
	if err != nil {
		return nil
	}
	out, err := exec.CommandContext(ctx, "git", "-C", dir, "status", "--porcelain", "-z", "--untracked-files=all", "--no-renames", "--", ".").Output()
	if err != nil {
		return nil
	}
	changes := map[string]string{}
	for entry := range strings.SplitSeq(string(out), "\x00") {
		if len(entry) < 4 {
			continue
		}
		rel, ok := strings.CutPrefix(entry[3:], strings.TrimSpace(string(prefix)))
		if !ok {
			continue
		}
		change := changeOf(entry[0], entry[1])
		changes[rel] = change
		for d := path.Dir(rel); d != "."; d = path.Dir(d) {
			if changeRank[change] > changeRank[changes[d]] {
				changes[d] = change
			}
		}
	}
	return changes
}

func pullRequestOf(ctx context.Context, dir string) *pullRequest {
	ctx, cancel := context.WithTimeout(ctx, prTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "pr", "view", "--json", "number,title,url,state,isDraft,statusCheckRollup")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var v struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		URL     string `json:"url"`
		State   string `json:"state"`
		IsDraft bool   `json:"isDraft"`
		Rollup  []struct {
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
			State      string `json:"state"`
		} `json:"statusCheckRollup"`
	}
	if json.Unmarshal(out, &v) != nil {
		return nil
	}
	pr := &pullRequest{Number: v.Number, Title: v.Title, URL: v.URL, State: v.State, Draft: v.IsDraft}
	for _, c := range v.Rollup {
		pr.Checks.Total++
		switch {
		case c.Status != "" && c.Status != "COMPLETED", c.State == "PENDING", c.State == "EXPECTED":
			pr.Checks.Pending++
		case !slices.Contains(passingChecks, cmp.Or(c.Conclusion, c.State)):
			pr.Checks.Failed++
		}
	}
	return pr
}
