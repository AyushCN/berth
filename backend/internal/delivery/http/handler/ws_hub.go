package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/AyushCN/berth/internal/infrastructure/redis"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"log/slog"
)

type WSMessage struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
	UserID    string          `json:"user_id,omitempty"`
	SandboxID string          `json:"sandbox_id,omitempty"`
}

type WSConnection struct {
	ID         string
	UserID     string
	SandboxID  string
	Conn       *websocket.Conn
	Send       chan []byte
	CloseOnce  sync.Once
}

type WSHub struct {
	pubsub       *redis.PubSub
	connections  map[string]*WSConnection // connection ID -> connection
	sandboxConns map[string]map[string]bool // sandbox ID -> connection IDs
	userConns    map[string]map[string]bool // user ID -> connection IDs
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	upgrader     *websocket.Upgrader
}

func NewWSHub(pubsub *redis.PubSub, allowedOrigins ...string) *WSHub {
	origins := allowedOrigins
	if len(origins) == 0 {
		origins = []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	h := &WSHub{
		pubsub:       pubsub,
		connections:  make(map[string]*WSConnection),
		sandboxConns: make(map[string]map[string]bool),
		userConns:    make(map[string]map[string]bool),
		ctx:          ctx,
		cancel:       cancel,
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				for _, allowed := range origins {
					if origin == allowed {
						return true
					}
				}
				slog.Warn("websocket origin rejected", "origin", origin)
				return false
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	// Start Redis subscriber
	go h.runSubscriber()

	// Start cleanup routine
	go h.cleanupRoutine()

	return h
}

func (h *WSHub) runSubscriber() {
	pubsub := h.pubsub.Subscribe(h.ctx, "ws:broadcast:*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-h.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			h.handleBroadcast([]byte(msg.Payload))
		}
	}
}

func (h *WSHub) handleBroadcast(data []byte) {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		slog.Error("failed to unmarshal broadcast message", "error", err)
		return
	}

	h.mu.RLock()
	connIDs := h.sandboxConns[msg.SandboxID]
	h.mu.RUnlock()

	for connID := range connIDs {
		h.mu.RLock()
		conn := h.connections[connID]
		h.mu.RUnlock()

		if conn != nil {
			select {
			case conn.Send <- data:
			default:
				slog.Warn("connection send buffer full", "conn_id", connID)
			}
		}
	}
}

func (h *WSHub) BroadcastToSandbox(sandboxID string, msg WSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return h.pubsub.Publish(context.Background(), "ws:broadcast:"+sandboxID, data)
}

func (h *WSHub) BroadcastToUser(userID string, msg WSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	h.mu.RLock()
	connIDs := h.userConns[userID]
	h.mu.RUnlock()

	for connID := range connIDs {
		h.mu.RLock()
		conn := h.connections[connID]
		h.mu.RUnlock()

		if conn != nil {
			select {
			case conn.Send <- data:
			default:
				slog.Warn("connection send buffer full", "conn_id", connID)
			}
		}
	}
	return nil
}

func (h *WSHub) Register(conn *WSConnection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.connections[conn.ID] = conn
	h.sandboxConns[conn.SandboxID][conn.ID] = true
	h.userConns[conn.UserID][conn.ID] = true

	// Publish presence event
	h.publishPresence(conn.SandboxID, conn.UserID, "join")
}

func (h *WSHub) Unregister(connID string) {
	h.mu.Lock()
	conn, exists := h.connections[connID]
	if exists {
		delete(h.connections, connID)
		delete(h.sandboxConns[conn.SandboxID], connID)
		delete(h.userConns[conn.UserID], connID)
	}
	h.mu.Unlock()

	if exists {
		h.publishPresence(conn.SandboxID, conn.UserID, "leave")
	}
}

func (h *WSHub) publishPresence(sandboxID, userID, action string) {
	msg := WSMessage{
		Type:      "presence",
		Payload:   json.RawMessage(`{"user_id":"` + userID + `","action":"` + action + `"}`),
		Timestamp: time.Now(),
		UserID:    userID,
		SandboxID: sandboxID,
	}
	h.BroadcastToSandbox(sandboxID, msg)
}

func (h *WSHub) cleanupRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.mu.Lock()
			for connID, conn := range h.connections {
				// Check if connection is still alive
				conn.Conn.SetReadDeadline(time.Now().Add(1 * time.Second))
				if _, _, err := conn.Conn.ReadMessage(); err != nil {
					// Connection dead, remove it
					delete(h.connections, connID)
					delete(h.sandboxConns[conn.SandboxID], connID)
					delete(h.userConns[conn.UserID], connID)
					h.publishPresence(conn.SandboxID, conn.UserID, "leave")
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *WSHub) Shutdown() {
	h.cancel()
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, conn := range h.connections {
		conn.CloseOnce.Do(func() {
			close(conn.Send)
			conn.Conn.Close()
		})
	}
}

func (h *WSHub) Upgrader() *websocket.Upgrader {
	return h.upgrader
}

func (h *WSHub) HandleWS(c *gin.Context) {
	sandboxID := c.Param("sandbox_id")
	if sandboxID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sandbox_id required"})
		return
	}

	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}

	connID := sandboxID + "-" + userID + "-" + time.Now().Format("20060102150405.000000000")
	wsConn := &WSConnection{
		ID:        connID,
		UserID:    userID,
		SandboxID: sandboxID,
		Conn:      conn,
		Send:      make(chan []byte, 256),
	}

	h.Register(wsConn)

	// Write pump
	go func() {
		defer h.Unregister(connID)
		for {
			select {
			case data, ok := <-wsConn.Send:
				if !ok {
					conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
					slog.Error("websocket write failed", "error", err)
					return
				}
			case <-h.ctx.Done():
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
		}
	}()

	// Read pump
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("websocket read error", "error", err)
			}
			break
		}

		if msgType == websocket.TextMessage || msgType == websocket.BinaryMessage {
			var msg WSMessage
			if err := json.Unmarshal(data, &msg); err == nil {
				msg.UserID = userID
				msg.SandboxID = sandboxID
				msg.Timestamp = time.Now()
				h.BroadcastToSandbox(sandboxID, msg)
			}
		}
	}
}