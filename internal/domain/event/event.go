package event

import "time"

type Event struct {
	Type      EventType
	Payload   any
	Timestamp time.Time
	TaskID    string
}
