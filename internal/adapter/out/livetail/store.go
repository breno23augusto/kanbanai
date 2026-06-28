// Package livetail holds an in-memory ring buffer of each running harness's
// stdout/stderr and broadcasts new chunks to SSE subscribers in real time.
//
// The harness adapter captures process output into a bytes.Buffer (used only to
// explain a failure after the fact). livetail mirrors the same stream live so
// the operator can watch an agent work instead of staring at a static card.
package livetail

import (
	"io"
	"sync"
	"time"
)

const (
	// maxBytes bounds the replay snapshot kept per task so a long-running
	// harness cannot grow memory unbounded. New viewers get the recent tail.
	maxBytes = 64 * 1024
	// subChanCap lets a slow viewer drop chunks instead of blocking the writer
	// (the harness pipe must never stall on a web client).
	subChanCap = 256
)

// Chunk is one streamed piece of harness output.
type Chunk struct {
	Text string `json:"text"`
	End  bool   `json:"end"`
}

// Stream is the live buffer for a single task.
type Stream struct {
	mu    sync.Mutex
	buf   []byte
	subs  map[chan Chunk]struct{}
	ended bool
	mtime time.Time
}

// Store keeps a Stream per taskID.
type Store struct {
	mu      sync.Mutex
	streams map[string]*Stream
}

func NewStore() *Store {
	return &Store{streams: make(map[string]*Stream)}
}

// Open creates a fresh stream for a task, replacing any prior one. Called by
// the harness adapter at Dispatch so each phase run starts a clean tail.
func (s *Store) Open(taskID string) *Stream {
	st := &Stream{subs: make(map[chan Chunk]struct{}), mtime: time.Now()}
	s.mu.Lock()
	s.streams[taskID] = st
	s.mu.Unlock()
	return st
}

// Get returns the stream for a task, or nil if none was ever opened.
func (s *Store) Get(taskID string) *Stream {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.streams[taskID]
}

// Delete removes the stream (e.g. when a task is deleted).
func (s *Store) Delete(taskID string) {
	s.mu.Lock()
	st := s.streams[taskID]
	delete(s.streams, taskID)
	s.mu.Unlock()
	if st != nil {
		st.End()
	}
}

// Writer returns an io.Writer that appends everything written to the stream
// and broadcasts it to subscribers. Writes never block on subscribers.
func (s *Store) Writer(taskID string) io.Writer {
	return &streamWriter{store: s, taskID: taskID}
}

// Append writes raw bytes into the stream buffer and fans them out.
func (s *Store) Append(taskID string, p []byte) {
	st := s.Get(taskID)
	if st == nil {
		return
	}
	st.append(p)
}

// Snapshot returns a copy of the buffered tail for replay on connect.
func (s *Store) Snapshot(taskID string) ([]byte, bool) {
	st := s.Get(taskID)
	if st == nil {
		return nil, false
	}
	return st.snapshot()
}

// Subscribe registers a viewer. The returned channel receives the buffered
// snapshot is NOT sent here (the caller sends it first); this channel only
// carries chunks written AFTER subscription. A final Chunk{End:true} is sent
// when the stream ends or is closed.
func (s *Store) Subscribe(taskID string) (chan Chunk, *Stream) {
	st := s.Get(taskID)
	if st == nil {
		return nil, nil
	}
	ch := make(chan Chunk, subChanCap)
	st.subscribe(ch)
	return ch, st
}

func (st *Stream) append(p []byte) {
	if len(p) == 0 {
		return
	}
	chunk := Chunk{Text: string(p)}
	st.mu.Lock()
	st.buf = append(st.buf, p...)
	if len(st.buf) > maxBytes {
		st.buf = st.buf[len(st.buf)-maxBytes:]
	}
	st.mtime = time.Now()
	subs := st.subs
	st.mu.Unlock()
	for ch := range subs {
		select {
		case ch <- chunk:
		default:
			// drop — slow viewer; the snapshot on (re)connect recovers context.
		}
	}
}

func (st *Stream) snapshot() ([]byte, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	out := make([]byte, len(st.buf))
	copy(out, st.buf)
	return out, st.ended
}

func (st *Stream) subscribe(ch chan Chunk) {
	st.mu.Lock()
	st.subs[ch] = struct{}{}
	ended := st.ended
	st.mu.Unlock()
	if ended {
		// Stream already finished — tell the subscriber immediately.
		select {
		case ch <- Chunk{End: true}:
		default:
		}
	}
}

func (st *Stream) Unsubscribe(ch chan Chunk) {
	st.mu.Lock()
	delete(st.subs, ch)
	st.mu.Unlock()
}

// End marks the stream finished and notifies all subscribers. The buffer is
// retained so a late viewer still gets the final tail.
func (st *Stream) End() {
	st.mu.Lock()
	if st.ended {
		st.mu.Unlock()
		return
	}
	st.ended = true
	subs := st.subs
	st.mu.Unlock()
	for ch := range subs {
		select {
		case ch <- Chunk{End: true}:
		default:
		}
	}
}

// Write implements io.Writer so the stream can be plugged directly into an
// io.MultiWriter alongside the failure-tail bytes.Buffer.
func (st *Stream) Write(p []byte) (int, error) {
	st.append(p)
	return len(p), nil
}

type streamWriter struct {
	store  *Store
	taskID string
}

func (w *streamWriter) Write(p []byte) (int, error) {
	w.store.Append(w.taskID, p)
	return len(p), nil
}
