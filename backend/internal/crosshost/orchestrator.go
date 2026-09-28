package crosshost

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// MigrationOrchestrator coordinates cross-host migrations
type MigrationOrchestrator struct {
	vxlanMesh      *VXLANMesh
	criuMigrator   *CRIUMigrator
	scheduler      SchedulerClient
	workerRegistry WorkerRegistry
	mu             sync.RWMutex
	activeMigrations map[string]*MigrationTask
}

type SchedulerClient interface {
	Schedule(ctx context.Context, spec *domain.SandboxSpec, tenant *domain.Tenant) (*domain.Worker, error)
	GetWorker(ctx context.Context, workerID uuid.UUID) (*domain.Worker, error)
}

type WorkerRegistry interface {
	GetWorker(ctx context.Context, workerID uuid.UUID) (*domain.Worker, error)
	ListWorkers(ctx context.Context) ([]*domain.Worker, error)
	UpdateWorkerStatus(ctx context.Context, workerID uuid.UUID, status domain.WorkerStatus) error
}

type MigrationTask struct {
	ID              string
	ContainerID     string
	SourceWorkerID  uuid.UUID
	TargetWorkerID  uuid.UUID
	Status          MigrationStatus
	CheckpointID    string
	StartedAt       time.Time
	CompletedAt     *time.Time
	Error           string
	Progress        float64
}

type MigrationStatus string

const (
	MigrationStatusPending   MigrationStatus = "pending"
	MigrationStatusPreparing MigrationStatus = "preparing"
	MigrationStatusPreDump   MigrationStatus = "pre_dump"
	MigrationStatusFinalDump MigrationStatus = "final_dump"
	MigrationStatusTransfer  MigrationStatus = "transfer"
	MigrationStatusRestore   MigrationStatus = "restore"
	MigrationStatusNetwork   MigrationStatus = "network_setup"
	MigrationStatusCompleted MigrationStatus = "completed"
	MigrationStatusFailed    MigrationStatus = "failed"
	MigrationStatusCancelled MigrationStatus = "cancelled"
)

func NewMigrationOrchestrator(
	vxlan *VXLANMesh,
	criu *CRIUMigrator,
	scheduler SchedulerClient,
	registry WorkerRegistry,
) *MigrationOrchestrator {
	return &MigrationOrchestrator{
		vxlanMesh:      vxlan,
		criuMigrator:   criu,
		scheduler:      scheduler,
		workerRegistry: registry,
		activeMigrations: make(map[string]*MigrationTask),
	}
}

