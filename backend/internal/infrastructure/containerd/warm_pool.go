package containerd

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// WarmContainer holds metadata for a pre-created container.
type WarmContainer struct {
	ID          string
	BaseImage   string
	CreatedAt   time.Time
	MemoryBytes int64
}

// WarmPool maintains pre-created containers for fast assignment with resource awareness.
type WarmPool struct {
	mu              sync.RWMutex
	available       map[string][]*WarmContainer // BaseImage -> queue
	maxMemoryBytes  int64
	usedMemoryBytes int64
	destroyer       func(ctx context.Context, id string) error
	stopReaper      chan struct{}
}

// NewWarmPool creates a resource-aware warm pool manager.
func NewWarmPool(maxMem int64, destroyer func(context.Context, string) error) *WarmPool {
	wp := &WarmPool{
		available:      make(map[string][]*WarmContainer),
		maxMemoryBytes: maxMem,
		destroyer:      destroyer,
		stopReaper:     make(chan struct{}),
	}
	go wp.startReaper(15 * time.Minute)
	return wp
}

// Stop halts the background reaper.
func (wp *WarmPool) Stop() {
	close(wp.stopReaper)
}

// PreWarm attempts to pre-create a container if resources allow.
// creator is a callback that actually provisions the container via containerd.
func (wp *WarmPool) PreWarm(ctx context.Context, baseImage string, memoryBytes int64, creator func() (string, error)) error {
	wp.mu.Lock()
	if wp.usedMemoryBytes+memoryBytes > wp.maxMemoryBytes {
		// Eviction policy: simple for now - find oldest container across all pools
		if !wp.evictOne() {
			wp.mu.Unlock()
			return fmt.Errorf("warm pool at capacity (%d bytes) and eviction failed", wp.maxMemoryBytes)
		}
	}
	wp.usedMemoryBytes += memoryBytes
	wp.mu.Unlock()

	// Provision outside the lock
	id, err := creator()
	if err != nil {
		// Revert memory
		wp.mu.Lock()
		wp.usedMemoryBytes -= memoryBytes
		wp.mu.Unlock()
		return err
	}

	wp.mu.Lock()
	defer wp.mu.Unlock()
	wp.available[baseImage] = append(wp.available[baseImage], &WarmContainer{
		ID:          id,
		BaseImage:   baseImage,
		CreatedAt:   time.Now(),
		MemoryBytes: memoryBytes,
	})
	slog.Info("container pre-warmed", "base_image", baseImage, "id", id, "used_mem", wp.usedMemoryBytes)
	return nil
}

// evictOne removes and destroys the oldest container in the pool.
// Must be called with lock held.
func (wp *WarmPool) evictOne() bool {
	var oldest *WarmContainer
	var oldestImage string
	var oldestIdx int

	for img, queue := range wp.available {
		for i, c := range queue {
			if oldest == nil || c.CreatedAt.Before(oldest.CreatedAt) {
				oldest = c
				oldestImage = img
				oldestIdx = i
			}
		}
	}

	if oldest == nil {
		return false
	}

	// Remove from queue
	queue := wp.available[oldestImage]
	wp.available[oldestImage] = append(queue[:oldestIdx], queue[oldestIdx+1:]...)
	wp.usedMemoryBytes -= oldest.MemoryBytes

	// Destroy asynchronously so we don't block
	go func(id string) {
		slog.Info("evicting warm container for capacity", "id", id)
		_ = wp.destroyer(context.Background(), id)
	}(oldest.ID)

	return true
}

// Take removes a container from the warm pool for assignment.
// Returns empty string if no warm container is available.
func (wp *WarmPool) Take(baseImage string) string {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	queue := wp.available[baseImage]
	if len(queue) == 0 {
		return ""
	}

	c := queue[0]
	wp.available[baseImage] = queue[1:]
	wp.usedMemoryBytes -= c.MemoryBytes // Memory is now accounted to the active runtime
	return c.ID
}

// Return attempts to return a stopped container to the warm pool.
func (wp *WarmPool) Return(containerID string) bool {
	// In Phase 1, we don't return dirty containers.
	return false
}

// startReaper periodically kills containers older than maxAge.
func (wp *WarmPool) startReaper(maxAge time.Duration) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wp.stopReaper:
			return
		case <-ticker.C:
			wp.reap(maxAge)
		}
	}
}

func (wp *WarmPool) reap(maxAge time.Duration) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	now := time.Now()
	for img, queue := range wp.available {
		var active []*WarmContainer
		for _, c := range queue {
			if now.Sub(c.CreatedAt) > maxAge {
				wp.usedMemoryBytes -= c.MemoryBytes
				go func(id string) {
					slog.Info("reaping stale warm container", "id", id)
					_ = wp.destroyer(context.Background(), id)
				}(c.ID)
			} else {
				active = append(active, c)
			}
		}
		wp.available[img] = active
	}
}
