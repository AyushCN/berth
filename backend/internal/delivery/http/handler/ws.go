package handler

import (
	"net/http"

	"github.com/AyushCN/berth/internal/infrastructure/redis"
	"github.com/gin-gonic/gin"
)

// WSHandler handles WebSocket connections for real-time sync using distributed hub.
type WSHandler struct {
	hub            *WSHub
	allowedOrigins []string
}

// NewWSHandler creates a WebSocket handler using distributed hub.
// allowedOrigins defaults to local dev addresses if none are provided.
// In production, pass cfg.FrontendURL: handler.NewWSHandler(redisPubSub, cfg.FrontendURL)
func NewWSHandler(pubsub *redis.PubSub, allowedOrigins ...string) *WSHandler {
	origins := allowedOrigins
	if len(origins) == 0 {
		origins = []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		}
	}

	hub := NewWSHub(pubsub, allowedOrigins...)

	return &WSHandler{hub: hub, allowedOrigins: allowedOrigins}
}

// HandleSandboxWS upgrades to WebSocket and bridges to distributed hub.
func (h *WSHandler) HandleSandboxWS(c *gin.Context) {
	if h.hub == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "real-time sync unavailable"})
		return
	}

	// Use the distributed hub's handler
	h.hub.HandleWS(c)
}

// Shutdown gracefully shuts down the WebSocket hub
func (h *WSHandler) Shutdown() {
	if h.hub != nil {
		h.hub.Shutdown()
	}
}
