package event

type Handler func(event Event)

type Dispatcher interface {
	Subscribe(eventType EventType, handler Handler)
	SubscribeAll(handler Handler)
	Publish(event Event)
}
