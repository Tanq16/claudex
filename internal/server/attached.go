package server

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tanq16/claudex/internal/accounts"
	"github.com/tanq16/claudex/internal/attached"
	"github.com/tanq16/claudex/internal/convo"
	"github.com/tanq16/claudex/internal/session"
	"github.com/tanq16/claudex/internal/state"
)

const (
	commandTimeout = 30 * time.Second
	heartbeat      = 15 * time.Second
)

func (s *Server) startAttached(name, dir string, p state.Project, continueLast bool) error {
	if err := accounts.Trust(p.Account, s.launcher.DefaultAccount, dir); err != nil {
		return fmt.Errorf("trusting %s for the account: %w", dir, err)
	}
	if continueLast {
		ids, err := convo.SessionIDs(p.Account, dir)
		continueLast = err == nil && len(ids) > 0
	}
	cfg := attached.Config{
		Name:     name,
		Dir:      dir,
		Env:      accounts.Env(p.Account, s.launcher.DefaultAccount),
		Args:     session.SharedArgs(p.Connectors, s.launcher.PluginDir),
		Continue: continueLast,
	}
	if a := p.Attached; a != nil {
		cfg.Model, cfg.Effort, cfg.PermissionMode = a.Model, a.Effort, a.PermissionMode
	}
	return s.host.Start(cfg)
}

func (s *Server) syncAttached() {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadState()
	if err != nil {
		log.Warn().Err(err).Msg("could not load state to record attached sessions")
		return
	}
	live := s.host.Settings()
	changed := false
	for dir, p := range st.Projects {
		if p.SessionMode() != state.ModeAttached || filepath.Dir(dir) != s.root {
			continue
		}
		next := state.Attached{}
		if cur, ok := live[filepath.Base(dir)]; ok {
			next = state.Attached{Model: cur.Model, Effort: cur.Effort, PermissionMode: cur.PermissionMode, PID: cur.PID, StartedAt: cur.StartedAt.UnixMilli()}
		} else if p.Attached != nil {
			next = state.Attached{Model: p.Attached.Model, Effort: p.Attached.Effort, PermissionMode: p.Attached.PermissionMode}
		}
		if p.Attached != nil && *p.Attached == next {
			continue
		}
		p.Attached = &next
		st.Projects[dir] = p
		changed = true
	}
	if !changed {
		return
	}
	if err := s.saveState(st); err != nil {
		log.Warn().Err(err).Msg("could not record attached sessions")
	}
}

func (s *Server) resumeAttached() error {
	st, err := s.loadState()
	if err != nil {
		return err
	}
	for dir, p := range st.Projects {
		if p.SessionMode() != state.ModeAttached || p.Attached == nil || p.Attached.PID == 0 || filepath.Dir(dir) != s.root {
			continue
		}
		name := filepath.Base(dir)
		if attached.KillLeftover(p.Attached.PID, time.UnixMilli(p.Attached.StartedAt)) {
			log.Info().Str("project", name).Int("pid", p.Attached.PID).Msg("stopped a leftover attached session")
		}
		if err := s.startAttached(name, dir, p, true); err != nil {
			log.Warn().Err(err).Str("project", name).Msg("could not respawn the attached session")
			continue
		}
		log.Info().Str("project", name).Msg("attached session respawned with --continue")
	}
	return nil
}

func (s *Server) Busy() map[string][]string {
	return s.host.Busy()
}

func (s *Server) Shutdown() {
	s.cancel()
	s.syncAttached()
	s.host.Shutdown()
}

func (s *Server) attachedSession(name string) (*attached.Session, error) {
	if _, err := s.dirOf(name); err != nil {
		return nil, err
	}
	sess, err := s.host.Session(name)
	if err != nil {
		return nil, fail(http.StatusConflict, "%s has no running attached session", name)
	}
	return sess, nil
}

func commandContext(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), commandTimeout)
}

func payloadOrEmpty(v jsontext.Value) jsontext.Value {
	if len(v) == 0 {
		return jsontext.Value("{}")
	}
	return v
}

