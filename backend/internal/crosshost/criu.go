package crosshost

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type CRIUMigrator struct {
	criuPath       string
	checkpointDir  string
	workDir        string
	networkPlugin  NetworkMigrator
}

type NetworkMigrator interface {
	SaveNetworkState(ctx context.Context, containerID string) (*NetworkState, error)
	RestoreNetworkState(ctx context.Context, state *NetworkState) error
}

type NetworkState struct {
	ContainerID   string            `json:"container_id"`
	Interfaces    []InterfaceState  `json:"interfaces"`
	Routes        []RouteState      `json:"routes"`
	IPTablesRules []string          `json:"iptables_rules"`
}

type InterfaceState struct {
	Name       string   `json:"name"`
	Index      int      `json:"index"`
	MAC        string   `json:"mac"`
	IPs        []string `json:"ips"`
	MTU        int      `json:"mtu"`
	Up         bool     `json:"up"`
	Master     string   `json:"master,omitempty"`
	PeerIndex  int      `json:"peer_index,omitempty"`
}

type RouteState struct {
	Dst     string `json:"dst"`
	Src     string `json:"src,omitempty"`
	Gateway string `json:"gateway,omitempty"`
	Dev     string `json:"dev"`
	Scope   int    `json:"scope"`
	Table   int    `json:"table"`
}

type CheckpointConfig struct {
	CRIUPath      string   `json:"criu_path"`
	CheckpointDir string   `json:"checkpoint_dir"`
	WorkDir       string   `json:"work_dir"`
	ImagesDir     string   `json:"images_dir"`
	LeaveRunning  bool     `json:"leave_running"`
	TcpEstablished bool    `json:"tcp_established"`
	ShellJob      bool     `json:"shell_job"`
	FileLocks     bool     `json:"file_locks"`
	ExtUnixSk     bool     `json:"ext_unix_sk"`
}

type CheckpointResult struct {
	ContainerID   string    `json:"container_id"`
	CheckpointID  string    `json:"checkpoint_id"`
	Path          string    `json:"path"`
	Size          int64     `json:"size"`
	CreatedAt     time.Time `json:"created_at"`
	Config        CheckpointConfig `json:"config"`
	NetworkState  *NetworkState    `json:"network_state,omitempty"`
}

type RestoreConfig struct {
	CRIUPath      string `json:"criu_path"`
	CheckpointDir string   `json:"checkpoint_dir"`
	WorkDir       string   `json:"work_dir"`
	ImagesDir     string   `json:"images_dir"`
	NewContainerID string `json:"new_container_id,omitempty"`
	Detach        bool     `json:"detach"`
	PidFile       string   `json:"pid_file"`
}

type MigrationTarget struct {
	Address        string `json:"address"`
	CRIUPath       string `json:"criu_path"`
	CheckpointDir  string `json:"checkpoint_dir"`
	WorkDir        string `json:"work_dir"`
	ImagesDir      string `json:"images_dir"`
	SSHUser        string `json:"ssh_user"`
	SSHKey         string `json:"ssh_key"`
}

type MigrationResult struct {
	TaskID           string            `json:"task_id"`
	ContainerID      string            `json:"container_id"`
	SourceWorkerID   uuid.UUID         `json:"source_worker_id"`
	TargetWorkerID   uuid.UUID         `json:"target_worker_id"`
	CheckpointID     string            `json:"checkpoint_id"`
	RestoredPID      string            `json:"restored_pid"`
	Status           MigrationStatus   `json:"status"`
	StartedAt        time.Time         `json:"started_at"`
	CompletedAt      time.Time         `json:"completed_at"`
	Duration         time.Duration     `json:"duration"`
}

type PageServer struct {
	mu           sync.Mutex
	checkpointDir string
	activeTransfers map[string]*PageTransfer
}

type PageTransfer struct {
	ContainerID  string
	CheckpointID string
	RequestedPages map[int]bool
	LastRequest  time.Time
}

