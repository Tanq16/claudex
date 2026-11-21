package server

import (
	"embed"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"
	"github.com/tanq16/claudex/internal/embedded"
	"github.com/tanq16/claudex/internal/paths"
	"github.com/tanq16/claudex/internal/plugins"
	"github.com/tanq16/claudex/internal/session"
	"github.com/tanq16/claudex/internal/state"
	"github.com/tanq16/claudex/internal/workspace"
)

//go:embed static
var staticFiles embed.FS

const maxJSONBody = 32 << 20

type Config struct {
	Host string
	Port int
	Root string
}

type Server struct {
	cfg         Config
	home        string
	root        string
	launcher    session.Launcher
	mu          sync.Mutex
	mux         *http.ServeMux
	schemaTried atomic.Bool
}

func New(cfg Config) *Server {
	return &Server{cfg: cfg, mux: http.NewServeMux()}
}

func (s *Server) Setup() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolving the home directory: %w", err)
	}
	s.home = home

	if s.cfg.Root == "" {
		s.cfg.Root = paths.ProjectsDir(home)
		if err := os.MkdirAll(s.cfg.Root, 0o700); err != nil {
			return fmt.Errorf("creating the projects directory: %w", err)
		}
	}
	root, err := filepath.Abs(s.cfg.Root)
	if err == nil {
		root, err = filepath.EvalSymlinks(root)
	}
	if err != nil {
		return fmt.Errorf("resolving the root %s: %w", s.cfg.Root, err)
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return fmt.Errorf("root %s is not a directory", root)
	}
	s.root = root

	if err := workspace.EnsurePresets(embedded.PresetsFS, "presets", paths.PresetsDir(home)); err != nil {
		return fmt.Errorf("laying down the built-in presets: %w", err)
	}
	if err := workspace.EnsureRemotePresets(paths.RemotePresetsDir(home)); err != nil {
		return fmt.Errorf("creating the remote presets directory: %w", err)
	}
	if _, err := state.Load(paths.StateFile(home)); err != nil {
		return err
	}

	s.launcher = session.Launcher{DefaultAccount: paths.DefaultAccount(home)}
	pluginDir := paths.GlobalPluginDir(home)
	if err := plugins.BuildGlobalPlugin(pluginDir); err != nil {
		log.Warn().Err(err).Msg("could not prepare the global plugin; sessions start without it")
	} else {
		s.launcher.PluginDir = pluginDir
	}

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return err
	}
	s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	s.mux.HandleFunc("GET /api/state", s.api(s.handleState))
	s.mux.HandleFunc("GET /api/directories", s.api(s.handleDirectories))
	s.mux.HandleFunc("GET /api/repos", s.api(s.handleRepos))
	s.mux.HandleFunc("GET /api/presets/validate", s.api(s.handleValidatePreset))

	s.mux.HandleFunc("POST /api/sessions", s.api(s.handleStart))
	s.mux.HandleFunc("POST /api/projects/{name}/stop", s.api(s.handleStop))
	s.mux.HandleFunc("POST /api/projects/{name}/restart", s.api(s.handleRestart))
	s.mux.HandleFunc("POST /api/projects/{name}/switch", s.api(s.handleSwitch))
	s.mux.HandleFunc("DELETE /api/projects/{name}", s.api(s.handleUntrack))
	s.mux.HandleFunc("POST /api/projects/{name}/track", s.api(s.handleTrack))
	s.mux.HandleFunc("PUT /api/projects/{name}/connectors", s.api(s.handleSetConnectors))

	s.mux.HandleFunc("POST /api/projects/{name}/clean", s.api(s.handleClean))
	s.mux.HandleFunc("POST /api/projects/{name}/restore-resume", s.api(s.handleRestoreResume))
	s.mux.HandleFunc("PUT /api/projects/{name}/presets", s.api(s.handleApplyPresets))
	s.mux.HandleFunc("PUT /api/projects/{name}/mcp", s.api(s.handleSetMCP))

	s.mux.HandleFunc("GET /api/projects/{name}/tree", s.api(s.handleTree))
	s.mux.HandleFunc("GET /api/projects/{name}/file", s.api(s.handleFile))
	s.mux.HandleFunc("GET /api/projects/{name}/raw", s.handleRaw)
	s.mux.HandleFunc("POST /api/projects/{name}/files", s.api(s.handleCreateFile))
	s.mux.HandleFunc("POST /api/projects/{name}/upload", s.api(s.handleUpload))

	s.mux.HandleFunc("GET /api/settings", s.api(s.handleSettings))
	s.mux.HandleFunc("POST /api/accounts/apply-defaults", s.api(s.handleApplyDefaults))
	s.mux.HandleFunc("GET /api/defaults", s.api(s.handleDefaults))
	s.mux.HandleFunc("PUT /api/defaults", s.api(s.handleSetDefault))
	s.mux.HandleFunc("DELETE /api/defaults", s.api(s.handleResetDefault))
	s.mux.HandleFunc("POST /api/defaults/refresh", s.api(s.handleRefreshSchema))
	s.mux.HandleFunc("POST /api/presets/remote", s.api(s.handleAddRemotePreset))
	s.mux.HandleFunc("POST /api/presets/remote/update", s.api(s.handleUpdateRemotePreset))
	s.mux.HandleFunc("POST /api/presets/remote/remove", s.api(s.handleRemoveRemotePreset))
	s.mux.HandleFunc("PUT /api/mcp", s.api(s.handleSaveMCP))
	s.mux.HandleFunc("DELETE /api/mcp/{name}", s.api(s.handleDeleteMCP))

	s.mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "no such API route: "+r.Method+" "+r.URL.Path)
	})
	s.mux.HandleFunc("/", s.handleIndex)
	return nil
}

