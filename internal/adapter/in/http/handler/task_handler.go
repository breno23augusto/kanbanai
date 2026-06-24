package handler

import (
	"kanbanai/internal/application/dto"
	"kanbanai/internal/application/usecase"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/adapter/in/http/response"
	"kanbanai/internal/domain/query"

	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	createTaskUC   *usecase.CreateTask
	updateTaskUC   *usecase.UpdateTask
	deleteTaskUC   *usecase.DeleteTask
	getTaskUC      *usecase.GetTask
	listTasksUC    *usecase.ListTasks
	advancePhaseUC *usecase.AdvancePhase
	timelineQuery  query.TaskTimelineQuery
}

func NewTaskHandler(
	createTaskUC *usecase.CreateTask,
	updateTaskUC *usecase.UpdateTask,
	deleteTaskUC *usecase.DeleteTask,
	getTaskUC *usecase.GetTask,
	listTasksUC *usecase.ListTasks,
	advancePhaseUC *usecase.AdvancePhase,
	timelineQuery query.TaskTimelineQuery,
) *TaskHandler {
	return &TaskHandler{
		createTaskUC:   createTaskUC,
		updateTaskUC:   updateTaskUC,
		deleteTaskUC:   deleteTaskUC,
		getTaskUC:      getTaskUC,
		listTasksUC:    listTasksUC,
		advancePhaseUC: advancePhaseUC,
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
		if err.Error() == "concurrent modification: version mismatch" {
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

	// Build response with task + phase outputs
	taskOutput := dto.TaskOutput{
		ID:           result.Task.ID,
		Title:        result.Task.Title,
		Description:  result.Task.Description,
		CurrentPhase: result.Task.CurrentPhase,
		Status:       result.Task.Status,
		Priority:     result.Task.Priority,
		Version:      result.Task.Version,
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

func (h *TaskHandler) Retry(c *gin.Context) {
	id := c.Param("id")

	// For now, just acknowledge the retry request
	// The actual retry logic is handled by the orchestrator
	response.OK(c, gin.H{
		"task_id": id,
		"message": "retry initiated",
	})
}
