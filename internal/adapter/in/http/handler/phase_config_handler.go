package handler

import (
	"kanbanai/internal/adapter/in/http/response"
	"kanbanai/internal/application/dto"
	"kanbanai/internal/application/usecase"

	"github.com/gin-gonic/gin"
)

// PhaseConfigHandler exposes the per-lane harness/model configuration that
// operators edit from the UI.
type PhaseConfigHandler struct {
	getUC    *usecase.GetPhaseConfigs
	updateUC *usecase.UpdatePhaseConfigs
}

func NewPhaseConfigHandler(getUC *usecase.GetPhaseConfigs, updateUC *usecase.UpdatePhaseConfigs) *PhaseConfigHandler {
	return &PhaseConfigHandler{getUC: getUC, updateUC: updateUC}
}

// GET /api/v1/config/phases — effective per-lane config + env defaults.
func (h *PhaseConfigHandler) List(c *gin.Context) {
	configs, err := h.getUC.Execute(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"phases": configs})
}

type updatePhaseConfigsRequest struct {
	Phases []dto.PhaseConfigInput `json:"phases"`
}

// PUT /api/v1/config/phases — replace all per-lane overrides.
func (h *PhaseConfigHandler) Update(c *gin.Context) {
	var req updatePhaseConfigsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	configs, err := h.updateUC.Execute(c.Request.Context(), req.Phases)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"phases": configs})
}