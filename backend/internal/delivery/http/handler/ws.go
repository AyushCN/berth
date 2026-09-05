package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	infranats "github.com/AyushCN/berth/internal/infrastructure/nats"
	"github.com/nats-io/nats.go"
	"log/slog"
)

// WSHandler handles WebSocket connections for real-time sync.
type WSHandler struct {
	natsClient     *infranats.Client
	allowedOrigins []string
}

// NewWSHandler creates a WebSocket handler with strict origin validation.
// allowedOrigins defaults to local dev addresses if none are provided.
// In production, pass cfg.FrontendURL: handler.NewWSHandler(natsClient, cfg.FrontendURL)
func NewWSHandler(nc *infranats.Client, allowedOrigins ...string) *WSHandler {
	origins := allowedOrigins
	if len(origins) == 0 {
		origins = []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		}
	}
	return &WSHandler{natsClient: nc, allowedOrigins: origins}
}

func (h *WSHandler) upgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			for _, allowed := range h.allowedOrigins {
				if origin == allowed {
					return true
				}
			}
			slog.Warn("websocket origin rejected", "origin", origin)
			return false
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
}

// HandleSandboxWS upgrades to WebSocket and bridges to NATS.
func (h *WSHandler) HandleSandboxWS(c *gin.Context) {
	if h.natsClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "real-time sync unavailable"})
		return
	}
	sandboxID := c.Param("id")
	if sandboxID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sandbox id required"})
		return
	}

	conn, err := h.upgrader().Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	subject := "sandbox." + sandboxID + ".events"
	sub, err := h.natsClient.Subscribe(subject, "", func(msg *nats.Msg) {
		if err := conn.WriteMessage(websocket.TextMessage, msg.Data); err != nil {
			slog.Error("websocket write failed", "error", err)
		}
		_ = msg.Ack()
	})
	if err != nil {
		slog.Error("nats subscribe failed", "error", err)
		return
	}
	defer sub.Unsubscribe()

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("websocket read error", "error", err)
			}
			break
		}
		if msgType == websocket.TextMessage || msgType == websocket.BinaryMessage {
			if err := h.natsClient.Publish(subject, data); err != nil {
				slog.Error("nats publish failed", "error", err)
			}
		}
	}
}

// HandleFileSyncWS upgrades to WebSocket and bridges to NATS for file sync.
func (h *WSHandler) HandleFileSyncWS(c *gin.Context) {
	if h.natsClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "real-time sync unavailable"})
		return
	}
	sandboxID := c.Param("id")
	filePath := c.Query("path")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file path required"})
		return
	}

	conn, err := h.upgrader().Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	subject := "file." + sandboxID + "." + filePath
	sub, err := h.natsClient.Subscribe(subject, "", func(msg *nats.Msg) {
		_ = conn.WriteMessage(websocket.BinaryMessage, msg.Data)
		_ = msg.Ack()
	})
	if err != nil {
		return
	}
	defer sub.Unsubscribe()

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if msgType == websocket.BinaryMessage {
			_ = h.natsClient.Publish(subject, data)
		}
	}
}
