package sse

import (
	"kanbanai/internal/domain/event"
	"sync"
)

type Broker struct {
	mu         sync.RWMutex
	clients    map[string]chan event.Event
	dispatcher event.Dispatcher
}

func NewBroker(dispatcher event.Dispatcher) *Broker {
	b := &Broker{
		clients:    make(map[string]chan event.Event),
		dispatcher: dispatcher,
	}
	dispatcher.SubscribeAll(b.onEvent)
	return b
}

func (b *Broker) onEvent(evt event.Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.clients {
		select {
		case ch <- evt:
		default:
		}
	}
}

func (b *Broker) Subscribe(clientID string) <-chan event.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan event.Event, 64)
	b.clients[clientID] = ch
	return ch
}

func (b *Broker) Unsubscribe(clientID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ch, ok := b.clients[clientID]; ok {
		close(ch)
		delete(b.clients, clientID)
	}
}