func NewCRIUMigrator(config CheckpointConfig) (*CRIUMigrator, error) {
	if config.CRIUPath == "" {
		config.CRIUPath = "criu"
	}
	if config.CheckpointDir == "" {
		config.CheckpointDir = "/var/lib/berth/checkpoints"
	}
	if config.WorkDir == "" {
		config.WorkDir = "/var/lib/berth/criu-work"
	}
	if config.ImagesDir == "" {
		config.ImagesDir = "/var/lib/berth/criu-images"
	}

	if _, err := exec.LookPath(config.CRIUPath); err != nil {
		return nil, fmt.Errorf("criu not found in PATH: %w", err)
	}

	out, err := exec.Command(config.CRIUPath, "--version").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get criu version: %w", err)
	}
	slog.Info("CRIU version", "output", string(out))

	for _, dir := range []string{config.CheckpointDir, config.WorkDir, config.ImagesDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return &CRIUMigrator{
		criuPath:      config.CRIUPath,
		checkpointDir: config.CheckpointDir,
		workDir:       config.WorkDir,
	}, nil
}

func (m *CRIUMigrator) Checkpoint(ctx context.Context, containerID string, config CheckpointConfig) (*CheckpointResult, error) {
	checkpointID := fmt.Sprintf("checkpoint-%s-%d", containerID[:8], time.Now().Unix())
	checkpointPath := filepath.Join(config.CheckpointDir, checkpointID)

	if err := os.MkdirAll(checkpointPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	args := []string{
		"dump",
		"-t", containerID,
		"-D", checkpointPath,
		"-W", config.WorkDir,
		"--images-dir", config.ImagesDir,
		"-v4",
		"-o",
	}

	if config.LeaveRunning {
		args = append(args, "--leave-running")
	}
	if config.TcpEstablished {
		args = append(args, "--tcp-established")
	}
	if config.ShellJob {
		args = append(args, "--shell-job")
	}
	if config.FileLocks {
		args = append(args, "--file-locks")
	}
	if config.ExtUnixSk {
		args = append(args, "--ext-unix-sk")
	}

	slog.Info("Starting CRIU checkpoint", "container", containerID, "path", checkpointPath)

	cmd := exec.CommandContext(ctx, config.CRIUPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.RemoveAll(checkpointPath)
		return nil, fmt.Errorf("criu dump failed: %w", err)
	}

	size, err := getDirSize(checkpointPath)
	if err != nil {
		slog.Warn("failed to calculate checkpoint size", "error", err)
	}

	result := &CheckpointResult{
		ContainerID:  containerID,
		CheckpointID: checkpointID,
		Path:         checkpointPath,
		Size:         size,
		CreatedAt:    time.Now(),
		Config:       config,
	}

	slog.Info("Checkpoint completed", "container", containerID, "checkpoint", checkpointID, "size", size)
	return result, nil
}

func (m *CRIUMigrator) Restore(ctx context.Context, checkpointID string, config RestoreConfig) (string, error) {
	checkpointPath := filepath.Join(config.CheckpointDir, checkpointID)

	if _, err := os.Stat(checkpointPath); os.IsNotExist(err) {
		return "", fmt.Errorf("checkpoint not found: %s", checkpointPath)
	}

	args := []string{
		"restore",
		"-D", checkpointPath,
		"-W", config.WorkDir,
		"--images-dir", config.ImagesDir,
		"-v4",
		"-o",
	}

	if config.Detach {
		args = append(args, "-d")
	}
	if config.PidFile != "" {
		args = append(args, "--pid-file", config.PidFile)
	}
	if config.NewContainerID != "" {
		args = append(args, "--new-pid", config.NewContainerID)
	}

	slog.Info("Starting CRIU restore", "checkpoint", checkpointID, "path", checkpointPath)

	cmd := exec.CommandContext(ctx, config.CRIUPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("criu restore failed: %w", err)
	}

	restoredPID := config.NewContainerID
	if restoredPID == "" {
		if config.PidFile != "" {
			if data, err := os.ReadFile(config.PidFile); err == nil {
				restoredPID = strings.TrimSpace(string(data))
			}
		}
	}

	slog.Info("Restore completed", "checkpoint", checkpointID, "pid", restoredPID)
	return restoredPID, nil
}

func (m *CRIUMigrator) PreDump(ctx context.Context, containerID string, config CheckpointConfig) (*CheckpointResult, error) {
	config.LeaveRunning = true
	return m.Checkpoint(ctx, containerID, config)
}

func (m *CRIUMigrator) Migrate(ctx context.Context, containerID string, targetNode *MigrationTarget) (*MigrationResult, error) {
	slog.Info("Starting live migration", "container", containerID, "target", targetNode.Address)

	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		slog.Info("Pre-dump iteration", "iteration", i+1)
		config := CheckpointConfig{
			CRIUPath:      m.criuPath,
			CheckpointDir: m.checkpointDir,
			WorkDir:       m.workDir,
			ImagesDir:     m.criuPath + "-images",
			LeaveRunning:  true,
			TcpEstablished: true,
		}

		result, err := m.Checkpoint(ctx, containerID, config)
		if err != nil {
			return nil, fmt.Errorf("pre-dump iteration %d failed: %w", i+1, err)
		}

		if err := m.transferCheckpoint(ctx, result, targetNode); err != nil {
			slog.Warn("checkpoint transfer failed", "error", err)
		}

		time.Sleep(2 * time.Second)
	}

	slog.Info("Final checkpoint (container will pause)")
	finalConfig := CheckpointConfig{
		CRIUPath:      m.criuPath,
		CheckpointDir: m.checkpointDir,
		WorkDir:       m.workDir,
		ImagesDir:     m.criuPath + "-images",
		LeaveRunning:  false,
		TcpEstablished: true,
	}

	finalResult, err := m.Checkpoint(ctx, containerID, finalConfig)
	if err != nil {
		return nil, fmt.Errorf("final checkpoint failed: %w", err)
	}

	if err := m.transferCheckpoint(ctx, finalResult, targetNode); err != nil {
		return nil, fmt.Errorf("final checkpoint transfer failed: %w", err)
	}

	restoreConfig := RestoreConfig{
		CRIUPath:      targetNode.CRIUPath,
		CheckpointDir: targetNode.CheckpointDir,
		WorkDir:       targetNode.WorkDir,
		ImagesDir:     targetNode.ImagesDir,
		Detach:        true,
	}

	restoredPID, err := m.Restore(ctx, finalResult.CheckpointID, restoreConfig)
	if err != nil {
		return nil, fmt.Errorf("restore on target failed: %w", err)
	}

	slog.Info("Live migration completed", "container", containerID, "target", targetNode.Address, "pid", restoredPID)

	return &MigrationResult{
		ContainerID:   containerID,
		CheckpointID:  finalResult.CheckpointID,
		RestoredPID:   restoredPID,
		CompletedAt:   time.Now(),
	}, nil
}

func (m *CRIUMigrator) transferCheckpoint(ctx context.Context, result *CheckpointResult, target *MigrationTarget) error {
	src := result.Path
	dst := fmt.Sprintf("%s@%s:%s", target.SSHUser, target.Address, target.CheckpointDir)

	cmd := exec.CommandContext(ctx, "rsync", "-avz", "-e", "ssh -i "+target.SSHKey, src+"/", dst+"/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (m *CRIUMigrator) SaveNetworkState(ctx context.Context, containerID string) (*NetworkState, error) {
	return &NetworkState{
		ContainerID: containerID,
		Interfaces:  []InterfaceState{},
		Routes:      []RouteState{},
		IPTablesRules: []string{},
	}, nil
}

func (m *CRIUMigrator) RestoreNetworkState(ctx context.Context, state *NetworkState) error {
	return nil
}

func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
