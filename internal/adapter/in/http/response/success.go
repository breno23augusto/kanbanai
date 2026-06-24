package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    any         `json:"data"`
	Meta    MetaData    `json:"meta"`
}

type MetaData struct {
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
		Meta: MetaData{
			RequestID: c.GetString("request_id"),
			Timestamp: time.Now(),
		},
	})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    data,
		Meta: MetaData{
			RequestID: c.GetString("request_id"),
			Timestamp: time.Now(),
		},
	})
}

func NoContent(c *gin.Context) {
	c.JSON(http.StatusNoContent, gin.H{})
}
