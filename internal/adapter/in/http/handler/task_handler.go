package handler

import (
	"errors"
	"fmt"

	"kanbanai/internal/adapter/in/http/response"
	"kanbanai/internal/application/dto"
	"kanbanai/internal/application/usecase"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/port"
	"kanbanai/internal/domain/query"
	"kanbanai/internal/domain/repository"

	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	createTaskUC   *usecase.CreateTask
	updateTaskUC   *usecase.UpdateTask
	deleteTaskUC   *usecase.DeleteTask
	getTaskUC      *usecase.GetTask
	listTasksUC    *usecase.ListTasks
	advancePhaseUC *usecase.AdvancePhase
	orchestrator   port.PhaseOrchestratorPort
	timelineQuery  query.TaskTimelineQuery
}

func NewTaskHandler(
	createTaskUC *usecase.CreateTask,
	updateTaskUC *usecase.UpdateTask,
	deleteTaskUC *usecase.DeleteTask,
	getTaskUC *usecase.GetTask,
	listTasksUC *usecase.ListTasks,
	advancePhaseUC *usecase.AdvancePhase,
	orchestrator port.PhaseOrchestratorPort,
	timelineQuery query.TaskTimelineQuery,
) *TaskHandler {
	return &TaskHandler{
		createTaskUC:   createTaskUC,
		updateTaskUC:   updateTaskUC,
		deleteTaskUC:   deleteTaskUC,
		getTaskUC:      getTaskUC,
		listTasksUC:    listTasksUC,
		advancePhaseUC: advancePhaseUC,
		orchestrator:   orchestrator,
		timelineQuery:  timelineQuery,
	}
}

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
}

type updateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	Version     int    `json:"version"`
}

func (h *TaskHandler) Create(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	input := dto.CreateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
	}

	result, err := h.createTaskUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Created(c, result)
}

func (h *TaskHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req updateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	input := dto.CreateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
	}

	result, err := h.updateTaskUC.Execute(c.Request.Context(), id, input, req.Version)
	if err != nil {
		if errors.Is(err, repository.ErrConcurrentModification) {
			response.Conflict(c, "The task version has changed. Please reload the data and try again.")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, result)
}

func (h *TaskHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.deleteTaskUC.Execute(c.Request.Context(), id); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.NoContent(c)
}

func (h *TaskHandler) Get(c *gin.Context) {
	id := c.Param("id")

	result, err := h.getTaskUC.Execute(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "task not found")
		return
	}

	taskOutput := dto.TaskOutput{
		ID:           result.Task.ID,
		Title:        result.Task.Title,
		Description:  result.Task.Description,
		CurrentPhase: result.Task.CurrentPhase,
		Status:       result.Task.Status,
		Priority:     result.Task.Priority,
		Version:      result.Task.Version,
		ErrorMessage: result.Task.ErrorMessage,
		CreatedAt:    result.Task.CreatedAt,
		UpdatedAt:    result.Task.UpdatedAt,
	}

	var phaseOutputs []dto.PhaseOutputDTO
	for _, po := range result.PhaseOutputs {
		phaseOutputs = append(phaseOutputs, dto.PhaseOutputDTO{
			ID:        po.ID,
			TaskID:    po.TaskID,
			Phase:     po.Phase,
			Output:    po.Output,
			Summary:   po.Summary,
			CreatedAt: po.CreatedAt,
			UpdatedAt: po.UpdatedAt,
		})
	}

	response.OK(c, gin.H{
		"task":          taskOutput,
		"phase_outputs": phaseOutputs,
	})
}

func (h *TaskHandler) List(c *gin.Context) {
	filter := dto.TaskFilter{}

	if phase := c.Query("phase"); phase != "" {
		p := entity.Phase(phase)
		filter.Phase = &p
	}
	if status := c.Query("status"); status != "" {
		s := entity.Status(status)
		filter.Status = &s
	}

	tasks, err := h.listTasksUC.Execute(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, tasks)
}

func (h *TaskHandler) GetTimeline(c *gin.Context) {
	id := c.Param("id")

	result, err := h.timelineQuery.Get(id)
	if err != nil {
		response.NotFound(c, "task not found")
		return
	}

	response.OK(c, result)
}

