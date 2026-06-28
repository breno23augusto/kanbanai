package usecase

import (
	"context"
	"fmt"
	"time"

	"kanbanai/internal/application/dto"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/repository"
)

// UpdateSubtaskStatus advances a single subtask's status. It is the MCP entry
// point the doing/validating/testing harness calls as it completes each
// sub-activity, so the card and drawer reflect live per-subtask progress.
type UpdateSubtaskStatus struct {
	subtaskRepo repository.SubtaskRepository
	dispatcher  event.Dispatcher
}

func NewUpdateSubtaskStatus(repo repository.SubtaskRepository, disp event.Dispatcher) *UpdateSubtaskStatus {
	return &UpdateSubtaskStatus{subtaskRepo: repo, dispatcher: disp}
}

func (uc *UpdateSubtaskStatus) Execute(ctx context.Context, taskID, subtaskID string, status entity.SubtaskStatus) (*dto.SubtaskDTO, error) {
	if taskID == "" || subtaskID == "" {
		return nil, fmt.Errorf("task_id and subtask_id are required")
	}
	if !isValidSubtaskStatus(status) {
		return nil, fmt.Errorf("invalid subtask status: %s", status)
	}

	st, err := uc.subtaskRepo.Find(ctx, subtaskID)
	if err != nil {
		return nil, fmt.Errorf("find subtask: %w", err)
	}
	if st.TaskID != taskID {
		return nil, fmt.Errorf("subtask %s does not belong to task %s", subtaskID, taskID)
	}

	st.Status = status
	st.UpdatedAt = time.Now()
	if err := uc.subtaskRepo.Update(ctx, st); err != nil {
		return nil, fmt.Errorf("update subtask: %w", err)
	}

	dto := toSubtaskDTO(st)
	uc.dispatcher.Publish(event.Event{
		Type:      event.SubtaskUpdated,
		TaskID:    taskID,
		Payload:   map[string]any{"subtask": dto},
		Timestamp: st.UpdatedAt,
	})

	return dto, nil
}

func isValidSubtaskStatus(s entity.SubtaskStatus) bool {
	switch s {
	case entity.SubtaskPending, entity.SubtaskInProgress, entity.SubtaskCompleted:
		return true
	}
	return false
}