func (s *Server) handlePrompt(r *http.Request) (any, error) {
	var req struct {
		Text string `json:"text"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	sess, err := s.attachedSession(r.PathValue("name"))
	if err != nil {
		return nil, err
	}
	ctx, cancel := commandContext(r)
	defer cancel()
	id, err := sess.Prompt(ctx, req.Text)
	if err != nil {
		return nil, err
	}
	return map[string]string{"uuid": id}, nil
}

func (s *Server) handleAnswer(r *http.Request) (any, error) {
	var payload jsontext.Value
	if err := decode(r, &payload); err != nil {
		return nil, err
	}
	sess, err := s.attachedSession(r.PathValue("name"))
	if err != nil {
		return nil, err
	}
	ctx, cancel := commandContext(r)
	defer cancel()
	if err := sess.Answer(ctx, r.PathValue("id"), payload); err != nil {
		return nil, err
	}
	return map[string]string{"requestId": r.PathValue("id")}, nil
}

func (s *Server) handleInterrupt(r *http.Request) (any, error) {
	sess, err := s.attachedSession(r.PathValue("name"))
	if err != nil {
		return nil, err
	}
	ctx, cancel := commandContext(r)
	defer cancel()
	resp, err := sess.Interrupt(ctx)
	return payloadOrEmpty(resp), err
}

func (s *Server) handleStopTask(r *http.Request) (any, error) {
	sess, err := s.attachedSession(r.PathValue("name"))
	if err != nil {
		return nil, err
	}
	ctx, cancel := commandContext(r)
	defer cancel()
	resp, err := sess.StopTask(ctx, r.PathValue("task"))
	return payloadOrEmpty(resp), err
}

func (s *Server) handleTaskOutput(r *http.Request) (any, error) {
	sess, err := s.attachedSession(r.PathValue("name"))
	if err != nil {
		return nil, err
	}
	ctx, cancel := commandContext(r)
	defer cancel()
	resp, err := sess.TaskOutput(ctx, r.PathValue("task"))
	return payloadOrEmpty(resp), err
}

func (s *Server) handleSetting(apply func(*attached.Session, context.Context, string) error, field string) func(r *http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var req map[string]string
		if err := decode(r, &req); err != nil {
			return nil, err
		}
		value := strings.TrimSpace(req[field])
		if value == "" {
			return nil, fail(http.StatusBadRequest, "%s is required", field)
		}
		sess, err := s.attachedSession(r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		ctx, cancel := commandContext(r)
		defer cancel()
		if err := apply(sess, ctx, value); err != nil {
			return nil, err
		}
		return map[string]string{field: value}, nil
	}
}

func (s *Server) handleContext(r *http.Request) (any, error) {
	sess, err := s.attachedSession(r.PathValue("name"))
	if err != nil {
		return nil, err
	}
	ctx, cancel := commandContext(r)
	defer cancel()
	return sess.ContextUsage(ctx, "")
}

func (s *Server) handleImage(w http.ResponseWriter, r *http.Request) {
	path, ok := s.images.Path(r.PathValue("file"))
	if !ok {
		writeError(w, http.StatusNotFound, "no such image")
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=604800, immutable")
	http.ServeFile(w, r, path)
}

type sseWriter struct {
	w           http.ResponseWriter
	lastSummary []byte
}

func (sw *sseWriter) event(id, name string, data []byte) {
	if id != "" {
		fmt.Fprintf(sw.w, "id: %s\n", id)
	}
	fmt.Fprintf(sw.w, "event: %s\ndata: %s\n\n", name, data)
}

func (sw *sseWriter) summary(host *attached.Host) {
	data, err := json.Marshal(host.Summary(), json.Deterministic(true))
	if err != nil || bytes.Equal(data, sw.lastSummary) {
		return
	}
	sw.lastSummary = data
	sw.event("", "summary", data)
}

func (sw *sseWriter) open(sub *attached.Subscription) {
	if sub.Reset != nil {
		sw.event(sub.ResetID, "reset", sub.Reset)
		return
	}
	if data, ok := sub.Snapshot(); ok {
		sw.event("", "session", data)
	}
	for _, ev := range sub.Backlog {
		sw.event(ev.ID, ev.Name, ev.Data)
	}
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("session")
	if name != "" {
		if _, err := s.dirOf(name); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	cursor := cmp.Or(r.Header.Get("Last-Event-ID"), r.URL.Query().Get("after"))
	wake, unwatch := s.host.Watch()
	defer unwatch()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	rc := http.NewResponseController(w)
	sw := &sseWriter{w: w}
	sw.summary(s.host)

	var sub *attached.Subscription
	if name != "" {
		sub = s.host.Subscribe(name, cursor)
		defer func() { sub.Close() }()
		sw.open(sub)
	}
	if rc.Flush() != nil {
		return
	}

	beat := time.Tick(heartbeat)
	for {
		var live <-chan attached.Event
		if sub != nil {
			live = sub.Live
		}
		select {
		case <-r.Context().Done():
			return
		case <-beat:
			fmt.Fprint(w, ": ping\n\n")
		case ev, ok := <-live:
			if !ok {
				return
			}
			sw.event(ev.ID, ev.Name, ev.Data)
		case <-wake:
			sw.summary(s.host)
			if sub == nil {
				break
			}
			if s.host.Epoch(name) != sub.Epoch {
				sub.Close()
				sub = s.host.Subscribe(name, "")
				sw.open(sub)
			} else if data, ok := sub.Snapshot(); ok {
				sw.event("", "session", data)
			}
		}
		if rc.Flush() != nil {
			return
		}
	}
}
