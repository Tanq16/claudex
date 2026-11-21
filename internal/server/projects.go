package server

import (
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tanq16/claudex/internal/convo"
	"github.com/tanq16/claudex/internal/embedded"
	"github.com/tanq16/claudex/internal/mcp"
	"github.com/tanq16/claudex/internal/paths"
	"github.com/tanq16/claudex/internal/session"
	"github.com/tanq16/claudex/internal/state"
	"github.com/tanq16/claudex/internal/workspace"
)

var (
	newDirName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	repoName   = regexp.MustCompile(`^[\w.-]+/[\w.-]+$`)
)

type projectView struct {
	Dir        string   `json:"dir"`
	Name       string   `json:"name"`
	RCName     string   `json:"rcName"`
	Account    string   `json:"account"`
	Active     bool     `json:"active"`
	StartedAt  *int64   `json:"startedAt"`
	LastActive *int64   `json:"lastActive"`
	Presets    []string `json:"presets"`
	MCP        []string `json:"mcp"`
	Connectors bool     `json:"connectors"`
	Stash      bool     `json:"stash"`
}

type mcpSummary struct {
	Name      string `json:"name"`
	Transport string `json:"transport"`
}

func millis(t time.Time) *int64 {
	if t.IsZero() {
		return nil
	}
	return new(t.UnixMilli())
}

func (s *Server) handleState(r *http.Request) (any, error) {
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	lib, err := mcp.LoadLibrary(paths.MCPLibraryFile(s.home))
	if err != nil {
		return nil, err
	}
	summaries := make([]mcpSummary, len(lib))
	for i, e := range lib {
		summaries[i] = mcpSummary{Name: e.Name, Transport: e.Transport}
	}
	return map[string]any{
		"root":       s.abbrev(s.root),
		"accounts":   s.accounts(),
		"projects":   s.projects(st),
		"presets":    s.presets(st),
		"mcpLibrary": summaries,
	}, nil
}

func (s *Server) windows() map[string]session.Window {
	windows, err := session.Windows()
	if err != nil {
		log.Warn().Err(err).Msg("could not list tmux windows; every project shows as inactive")
	}
	return windows
}

func (s *Server) projects(st *state.State) []projectView {
	windows := s.windows()
	var pids []int
	for _, w := range windows {
		if !w.Dead {
			pids = append(pids, w.PID)
		}
	}
	started := session.StartTimes(pids)

	views := []projectView{}
	for dir, p := range st.Projects {
		if filepath.Dir(dir) != s.root {
			continue
		}
		name := filepath.Base(dir)
		v := projectView{
			Dir:        s.abbrev(dir),
			Name:       name,
			RCName:     p.RCName,
			Account:    s.abbrev(p.Account),
			Active:     session.Active(windows, name),
			Presets:    workspace.Presets(dir),
			MCP:        mcp.ProjectNames(dir),
			Connectors: p.Connectors,
			Stash:      fileExists(s.stashPath(dir)) && !fileExists(filepath.Join(dir, workspace.ResumeFile)),
		}
		if v.Active {
			v.StartedAt = millis(started[windows[name].PID])
		}
		last := convo.LastActivity(p.Account, dir)
		if stored := time.UnixMilli(p.LastActive); p.LastActive > 0 && stored.After(last) {
			last = stored
		}
		v.LastActive = millis(last)
		views = append(views, v)
	}
	slices.SortFunc(views, func(a, b projectView) int {
		if a.Active != b.Active {
			if a.Active {
				return -1
			}
			return 1
		}
		if c := cmp.Compare(deref(b.LastActive), deref(a.LastActive)); c != 0 {
			return c
		}
		return strings.Compare(a.Name, b.Name)
	})
	return views
}

