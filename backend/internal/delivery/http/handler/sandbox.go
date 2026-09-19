package handler

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/AyushCN/berth/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SandboxHandler handles environment HTTP requests.
type SandboxHandler struct {
	sandboxUC *usecase.SandboxUsecase
}

func NewSandboxHandler(uc *usecase.SandboxUsecase) *SandboxHandler {
	return &SandboxHandler{sandboxUC: uc}
}

// ListEnvironments returns all environments for the authenticated user.
func (h *SandboxHandler) ListEnvironments(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	envs, err := h.sandboxUC.ListEnvironments(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sandboxes": envs})
}

// CreateEnvironment creates a new sandbox environment.
func (h *SandboxHandler) CreateEnvironment(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req usecase.CreateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	env, err := h.sandboxUC.CreateEnvironment(c.Request.Context(), uid, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, env)
}

// GetEnvironment returns a single environment.
func (h *SandboxHandler) GetEnvironment(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	env, err := h.sandboxUC.GetEnvironment(c.Request.Context(), uid, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "environment not found"})
		return
	}
	c.JSON(http.StatusOK, env)
}

// DeleteEnvironment destroys an environment.
func (h *SandboxHandler) DeleteEnvironment(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.sandboxUC.DeleteEnvironment(c.Request.Context(), uid, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ExecCommand runs a command in a sandbox.
func (h *SandboxHandler) ExecCommand(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Command []string `json:"command"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := h.sandboxUC.ExecCommand(c.Request.Context(), uid, id, req.Command)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// GetLogs returns logs for a sandbox.
func (h *SandboxHandler) GetLogs(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	logs, err := h.sandboxUC.GetLogs(c.Request.Context(), uid, id, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// ForkEnvironment forks an existing sandbox.
func (h *SandboxHandler) ForkEnvironment(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sandbox id"})
		return
	}

	var req usecase.CreateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	env, err := h.sandboxUC.ForkEnvironment(c.Request.Context(), uid, id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, env)
}

// PreviewProxy acts as a reverse proxy for sandbox previews.
func (h *SandboxHandler) PreviewProxy(c *gin.Context) {
	sandboxID := c.Param("id")
	uid, err := uuid.Parse(sandboxID)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	sandbox, err := h.sandboxUC.GetPreviewEnvironment(c.Request.Context(), uid)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if sandbox.Port == nil || *sandbox.Port == 0 {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "sandbox is not running or port is not assigned"})
		return
	}

	// Setup Reverse Proxy
	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = fmt.Sprintf("127.0.0.1:%d", *sandbox.Port)
		// Strip the `/p/<id>` prefix
		pathPrefix := fmt.Sprintf("/p/%s", sandboxID)
		if strings.HasPrefix(req.URL.Path, pathPrefix) {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, pathPrefix)
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
		}
	}

	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(c.Writer, c.Request)
}

// StopEnvironment stops an environment.
func (h *SandboxHandler) StopEnvironment(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.sandboxUC.StopEnvironment(c.Request.Context(), uid, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// RestartEnvironment restarts an environment.
func (h *SandboxHandler) RestartEnvironment(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.sandboxUC.RestartEnvironment(c.Request.Context(), uid, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
