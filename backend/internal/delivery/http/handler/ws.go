package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	infranats "github.com/AyushCN/berth/internal/infrastructure/nats"
	"github.com/nats-io/nats.go"
	"log/slog"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: restrict in production
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WSHandler handles WebSocket connections for real-time sync.
type WSHandler struct {
	natsClient *infranats.Client
}

func NewWSHandler(nc *infranats.Client) *WSHandler {
	return &WSHandler{natsClient: nc}
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

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	// Subscribe to NATS subject for this sandbox
	subject := "sandbox." + sandboxID + ".events"
	sub, err := h.natsClient.Subscribe(subject, "ws-"+sandboxID, func(msg *nats.Msg) {
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

	// Read from WebSocket and publish to NATS
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

// HandleFileSyncWS handles collaborative file editing via Yjs.
func (h *WSHandler) HandleFileSyncWS(c *gin.Context) {
	sandboxID := c.Param("id")
	filePath := c.Query("path")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file path required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	// Phase 2: simple echo. Phase 3: Yjs awareness + CRDT sync.
	subject := "file." + sandboxID + "." + filePath
	sub, err := h.natsClient.Subscribe(subject, "file-"+sandboxID, func(msg *nats.Msg) {
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
