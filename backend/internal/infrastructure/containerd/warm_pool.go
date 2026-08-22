package containerd

import (
	"sync"
	"time"
)

// WarmPool maintains pre-created containers for fast assignment.
type WarmPool struct {
	mu      sync.RWMutex
	pools   map[string][]string // baseImage -> []containerID
	maxSize int
}

// NewWarmPool creates a warm pool manager.
func NewWarmPool() *WarmPool {
	return &WarmPool{
		pools:   make(map[string][]string),
		maxSize: 5, // max 5 warm containers per base image
	}
}

// Register adds a newly created container to the warm pool.
func (wp *WarmPool) Register(containerID, baseImage string) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if len(wp.pools[baseImage]) >= wp.maxSize {
		return // pool full
	}
	wp.pools[baseImage] = append(wp.pools[baseImage], containerID)
}

// Take removes a container from the warm pool for assignment.
// Returns empty string if no warm container is available.
func (wp *WarmPool) Take(baseImage string) string {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	pool := wp.pools[baseImage]
	if len(pool) == 0 {
		return ""
	}

	containerID := pool[0]
	wp.pools[baseImage] = pool[1:]
	return containerID
}

// Return attempts to return a stopped container to the warm pool.
// Returns true if the container was accepted, false if it should be destroyed.
func (wp *WarmPool) Return(containerID string) bool {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	// In Phase 1, we don't actually return containers to the pool
	// because resetting a container's state is complex.
	// Phase 2 will implement snapshot reset.
	_ = containerID
	return false
}

// Reap removes stale containers from the warm pool.
func (wp *WarmPool) Reap(maxAge time.Duration) []string {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	// Phase 1: no-op. Phase 2 will track creation time and reap old entries.
	return nil
}
