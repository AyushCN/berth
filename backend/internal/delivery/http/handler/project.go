package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/AyushCN/berth/internal/usecase"
)

type ProjectHandler struct {
	projUC *usecase.ProjectUsecase
}

func NewProjectHandler(uc *usecase.ProjectUsecase) *ProjectHandler {
	return &ProjectHandler{projUC: uc}
}

func (h *ProjectHandler) Create(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req struct {
		Name                string    `json:"name" binding:"required"`
		Description         *string   `json:"description"`
		OwnerOrganizationID uuid.UUID `json:"owner_organization_id" binding:"required"`
		IsPublic            bool      `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	proj, err := h.projUC.Create(c.Request.Context(), uid, req.OwnerOrganizationID, req.Name, req.Description, req.IsPublic)
	if err != nil {
		if err == usecase.ErrProjectUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, proj)
}

func (h *ProjectHandler) ListForOrg(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org id"})
		return
	}

	projs, err := h.projUC.ListForOrg(c.Request.Context(), uid, orgID)
	if err != nil {
		if err == usecase.ErrProjectUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"projects": projs})
}

func (h *ProjectHandler) ListForUser(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	projs, err := h.projUC.ListForUser(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"projects": projs})
}

func (h *ProjectHandler) GetSandboxes(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	sandboxes, err := h.projUC.ListSandboxes(c.Request.Context(), uid, projectID)
	if err != nil {
		if err == usecase.ErrProjectUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sandboxes": sandboxes})
}
