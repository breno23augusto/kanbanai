package sse

import "kanbanai/internal/domain/event"

type Client struct {
	ID     string
	Events chan event.Event
}
