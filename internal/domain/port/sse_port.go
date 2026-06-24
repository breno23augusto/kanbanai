package port

import "kanbanai/internal/domain/event"

type SSEPort interface {
	Broadcast(event event.Event)
	Subscribe(clientID string) <-chan event.Event
	Unsubscribe(clientID string)
}
