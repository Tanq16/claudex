package attached

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"maps"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/tanq16/claudex/internal/images"
	"github.com/tanq16/claudex/internal/session"
)

var ErrClosing = errors.New("claudex is shutting down")

var noSession = []byte(`{"session":null,"init":null,"events":[]}`)

type Summary struct {
	State     string `json:"state"`
	Pending   int    `json:"pending"`
	Epoch     string `json:"epoch"`
	StartedAt int64  `json:"startedAt"`
}

type Anchor struct {
	Before string
	Whole  bool
}

type Host struct {
	images  *images.Store
	persist chan struct{}

	mu       sync.Mutex
	sessions map[string]*Session
	closing  bool

	watchMu  sync.Mutex
	watchers map[chan struct{}]struct{}
}

func NewHost(store *images.Store) *Host {
	return &Host{
		images:   store,
		persist:  make(chan struct{}, 1),
		sessions: map[string]*Session{},
		watchers: map[chan struct{}]struct{}{},
	}
}

func (h *Host) Start(cfg Config) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closing {
		return ErrClosing
	}
	if old, ok := h.sessions[cfg.Name]; ok && old.running() {
		return ErrRunning
	}
	s, err := spawn(h, cfg)
	if err != nil {
		return err
	}
	h.sessions[cfg.Name] = s
	h.notify()
	h.persistLater()
	return nil
}

func (h *Host) Stop(name string) error {
	h.mu.Lock()
	s, ok := h.sessions[name]
	delete(h.sessions, name)
	h.mu.Unlock()
	if !ok {
		return ErrNotRunning
	}
	s.kill()
	h.notify()
	h.persistLater()
	return nil
}

func (h *Host) Shutdown() {
	h.mu.Lock()
	h.closing = true
	list := slices.Collect(maps.Values(h.sessions))
	h.mu.Unlock()
	var wg sync.WaitGroup
	for _, s := range list {
		s.mu.Lock()
		s.shutdown = s.running()
		s.mu.Unlock()
		wg.Go(s.kill)
	}
	wg.Wait()
}

func (h *Host) Session(name string) (*Session, error) {
	h.mu.Lock()
	s, ok := h.sessions[name]
	h.mu.Unlock()
	if !ok || !s.running() {
		return nil, ErrNotRunning
	}
	return s, nil
}

func (h *Host) Live(name string) bool {
	_, err := h.Session(name)
	return err == nil
}

func (h *Host) list() map[string]*Session {
	h.mu.Lock()
	defer h.mu.Unlock()
	return maps.Clone(h.sessions)
}

func (h *Host) Settings() map[string]Settings {
	out := map[string]Settings{}
	for name, s := range h.list() {
		if set, ok := s.settings(); ok {
			out[name] = set
		}
	}
	return out
}

func (h *Host) Busy() map[string][]string {
	out := map[string][]string{}
	for name, s := range h.list() {
		if items := s.busy(); len(items) > 0 {
			out[name] = items
		}
	}
	return out
}

func (h *Host) Summary() map[string]Summary {
	out := map[string]Summary{}
	for name, s := range h.list() {
		s.mu.Lock()
		out[name] = Summary{State: s.r.snap.State, Pending: len(s.r.pending), Epoch: s.log.epoch, StartedAt: s.startedAt.UnixMilli()}
		s.mu.Unlock()
	}
	return out
}

func (h *Host) Epoch(name string) string {
	h.mu.Lock()
	s, ok := h.sessions[name]
	h.mu.Unlock()
	if !ok {
		return ""
	}
	return s.log.epoch
}

func (h *Host) History(name string) (Anchor, bool) {
	h.mu.Lock()
	s, ok := h.sessions[name]
	h.mu.Unlock()
	if !ok {
		return Anchor{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return Anchor{Before: s.log.anchor(), Whole: s.continued}, true
}

func (h *Host) Changed() <-chan struct{} {
	return h.persist
}

func (h *Host) persistLater() {
	select {
	case h.persist <- struct{}{}:
	default:
	}
}

func (h *Host) Watch() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	h.watchMu.Lock()
	h.watchers[ch] = struct{}{}
	h.watchMu.Unlock()
	return ch, func() {
		h.watchMu.Lock()
		delete(h.watchers, ch)
		h.watchMu.Unlock()
	}
}

func (h *Host) notify() {
	h.watchMu.Lock()
	defer h.watchMu.Unlock()
	for ch := range h.watchers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

type Subscription struct {
	Epoch   string
	Reset   []byte
	ResetID string
	Backlog []Event
	Live    <-chan Event
	ch      chan Event
	session *Session
	version uint64
}

func (h *Host) Subscribe(name, cursor string) *Subscription {
	h.mu.Lock()
	s, ok := h.sessions[name]
	h.mu.Unlock()
	if !ok {
		return &Subscription{Reset: noSession}
	}
	return s.subscribe(cursor)
}

func (s *Session) subscribe(cursor string) *Subscription {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub := &Subscription{Epoch: s.log.epoch, ch: make(chan Event, subscriberBuf), session: s, version: s.version}
	sub.Live = sub.ch
	s.subs[sub] = struct{}{}
	if backlog, ok := s.log.since(cursor); ok {
		sub.Backlog, sub.version = backlog, 0
		return sub
	}
	sub.ResetID = s.log.id(s.log.last)
	sub.Reset = s.resetLocked()
	return sub
}

func (s *Session) resetLocked() []byte {
	snap, _ := json.Marshal(s.snapshotLocked(), lenient)
	var b bytes.Buffer
	b.WriteString(`{"session":`)
	b.Write(snap)
	b.WriteString(`,"init":`)
	if len(s.r.init) > 0 {
		b.Write(s.r.init)
	} else {
		b.WriteString("null")
	}
	b.WriteString(`,"events":[`)
	for i, e := range s.log.entries {
		if i > 0 {
			b.WriteByte(',')
		}
		id, _ := jsontext.AppendQuote(nil, s.log.id(e.seq))
		name, _ := jsontext.AppendQuote(nil, e.name)
		b.WriteString(`{"id":`)
		b.Write(id)
		b.WriteString(`,"event":`)
		b.Write(name)
		b.WriteString(`,"data":`)
		b.Write(e.data)
		b.WriteByte('}')
	}
	b.WriteString("]}")
	return b.Bytes()
}

func (sub *Subscription) Close() {
	if sub.session == nil {
		return
	}
	s := sub.session
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subs[sub]; ok {
		delete(s.subs, sub)
		close(sub.ch)
	}
}

func (sub *Subscription) Snapshot() ([]byte, bool) {
	if sub.session == nil {
		return nil, false
	}
	s := sub.session
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.version == sub.version {
		return nil, false
	}
	sub.version = s.version
	data, err := json.Marshal(s.snapshotLocked(), lenient)
	return data, err == nil
}

func KillLeftover(pid int, startedAt time.Time) bool {
	if pid <= 0 {
		return false
	}
	started, ok := session.StartTimes([]int{pid})[pid]
	if !ok || started.Sub(startedAt).Abs() > 3*time.Second {
		return false
	}
	syscall.Kill(-pid, syscall.SIGTERM)
	for range 30 {
		if syscall.Kill(pid, 0) != nil {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	syscall.Kill(-pid, syscall.SIGKILL)
	return true
}
