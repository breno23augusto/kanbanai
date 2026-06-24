package sse

import (
	"encoding/json"
	"fmt"
	"kanbanai/internal/domain/event"
)

func FormatSSE(evt event.Event) (string, error) {
	data, err := json.Marshal(evt.Payload)
	if err != nil {
		return "", fmt.Errorf("marshal event payload: %w", err)
	}

	return fmt.Sprintf("event: %s\ndata: %s\n\n", evt.Type, string(data)), nil
}