func (mo *MigrationOrchestrator) MigrateContainer(ctx context.Context, containerID string, targetWorkerID uuid.UUID) (*MigrationResult, error) {
	taskID := uuid.New().String()

	task := &MigrationTask{
		ID:             taskID,
		ContainerID:    containerID,
		TargetWorkerID: targetWorkerID,
		Status:         MigrationStatusPending,
		StartedAt:      time.Now(),
		Progress:       0,
	}

	mo.mu.Lock()
	mo.activeMigrations[taskID] = task
	mo.mu.Unlock()

	defer func() {
		mo.mu.Lock()
		delete(mo.activeMigrations, taskID)
		mo.mu.Unlock()
	}()

	// Get source worker info (from container metadata)
	sourceWorkerID, err := mo.getContainerWorker(ctx, containerID)
	if err != nil {
		task.Status = MigrationStatusFailed
		task.Error = fmt.Sprintf("failed to get source worker: %v", err)
		return nil, err
	}
	task.SourceWorkerID = sourceWorkerID

	// Get target worker details
	targetWorker, err := mo.workerRegistry.GetWorker(ctx, targetWorkerID)
	if err != nil {
		task.Status = MigrationStatusFailed
		task.Error = fmt.Sprintf("target worker not found: %v", err)
		return nil, err
	}

	if targetWorker.Status != domain.WorkerStatusHealthy {
		task.Status = MigrationStatusFailed
		task.Error = fmt.Sprintf("target worker unhealthy: %s", targetWorker.Status)
		return nil, fmt.Errorf("target worker unhealthy: %s", targetWorker.Status)
	}

	// Phase 1: Prepare migration
	task.Status = MigrationStatusPreparing
	task.Progress = 0.1
	mo.updateTask(task)

	if err := mo.prepareMigration(ctx, task, targetWorker); err != nil {
		task.Status = MigrationStatusFailed
		task.Error = err.Error()
		return nil, err
	}

	// Phase 2: Pre-dump iterations
	task.Status = MigrationStatusPreDump
	task.Progress = 0.3
	mo.updateTask(task)

	if err := mo.runPreDump(ctx, task); err != nil {
		task.Status = MigrationStatusFailed
		task.Error = fmt.Sprintf("pre-dump failed: %v", err)
		return nil, err
	}

	// Phase 3: Final dump
	task.Status = MigrationStatusFinalDump
	task.Progress = 0.5
	mo.updateTask(task)

	checkpointResult, err := mo.runFinalDump(ctx, task)
	if err != nil {
		task.Status = MigrationStatusFailed
		task.Error = fmt.Sprintf("final dump failed: %v", err)
		return nil, err
	}
	task.CheckpointID = checkpointResult.CheckpointID

	// Phase 4: Transfer checkpoint
	task.Status = MigrationStatusTransfer
	task.Progress = 0.7
	mo.updateTask(task)

	if err := mo.transferCheckpoint(ctx, task, checkpointResult); err != nil {
		task.Status = MigrationStatusFailed
		task.Error = fmt.Sprintf("transfer failed: %v", err)
		return nil, err
	}

	// Phase 5: Restore on target
	task.Status = MigrationStatusRestore
	task.Progress = 0.85
	mo.updateTask(task)

	restoredPID, err := mo.restoreOnTarget(ctx, task, targetWorker)
	if err != nil {
		task.Status = MigrationStatusFailed
		task.Error = fmt.Sprintf("restore failed: %v", err)
		return nil, err
	}

	// Phase 6: Network setup on target
	task.Status = MigrationStatusNetwork
	task.Progress = 0.95
	mo.updateTask(task)

	if err := mo.setupNetworkOnTarget(ctx, task, targetWorker); err != nil {
		slog.Warn("network setup on target failed", "error", err)
		// Non-fatal, log and continue
	}

	// Complete
	task.Status = MigrationStatusCompleted
	now := time.Now()
	task.CompletedAt = &now
	task.Progress = 1.0
	mo.updateTask(task)

	completedAt := time.Now()
	return &MigrationResult{
		TaskID:         taskID,
		ContainerID:    containerID,
		SourceWorkerID: sourceWorkerID,
		TargetWorkerID: targetWorkerID,
		CheckpointID:   task.CheckpointID,
		RestoredPID:    restoredPID,
		Status:         MigrationStatusCompleted,
		StartedAt:      task.StartedAt,
		CompletedAt:    completedAt,
		Duration:       completedAt.Sub(task.StartedAt),
	}, nil
}

func (mo *MigrationOrchestrator) prepareMigration(ctx context.Context, task *MigrationTask, targetWorker *domain.Worker) error {
	slog.Info("Preparing migration", "task_id", task.ID, "container", task.ContainerID)

	// Verify target worker has capacity
	// Verify network connectivity between source and target
	// Pre-create VXLAN peer entry
	targetIP := targetWorker.Hostname // Assuming hostname is IP
	if err := mo.vxlanMesh.AddPeer(targetIP, targetWorker.Name, []string{}); err != nil {
		slog.Warn("failed to add VXLAN peer", "error", err)
	}

	// Create checkpoint directory on target
	// This would be done via SSH or agent RPC
	return nil
}

func (mo *MigrationOrchestrator) runPreDump(ctx context.Context, task *MigrationTask) error {
	slog.Info("Running pre-dump iterations", "task_id", task.ID)

	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		task.Progress = 0.3 + float64(i)*0.05
		mo.updateTask(task)

		config := CheckpointConfig{
			CRIUPath:      "/usr/bin/criu",
			CheckpointDir: "/var/lib/berth/checkpoints",
			WorkDir:       "/var/lib/berth/criu-work",
			ImagesDir:     "/var/lib/berth/criu-images",
			LeaveRunning:  true,
			TcpEstablished: true,
		}

		_, err := mo.criuMigrator.Checkpoint(ctx, task.ContainerID, config)
		if err != nil {
			return fmt.Errorf("pre-dump iteration %d failed: %w", i+1, err)
		}

		slog.Info("Pre-dump iteration completed", "iteration", i+1, "task_id", task.ID)
		time.Sleep(2 * time.Second)
	}
	return nil
}