func deref(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func fileExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func (s *Server) stashPath(dir string) string {
	return paths.ResumeStash(s.home, dir)
}

func (s *Server) handleDirectories(r *http.Request) (any, error) {
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	type directory struct {
		Name    string   `json:"name"`
		Foreign []string `json:"foreign"`
	}
	dirs := []directory{}
	for _, e := range entries {
		dir := filepath.Join(s.root, e.Name())
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if _, tracked := st.Projects[dir]; tracked {
			continue
		}
		dirs = append(dirs, directory{Name: e.Name(), Foreign: workspace.Foreign(dir)})
	}
	return dirs, nil
}

type startRequest struct {
	Name          string   `json:"name"`
	Source        string   `json:"source"`
	RCName        string   `json:"rcName"`
	Account       string   `json:"account"`
	Mode          string   `json:"mode"`
	Presets       []string `json:"presets"`
	MCP           []string `json:"mcp"`
	Connectors    bool     `json:"connectors"`
	Repo          string   `json:"repo"`
	ForeignChoice string   `json:"foreignChoice"`
}

func (s *Server) handleStart(r *http.Request) (any, error) {
	var req startRequest
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	if req.Mode != "new" && req.Mode != "continue" {
		return nil, fail(http.StatusBadRequest, "mode must be new or continue")
	}
	dir, err := s.dirOf(req.Name)
	if err != nil {
		return nil, err
	}
	account, err := s.resolveAccount(req.Account)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	_, tracked := st.Projects[dir]
	if session.Active(s.windows(), req.Name) {
		return nil, fail(http.StatusConflict, "%s already has a running session", req.Name)
	}

	if err := s.prepareSource(r, req, dir); err != nil {
		return nil, err
	}

	var foreign []string
	if !tracked {
		foreign = workspace.Foreign(dir)
	}
	if len(foreign) > 0 {
		choice := req.ForeignChoice
		if choice == "" && req.Source == "clone" {
			choice = "preserve"
		}
		if choice != "preserve" && choice != "clean" {
			return nil, fail(http.StatusConflict, "%s holds files claudex did not create (%s); choose preserve or clean", req.Name, strings.Join(foreign, ", "))
		}
		if zipped, err := workspace.ClearForeign(dir, foreign, choice == "preserve"); err != nil {
			if zipped != "" {
				return nil, fmt.Errorf("the old files are zipped in %s: %w", zipped, err)
			}
			return nil, err
		}
	}

	if len(req.Presets) > 0 {
		if err := s.applyPresets(st, dir, req.Presets); err != nil {
			return nil, err
		}
	}
	if err := s.selectLibraryMCP(dir, req.MCP); err != nil {
		return nil, err
	}

	p := state.Project{
		RCName:     cmp.Or(strings.TrimSpace(req.RCName), req.Name),
		Account:    account,
		Connectors: req.Connectors,
		LastActive: time.Now().UnixMilli(),
	}
	st.Projects[dir] = p
	if err := s.saveState(st); err != nil {
		return nil, err
	}
	if err := s.launcher.Start(s.launch(req.Name, dir, p, req.Mode == "continue")); err != nil {
		return nil, err
	}
	log.Info().Str("project", req.Name).Str("mode", req.Mode).Msg("session started")
	return map[string]string{"name": req.Name, "mode": req.Mode}, nil
}

func (s *Server) prepareSource(r *http.Request, req startRequest, dir string) error {
	if req.Source == "existing" {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return fail(http.StatusNotFound, "%s is not a directory under %s", req.Name, s.abbrev(s.root))
		}
		return nil
	}
	if req.Source != "empty" && req.Source != "clone" {
		return fail(http.StatusBadRequest, "source must be existing, empty, or clone")
	}
	if !newDirName.MatchString(req.Name) {
		return fail(http.StatusBadRequest, "%q is not a valid directory name; use lowercase letters, digits, dots, dashes, and underscores", req.Name)
	}
	if fileExists(dir) {
		return fail(http.StatusConflict, "%s already exists", s.abbrev(dir))
	}
	if req.Source == "empty" {
		return os.Mkdir(dir, 0o755)
	}
	if !repoName.MatchString(req.Repo) {
		return fail(http.StatusBadRequest, "repo must look like owner/name")
	}
	cmd := exec.CommandContext(r.Context(), "gh", "repo", "clone", req.Repo, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fail(http.StatusBadGateway, "cloning %s failed: %s", req.Repo, cmp.Or(strings.TrimSpace(string(out)), err.Error()))
	}
	return nil
}

func (s *Server) launch(name, dir string, p state.Project, continueMode bool) session.Launch {
	if continueMode {
		ids, err := convo.SessionIDs(p.Account, dir)
		continueMode = err == nil && len(ids) > 0
	}
	return session.Launch{
		Name:       name,
		Dir:        dir,
		RCName:     p.RCName,
		Account:    p.Account,
		Connectors: p.Connectors,
		Continue:   continueMode,
	}
}

