package handler

import (
	"net/http"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/AyushCN/berth/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChangeRequestHandler struct {
	changeRequestUC *usecase.ChangeRequestUsecase
}

func NewChangeRequestHandler(uc *usecase.ChangeRequestUsecase) *ChangeRequestHandler {
	return &ChangeRequestHandler{changeRequestUC: uc}
}

type createChangeRequestRequest struct {
	SourceWorkspaceID uuid.UUID `json:"source_workspace_id" binding:"required"`
	TargetWorkspaceID uuid.UUID `json:"target_workspace_id" binding:"required"`
	Title             string    `json:"title" binding:"required"`
	Description       string    `json:"description"`
}

type updateChangeRequestRequest struct {
	Title       *string                    `json:"title"`
	Description *string                    `json:"description"`
	State       *domain.ChangeRequestState `json:"state"`
}

// CreateChangeRequest creates a new change request
func (h *ChangeRequestHandler) CreateChangeRequest(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req createChangeRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cr, err := h.changeRequestUC.CreateChangeRequest(c.Request.Context(), usecase.CreateChangeRequestRequest{
		SourceWorkspaceID: req.SourceWorkspaceID,
		TargetWorkspaceID: req.TargetWorkspaceID,
		Title:             req.Title,
		Description:       req.Description,
		AuthorID:          uid,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, cr)
}

// GetChangeRequest returns a single change request
func (h *ChangeRequestHandler) GetChangeRequest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	cr, err := h.changeRequestUC.GetChangeRequest(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "change request not found"})
		return
	}

	c.JSON(http.StatusOK, cr)
}

// ListChangeRequests returns change requests for a project
func (h *ChangeRequestHandler) ListChangeRequests(c *gin.Context) {
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	limit := 20
	offset := 0

	crList, err := h.changeRequestUC.ListChangeRequests(c.Request.Context(), projectID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"change_requests": crList})
}

// ListChangeRequestsBySource returns change requests for a source workspace
func (h *ChangeRequestHandler) ListChangeRequestsBySource(c *gin.Context) {
	workspaceIDStr := c.Param("workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	crList, err := h.changeRequestUC.ListChangeRequestsBySource(c.Request.Context(), workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"change_requests": crList})
}

// UpdateChangeRequest updates a change request
func (h *ChangeRequestHandler) UpdateChangeRequest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req updateChangeRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cr, err := h.changeRequestUC.UpdateChangeRequest(c.Request.Context(), usecase.UpdateChangeRequestRequest{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
		State:       req.State,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cr)
}

// MergeChangeRequest merges a change request
func (h *ChangeRequestHandler) MergeChangeRequest(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	cr, err := h.changeRequestUC.MergeChangeRequest(c.Request.Context(), id, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cr)
}

// CloseChangeRequest closes a change request without merging
func (h *ChangeRequestHandler) CloseChangeRequest(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	cr, err := h.changeRequestUC.CloseChangeRequest(c.Request.Context(), id, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cr)
}

// GetDiff returns the diff for a change request
func (h *ChangeRequestHandler) GetDiff(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// This would need a method to get the diff
	// For now, return a placeholder
	c.JSON(http.StatusOK, gin.H{"diff": "diff not implemented yet", "change_request_id": id})
}
