package handler

import (
	"net/http"

	"github.com/AyushCN/berth/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ActivityHandler struct {
	activityUC *usecase.ActivityTracker
}

func NewActivityHandler(activityUC *usecase.ActivityTracker) *ActivityHandler {
	return &ActivityHandler{activityUC: activityUC}
}

type recordActivityRequest struct {
	EnvironmentID string `json:"environment_id" binding:"required"`
}

type resumeEnvironmentRequest struct {
	EnvironmentID string `json:"environment_id" binding:"required"`
}

// RecordActivity records user activity for an environment
func (h *ActivityHandler) RecordActivity(c *gin.Context) {
	var req recordActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	environmentID, err := uuid.Parse(req.EnvironmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid environment id"})
		return
	}

	if err := h.activityUC.RecordActivity(c.Request.Context(), environmentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "activity recorded"})
}

// RecordSessionStart records a new session for an environment
func (h *ActivityHandler) RecordSessionStart(c *gin.Context) {
	var req recordActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	environmentID, err := uuid.Parse(req.EnvironmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid environment id"})
		return
	}

	if err := h.activityUC.RecordSessionStart(c.Request.Context(), environmentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session started"})
}

// RecordSessionEnd records the end of a session
func (h *ActivityHandler) RecordSessionEnd(c *gin.Context) {
	var req recordActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	environmentID, err := uuid.Parse(req.EnvironmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid environment id"})
		return
	}

	if err := h.activityUC.RecordSessionEnd(c.Request.Context(), environmentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session ended"})
}

// ResumeEnvironment resumes a suspended environment
func (h *ActivityHandler) ResumeEnvironment(c *gin.Context) {
	var req resumeEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	environmentID, err := uuid.Parse(req.EnvironmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid environment id"})
		return
	}

	if err := h.activityUC.ResumeEnvironment(c.Request.Context(), environmentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "environment resuming"})
}

// GetIdleEnvironments returns environments that are candidates for suspension
func (h *ActivityHandler) GetIdleEnvironments(c *gin.Context) {
	// This would need a new repository method
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}
