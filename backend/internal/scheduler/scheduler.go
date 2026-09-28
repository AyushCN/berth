package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// DefaultScheduler implements domain.Scheduler with multiple placement strategies.
type DefaultScheduler struct {
	workerRepo     domain.WorkerRepository
	tenantRepo     domain.TenantRepository
	sandboxRepo    domain.SandboxRepository
	config         domain.SchedulerConfig
	mu             sync.RWMutex
	workerCache    map[uuid.UUID]*domain.Worker
	lastCacheUpdate time.Time
}

func NewDefaultScheduler(
	workerRepo domain.WorkerRepository,
	tenantRepo domain.TenantRepository,
	sandboxRepo domain.SandboxRepository,
	config domain.SchedulerConfig,
) *DefaultScheduler {
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
	}
	if config.HeartbeatTimeout == 0 {
		config.HeartbeatTimeout = 60 * time.Second
	}
	if config.MaxSandboxesPerWorker == 0 {
		config.MaxSandboxesPerWorker = 50
	}

	s := &DefaultScheduler{
		workerRepo:  workerRepo,
		tenantRepo:  tenantRepo,
		sandboxRepo: sandboxRepo,
		config:      config,
		workerCache: make(map[uuid.UUID]*domain.Worker),
	}

	go s.cacheWorkersLoop()
	go s.healthCheckLoop()

	return s
}

// Schedule finds the best worker for a sandbox spec.
func (s *DefaultScheduler) Schedule(ctx context.Context, spec *domain.SandboxSpec, tenant *domain.Tenant) (*domain.Worker, error) {
	// Check tenant quota first
	if err := s.checkTenantQuota(ctx, tenant, spec); err != nil {
		return nil, fmt.Errorf("tenant quota exceeded: %w", err)
	}

	// Get healthy workers
	workers, err := s.getHealthyWorkers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get healthy workers: %w", err)
	}

	if len(workers) == 0 {
		return nil, fmt.Errorf("no healthy workers available")
	}

	// Filter workers that can accommodate the sandbox
	candidates := s.filterCandidates(workers, spec, tenant)
	if len(candidates) == 0 {
		// Try preemption if enabled
		if s.config.EnablePreemption {
			return s.scheduleWithPreemption(ctx, spec, tenant, workers)
		}
		return nil, fmt.Errorf("no workers can accommodate sandbox requirements")
	}

	// Select best worker based on strategy
	selected := s.selectWorker(candidates, spec)
	if selected == nil {
		return nil, fmt.Errorf("no suitable worker found")
	}

	// Reserve resources on selected worker
	if err := s.reserveResources(ctx, selected, spec); err != nil {
		return nil, fmt.Errorf("failed to reserve resources: %w", err)
	}

	slog.Info("scheduled sandbox", "sandbox_id", spec.ID, "worker_id", selected.ID, "strategy", s.config.Strategy)
	return selected, nil
}

// Reschedule moves a sandbox to a different worker.
func (s *DefaultScheduler) Reschedule(ctx context.Context, sandboxID uuid.UUID, targetWorkerID uuid.UUID) error {
	// Get sandbox details
	sandbox, err := s.sandboxRepo.GetByID(ctx, sandboxID)
	if err != nil {
		return fmt.Errorf("failed to get sandbox: %w", err)
	}

	if sandbox.ContainerID == nil {
		return fmt.Errorf("sandbox has no container to reschedule")
	}

	// Get target worker
	targetWorker, err := s.workerRepo.GetByID(ctx, targetWorkerID)
	if err != nil {
		return fmt.Errorf("failed to get target worker: %w", err)
	}

	if targetWorker.Status != domain.WorkerStatusHealthy {
		return fmt.Errorf("target worker is not healthy: %s", targetWorker.Status)
	}

	// Check if target worker has capacity
	capacity := &domain.WorkerCapacity{WorkerID: targetWorkerID}
	if err := s.workerRepo.UpdateHeartbeat(ctx, targetWorkerID, capacity); err != nil {
		return fmt.Errorf("failed to check target worker capacity: %w", err)
	}

	// TODO: Implement actual container migration (CRIU or recreate)
	// For now, this is a placeholder for the migration logic
	slog.Info("reschedule requested", "sandbox_id", sandboxID, "target_worker", targetWorkerID)

	return fmt.Errorf("container migration not yet implemented")
}

