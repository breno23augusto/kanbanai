package handler

import (
	"io"
	"kanbanai/internal/adapter/out/sse"
	"kanbanai/pkg/uid"

	"github.com/gin-gonic/gin"
)

type SSEHandler struct {
	broker *sse.Broker
}

func NewSSEHandler(broker *sse.Broker) *SSEHandler {
	return &SSEHandler{broker: broker}
}

func (h *SSEHandler) Stream(c *gin.Context) {
	clientID := uid.New()
	ch := h.broker.Subscribe(clientID)
	defer h.broker.Unsubscribe(clientID)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	c.Stream(func(w io.Writer) bool {
		select {
		case evt, ok := <-ch:
			if !ok {
				return false
			}
			_, err := sse.FormatSSE(evt)
			if err != nil {
				return false
			}
			c.SSEvent(string(evt.Type), evt.Payload)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
