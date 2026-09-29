package handler

import (
	"log/slog"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/AyushCN/berth/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// EnvironmentHandler handles environment HTTP requests.
type EnvironmentHandler struct {
	envUC        *usecase.EnvironmentUsecase
	traefikDomain string
}

func NewEnvironmentHandler(uc *usecase.EnvironmentUsecase, traefikDomain string) *EnvironmentHandler {
	return &EnvironmentHandler{envUC: uc, traefikDomain: traefikDomain}
}

// ListEnvironments returns all environments for the authenticated user.
func (h *EnvironmentHandler) ListEnvironments(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	envs, err := h.envUC.ListEnvironments(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"environments": envs})
}

// CreateEnvironment creates a new sandbox environment.
func (h *EnvironmentHandler) CreateEnvironment(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req usecase.EnvironmentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	env, err := h.envUC.CreateEnvironment(c.Request.Context(), uid, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, env)
}

// GetEnvironment returns a single environment.
func (h *EnvironmentHandler) GetEnvironment(c *gin.Context) {
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

	env, err := h.envUC.GetEnvironment(c.Request.Context(), uid, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, env)
}

// DeleteEnvironment deletes an environment.
func (h *EnvironmentHandler) DeleteEnvironment(c *gin.Context) {
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

	if err := h.envUC.DeleteEnvironment(c.Request.Context(), uid, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ForkEnvironment forks an environment.
func (h *EnvironmentHandler) ForkEnvironment(c *gin.Context) {
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

	var req usecase.EnvironmentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	env, err := h.envUC.ForkEnvironment(c.Request.Context(), uid, id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, env)
}

// StopEnvironment stops an environment.
func (h *EnvironmentHandler) StopEnvironment(c *gin.Context) {
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

	if err := h.envUC.StopEnvironment(c.Request.Context(), uid, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// RestartEnvironment restarts an environment.
func (h *EnvironmentHandler) RestartEnvironment(c *gin.Context) {
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

	if err := h.envUC.RestartEnvironment(c.Request.Context(), uid, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// StartEnvironment starts a stopped environment.
func (h *EnvironmentHandler) StartEnvironment(c *gin.Context) {
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

	if err := h.envUC.StartEnvironment(c.Request.Context(), uid, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ExecCommand executes a command in an environment.
func (h *EnvironmentHandler) ExecCommand(c *gin.Context) {
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
		Command []string `json:"command" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := h.envUC.ExecCommand(c.Request.Context(), uid, id, req.Command)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// GetLogs returns logs for an environment.
func (h *EnvironmentHandler) GetLogs(c *gin.Context) {
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

	lines := 100
	if l := c.Query("lines"); l != "" {
		fmt.Sscanf(l, "%d", &lines)
	}

	logs, err := h.envUC.GetLogs(c.Request.Context(), uid, id, lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// PreviewProxy acts as a reverse proxy for environment previews.
// Uses Traefik's routing (Host-based) instead of direct localhost connection.
func (h *EnvironmentHandler) PreviewProxy(c *gin.Context) {
	environmentID := c.Param("id")
	uid, err := uuid.Parse(environmentID)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	env, err := h.envUC.GetPreviewEnvironment(c.Request.Context(), uid)
	if err != nil {
		slog.Error("preview proxy: environment not found", "environment_id", environmentID, "error", err)
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if env.Port == 0 {
		slog.Error("preview proxy: environment not running or port not assigned", "environment_id", environmentID)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "environment is not running or port is not assigned"})
		return
	}

	// Use Traefik's Host-based routing instead of direct localhost
	// The container has Traefik labels: Host(`{environmentID}.{traefikDomain}`)
	targetHost := fmt.Sprintf("%s.%s", environmentID, h.traefikDomain)
	if h.traefikDomain == "" {
		targetHost = fmt.Sprintf("127.0.0.1:%d", env.Port)
	}

	slog.Info("preview proxy: forwarding request", "environment_id", environmentID, "target_host", targetHost, "port", env.Port)

	// Setup Reverse Proxy
	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = targetHost
		req.Host = targetHost // Important for Traefik Host-based routing
		// Strip the `/p/<id>` prefix
		pathPrefix := fmt.Sprintf("/p/%s", environmentID)
		if strings.HasPrefix(req.URL.Path, pathPrefix) {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, pathPrefix)
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
		}
	}

	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		slog.Error("preview proxy error", "environment_id", environmentID, "target_host", targetHost, "error", err)
		rw.WriteHeader(http.StatusBadGateway)
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}