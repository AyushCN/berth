package handler

import (
	"net/http"

	"github.com/AyushCN/berth/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GitHandler struct {
	gitUC *usecase.GitUsecase
}

func NewGitHandler(uc *usecase.GitUsecase) *GitHandler {
	return &GitHandler{gitUC: uc}
}

func (h *GitHandler) Status(c *gin.Context) {
	sandboxID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sandbox id"})
		return
	}

	status, err := h.gitUC.GetStatus(c.Request.Context(), sandboxID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *GitHandler) ListBranches(c *gin.Context) {
	sandboxID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sandbox id"})
		return
	}

	branches, err := h.gitUC.ListBranches(c.Request.Context(), sandboxID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"branches": branches})
}

type CheckoutRequest struct {
	Branch string `json:"branch" binding:"required"`
}

func (h *GitHandler) Checkout(c *gin.Context) {
	sandboxID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sandbox id"})
		return
	}

	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "branch required"})
		return
	}

	if err := h.gitUC.Checkout(c.Request.Context(), sandboxID, req.Branch); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "checked out " + req.Branch})
}

func (h *GitHandler) CreateBranch(c *gin.Context) {
	sandboxID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sandbox id"})
		return
	}

	var req CheckoutRequest // reuse struct
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "branch name required"})
		return
	}

	if err := h.gitUC.CreateBranch(c.Request.Context(), sandboxID, req.Branch); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "created branch " + req.Branch})
}

func (h *GitHandler) Pull(c *gin.Context) {
	sandboxID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sandbox id"})
		return
	}

	if err := h.gitUC.Pull(c.Request.Context(), sandboxID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "pull successful"})
}

type CommitRequest struct {
	Message string `json:"message" binding:"required"`
}

func (h *GitHandler) Commit(c *gin.Context) {
	sandboxID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sandbox id"})
		return
	}

	var req CommitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message required"})
		return
	}

	if err := h.gitUC.Commit(c.Request.Context(), sandboxID, req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "commit successful"})
}

func (h *GitHandler) Push(c *gin.Context) {
	sandboxID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sandbox id"})
		return
	}

	if err := h.gitUC.Push(c.Request.Context(), sandboxID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "push successful"})
}

func (h *GitHandler) Log(c *gin.Context) {
	sandboxID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sandbox id"})
		return
	}

	commits, err := h.gitUC.Log(c.Request.Context(), sandboxID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"commits": commits})
}
