package usecase

import (
	"context"
	"fmt"
	"time"

	"kanbanai/internal/application/dto"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/repository"
	"kanbanai/pkg/uid"
)

// CreateSubtasks replaces the entire subtask set of a task with the provided
// list. It is the MCP entry point the planning harness calls to persist the
// subtasks/acceptance-criteria it identified. Replacing (rather than appending)
// keeps the set authoritative: if planning is re-dispatched, the fresh list
// wins. Existing subtask progress is NOT preserved on replace, because a
// re-plan means the breakdown itself changed.
type CreateSubtasks struct {
	subtaskRepo repository.SubtaskRepository
	dispatcher  event.Dispatcher
}

func NewCreateSubtasks(repo repository.SubtaskRepository, disp event.Dispatcher) *CreateSubtasks {
	return &CreateSubtasks{subtaskRepo: repo, dispatcher: disp}
}

func (uc *CreateSubtasks) Execute(ctx context.Context, taskID string, items []dto.SubtaskInput) ([]*dto.SubtaskDTO, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("at least one subtask is required")
	}

	// Replace semantics: clear any prior set for this task, then create the new
	// batch in order.
	if err := uc.subtaskRepo.DeleteByTask(ctx, taskID); err != nil {
		return nil, fmt.Errorf("clear subtasks: %w", err)
	}

	now := time.Now()
	result := make([]*dto.SubtaskDTO, 0, len(items))
	created := make([]*entity.Subtask, 0, len(items))
	for i, in := range items {
		if in.Title == "" {
			return nil, fmt.Errorf("subtask %d: title is required", i)
		}
		st := &entity.Subtask{
			ID:        uid.New(),
			TaskID:    taskID,
			Title:     in.Title,
			Status:    entity.SubtaskPending,
			Order:     i,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := uc.subtaskRepo.Create(ctx, st); err != nil {
			return nil, fmt.Errorf("create subtask %d: %w", i, err)
		}
		created = append(created, st)
		result = append(result, toSubtaskDTO(st))
	}

	uc.dispatcher.Publish(event.Event{
		Type:      event.SubtaskCreated,
		TaskID:    taskID,
		Payload:   map[string]any{"subtasks": result},
		Timestamp: now,
	})

	return result, nil
}

func toSubtaskDTO(st *entity.Subtask) *dto.SubtaskDTO {
	return &dto.SubtaskDTO{
		ID:        st.ID,
		TaskID:    st.TaskID,
		Title:     st.Title,
		Status:    st.Status,
		Order:     st.Order,
		CreatedAt: st.CreatedAt,
		UpdatedAt: st.UpdatedAt,
	}
}

func toSubtaskDTOs(items []*entity.Subtask) []dto.SubtaskDTO {
	if len(items) == 0 {
		return []dto.SubtaskDTO{}
	}
	out := make([]dto.SubtaskDTO, 0, len(items))
	for _, st := range items {
		out = append(out, *toSubtaskDTO(st))
	}
	return out
}

func toEntitySubtasks(items []*entity.Subtask) []entity.Subtask {
	out := make([]entity.Subtask, 0, len(items))
	for _, st := range items {
		out = append(out, *st)
	}
	return out
}