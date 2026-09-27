package handler

import (
	"net/http"
	"time"

	"github.com/AyushCN/berth/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ShareLinkHandler struct {
	shareLinkUC *usecase.ShareLinkUsecase
}

func NewShareLinkHandler(uc *usecase.ShareLinkUsecase) *ShareLinkHandler {
	return &ShareLinkHandler{shareLinkUC: uc}
}

type createShareLinkRequest struct {
	Role      string  `json:"role" binding:"required,oneof=VIEWER EDITOR"`
	ExpiresAt *string `json:"expires_at,omitempty"`
	MaxUses   *int    `json:"max_uses,omitempty"`
}

type joinShareLinkRequest struct {
	Code string `json:"code" binding:"required"`
}

// CreateShareLink creates a new share link for a project
func (h *ShareLinkHandler) CreateShareLink(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	var req createShareLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expires_at format, use RFC3339"})
			return
		}
		expiresAt = &t
	}

	link, err := h.shareLinkUC.CreateShareLink(c.Request.Context(), usecase.CreateShareLinkRequest{
		ProjectID: projectID,
		Role:      req.Role,
		CreatedBy: uid,
		ExpiresAt: expiresAt,
		MaxUses:   req.MaxUses,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, link)
}

// GetShareLinks returns all share links for a project
func (h *ShareLinkHandler) GetShareLinks(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	links, err := h.shareLinkUC.GetShareLinks(c.Request.Context(), projectID, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"links": links})
}

// RevokeShareLink revokes a share link
func (h *ShareLinkHandler) RevokeShareLink(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	linkIDStr := c.Param("linkId")
	linkID, err := uuid.Parse(linkIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid link id"})
		return
	}

	if err := h.shareLinkUC.RevokeShareLink(c.Request.Context(), linkID, uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// JoinViaShareLink handles joining a project via share link
func (h *ShareLinkHandler) JoinViaShareLink(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req joinShareLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workspace, err := h.shareLinkUC.JoinViaShareLink(c.Request.Context(), usecase.JoinViaShareLinkRequest{
		Code:   req.Code,
		UserID: uid,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"workspace": workspace})
}