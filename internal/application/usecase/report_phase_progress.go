package usecase

import (
	"context"
	"fmt"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/repository"
	"kanbanai/pkg/uid"
	"time"
)

type ReportPhaseProgress struct {
	eventLogRepo repository.TaskEventLogRepository
	dispatcher   event.Dispatcher
}

func NewReportPhaseProgress(eventLogRepo repository.TaskEventLogRepository, dispatcher event.Dispatcher) *ReportPhaseProgress {
	return &ReportPhaseProgress{eventLogRepo: eventLogRepo, dispatcher: dispatcher}
}

func (uc *ReportPhaseProgress) Execute(ctx context.Context, taskID string, phase entity.Phase, message string) error {
	log := &entity.TaskEventLog{
		ID:        uid.New(),
		TaskID:    taskID,
		EventType: string(event.PhaseEvent(string(phase), "progress")),
		Phase:     phase,
		Message:   message,
		CreatedAt: time.Now(),
	}

	if err := uc.eventLogRepo.Create(ctx, log); err != nil {
		return fmt.Errorf("create event log: %w", err)
	}

	uc.dispatcher.Publish(event.Event{
		Type:    event.PhaseEvent(string(phase), "progress"),
		TaskID:  taskID,
		Payload: map[string]any{"phase": phase, "message": message},
		Timestamp: time.Now(),
	})

	return nil
}
