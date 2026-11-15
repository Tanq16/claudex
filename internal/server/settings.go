package server

import (
	"cmp"
	"encoding/json/v2"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/tanq16/claudex/internal/accounts"
	"github.com/tanq16/claudex/internal/embedded"
	"github.com/tanq16/claudex/internal/mcp"
	"github.com/tanq16/claudex/internal/parser"
	"github.com/tanq16/claudex/internal/paths"
	"github.com/tanq16/claudex/internal/state"
	"github.com/tanq16/claudex/internal/workspace"
)

type accountView struct {
	Dir       string `json:"dir"`
	Label     string `json:"label"`
	IsDefault bool   `json:"isDefault"`
}

type accountSettings struct {
	accountView
	SettingsFound bool             `json:"settingsFound"`
	Drift         []accounts.Drift `json:"drift"`
	Projects      int              `json:"projects"`
}

type builtinPreset struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	DefaultOn   bool   `json:"defaultOn"`
}

type remotePreset struct {
	Name        string `json:"name"`
	Repo        string `json:"repo"`
	Description string `json:"description"`
	UpdatedAt   *int64 `json:"updatedAt"`
}

type presetsView struct {
	Builtin []builtinPreset `json:"builtin"`
	Remote  []remotePreset  `json:"remote"`
}

func (s *Server) accounts() []accountView {
	def := paths.DefaultAccount(s.home)
	var views []accountView
	for _, dir := range paths.DiscoverAccounts(s.home) {
		label := s.abbrev(dir)
		if info, err := parser.ParseAccount(dir); err == nil && info.Email != "" {
			label = info.Email
		}
		views = append(views, accountView{Dir: s.abbrev(dir), Label: label, IsDefault: dir == def})
	}
	return views
}

func baseDescription() string {
	entries, _ := fs.ReadDir(embedded.DefaultSkillsFS, "default-skills")
	names := []string{"AGENTS.md base rules"}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return strings.Join(names, ", ")
}

func (s *Server) presets(st *state.State) presetsView {
	view := presetsView{
		Builtin: []builtinPreset{{Name: workspace.BasePreset, Description: baseDescription(), DefaultOn: true}},
		Remote:  []remotePreset{},
	}
	for _, p := range workspace.ListPresets(paths.PresetsDir(s.home), paths.RemotePresetsDir(s.home)) {
		if !strings.Contains(p.Ref, "/") {
			view.Builtin = append(view.Builtin, builtinPreset{Name: p.Ref, Description: p.Description})
			continue
		}
		tracked := st.RemotePresets[p.Ref]
		updated := time.UnixMilli(tracked.UpdatedAt)
		if tracked.UpdatedAt == 0 {
			updated = time.Time{}
			if info, err := os.Stat(p.Dir); err == nil {
				updated = info.ModTime()
			}
		}
		view.Remote = append(view.Remote, remotePreset{Name: p.Ref, Repo: tracked.Repo, Description: p.Description, UpdatedAt: millis(updated)})
	}
	return view
}

func (s *Server) handleSettings(r *http.Request) (any, error) {
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	lib, err := mcp.LoadLibrary(paths.MCPLibraryFile(s.home))
	if err != nil {
		return nil, err
	}
	var views []accountSettings
	for _, a := range s.accounts() {
		dir := paths.Expand(s.home, a.Dir)
		found, drift := accounts.Compliance(dir)
		for i := range drift {
			drift[i].Expected = s.abbrev(drift[i].Expected)
			drift[i].Actual = s.abbrev(drift[i].Actual)
		}
		count := 0
		for _, p := range st.Projects {
			if p.Account == dir {
				count++
			}
		}
		views = append(views, accountSettings{accountView: a, SettingsFound: found, Drift: drift, Projects: count})
	}
	return map[string]any{
		"root":              s.abbrev(s.root),
		"accounts":          views,
		"configureDefaults": accounts.Defaults(),
		"presets":           s.presets(st),
		"mcpLibrary":        lib,
	}, nil
}