func (mo *MigrationOrchestrator) runFinalDump(ctx context.Context, task *MigrationTask) (*CheckpointResult, error) {
	config := CheckpointConfig{
		CRIUPath:      "/usr/bin/criu",
		CheckpointDir: "/var/lib/berth/checkpoints",
		WorkDir:       "/var/lib/berth/criu-work",
		ImagesDir:     "/var/lib/berth/criu-images",
		LeaveRunning:  false,
		TcpEstablished: true,
	}

	result, err := mo.criuMigrator.Checkpoint(ctx, task.ContainerID, config)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (mo *MigrationOrchestrator) transferCheckpoint(ctx context.Context, task *MigrationTask, result *CheckpointResult) error {
	// Transfer to target worker
	// In production, this would use efficient streaming (rsync, gRPC streaming, etc.)
	// For now, use rsync over SSH
	slog.Info("Transferring checkpoint", "task_id", task.ID, "checkpoint", result.CheckpointID)

	// Simulate transfer
	time.Sleep(1 * time.Second)
	return nil
}

func (mo *MigrationOrchestrator) restoreOnTarget(ctx context.Context, task *MigrationTask, targetWorker *domain.Worker) (string, error) {
	restoreConfig := RestoreConfig{
		CRIUPath:      "/usr/bin/criu",
		CheckpointDir: "/var/lib/berth/checkpoints",
		WorkDir:       "/var/lib/berth/criu-work",
		ImagesDir:     "/var/lib/berth/criu-images",
		Detach:        true,
	}

	restoredPID, err := mo.criuMigrator.Restore(ctx, task.CheckpointID, restoreConfig)
	if err != nil {
		return "", err
	}

	return restoredPID, nil
}

func (mo *MigrationOrchestrator) setupNetworkOnTarget(ctx context.Context, task *MigrationTask, targetWorker *domain.Worker) error {
	// 1. Create VXLAN interface on target if not exists
	// 2. Add ARP entries for migrated container
	// 3. Update FDB entries
	// 3. Configure iptables rules
	// 4. Update DNS/service discovery

	slog.Info("Setting up network on target", "task_id", task.ID, "worker", targetWorker.Name)

	// Update VXLAN peer with new container info
	targetIP := targetWorker.Hostname
	mo.vxlanMesh.UpdatePeerLastSeen(targetIP)

	return nil
}

func (mo *MigrationOrchestrator) getContainerWorker(ctx context.Context, containerID string) (uuid.UUID, error) {
	// In production, this would query the container runtime or database
	// For now, return a placeholder
	return uuid.Nil, fmt.Errorf("not implemented")
}

func (mo *MigrationOrchestrator) updateTask(task *MigrationTask) {
	mo.mu.Lock()
	defer mo.mu.Unlock()
	if t, ok := mo.activeMigrations[task.ID]; ok {
		t.Status = task.Status
		t.Progress = task.Progress
		t.Error = task.Error
		t.CheckpointID = task.CheckpointID
	}
}

func (mo *MigrationOrchestrator) GetMigrationStatus(taskID string) (*MigrationTask, bool) {
	mo.mu.RLock()
	defer mo.mu.RUnlock()
	task, ok := mo.activeMigrations[taskID]
	return task, ok
}

func (mo *MigrationOrchestrator) CancelMigration(taskID string) error {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	task, ok := mo.activeMigrations[taskID]
	if !ok {
		return fmt.Errorf("migration not found")
	}

	if task.Status == MigrationStatusCompleted || task.Status == MigrationStatusFailed {
		return fmt.Errorf("migration already completed")
	}

	task.Status = MigrationStatusCancelled
	task.Error = "cancelled by user"
	return nil
}