func (s *Server) handleStop(r *http.Request) (any, error) {
	name := r.PathValue("name")
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	dir, p, err := s.tracked(st, name)
	if err != nil {
		return nil, err
	}
	if err := session.Stop(name); errors.Is(err, session.ErrNotRunning) {
		return nil, fail(http.StatusConflict, "%s is not running", name)
	} else if err != nil {
		return nil, err
	}
	p.LastActive = time.Now().UnixMilli()
	st.Projects[dir] = p
	if err := s.saveState(st); err != nil {
		return nil, err
	}
	return map[string]string{"name": name}, nil
}

func (s *Server) handleRestart(r *http.Request) (any, error) {
	name := r.PathValue("name")
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	dir, p, err := s.tracked(st, name)
	if err != nil {
		return nil, err
	}
	if err := s.launcher.Restart(s.launch(name, dir, p, true)); err != nil {
		return nil, err
	}
	p.LastActive = time.Now().UnixMilli()
	st.Projects[dir] = p
	if err := s.saveState(st); err != nil {
		return nil, err
	}
	return map[string]string{"name": name}, nil
}

func (s *Server) handleSwitch(r *http.Request) (any, error) {
	name := r.PathValue("name")
	var req struct {
		Account string `json:"account"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	target, err := s.resolveAccount(req.Account)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	dir, p, err := s.tracked(st, name)
	if err != nil {
		return nil, err
	}
	if target == p.Account {
		return nil, fail(http.StatusBadRequest, "%s already runs under %s", name, s.abbrev(target))
	}
	if err := session.Stop(name); err != nil && !errors.Is(err, session.ErrNotRunning) {
		return nil, err
	}
	if _, err := convo.MoveProject(dir, p.Account, target); err != nil {
		return nil, err
	}
	p.Account = target
	p.LastActive = time.Now().UnixMilli()
	st.Projects[dir] = p
	if err := s.saveState(st); err != nil {
		return nil, err
	}
	if err := s.launcher.Start(s.launch(name, dir, p, true)); err != nil {
		return nil, err
	}
	return map[string]string{"name": name, "account": s.abbrev(target)}, nil
}

func (s *Server) handleUntrack(r *http.Request) (any, error) {
	name := r.PathValue("name")
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	dir, _, err := s.tracked(st, name)
	if err != nil {
		return nil, err
	}
	if session.Active(s.windows(), name) {
		return nil, fail(http.StatusConflict, "stop the session before untracking %s", name)
	}
	delete(st.Projects, dir)
	if err := s.saveState(st); err != nil {
		return nil, err
	}
	return map[string]string{"name": name}, nil
}

func (s *Server) handleClean(r *http.Request) (any, error) {
	name := r.PathValue("name")
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	dir, _, err := s.tracked(st, name)
	if err != nil {
		return nil, err
	}
	if _, err := workspace.StashResume(dir, s.stashPath(dir)); err != nil {
		return nil, err
	}
	if err := workspace.Clean(dir); err != nil {
		return nil, err
	}
	return map[string]string{"name": name}, nil
}

func (s *Server) handleRestoreResume(r *http.Request) (any, error) {
	name := r.PathValue("name")
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	dir, _, err := s.tracked(st, name)
	if err != nil {
		return nil, err
	}
	if err := workspace.RestoreResume(dir, s.stashPath(dir)); err != nil {
		return nil, err
	}
	return map[string]string{"name": name}, nil
}

func (s *Server) handleApplyPresets(r *http.Request) (any, error) {
	name := r.PathValue("name")
	var req struct {
		Presets []string `json:"presets"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	if len(req.Presets) == 0 {
		return nil, fail(http.StatusBadRequest, "pick at least one preset")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	dir, _, err := s.tracked(st, name)
	if err != nil {
		return nil, err
	}
	if err := s.applyPresets(st, dir, req.Presets); err != nil {
		return nil, err
	}
	if err := s.saveState(st); err != nil {
		return nil, err
	}
	return map[string]any{"name": name, "presets": workspace.Presets(dir)}, nil
}

func (s *Server) applyPresets(st *state.State, dir string, refs []string) error {
	s.refreshRemotePresets(st, refs)
	local, remote := paths.PresetsDir(s.home), paths.RemotePresetsDir(s.home)
	for _, ref := range refs {
		if ref == workspace.BasePreset {
			continue
		}
		if _, err := workspace.FindPreset(local, remote, ref); errors.Is(err, fs.ErrNotExist) {
			return fail(http.StatusBadRequest, "no preset named %s", ref)
		} else if err != nil {
			return fail(http.StatusBadRequest, "preset %s: %v", ref, err)
		}
	}
	return workspace.Apply(workspace.ApplyConfig{
		Root:          dir,
		Base:          embedded.AgentsBase,
		BaseSkills:    paths.BaseSkillsDir(s.home),
		LocalPresets:  local,
		RemotePresets: remote,
		Presets:       refs,
	})
}

func (s *Server) refreshRemotePresets(st *state.State, refs []string) {
	pulled := map[state.RemotePreset]bool{}
	for _, ref := range refs {
		tracked, ok := st.RemotePresets[ref]
		source := state.RemotePreset{Repo: tracked.Repo, Path: tracked.Path}
		if !ok || tracked.Repo == "" || pulled[source] {
			continue
		}
		pulled[source] = true
		if _, err := s.pullRemote(st, tracked.Repo, tracked.Path); err != nil {
			log.Warn().Err(err).Str("preset", ref).Msg("could not update the remote preset; applying the local copy")
		}
	}
}

func (s *Server) handleSetMCP(r *http.Request) (any, error) {
	name := r.PathValue("name")
	var req struct {
		Servers []jsontext.Value `json:"servers"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	var names []string
	var custom []mcp.Entry
	for _, raw := range req.Servers {
		if raw.Kind() == '"' {
			var n string
			if err := json.Unmarshal(raw, &n); err != nil {
				return nil, fail(http.StatusBadRequest, "invalid server name: %v", err)
			}
			names = append(names, n)
			continue
		}
		var e mcp.Entry
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil, fail(http.StatusBadRequest, "invalid server entry: %v", err)
		}
		if err := e.Validate(); err != nil {
			return nil, fail(http.StatusBadRequest, "%v", err)
		}
		custom = append(custom, e)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	dir, _, err := s.tracked(st, name)
	if err != nil {
		return nil, err
	}
	if err := s.writeProjectMCP(dir, names, custom); err != nil {
		return nil, err
	}
	return map[string]any{"name": name, "mcp": mcp.ProjectNames(dir)}, nil
}

func (s *Server) selectLibraryMCP(dir string, selected []string) error {
	lib, err := mcp.LoadLibrary(paths.MCPLibraryFile(s.home))
	if err != nil {
		return err
	}
	names := slices.Clone(selected)
	for _, n := range mcp.ProjectNames(dir) {
		inLibrary := slices.ContainsFunc(lib, func(e mcp.Entry) bool { return e.Name == n })
		if !inLibrary && !slices.Contains(names, n) {
			names = append(names, n)
		}
	}
	if len(names) == 0 && !fileExists(filepath.Join(dir, workspace.MCPFile)) {
		return nil
	}
	return s.writeProjectMCP(dir, names, nil)
}

func (s *Server) writeProjectMCP(dir string, names []string, custom []mcp.Entry) error {
	lib, err := mcp.LoadLibrary(paths.MCPLibraryFile(s.home))
	if err != nil {
		return err
	}
	existing, err := mcp.ProjectServers(dir)
	if err != nil {
		return fail(http.StatusConflict, "%v", err)
	}
	servers := map[string]jsontext.Value{}
	for _, n := range names {
		if i := slices.IndexFunc(lib, func(e mcp.Entry) bool { return e.Name == n }); i >= 0 {
			if servers[n], err = lib[i].Config(); err != nil {
				return err
			}
			continue
		}
		raw, ok := existing[n]
		if !ok {
			return fail(http.StatusBadRequest, "%s is neither in the MCP library nor in the project's %s", n, workspace.MCPFile)
		}
		servers[n] = raw
	}
	for _, e := range custom {
		if servers[e.Name], err = e.Config(); err != nil {
			return err
		}
	}
	if err := mcp.WriteProject(dir, servers); err != nil {
		return err
	}
	_, _, err = workspace.WriteGitExclude(dir)
	return err
}
