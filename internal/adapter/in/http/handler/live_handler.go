package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"kanbanai/internal/adapter/out/livetail"

	"github.com/gin-gonic/gin"
)

// LiveHandler streams a running harness's stdout/stderr in real time over SSE.
// On connect it replays the buffered tail (so a viewer that joins mid-run sees
// recent context), then streams new chunks until the harness exits.
type LiveHandler struct {
	store *livetail.Store
}

func NewLiveHandler(store *livetail.Store) *LiveHandler {
	return &LiveHandler{store: store}
}

func (h *LiveHandler) Stream(c *gin.Context) {
	taskID := c.Param("id")

	// No stream yet (harness never ran, or task unknown): return an empty
	// ended stream so the frontend can show "no live output" cleanly.
	st := h.store.Get(taskID)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // disable proxy buffering

	flusher, _ := c.Writer.(http.Flusher)

	write := func(data any) {
		b, _ := json.Marshal(data)
		_, _ = c.Writer.Write([]byte("data: "))
		_, _ = c.Writer.Write(b)
		_, _ = c.Writer.Write([]byte("\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
	}

	// Replay the snapshot first.
	if st != nil {
		snap, _ := h.store.Snapshot(taskID)
		if len(snap) > 0 {
			write(livetail.Chunk{Text: string(snap)})
		}
	}

	// If no stream or already ended, signal end and close.
	if st == nil {
		write(livetail.Chunk{End: true})
		return
	}

	ch, st := h.store.Subscribe(taskID)
	if ch == nil {
		write(livetail.Chunk{End: true})
		return
	}
	defer func() {
		if st != nil {
			st.Unsubscribe(ch)
		}
	}()

	c.Stream(func(w io.Writer) bool {
		select {
		case chunk, ok := <-ch:
			if !ok {
				return false
			}
			write(chunk)
			return !chunk.End
		case <-c.Request.Context().Done():
			return false
		}
	})
}