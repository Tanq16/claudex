package attached

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"uuid"
)

const (
	stopGrace      = 5 * time.Second
	drainGrace     = 3 * time.Second
	stderrTail     = 8 << 10
	contextTimeout = 30 * time.Second
)

var (
	ErrRunning    = errors.New("an attached session is already running")
	ErrNotRunning = errors.New("no attached session is running")
	ErrNoRequest  = errors.New("no such pending request")
	ErrBadAnswer  = errors.New("invalid answer")
)

type RefusedError struct {
	Subtype string
	Message string
}

func (e *RefusedError) Error() string { return e.Subtype + ": " + e.Message }

var ignored = map[string]bool{
	"hook_started":             true,
	"hook_progress":            true,
	"hook_response":            true,
	"plugin_install":           true,
	"worker_shutting_down":     true,
	"mirror_error":             true,
	"control_request_progress": true,
	"files_persisted":          true,
}

var stripped = []string{"CLAUDECODE", "CLAUDE_CODE_CHILD_SESSION", "CLAUDE_CODE_ENTRYPOINT"}

type Config struct {
	Name           string
	Dir            string
	Env            []string
	Args           []string
	Continue       bool
	Model          string
	Effort         string
	PermissionMode string
}

type Settings struct {
	Model          string
	Effort         string
	PermissionMode string
	PID            int
	StartedAt      time.Time
}

type reply struct {
	payload jsontext.Value
	err     error
}

type Session struct {
	host      *Host
	name      string
	continued bool
	cmd       *exec.Cmd
	startedAt time.Time
	queue     chan []byte
	exited    chan struct{}
	drained   chan struct{}
	finished  chan struct{}
	requests  atomic.Uint64
	wmu       sync.Mutex
	stderr    tail

	mu       sync.Mutex
	log      eventLog
	r        reduced
	version  uint64
	subs     map[*Subscription]struct{}
	waiting  map[string]chan reply
	shutdown bool
}

func spawn(h *Host, cfg Config) (*Session, error) {
	claude, err := exec.LookPath("claude")
	if err != nil {
		return nil, err
	}
	args := []string{
		"-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose",
		"--permission-prompt-tool=stdio", "--thinking-display", "summarized", "--allow-dangerously-skip-permissions",
	}
	if cfg.Continue {
		args = append(args, "--continue")
	}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	if cfg.Effort != "" {
		args = append(args, "--effort", cfg.Effort)
	}
	if cfg.PermissionMode != "" && cfg.PermissionMode != "default" {
		args = append(args, "--permission-mode", cfg.PermissionMode)
	}
	cmd := exec.Command(claude, append(args, cfg.Args...)...)
	cmd.Dir = cfg.Dir
	cmd.Env = environ(cfg.Env)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = drainGrace

	s := &Session{
		host:      h,
		name:      cfg.Name,
		continued: cfg.Continue,
		cmd:       cmd,
		queue:     make(chan []byte, 256),
		exited:    make(chan struct{}),
		drained:   make(chan struct{}),
		finished:  make(chan struct{}),
		version:   1,
		subs:      map[*Subscription]struct{}{},
		waiting:   map[string]chan reply{},
	}
	s.r = reduced{
		snap: Snapshot{
			State:          StateStarting,
			Model:          cfg.Model,
			Effort:         cfg.Effort,
			PermissionMode: cfg.PermissionMode,
			Tasks:          []Task{},
			Todos:          []Todo{},
		},
		taskInfo:  map[string]Task{},
		todoCalls: map[string]todoCall{},
	}
	cmd.Stderr = &s.stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdout = pw
	if err := cmd.Start(); err != nil {
		stdout.Close()
		pw.Close()
		return nil, err
	}
	pw.Close()
	s.startedAt = time.Now()
	s.log.epoch = strconv.FormatInt(s.startedAt.UnixMilli(), 36)

	go s.write(stdin)
	go s.read(stdout)
	go s.wait(stdout)
	go s.initialize()
	return s, nil
}

