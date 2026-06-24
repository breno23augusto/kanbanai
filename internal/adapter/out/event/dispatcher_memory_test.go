package eventimpl

import (
	"sync"
	"testing"
	"kanbanai/internal/domain/event"
)

func TestDispatcherSubscribe(t *testing.T) {
	d := NewDispatcherMemory()

	var received []event.EventType
	var mu sync.Mutex

	d.Subscribe(event.TaskCreated, func(evt event.Event) {
		mu.Lock()
		received = append(received, evt.Type)
		mu.Unlock()
	})

	d.Publish(event.Event{Type: event.TaskCreated, TaskID: "t1"})
	d.Publish(event.Event{Type: event.TaskUpdated, TaskID: "t1"})
	d.Publish(event.Event{Type: event.TaskCreated, TaskID: "t2"})

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 {
		t.Fatalf("expected 2 events, got %d", len(received))
	}
	if received[0] != event.TaskCreated || received[1] != event.TaskCreated {
		t.Errorf("unexpected events: %v", received)
	}
}

func TestDispatcherSubscribeAll(t *testing.T) {
	d := NewDispatcherMemory()

	var count int
	var mu sync.Mutex

	d.SubscribeAll(func(evt event.Event) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	d.Subscribe(event.TaskCreated, func(evt event.Event) {})

	d.Publish(event.Event{Type: event.TaskCreated})
	d.Publish(event.Event{Type: event.TaskUpdated})
	d.Publish(event.Event{Type: event.TaskDeleted})

	mu.Lock()
	defer mu.Unlock()
	// SubscribeAll receives all 3 events regardless of type
	if count != 3 {
		t.Errorf("expected 3 events via SubscribeAll, got %d", count)
	}
}

func TestDispatcherMultipleHandlers(t *testing.T) {
	d := NewDispatcherMemory()

	var callOrder []string
	var mu sync.Mutex

	d.Subscribe(event.TaskCreated, func(evt event.Event) {
		mu.Lock()
		callOrder = append(callOrder, "first")
		mu.Unlock()
	})
	d.Subscribe(event.TaskCreated, func(evt event.Event) {
		mu.Lock()
		callOrder = append(callOrder, "second")
		mu.Unlock()
	})

	d.Publish(event.Event{Type: event.TaskCreated})

	mu.Lock()
	defer mu.Unlock()
	if len(callOrder) != 2 {
		t.Fatalf("expected 2 handlers called, got %d", len(callOrder))
	}
}