func (s *Server) Run() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	log.Info().Str("addr", addr).Str("root", s.root).Msg("starting")
	return http.ListenAndServe(addr, http.NewCrossOriginProtection().Handler(s.mux))
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
}

type apiError struct {
	status int
	msg    string
}

func (e *apiError) Error() string { return e.msg }

func fail(status int, format string, args ...any) error {
	return &apiError{status: status, msg: fmt.Sprintf(format, args...)}
}

func statusOf(err error) int {
	if e, ok := errors.AsType[*apiError](err); ok {
		return e.status
	}
	if _, ok := errors.AsType[*workspace.ConflictError](err); ok {
		return http.StatusConflict
	}
	for _, target := range []error{session.ErrRunning, session.ErrNotRunning, workspace.ErrNoStash, workspace.ErrResumeExists} {
		if errors.Is(err, target) {
			return http.StatusConflict
		}
	}
	return http.StatusInternalServerError
}

func (s *Server) api(fn func(r *http.Request) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, err := fn(r)
		if err != nil {
			status := statusOf(err)
			if status >= http.StatusInternalServerError {
				log.Error().Err(err).Str("method", r.Method).Str("path", r.URL.Path).Msg("request failed")
			}
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, v)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.MarshalWrite(w, v); err != nil {
		log.Error().Err(err).Msg("failed to encode a response")
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	if err := json.UnmarshalRead(http.MaxBytesReader(nil, r.Body, maxJSONBody), v); err != nil {
		return fail(http.StatusBadRequest, "the request body is not valid JSON: %v", err)
	}
	return nil
}

func (s *Server) abbrev(path string) string {
	return paths.Abbreviate(s.home, path)
}

func (s *Server) loadState() (*state.State, error) {
	return state.Load(paths.StateFile(s.home))
}

func (s *Server) saveState(st *state.State) error {
	return st.Save(paths.StateFile(s.home))
}

func (s *Server) dirOf(name string) (string, error) {
	if name == "" || name != filepath.Base(name) || strings.HasPrefix(name, ".") {
		return "", fail(http.StatusBadRequest, "%q is not a project name", name)
	}
	return filepath.Join(s.root, name), nil
}

func (s *Server) tracked(st *state.State, name string) (string, state.Project, error) {
	dir, err := s.dirOf(name)
	if err != nil {
		return "", state.Project{}, err
	}
	p, ok := st.Projects[dir]
	if !ok {
		return "", state.Project{}, fail(http.StatusNotFound, "%s is not tracked", name)
	}
	return dir, p, nil
}

func (s *Server) resolveAccount(v string) (string, error) {
	discovered := paths.DiscoverAccounts(s.home)
	if v == "" {
		if def := paths.DefaultAccount(s.home); slices.Contains(discovered, def) {
			return def, nil
		}
		return discovered[0], nil
	}
	dir := filepath.Clean(paths.Expand(s.home, v))
	if !slices.Contains(discovered, dir) {
		return "", fail(http.StatusBadRequest, "%s is not a discovered account", v)
	}
	return dir, nil
}
