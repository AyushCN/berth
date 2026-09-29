package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/AyushCN/berth/internal/infrastructure/redis"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	ID        string
	UserID    string
	SandboxID string
	Conn      *websocket.Conn
	Send      chan []byte
	CloseOnce sync.Once
}

const (
	// pongWait is how long a connection may go without a pong before the read
	// pump gives up on it.
	pongWait = 60 * time.Second
	// pingPeriod must be comfortably shorter than pongWait.
	pingPeriod = (pongWait * 9) / 10
	writeWait  = 10 * time.Second
)

type WSHub struct {
	pubsub       *redis.PubSub
	connections  map[string]*WSConnection   // connection ID -> connection
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
	// Redis is the cross-process fan-out channel. If it is unavailable a
	// connection must still be registrable, so report rather than dereference.
	if h.pubsub == nil {
		return errors.New("websocket broadcast unavailable: redis pubsub not initialised")
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
	h.connections[conn.ID] = conn
	// The inner maps are created lazily. NewWSHub only allocated the outer
	// maps, so writing through h.sandboxConns[conn.SandboxID] was an
	// assignment into a nil map and panicked on the first connection.
	if h.sandboxConns[conn.SandboxID] == nil {
		h.sandboxConns[conn.SandboxID] = make(map[string]bool)
	}
	if h.userConns[conn.UserID] == nil {
		h.userConns[conn.UserID] = make(map[string]bool)
	}
	h.sandboxConns[conn.SandboxID][conn.ID] = true
	h.userConns[conn.UserID][conn.ID] = true
	sandboxID, userID := conn.SandboxID, conn.UserID
	h.mu.Unlock()

	// Publish presence outside the lock: this path goes to redis, not the hub.
	h.publishPresence(sandboxID, userID, "join")
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
			// Liveness is enforced by read deadlines and the pong handler set
			// in HandleWS, not by reading here. Calling ReadMessage from this
			// goroutine while the per-connection read pump is also reading is
			// a concurrent read on the same gorilla websocket, which panics.
			// This sweep only prunes connections whose send channel the write
			// pump has already closed.
			var dead []string
			h.mu.RLock()
			for connID, conn := range h.connections {
				select {
				case _, ok := <-conn.Send:
					if !ok {
						dead = append(dead, connID)
					}
				default:
				}
			}
			h.mu.RUnlock()

			for _, connID := range dead {
				slog.Info("pruning closed websocket connection", "conn_id", connID)
				h.Unregister(connID)
			}
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
	// The routes register the parameter as :id (router.go:
	// /ws/environments/:id, /ws/sandbox/:id, /ws/sandboxes/:id). Reading
	// "sandbox_id" always returned an empty string, so every terminal
	// connection was rejected with 400.
	sandboxID := c.Param("id")
	if sandboxID == "" {
		sandboxID = c.Param("sandbox_id")
	}
	if sandboxID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "environment id required"})
		return
	}
	if _, err := uuid.Parse(sandboxID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "environment id must be a uuid"})
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

	// Liveness is enforced here and in the read pump below, which is the only
	// reader on this connection. It fails once the deadline passes without a
	// pong, which is how dead peers get detected.
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

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
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case data, ok := <-wsConn.Send:
				if !ok {
					conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
					slog.Error("websocket write failed", "error", err)
					return
				}
			case <-ticker.C:
				// Without pings a client that never sends anything never
				// sends a pong, and the read deadline would expire on a
				// perfectly healthy connection.
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
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