// Evict evicts sandboxes to make room for higher priority work.
func (s *DefaultScheduler) Evict(ctx context.Context, workerID uuid.UUID, neededMemory, neededCPU int64) ([]uuid.UUID, error) {
	// Get sandboxes running on this worker
	sandboxes, err := s.sandboxRepo.ListByWorker(ctx, workerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sandboxes on worker: %w", err)
	}

	// Sort by priority (lower priority first)
	// For now, sort by creation time (oldest first)
	evicted := []uuid.UUID{}
	freedMemory := int64(0)
	freedCPU := int64(0)

	for _, sandbox := range sandboxes {
		if freedMemory >= neededMemory && freedCPU >= neededCPU {
			break
		}

		// Skip sandboxes with active interactive sessions
		// TODO: Check for active PTY sessions

		evicted = append(evicted, sandbox.ID)
		// Use execution profile limits if available, otherwise use defaults
		memLimit := int64(512 * 1024 * 1024)
		cpuLimit := int64(1000)
		if sandbox.ExecutionProfile != nil {
			memLimit = sandbox.ExecutionProfile.MemoryLimit
			cpuLimit = sandbox.ExecutionProfile.CPUQuota
		}
		freedMemory += memLimit
		freedCPU += cpuLimit
	}

	if freedMemory < neededMemory || freedCPU < neededCPU {
		return evicted, fmt.Errorf("insufficient resources freed after eviction")
	}

	slog.Warn("evicted sandboxes for capacity", "worker_id", workerID, "evicted", evicted)
	return evicted, nil
}

// checkTenantQuota verifies the tenant has capacity for the new sandbox.
func (s *DefaultScheduler) checkTenantQuota(ctx context.Context, tenant *domain.Tenant, spec *domain.SandboxSpec) error {
	quota, err := s.tenantRepo.GetQuota(ctx, tenant.ID)
	if err != nil {
		return fmt.Errorf("failed to get tenant quota: %w", err)
	}

	settings := tenant.Settings

	if quota.ActiveSandboxes >= settings.MaxSandboxes {
		return fmt.Errorf("max sandboxes reached: %d/%d", quota.ActiveSandboxes, settings.MaxSandboxes)
	}

	if spec.MemoryLimit > settings.MaxMemoryPerSandbox {
		return fmt.Errorf("memory limit exceeds tenant max: %d > %d", spec.MemoryLimit, settings.MaxMemoryPerSandbox)
	}

	if spec.CPULimit > settings.MaxCPUPerSandbox {
		return fmt.Errorf("CPU limit exceeds tenant max: %d > %d", spec.CPULimit, settings.MaxCPUPerSandbox)
	}

	if spec.DiskLimit > settings.MaxDiskPerSandbox {
		return fmt.Errorf("disk limit exceeds tenant max: %d > %d", spec.DiskLimit, settings.MaxDiskPerSandbox)
	}

	// Check total resource usage
	if quota.TotalMemoryUsed+spec.MemoryLimit > int64(settings.MaxSandboxes)*settings.MaxMemoryPerSandbox {
		return fmt.Errorf("tenant total memory quota would be exceeded")
	}

	if quota.TotalCPUUsed+spec.CPULimit > int64(settings.MaxSandboxes)*settings.MaxCPUPerSandbox {
		return fmt.Errorf("tenant total CPU quota would be exceeded")
	}

	return nil
}

// getHealthyWorkers returns all workers that are healthy and accepting work.
func (s *DefaultScheduler) getHealthyWorkers(ctx context.Context) ([]*domain.Worker, error) {
	// Check cache first
	s.mu.RLock()
	if time.Since(s.lastCacheUpdate) < 10*time.Second && len(s.workerCache) > 0 {
		workers := make([]*domain.Worker, 0, len(s.workerCache))
		for _, w := range s.workerCache {
			if w.Status == domain.WorkerStatusHealthy || w.Status == domain.WorkerStatusDegraded {
				workers = append(workers, w)
			}
		}
		s.mu.RUnlock()
		if len(workers) > 0 {
			return workers, nil
		}
	}
	s.mu.RUnlock()

	// Refresh from database
	workers, err := s.workerRepo.ListHealthy(ctx)
	if err != nil {
		return nil, err
	}

	// Update cache
	s.mu.Lock()
	s.workerCache = make(map[uuid.UUID]*domain.Worker)
	for _, w := range workers {
		s.workerCache[w.ID] = w
	}
	s.lastCacheUpdate = time.Now()
	s.mu.Unlock()

	return workers, nil
}

