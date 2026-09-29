package handler

import (
	"testing"

	"github.com/google/uuid"
)

// newBareHub builds a hub without starting the redis subscriber and cleanup
// goroutines, so the connection bookkeeping can be tested in isolation.
func newBareHub() *WSHub {
	return &WSHub{
		connections:  make(map[string]*WSConnection),
		sandboxConns: make(map[string]map[string]bool),
		userConns:    make(map[string]map[string]bool),
	}
}

func TestRegisterAllocatesInnerMaps(t *testing.T) {
	// Regression: NewWSHub only allocated the outer maps, so
	// h.sandboxConns[conn.SandboxID][conn.ID] = true was an assignment into a
	// nil map and panicked on the very first websocket connection.
	h := newBareHub()
	envID := uuid.New().String()
	userID := uuid.New().String()

	h.Register(&WSConnection{
		ID:        "conn-1",
		SandboxID: envID,
		UserID:    userID,
		Send:      make(chan []byte, 1),
	})

	if got := h.sandboxConns[envID]["conn-1"]; !got {
		t.Error("connection was not indexed by sandbox id")
	}
	if got := h.userConns[userID]["conn-1"]; !got {
		t.Error("connection was not indexed by user id")
	}
	if h.connections["conn-1"] == nil {
		t.Error("connection was not stored")
	}
}

func TestRegisterMultipleConnectionsSameSandbox(t *testing.T) {
	h := newBareHub()
	envID := uuid.New().String()

	for _, id := range []string{"a", "b", "c"} {
		h.Register(&WSConnection{ID: id, SandboxID: envID, UserID: uuid.New().String(), Send: make(chan []byte, 1)})
	}

	if got := len(h.sandboxConns[envID]); got != 3 {
		t.Errorf("expected 3 connections for sandbox, got %d", got)
	}
}

func TestUnregisterRemovesFromIndexes(t *testing.T) {
	h := newBareHub()
	envID := uuid.New().String()
	userID := uuid.New().String()

	h.Register(&WSConnection{ID: "conn-1", SandboxID: envID, UserID: userID, Send: make(chan []byte, 1)})
	h.Unregister("conn-1")

	if h.sandboxConns[envID]["conn-1"] {
		t.Error("connection still indexed by sandbox after unregister")
	}
	if h.userConns[userID]["conn-1"] {
		t.Error("connection still indexed by user after unregister")
	}
	if h.connections["conn-1"] != nil {
		t.Error("connection still present after unregister")
	}
}

func TestUnregisterUnknownConnectionIsSafe(t *testing.T) {
	h := newBareHub()
	h.Unregister("does-not-exist")
}
