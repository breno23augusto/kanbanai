package eventimpl

import (
	"kanbanai/internal/domain/event"
	"sync"
)

type DispatcherMemory struct {
	mu            sync.RWMutex
	handlers      map[event.EventType][]event.Handler
	allHandler    []event.Handler
}

func NewDispatcherMemory() *DispatcherMemory {
	return &DispatcherMemory{
		handlers: make(map[event.EventType][]event.Handler),
	}
}

func (d *DispatcherMemory) Subscribe(eventType event.EventType, handler event.Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[eventType] = append(d.handlers[eventType], handler)
}

func (d *DispatcherMemory) SubscribeAll(handler event.Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.allHandler = append(d.allHandler, handler)
}

func (d *DispatcherMemory) Publish(evt event.Event) {
	d.mu.RLock()
	handlers := d.handlers[evt.Type]
	allHandlers := d.allHandler
	d.mu.RUnlock()

	for _, h := range handlers {
		h(evt)
	}
	for _, h := range allHandlers {
		h(evt)
	}
}
