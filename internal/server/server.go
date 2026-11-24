package server

import (
	"context"
	"embed"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"
	"github.com/tanq16/claudex/internal/attached"
	"github.com/tanq16/claudex/internal/embedded"
	"github.com/tanq16/claudex/internal/images"
	"github.com/tanq16/claudex/internal/paths"
	"github.com/tanq16/claudex/internal/plugins"
	"github.com/tanq16/claudex/internal/session"
	"github.com/tanq16/claudex/internal/state"
	"github.com/tanq16/claudex/internal/workspace"
)

//go:embed static
var staticFiles embed.FS

const (
	maxJSONBody = 32 << 20
	gzipMin     = 1 << 10
)

type Config struct {
	Host string
	Port int
	Root string
}

type Server struct {
	cfg         Config
	ctx         context.Context
	cancel      context.CancelFunc
	home        string
	root        string
	launcher    session.Launcher
	host        *attached.Host
	images      *images.Store
	assets      map[string]func() (*asset, error)
	mu          sync.Mutex
	mux         *http.ServeMux
	schemaTried atomic.Bool
}

func New(cfg Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{cfg: cfg, ctx: ctx, cancel: cancel, mux: http.NewServeMux()}
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

	if err := workspace.Extract(embedded.DefaultSkillsFS, "default-skills", paths.BaseSkillsDir(home)); err != nil {
		return fmt.Errorf("laying down the base skills: %w", err)
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

	if s.images, err = images.New(paths.ImagesDir(home), "/api/images/"); err != nil {
		return fmt.Errorf("creating the image store: %w", err)
	}
	s.host = attached.NewHost(s.images, logAttached)
	if err := s.resumeAttached(); err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-s.host.Changed():
				s.syncAttached()
			}
		}
	}()

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return err
	}
	if s.assets, err = loadAssets(staticFS); err != nil {
		return fmt.Errorf("indexing the static assets: %w", err)
	}
	s.mux.HandleFunc("GET /static/", s.handleStatic)
	s.mux.HandleFunc("GET /sw.js", s.handleServiceWorker)

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

	s.mux.HandleFunc("GET /api/events", s.handleEvents)
	s.mux.HandleFunc("GET /api/images/{file}", s.handleImage)
	s.mux.HandleFunc("GET /api/projects/{name}/history", s.api(s.handleHistory))
	s.mux.HandleFunc("POST /api/projects/{name}/shell", s.api(s.handleShell))
	s.mux.HandleFunc("POST /api/projects/{name}/prompt", s.api(s.handlePrompt))
	s.mux.HandleFunc("POST /api/projects/{name}/approvals/{id}", s.api(s.handleAnswer))
	s.mux.HandleFunc("POST /api/projects/{name}/interrupt", s.api(s.handleInterrupt))
	s.mux.HandleFunc("POST /api/projects/{name}/tasks/{task}/stop", s.api(s.handleStopTask))
	s.mux.HandleFunc("GET /api/projects/{name}/tasks/{task}/output", s.api(s.handleTaskOutput))
	s.mux.HandleFunc("PUT /api/projects/{name}/model", s.api(s.handleSetting((*attached.Session).SetModel, "model")))
	s.mux.HandleFunc("PUT /api/projects/{name}/effort", s.api(s.handleSetting((*attached.Session).SetEffort, "effort")))
	s.mux.HandleFunc("PUT /api/projects/{name}/permission-mode", s.api(s.handleSetting((*attached.Session).SetPermissionMode, "mode")))
	s.mux.HandleFunc("GET /api/projects/{name}/context", s.api(s.handleContext))

	s.mux.HandleFunc("GET /api/projects/{name}/tree", s.api(s.handleTree))
	s.mux.HandleFunc("GET /api/projects/{name}/git", s.api(s.handleGit))
	s.mux.HandleFunc("GET /api/projects/{name}/file", s.api(s.handleFile))
	s.mux.HandleFunc("GET /api/projects/{name}/raw", s.handleRaw)
	s.mux.HandleFunc("POST /api/projects/{name}/files", s.api(s.handleCreateFile))
	s.mux.HandleFunc("POST /api/projects/{name}/upload", s.api(s.handleUpload))

	s.mux.HandleFunc("GET /api/settings", s.api(s.handleSettings))
	s.mux.HandleFunc("POST /api/accounts/apply-defaults", s.api(s.handleApplyDefaults))
	s.mux.HandleFunc("POST /api/accounts/adopt", s.api(s.handleAdoptKey))
	s.mux.HandleFunc("POST /api/accounts/remove-key", s.api(s.handleRemoveKey))
	s.mux.HandleFunc("GET /api/defaults", s.api(s.handleDefaults))
	s.mux.HandleFunc("PUT /api/defaults", s.api(s.handleSetDefault))
	s.mux.HandleFunc("PUT /api/defaults/document", s.api(s.handleReplaceDefaults))
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
	addr := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))
	log.Info().Str("addr", addr).Str("root", s.root).Msg("starting")
	srv := &http.Server{
		Addr:        addr,
		Handler:     http.NewCrossOriginProtection().Handler(s.mux),
		BaseContext: func(net.Listener) context.Context { return s.ctx },
	}
	return srv.ListenAndServe()
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

func (s *Server) handleServiceWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript")
	http.ServeFileFS(w, r, staticFiles, "static/sw.js")
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
	if _, ok := errors.AsType[*attached.RefusedError](err); ok {
		return http.StatusBadGateway
	}
	for _, target := range []error{session.ErrRunning, session.ErrNotRunning, workspace.ErrNoStash, workspace.ErrResumeExists, attached.ErrRunning, attached.ErrNotRunning, attached.ErrClosing} {
		if errors.Is(err, target) {
			return http.StatusConflict
		}
	}
	switch {
	case errors.Is(err, attached.ErrNoRequest):
		return http.StatusNotFound
	case errors.Is(err, attached.ErrBadAnswer):
		return http.StatusBadRequest
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout
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
		data, err := json.Marshal(v, json.Deterministic(true))
		if err != nil {
			log.Error().Err(err).Str("method", r.Method).Str("path", r.URL.Path).Msg("failed to encode a response")
			writeError(w, http.StatusInternalServerError, "the response could not be encoded")
			return
		}
		h := w.Header()
		h.Set("Content-Type", "application/json")
		h.Add("Vary", "Accept-Encoding")
		if len(data) >= gzipMin && acceptsGzip(r) {
			h.Set("Content-Encoding", "gzip")
			data = gzipped(data)
		}
		w.Write(data)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.MarshalWrite(w, v, json.Deterministic(true)); err != nil {
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