func (s *Server) handleApplyDefaults(r *http.Request) (any, error) {
	var req struct {
		Dir string `json:"dir"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	dir, err := s.resolveAccount(req.Dir)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := accounts.Configure(dir, "", embedded.StatuslineScript); err != nil {
		return nil, fail(http.StatusConflict, "%v", err)
	}
	return map[string]string{"dir": s.abbrev(dir)}, nil
}

func (s *Server) handleValidatePreset(r *http.Request) (any, error) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		return map[string]any{"ok": false, "name": name, "error": "Enter a preset name"}, nil
	}
	if name == workspace.BasePreset {
		return map[string]any{"ok": true, "name": name}, nil
	}
	if _, err := workspace.FindPreset(paths.PresetsDir(s.home), paths.RemotePresetsDir(s.home), name); err != nil {
		msg := err.Error()
		if errors.Is(err, fs.ErrNotExist) {
			msg = "No preset.yaml for " + name + "; track its repository in Settings first"
		}
		return map[string]any{"ok": false, "name": name, "error": msg}, nil
	}
	return map[string]any{"ok": true, "name": name, "note": "Found, preset.yaml parsed"}, nil
}

func (s *Server) handleRepos(r *http.Request) (any, error) {
	cmd := exec.CommandContext(r.Context(), "gh", "api", "--paginate", "--slurp", "user/repos?per_page=100")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fail(http.StatusBadGateway, "listing repositories with gh failed: %s", cmp.Or(strings.TrimSpace(stderr.String()), err.Error()))
	}
	var pages [][]struct {
		FullName    string    `json:"full_name"`
		Description string    `json:"description"`
		Private     bool      `json:"private"`
		UpdatedAt   time.Time `json:"updated_at"`
	}
	if err := json.Unmarshal(out, &pages); err != nil {
		return nil, fail(http.StatusBadGateway, "gh returned output that is not a list of repositories: %v", err)
	}
	type repo struct {
		Full        string `json:"full"`
		Description string `json:"description"`
		Private     bool   `json:"private"`
		UpdatedAt   int64  `json:"updatedAt"`
	}
	repos := []repo{}
	for _, page := range pages {
		for _, r := range page {
			repos = append(repos, repo{Full: r.FullName, Description: r.Description, Private: r.Private, UpdatedAt: r.UpdatedAt.UnixMilli()})
		}
	}
	slices.SortFunc(repos, func(a, b repo) int { return cmp.Compare(b.UpdatedAt, a.UpdatedAt) })
	return repos, nil
}

func (s *Server) pullRemote(st *state.State, repo, path string) ([]string, error) {
	res, err := workspace.PullPresets(workspace.PullConfig{Repo: repo, Path: path, Dir: paths.RemotePresetsDir(s.home)})
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	refs := make([]string, len(res.Presets))
	for i, name := range res.Presets {
		refs[i] = res.Slug + "/" + name
		st.RemotePresets[refs[i]] = state.RemotePreset{Repo: repo, Path: path, UpdatedAt: now}
	}
	return refs, nil
}

func (s *Server) handleAddRemotePreset(r *http.Request) (any, error) {
	var req struct {
		Repo string `json:"repo"`
		Path string `json:"path"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	repo, path := strings.TrimSpace(req.Repo), strings.TrimSpace(req.Path)
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	refs, err := s.pullRemote(st, repo, path)
	if err != nil {
		return nil, fail(http.StatusBadRequest, "pulling presets from %s failed: %v", repo, err)
	}
	if err := s.saveState(st); err != nil {
		return nil, err
	}
	return map[string]string{"name": refs[0]}, nil
}

func (s *Server) handleUpdateRemotePreset(r *http.Request) (any, error) {
	var req struct {
		Name string `json:"name"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	tracked, ok := st.RemotePresets[req.Name]
	if !ok || tracked.Repo == "" {
		return nil, fail(http.StatusNotFound, "%s has no tracked repository; track it again from its repository", req.Name)
	}
	if _, err := s.pullRemote(st, tracked.Repo, tracked.Path); err != nil {
		return nil, fail(http.StatusBadGateway, "updating %s failed: %v", req.Name, err)
	}
	if err := s.saveState(st); err != nil {
		return nil, err
	}
	return map[string]string{"name": req.Name}, nil
}

func (s *Server) handleRemoveRemotePreset(r *http.Request) (any, error) {
	var req struct {
		Name string `json:"name"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadState()
	if err != nil {
		return nil, err
	}
	remoteDir := paths.RemotePresetsDir(s.home)
	_, tracked := st.RemotePresets[req.Name]
	if !tracked && !fileExists(filepath.Join(remoteDir, filepath.FromSlash(req.Name))) {
		return nil, fail(http.StatusNotFound, "%s is not a tracked remote preset", req.Name)
	}
	if err := workspace.RemoveRemotePreset(remoteDir, req.Name); err != nil {
		return nil, fail(http.StatusBadRequest, "%v", err)
	}
	delete(st.RemotePresets, req.Name)
	if err := s.saveState(st); err != nil {
		return nil, err
	}
	return map[string]string{"name": req.Name}, nil
}

func (s *Server) handleSaveMCP(r *http.Request) (any, error) {
	var req struct {
		Entry        mcp.Entry `json:"entry"`
		PreviousName string    `json:"previousName"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	if err := req.Entry.Validate(); err != nil {
		return nil, fail(http.StatusBadRequest, "%v", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := paths.MCPLibraryFile(s.home)
	lib, err := mcp.LoadLibrary(path)
	if err != nil {
		return nil, err
	}
	index := func(name string) int { return slices.IndexFunc(lib, func(e mcp.Entry) bool { return e.Name == name }) }
	key := cmp.Or(req.PreviousName, req.Entry.Name)
	if req.Entry.Name != key && index(req.Entry.Name) >= 0 {
		return nil, fail(http.StatusConflict, "%s already exists", req.Entry.Name)
	}
	if i := index(key); i >= 0 {
		lib[i] = req.Entry
	} else {
		lib = append(lib, req.Entry)
	}
	if err := mcp.SaveLibrary(path, lib); err != nil {
		return nil, err
	}
	return map[string]string{"name": req.Entry.Name}, nil
}

func (s *Server) handleDeleteMCP(r *http.Request) (any, error) {
	name := r.PathValue("name")
	s.mu.Lock()
	defer s.mu.Unlock()
	path := paths.MCPLibraryFile(s.home)
	lib, err := mcp.LoadLibrary(path)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(lib, func(e mcp.Entry) bool { return e.Name == name })
	if i < 0 {
		return nil, fail(http.StatusNotFound, "%s is not in the MCP library", name)
	}
	if err := mcp.SaveLibrary(path, slices.Delete(lib, i, i+1)); err != nil {
		return nil, err
	}
	return map[string]string{"name": name}, nil
}
