package usecase

import (
	"context"
	"fmt"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/repository"
	"kanbanai/internal/application/dto"
	"kanbanai/pkg/uid"
	"time"
)

type SavePhaseOutput struct {
	phaseOutputRepo repository.PhaseOutputRepository
	dispatcher      event.Dispatcher
}

func NewSavePhaseOutput(repo repository.PhaseOutputRepository, disp event.Dispatcher) *SavePhaseOutput {
	return &SavePhaseOutput{phaseOutputRepo: repo, dispatcher: disp}
}

func (uc *SavePhaseOutput) Execute(ctx context.Context, input dto.SavePhaseOutputInput) (*dto.PhaseOutputDTO, error) {
	now := time.Now()
	output := &entity.PhaseOutput{
		ID:        uid.New(),
		TaskID:    input.TaskID,
		Phase:     input.Phase,
		Output:    input.Output,
		Summary:   input.Summary,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := uc.phaseOutputRepo.Create(ctx, output); err != nil {
		return nil, fmt.Errorf("create phase output: %w", err)
	}

	uc.dispatcher.Publish(event.Event{
		Type:    event.PhaseEvent(string(input.Phase), "progress"),
		TaskID:  input.TaskID,
		Payload: map[string]any{"phase": input.Phase, "output": input.Output, "summary": input.Summary},
		Timestamp: now,
	})

	return &dto.PhaseOutputDTO{
		ID:        output.ID,
		TaskID:    output.TaskID,
		Phase:     output.Phase,
		Output:    output.Output,
		Summary:   output.Summary,
		CreatedAt: output.CreatedAt,
		UpdatedAt: output.UpdatedAt,
	}, nil
}