func environ(base []string) []string {
	env := slices.DeleteFunc(slices.Clone(base), func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		return slices.Contains(stripped, name)
	})
	return append(env,
		"CLAUDE_CODE_ENTRYPOINT=claudex",
		"CLAUDE_CODE_EMIT_SESSION_STATE_EVENTS=1",
		"CLAUDE_CODE_ENABLE_TODO_TOOLS=1",
	)
}

func (s *Session) write(stdin io.WriteCloser) {
	defer stdin.Close()
	for {
		select {
		case line := <-s.queue:
			if _, err := stdin.Write(line); err != nil {
				s.report(WriteFailed, err)
				return
			}
		case <-s.exited:
			return
		}
	}
}

func (s *Session) read(stdout io.Reader) {
	defer close(s.drained)
	br := bufio.NewReaderSize(stdout, 64<<10)
	for {
		line, err := br.ReadBytes('\n')
		if line = bytes.TrimSpace(line); len(line) > 0 {
			s.handle(line)
		}
		if err != nil {
			return
		}
	}
}

func (s *Session) wait(stdout *os.File) {
	err := s.cmd.Wait()
	close(s.exited)
	select {
	case <-s.drained:
	case <-time.After(drainGrace):
		stdout.Close()
		<-s.drained
	}
	stdout.Close()
	syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)

	exit := &Exit{Code: s.cmd.ProcessState.ExitCode(), Stderr: s.stderr.String()}
	if err != nil {
		exit.Error = err.Error()
	}
	s.mu.Lock()
	s.r.snap.State, s.r.snap.Exit, s.r.pending = StateExited, exit, nil
	s.r.snap.Tasks = []Task{}
	s.changedLocked()
	s.mu.Unlock()
	s.host.report(Report{Session: s.name, Kind: Exited, Code: exit.Code})
	close(s.finished)
	s.host.notify()
	s.host.persistLater()
}

func (s *Session) initialize() {
	reply, err := s.request(context.Background(), "initialize", map[string]any{"perTaskStopAffordance": true})
	if err != nil {
		s.report(InitializeFailed, err)
		return
	}
	s.mu.Lock()
	s.r.applyInit(reply)
	if s.r.snap.State == StateStarting {
		s.r.snap.State = StateIdle
	}
	s.changedLocked()
	s.mu.Unlock()
	s.host.notify()
	s.refreshSettings()
	s.refreshContext()
}