// filterCandidates filters workers that can accommodate the sandbox.
func (s *DefaultScheduler) filterCandidates(workers []*domain.Worker, spec *domain.SandboxSpec, tenant *domain.Tenant) []*domain.Worker {
	candidates := []*domain.Worker{}

	for _, w := range workers {
		// Check runtime compatibility
		if !s.isRuntimeAllowed(w, spec, tenant) {
			continue
		}

		// Check resource availability
		if !s.hasCapacity(w, spec) {
			continue
		}

		// Check tenant restrictions
		if !s.matchesTenantRestrictions(w, tenant) {
			continue
		}

		candidates = append(candidates, w)
	}

	return candidates
}

// isRuntimeAllowed checks if the worker supports the required runtime.
func (s *DefaultScheduler) isRuntimeAllowed(w *domain.Worker, spec *domain.SandboxSpec, tenant *domain.Tenant) bool {
	// Check tenant allowed runtimes
	allowedRuntimes := tenant.Settings.AllowedRuntimes
	if len(allowedRuntimes) > 0 {
		found := false
		for _, rt := range allowedRuntimes {
			if rt == spec.ExecutionProfile.Runtime {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check worker labels for runtime support
	if runtimeLabel, ok := w.Labels["runtime"]; ok {
		if runtimeLabel != string(spec.ExecutionProfile.Runtime) {
			return false
		}
	}

	return true
}

// hasCapacity checks if the worker has enough resources.
func (s *DefaultScheduler) hasCapacity(w *domain.Worker, spec *domain.SandboxSpec) bool {
	// Use reserved resources
	availableMem := w.MaxMemory - s.config.ReservedMemory
	availableCPU := w.MaxCPU - s.config.ReservedCPU

	// Get current usage (would be better from real-time metrics)
	// For now, estimate based on active sandboxes
	usedMem := int64(0)
	usedCPU := int64(0)
	// This would be populated from WorkerCapacity in practice

	return (availableMem-usedMem) >= spec.MemoryLimit && (availableCPU-usedCPU) >= spec.CPULimit
}

// matchesTenantRestrictions checks worker against tenant-specific restrictions.
func (s *DefaultScheduler) matchesTenantRestrictions(w *domain.Worker, tenant *domain.Tenant) bool {
	// Check network mode restrictions
	allowedNetworkModes := tenant.Settings.AllowedNetworkModes
	if len(allowedNetworkModes) > 0 {
		// This would be checked against the worker's network capabilities
		// For now, assume all workers support all network modes
	}

	// Check filesystem mode restrictions
	allowedFSModes := tenant.Settings.AllowedFilesystemModes
	if len(allowedFSModes) > 0 {
		// Similar check for filesystem modes
	}

	// Check custom labels
	if tenantLabel, ok := w.Labels["tenant"]; ok {
		if tenantLabel != tenant.Slug {
			// Worker is dedicated to a different tenant
			return false
		}
	}

	return true
}

// selectWorker selects the best worker from candidates based on strategy.
func (s *DefaultScheduler) selectWorker(candidates []*domain.Worker, spec *domain.SandboxSpec) *domain.Worker {
	switch s.config.Strategy {
	case domain.PlacementStrategyBinPack:
		return s.selectBinPack(candidates, spec)
	case domain.PlacementStrategySpread:
		return s.selectSpread(candidates, spec)
	case domain.PlacementStrategyLeastUsed:
		return s.selectLeastUsed(candidates)
	default:
		return s.selectLeastUsed(candidates)
	}
}

// selectBinPack chooses the worker with the tightest fit (least remaining capacity after placement).
func (s *DefaultScheduler) selectBinPack(candidates []*domain.Worker, spec *domain.SandboxSpec) *domain.Worker {
	var best *domain.Worker
	bestScore := math.MaxFloat64

	for _, w := range candidates {
		// Score = remaining capacity after placement (lower is better for bin packing)
		memAfter := float64(w.MaxMemory - spec.MemoryLimit) / float64(w.MaxMemory)
		cpuAfter := float64(w.MaxCPU - spec.CPULimit) / float64(w.MaxCPU)
		score := memAfter + cpuAfter

		if score < bestScore {
			bestScore = score
			best = w
		}
	}

	return best
}

// selectSpread chooses the worker with the most remaining capacity (spread load).
func (s *DefaultScheduler) selectSpread(candidates []*domain.Worker, spec *domain.SandboxSpec) *domain.Worker {
	var best *domain.Worker
	bestScore := -1.0

	for _, w := range candidates {
		// Score = remaining capacity (higher is better for spreading)
		memAfter := float64(w.MaxMemory - spec.MemoryLimit) / float64(w.MaxMemory)
		cpuAfter := float64(w.MaxCPU - spec.CPULimit) / float64(w.MaxCPU)
		score := memAfter + cpuAfter

		if score > bestScore {
			bestScore = score
			best = w
		}
	}

	return best
}

// selectLeastUsed chooses the worker with the fewest active sandboxes.
func (s *DefaultScheduler) selectLeastUsed(candidates []*domain.Worker) *domain.Worker {
	var best *domain.Worker
	bestCount := math.MaxInt32

	for _, w := range candidates {
		// Get active sandbox count for this worker
		// In practice, this would come from WorkerCapacity
		count := 0 // Placeholder - would query actual count
		if count < bestCount {
			bestCount = count
			best = w
		}
	}

	return best
}

// scheduleWithPreemption attempts to evict lower priority work to make room.
func (s *DefaultScheduler) scheduleWithPreemption(ctx context.Context, spec *domain.SandboxSpec, tenant *domain.Tenant, workers []*domain.Worker) (*domain.Worker, error) {
	for _, w := range workers {
		if !s.isRuntimeAllowed(w, spec, tenant) {
			continue
		}

		// Try to evict enough resources
		neededMem := spec.MemoryLimit
		neededCPU := spec.CPULimit

		evicted, err := s.Evict(ctx, w.ID, neededMem, neededCPU)
		if err != nil {
			continue // Try next worker
		}

		slog.Info("preempted sandboxes for scheduling", "worker_id", w.ID, "evicted_count", len(evicted))

		// Reserve resources after eviction
		if err := s.reserveResources(ctx, w, spec); err != nil {
			continue
		}

		return w, nil
	}

	return nil, fmt.Errorf("no workers available even with preemption")
}

// reserveResources reserves resources on the selected worker.
func (s *DefaultScheduler) reserveResources(ctx context.Context, w *domain.Worker, spec *domain.SandboxSpec) error {
	// In a real implementation, this would use atomic operations or distributed locks
	// For now, we just update the worker's capacity tracking
	capacity := &domain.WorkerCapacity{
		WorkerID:        w.ID,
		AvailableMemory: w.MaxMemory - spec.MemoryLimit,
		AvailableCPU:    w.MaxCPU - spec.CPULimit,
	}

	return s.workerRepo.UpdateHeartbeat(ctx, w.ID, capacity)
}

// cacheWorkersLoop periodically refreshes the worker cache.
func (s *DefaultScheduler) cacheWorkersLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		workers, err := s.workerRepo.ListHealthy(ctx)
		if err != nil {
			slog.Error("failed to refresh worker cache", "error", err)
			continue
		}

		s.mu.Lock()
		s.workerCache = make(map[uuid.UUID]*domain.Worker)
		for _, w := range workers {
			s.workerCache[w.ID] = w
		}
		s.lastCacheUpdate = time.Now()
		s.mu.Unlock()
	}
}

// healthCheckLoop periodically checks worker health.
func (s *DefaultScheduler) healthCheckLoop() {
	ticker := time.NewTicker(s.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		workers, err := s.workerRepo.List(ctx)
		if err != nil {
			slog.Error("health check: failed to list workers", "error", err)
			continue
		}

		now := time.Now()
		for _, w := range workers {
			// Check heartbeat timeout
			if now.Sub(w.LastHeartbeat) > s.config.HeartbeatTimeout {
				if w.Status != domain.WorkerStatusOffline {
					slog.Warn("worker heartbeat timeout, marking offline", "worker_id", w.ID)
					s.workerRepo.UpdateStatus(ctx, w.ID, domain.WorkerStatusOffline)
				}
				continue
			}

			// Check resource exhaustion
			if w.Status == domain.WorkerStatusHealthy {
				// Check if resources are critically low
				// This would use WorkerCapacity in practice
			}
		}
	}
}