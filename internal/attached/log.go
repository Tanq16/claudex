package attached

import (
	"slices"
	"strconv"
	"strings"
)

const (
	maxLogEntries = 20000
	maxLogBytes   = 64 << 20
	subscriberBuf = 512
)

type Event struct {
	ID   string
	Name string
	Data []byte
}

type entry struct {
	seq    uint64
	name   string
	data   []byte
	anchor string
}

type eventLog struct {
	epoch    string
	entries  []entry
	last     uint64
	size     int
	complete bool
}

func (l *eventLog) append(name string, data []byte, anchor string) Event {
	l.last++
	l.entries = append(l.entries, entry{seq: l.last, name: name, data: data, anchor: anchor})
	l.size += len(data)
	if len(l.entries) > maxLogEntries || l.size > maxLogBytes {
		drop := max(len(l.entries)/10, 1)
		for _, e := range l.entries[:drop] {
			l.size -= len(e.data)
		}
		l.entries = slices.Delete(l.entries, 0, drop)
		l.complete = false
	}
	return l.event(l.entries[len(l.entries)-1])
}

func (l *eventLog) first() uint64 {
	return l.last + 1 - uint64(len(l.entries))
}

func (l *eventLog) restart(seq uint64) {
	first := l.first()
	if seq < first {
		return
	}
	drop := int(seq - first)
	for _, e := range l.entries[:drop] {
		l.size -= len(e.data)
	}
	l.entries = slices.Clone(l.entries[drop:])
	l.complete = true
}

func (l *eventLog) id(seq uint64) string {
	return l.epoch + ":" + strconv.FormatUint(seq, 10)
}

func (l *eventLog) event(e entry) Event {
	return Event{ID: l.id(e.seq), Name: e.name, Data: e.data}
}

func (l *eventLog) since(cursor string) ([]Event, bool) {
	epoch, rest, ok := strings.Cut(cursor, ":")
	if !ok || epoch != l.epoch {
		return nil, false
	}
	seq, err := strconv.ParseUint(rest, 10, 64)
	if err != nil || seq > l.last {
		return nil, false
	}
	first := l.first()
	if seq+1 < first {
		return nil, false
	}
	return l.events(l.entries[seq+1-first:]), true
}

func (l *eventLog) events(entries []entry) []Event {
	out := make([]Event, len(entries))
	for i, e := range entries {
		out[i] = l.event(e)
	}
	return out
}

func (l *eventLog) anchor() string {
	for _, e := range l.entries {
		if e.anchor != "" {
			return e.anchor
		}
	}
	return ""
}
