package handler

import (
	"net/http"
	"strconv"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PredictionHandler handles prediction-related HTTP requests
type PredictionHandler struct {
	predictionService domain.PredictionService
}

func NewPredictionHandler(predictionService domain.PredictionService) *PredictionHandler {
	return &PredictionHandler{
		predictionService: predictionService,
	}
}

// PredictBuildTime handles POST /api/predictions/build-time
func (h *PredictionHandler) PredictBuildTime(c *gin.Context) {
	var req struct {
		WorkspaceID string         `json:"workspace_id" binding:"required"`
		Features    map[string]any `json:"features"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workspaceID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace_id"})
		return
	}

	prediction, err := h.predictionService.PredictBuildTime(c.Request.Context(), workspaceID, req.Features)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prediction)
}

// PredictImageSize handles POST /api/predictions/image-size
func (h *PredictionHandler) PredictImageSize(c *gin.Context) {
	var req struct {
		WorkspaceID string         `json:"workspace_id" binding:"required"`
		Features    map[string]any `json:"features"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workspaceID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace_id"})
		return
	}

	prediction, err := h.predictionService.PredictImageSize(c.Request.Context(), workspaceID, req.Features)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prediction)
}

// PredictCacheHit handles POST /api/predictions/cache-hit
func (h *PredictionHandler) PredictCacheHit(c *gin.Context) {
	var req struct {
		WorkspaceID string         `json:"workspace_id" binding:"required"`
		Features    map[string]any `json:"features"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workspaceID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace_id"})
		return
	}

	prediction, err := h.predictionService.PredictCacheHit(c.Request.Context(), workspaceID, req.Features)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prediction)
}

// PredictFailureRisk handles POST /api/predictions/failure-risk
func (h *PredictionHandler) PredictFailureRisk(c *gin.Context) {
	var req struct {
		WorkspaceID string         `json:"workspace_id" binding:"required"`
		Features    map[string]any `json:"features"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workspaceID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace_id"})
		return
	}

	prediction, err := h.predictionService.PredictFailureRisk(c.Request.Context(), workspaceID, req.Features)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prediction)
}

// GetPredictionHistory handles GET /api/predictions/history
func (h *PredictionHandler) GetPredictionHistory(c *gin.Context) {
	workspaceIDStr := c.Query("workspace_id")
	if workspaceIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id required"})
		return
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace_id"})
		return
	}

	pTypeStr := c.Query("type")
	if pTypeStr == "" {
		pTypeStr = "BUILD_TIME"
	}
	pType := domain.PredictionType(pTypeStr)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	predictions, err := h.predictionService.GetPredictionHistory(c.Request.Context(), workspaceID, pType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, predictions)
}

// GetModelMetrics handles GET /api/predictions/models/metrics
func (h *PredictionHandler) GetModelMetrics(c *gin.Context) {
	pTypeStr := c.Query("type")
	if pTypeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type required"})
		return
	}
	pType := domain.PredictionType(pTypeStr)

	metrics, err := h.predictionService.GetModelMetrics(c.Request.Context(), pType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// RetrainModel handles POST /api/predictions/models/retrain
func (h *PredictionHandler) RetrainModel(c *gin.Context) {
	var req struct {
		Type      string `json:"type" binding:"required"`
		Algorithm string `json:"algorithm"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pType := domain.PredictionType(req.Type)
	if req.Algorithm == "" {
		req.Algorithm = "linear"
	}

	model, err := h.predictionService.RetrainModel(c.Request.Context(), pType, req.Algorithm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, model)
}

// ExportModelONNX handles POST /api/predictions/models/export
func (h *PredictionHandler) ExportModelONNX(c *gin.Context) {
	var req struct {
		ModelID string `json:"model_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	modelID, err := uuid.Parse(req.ModelID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid model_id"})
		return
	}

	path, err := h.predictionService.ExportModelONNX(c.Request.Context(), modelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"onnx_path": path})
}

// ActivateModel handles POST /api/predictions/models/activate
func (h *PredictionHandler) ActivateModel(c *gin.Context) {
	var req struct {
		ModelID string `json:"model_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	modelID, err := uuid.Parse(req.ModelID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid model_id"})
		return
	}

	err = h.predictionService.ActivateModel(c.Request.Context(), modelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}