// Retry restarts the flow for the task's current phase, resetting the retry
// counter. Used to unstick tasks in a failed state (SPEC §16.1 / §32.3).
func (h *TaskHandler) Retry(c *gin.Context) {
	id := c.Param("id")

	if err := h.orchestrator.RestartPhase(c.Request.Context(), id); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{
		"task_id": id,
		"message": "retry initiated",
	})
}

// Pause stops the running harness for the task and marks it as paused. Only
// valid for tasks currently in_progress. A paused task can be edited via the
// regular PUT /tasks/:id endpoint and later resumed.
func (h *TaskHandler) Pause(c *gin.Context) {
	id := c.Param("id")

	if err := h.orchestrator.PauseTask(c.Request.Context(), id); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{
		"task_id": id,
		"message": "task paused",
	})
}

// Resume re-dispatches the current phase of a paused task, returning it to
// in_progress. Only valid for tasks in the paused state.
func (h *TaskHandler) Resume(c *gin.Context) {
	id := c.Param("id")

	if err := h.orchestrator.ResumeTask(c.Request.Context(), id); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{
		"task_id": id,
		"message": "task resumed",
	})
}

type completePhaseRequest struct {
	Phase   string `json:"phase"`
	Summary string `json:"summary"`
}

type reopenPhaseRequest struct {
	TargetPhase string `json:"target_phase"`
	Reason      string `json:"reason"`
}

// CompletePhase bridges non-MCP harnesses (e.g. pi, which has no MCP support)
// into the phase-advancement flow. A harness wrapper, after finishing its work,
// POSTs here to mark the task's current phase completed. This invokes the same
// AdvancePhase use case the complete_phase MCP tool uses, so the orchestrator
// reacts via the phase.<phase>.completed event and advances the lane.
func (h *TaskHandler) CompletePhase(c *gin.Context) {
	id := c.Param("id")

	var req completePhaseRequest
	_ = c.ShouldBindJSON(&req) // body optional; phase defaults to current

	ctx := c.Request.Context()

	result, err := h.getTaskUC.Execute(ctx, id)
	if err != nil {
		response.NotFound(c, "task not found")
		return
	}

	phase := entity.Phase(req.Phase)
	if phase == "" {
		phase = result.Task.CurrentPhase
	}
	if phase != result.Task.CurrentPhase {
		response.BadRequest(c, fmt.Sprintf("phase %s is not the current phase (current: %s)", phase, result.Task.CurrentPhase))
		return
	}
	if result.Task.Status != entity.StatusInProgress && result.Task.Status != entity.StatusPending {
		response.Conflict(c, fmt.Sprintf("task is not active (status=%s)", result.Task.Status))
		return
	}

	if err := h.advancePhaseUC.Execute(ctx, id, phase, req.Summary); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{"task_id": id, "phase": phase, "status": "completed"})
}

// Reopen moves the task BACK to an earlier phase (e.g. from validating back
// to doing) and re-dispatches it, so problems found during validation get
// reworked instead of carried forward (SPEC §6.3.7). This is the HTTP fallback
// for harnesses that have no MCP client (e.g. pi): after a validation run
// detects failures, the harness wrapper POSTs here to send the task back to
// doing and trigger a fresh Doing dispatch. The body may omit target_phase, in
// which case the phase immediately preceding the current one is used.
func (h *TaskHandler) Reopen(c *gin.Context) {
	id := c.Param("id")

	var req reopenPhaseRequest
	_ = c.ShouldBindJSON(&req)

	ctx := c.Request.Context()

	result, err := h.getTaskUC.Execute(ctx, id)
	if err != nil {
		response.NotFound(c, "task not found")
		return
	}
	if result.Task.Status != entity.StatusInProgress && result.Task.Status != entity.StatusPending {
		response.Conflict(c, fmt.Sprintf("task is not active (status=%s)", result.Task.Status))
		return
	}

	target := entity.Phase(req.TargetPhase)
	if target == "" {
		prev, ok := result.Task.CurrentPhase.Prev()
		if !ok {
			response.BadRequest(c, fmt.Sprintf("no previous phase to reopen from %s", result.Task.CurrentPhase))
			return
		}
		target = prev
	}

	if err := h.orchestrator.ReopenPhase(ctx, id, target, req.Reason); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{
		"task_id":       id,
		"target_phase":  string(target),
		"status":        "reopened",
		"message":       "task moved back and target phase dispatched",
	})
}