func (s *Session) handle(line []byte) {
	var h struct {
		Type      string `json:"type"`
		Subtype   string `json:"subtype"`
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(line, &h, lenient); err != nil {
		s.report(LineUnreadable, err)
		return
	}
	kind := h.Type
	switch h.Type {
	case "control_response":
		s.deliver(line)
		return
	case "control_request":
		s.inbound(h.RequestID, line)
		return
	case "control_cancel_request":
		s.withdrawn(h.RequestID)
		return
	case "keep_alive":
		return
	case "system":
		if ignored[h.Subtype] {
			return
		}
		kind += "/" + h.Subtype
	case "user":
		rewritten, err := s.host.images.Rewrite(line)
		if err != nil {
			s.report(ImageNotStored, err)
		}
		line = rewritten
	}

	s.mu.Lock()
	c := s.r.reduce(kind, line)
	s.publishLocked(s.log.append("message", line, c.anchor))
	if c.snapshot {
		s.changedLocked()
	}
	s.mu.Unlock()
	if c.snapshot {
		s.host.notify()
	}
	if c.settings {
		s.host.persistLater()
	}
	if kind == "result" || kind == "system/compact_boundary" {
		go s.refreshContext()
	}
	if kind == "result" {
		go s.refreshSettings()
	}
}

func (s *Session) deliver(line []byte) {
	var m struct {
		Response struct {
			Subtype   string         `json:"subtype"`
			RequestID string         `json:"request_id"`
			Response  jsontext.Value `json:"response"`
			Error     string         `json:"error"`
		} `json:"response"`
	}
	if json.Unmarshal(line, &m, lenient) != nil {
		return
	}
	s.mu.Lock()
	ch, ok := s.waiting[m.Response.RequestID]
	s.mu.Unlock()
	if !ok {
		return
	}
	if m.Response.Subtype == "error" {
		ch <- reply{err: &RefusedError{Message: m.Response.Error}}
		return
	}
	ch <- reply{payload: m.Response.Response}
}

func (s *Session) inbound(id string, line []byte) {
	var m struct {
		Request struct {
			Subtype  string `json:"subtype"`
			ToolName string `json:"tool_name"`
		} `json:"request"`
	}
	json.Unmarshal(line, &m, lenient)
	switch m.Request.Subtype {
	case "can_use_tool", "elicitation":
		s.mu.Lock()
		s.r.addPending(id, m.Request.Subtype, m.Request.ToolName, line)
		s.publishLocked(s.log.append("message", line, ""))
		s.changedLocked()
		s.mu.Unlock()
		s.host.notify()
	default:
		s.respond(context.Background(), id, nil, "claudex does not handle "+m.Request.Subtype)
	}
}

func (s *Session) withdrawn(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.r.takePending(id); !ok {
		return
	}
	data, _ := json.Marshal(map[string]any{"request_id": id, "cancelled": true})
	s.publishLocked(s.log.append("approval", data, ""))
	s.changedLocked()
	s.host.notify()
}

func (s *Session) respond(ctx context.Context, id string, payload jsontext.Value, refusal string) error {
	resp := map[string]any{"subtype": "success", "request_id": id, "response": payload}
	if refusal != "" {
		resp = map[string]any{"subtype": "error", "request_id": id, "error": refusal}
	}
	line, err := json.Marshal(map[string]any{"type": "control_response", "response": resp})
	if err != nil {
		return err
	}
	return s.enqueue(ctx, line)
}

func (s *Session) enqueue(ctx context.Context, line []byte) error {
	select {
	case s.queue <- append(line, '\n'):
		return nil
	case <-s.exited:
		return ErrNotRunning
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Session) request(ctx context.Context, subtype string, fields map[string]any) (jsontext.Value, error) {
	id := "claudex-" + strconv.FormatUint(s.requests.Add(1), 10)
	req := map[string]any{"subtype": subtype}
	maps.Copy(req, fields)
	line, err := json.Marshal(map[string]any{"type": "control_request", "request_id": id, "request": req})
	if err != nil {
		return nil, err
	}
	ch := make(chan reply, 1)
	s.mu.Lock()
	s.waiting[id] = ch
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.waiting, id)
		s.mu.Unlock()
	}()
	if err := s.enqueue(ctx, line); err != nil {
		return nil, err
	}
	select {
	case r := <-ch:
		if refused, ok := errors.AsType[*RefusedError](r.err); ok {
			refused.Subtype = subtype
		}
		return r.payload, r.err
	case <-s.exited:
		return nil, ErrNotRunning
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Session) publishLocked(ev Event) {
	for sub := range s.subs {
		select {
		case sub.ch <- ev:
		default:
			close(sub.ch)
			delete(s.subs, sub)
		}
	}
}

func (s *Session) changedLocked() {
	s.version++
}

func (s *Session) running() bool {
	select {
	case <-s.exited:
		return false
	default:
		return true
	}
}

func (s *Session) kill() {
	pid := s.cmd.Process.Pid
	syscall.Kill(-pid, syscall.SIGTERM)
	select {
	case <-s.finished:
		return
	case <-time.After(stopGrace):
	}
	syscall.Kill(-pid, syscall.SIGKILL)
	<-s.finished
}

func (s *Session) settings() (Settings, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running() && !s.shutdown {
		return Settings{}, false
	}
	return Settings{
		Model:          s.r.snap.Model,
		Effort:         s.r.snap.Effort,
		PermissionMode: s.r.snap.PermissionMode,
		PID:            s.cmd.Process.Pid,
		StartedAt:      s.startedAt,
	}, true
}

func (s *Session) snapshotLocked() Snapshot {
	snap := s.r.public()
	snap.Name, snap.Epoch, snap.PID, snap.StartedAt = s.name, s.log.epoch, s.cmd.Process.Pid, s.startedAt.UnixMilli()
	return snap
}

func (s *Session) Prompt(ctx context.Context, text string, blocks []any) (string, error) {
	if strings.TrimSpace(text) == "" && len(blocks) == 0 {
		return "", fmt.Errorf("%w: the prompt is empty", ErrBadAnswer)
	}
	var content, shown any = text, text
	if len(blocks) > 0 {
		if strings.TrimSpace(text) != "" {
			blocks = append(blocks, map[string]any{"type": "text", "text": text})
		}
		content, shown = blocks, blocks
		raw, err := json.Marshal(blocks)
		if err != nil {
			return "", err
		}
		if stored, err := s.host.images.Rewrite(raw); err != nil {
			s.report(ImageNotStored, err)
		} else {
			shown = jsontext.Value(stored)
		}
	}
	id := uuid.New().String()
	wire, err := json.Marshal(map[string]any{
		"type":               "user",
		"uuid":               id,
		"session_id":         "",
		"parent_tool_use_id": nil,
		"message":            map[string]any{"role": "user", "content": content},
	})
	if err != nil {
		return "", err
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if !s.running() {
		return "", ErrNotRunning
	}
	s.mu.Lock()
	event, err := json.Marshal(map[string]any{
		"type":               "user",
		"uuid":               id,
		"session_id":         s.r.snap.SessionID,
		"parent_tool_use_id": nil,
		"timestamp":          time.Now().UTC().Format(time.RFC3339Nano),
		"message":            map[string]any{"role": "user", "content": shown},
	})
	if err == nil {
		anchor := id
		if strings.HasPrefix(strings.TrimSpace(text), "/") {
			anchor = ""
		}
		s.publishLocked(s.log.append("message", event, anchor))
	}
	s.mu.Unlock()
	if err != nil {
		return "", err
	}
	return id, s.enqueue(ctx, wire)
}

func (s *Session) Answer(ctx context.Context, id string, payload jsontext.Value) error {
	var answer struct {
		Behavior string `json:"behavior"`
		Message  string `json:"message"`
		Action   string `json:"action"`
	}
	if err := json.Unmarshal(payload, &answer, lenient); err != nil || payload.Kind() != '{' {
		return fmt.Errorf("%w: the answer must be a JSON object", ErrBadAnswer)
	}
	s.mu.Lock()
	i := slices.IndexFunc(s.r.pending, func(p pendingRequest) bool { return p.id == id })
	if i < 0 {
		s.mu.Unlock()
		return ErrNoRequest
	}
	subtype := s.r.pending[i].subtype
	s.mu.Unlock()

	switch {
	case subtype == "elicitation" && !slices.Contains([]string{"accept", "decline", "cancel"}, answer.Action):
		return fmt.Errorf("%w: action must be accept, decline, or cancel", ErrBadAnswer)
	case subtype == "can_use_tool" && answer.Behavior != "allow" && answer.Behavior != "deny":
		return fmt.Errorf("%w: behavior must be allow or deny", ErrBadAnswer)
	case subtype == "can_use_tool" && answer.Behavior == "deny" && answer.Message == "":
		var m map[string]any
		json.Unmarshal(payload, &m, lenient)
		m["message"] = "Denied in claudex"
		payload, _ = json.Marshal(m, lenient)
	}

	s.mu.Lock()
	_, ok := s.r.takePending(id)
	s.mu.Unlock()
	if !ok {
		return ErrNoRequest
	}
	if err := s.respond(ctx, id, payload, ""); err != nil {
		return err
	}
	data, err := json.Marshal(map[string]any{"request_id": id, "response": payload}, lenient)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.publishLocked(s.log.append("approval", data, ""))
	s.changedLocked()
	s.mu.Unlock()
	s.host.notify()
	return nil
}

func (s *Session) Interrupt(ctx context.Context) (jsontext.Value, error) {
	return s.request(ctx, "interrupt", nil)
}

func (s *Session) StopTask(ctx context.Context, taskID string) (jsontext.Value, error) {
	return s.request(ctx, "stop_task", map[string]any{"task_id": taskID})
}

func (s *Session) TaskOutput(ctx context.Context, taskID string) (jsontext.Value, error) {
	return s.request(ctx, "get_task_output", map[string]any{"task_id": taskID})
}

func (s *Session) SetModel(ctx context.Context, model string) error {
	if _, err := s.request(ctx, "set_model", map[string]any{"model": model}); err != nil {
		return err
	}
	s.update(func(r *reduced) { r.snap.Model = model })
	go s.refreshSettings()
	return nil
}

func (s *Session) SetEffort(ctx context.Context, effort string) error {
	if _, err := s.request(ctx, "apply_flag_settings", map[string]any{"settings": map[string]any{"effortLevel": effort}}); err != nil {
		return err
	}
	s.update(func(r *reduced) { r.snap.Effort, r.snap.AppliedEffort = effort, effort })
	go s.refreshSettings()
	return nil
}

func (s *Session) SetPermissionMode(ctx context.Context, mode string) error {
	if _, err := s.request(ctx, "set_permission_mode", map[string]any{"mode": mode}); err != nil {
		return err
	}
	s.update(func(r *reduced) { r.snap.PermissionMode = mode })
	return nil
}

func (s *Session) ContextUsage(ctx context.Context, detail string) (*Context, error) {
	fields := map[string]any{}
	if detail != "" {
		fields["detail"] = detail
	}
	raw, err := s.request(ctx, "get_context_usage", fields)
	if err != nil {
		return nil, err
	}
	var usage Context
	if err := json.Unmarshal(raw, &usage, lenient); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.r.snap.Context = &usage
	s.changedLocked()
	s.mu.Unlock()
	s.host.notify()
	return &usage, nil
}

func (s *Session) refreshContext() {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()
	if _, err := s.ContextUsage(ctx, "summary"); err != nil && !errors.Is(err, ErrNotRunning) {
		s.report(ContextRefreshFailed, err)
	}
}

func (s *Session) refreshSettings() {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()
	raw, err := s.request(ctx, "get_settings", nil)
	if err != nil {
		if !errors.Is(err, ErrNotRunning) {
			s.report(SettingsRefreshFailed, err)
		}
		return
	}
	var m struct {
		Applied struct {
			Effort string `json:"effort"`
		} `json:"applied"`
	}
	if err := json.Unmarshal(raw, &m, lenient); err != nil {
		s.report(SettingsRefreshFailed, err)
		return
	}
	s.mu.Lock()
	s.r.snap.AppliedEffort = m.Applied.Effort
	s.changedLocked()
	s.mu.Unlock()
	s.host.notify()
}

func (s *Session) report(kind ReportKind, err error) {
	s.host.report(Report{Session: s.name, Kind: kind, Err: err})
}

func (s *Session) update(fn func(r *reduced)) {
	s.mu.Lock()
	fn(&s.r)
	s.changedLocked()
	s.mu.Unlock()
	s.host.notify()
	s.host.persistLater()
}

func (s *Session) busy() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running() {
		return nil
	}
	var items []string
	switch s.r.snap.State {
	case StateRunning:
		items = append(items, "a turn is running")
	case StateRequiresAction:
		items = append(items, "a turn is waiting on input")
	}
	for _, p := range s.r.pending {
		if p.subtype == "can_use_tool" {
			items = append(items, "approval pending for "+p.tool)
		}
	}
	for _, t := range s.r.snap.Tasks {
		if !t.Ambient {
			items = append(items, fmt.Sprintf("%s task: %s", t.Type, t.Description))
		}
	}
	return items
}

type tail struct {
	mu  sync.Mutex
	buf []byte
}

func (t *tail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if over := len(t.buf) - stderrTail; over > 0 {
		t.buf = slices.Delete(t.buf, 0, over)
	}
	return len(p), nil
}

func (t *tail) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return strings.ToValidUTF8(string(t.buf), "�")
}
