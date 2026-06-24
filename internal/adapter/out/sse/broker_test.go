package sse

import (
	"testing"
	"kanbanai/internal/domain/event"
)

func TestBrokerSubscribeUnsubscribe(t *testing.T) {
	broker := NewBroker(NewMockDispatcher())
	ch1 := broker.Subscribe("client1")
	ch2 := broker.Subscribe("client2")

	broker.onEvent(event.Event{Type: "test.event", TaskID: "t1"})

	evt1 := <-ch1
	evt2 := <-ch2

	if evt1.TaskID != "t1" || evt2.TaskID != "t1" {
		t.Error("both clients should receive the event")
	}

	broker.Unsubscribe("client1")

	// After unsubscribe, no events should be sent to client1's closed channel
	broker.onEvent(event.Event{Type: "test.event2", TaskID: "t2"})

	// client2 should still receive
	evt2 = <-ch2
	if evt2.TaskID != "t2" {
		t.Error("client2 should receive event after client1 unsubscribed")
	}
}

func TestBrokerNonBlockingWhenBufferFull(t *testing.T) {
	broker := NewBroker(NewMockDispatcher())
	ch := broker.Subscribe("client1")

	// Fill the buffer (64 capacity)
	for i := 0; i < 64; i++ {
		broker.onEvent(event.Event{Type: "test.event", TaskID: "t1"})
	}

	// Publishing beyond buffer should not block (non-blocking send with default case)
	broker.onEvent(event.Event{Type: "test.event", TaskID: "overflow"})

	// Drain and verify we got 64 events
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			if count != 64 {
				t.Errorf("expected 64 buffered events, got %d", count)
			}
			return
		}
	}
}

// mockDispatcher implements event.Dispatcher for testing the broker in isolation.
type mockDispatcher struct{}

func (m *mockDispatcher) Subscribe(event.EventType, event.Handler)       {}
func (m *mockDispatcher) SubscribeAll(event.Handler)                       {}
func (m *mockDispatcher) Publish(event.Event)                             {}

func NewMockDispatcher() *mockDispatcher { return &mockDispatcher{} }