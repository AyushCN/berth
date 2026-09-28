package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Tenant represents an organization or team using the platform.
type Tenant struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"` // URL-friendly identifier
	OwnerID     uuid.UUID `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	IsActive    bool      `json:"is_active"`
	Settings    TenantSettings `json:"settings"`
}

// TenantSettings contains tenant-specific configuration.
type TenantSettings struct {
	MaxSandboxes         int   `json:"max_sandboxes"`          // Max concurrent sandboxes
	MaxMemoryPerSandbox  int64 `json:"max_memory_per_sandbox"` // Bytes
	MaxCPUPerSandbox     int64 `json:"max_cpu_per_sandbox"`    // Milli-cores
	MaxDiskPerSandbox    int64 `json:"max_disk_per_sandbox"`   // Bytes
	AllowedRuntimes      []SandboxRuntime `json:"allowed_runtimes"`
	AllowedNetworkModes  []NetworkMode    `json:"allowed_network_modes"`
	AllowedFilesystemModes []FilesystemMode `json:"allowed_filesystem_modes"`
	DefaultExecutionProfile *ExecutionProfile `json:"default_execution_profile"`
	NetworkEgressPolicy  string           `json:"network_egress_policy"` // default, restricted, none
	EnableInternetAccess bool             `json:"enable_internet_access"`
	CustomBaseImages     []string         `json:"custom_base_images"`
}

// TenantQuota tracks resource usage for a tenant.
type TenantQuota struct {
	TenantID          uuid.UUID `json:"tenant_id"`
	ActiveSandboxes   int       `json:"active_sandboxes"`
	TotalMemoryUsed   int64     `json:"total_memory_used"`   // Bytes
	TotalCPUUsed      int64     `json:"total_cpu_used"`      // Milli-cores
	TotalDiskUsed     int64     `json:"total_disk_used"`     // Bytes
	LastUpdated       time.Time `json:"last_updated"`
}

// TenantRepository defines the interface for tenant persistence.
type TenantRepository interface {
	Create(ctx context.Context, t *Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*Tenant, error)
	GetByOwner(ctx context.Context, ownerID uuid.UUID) (*Tenant, error)
	Update(ctx context.Context, t *Tenant) error
	UpdateQuota(ctx context.Context, q *TenantQuota) error
	GetQuota(ctx context.Context, tenantID uuid.UUID) (*TenantQuota, error)
	List(ctx context.Context) ([]*Tenant, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// Worker represents a compute node that can run sandboxes.
type Worker struct {
	ID              uuid.UUID         `json:"id"`
	Name            string            `json:"name"`            // Human-readable name
	Hostname        string            `json:"hostname"`        // Network hostname/IP
	APIPort         int               `json:"api_port"`        // gRPC/HTTP API port
	ContainerdSock  string            `json:"containerd_sock"` // Path to containerd socket
	Labels          map[string]string `json:"labels"`          // Custom labels (e.g., gpu=true, region=us-east)
	MaxMemory       int64             `json:"max_memory"`      // Total memory in bytes
	MaxCPU          int64             `json:"max_cpu"`         // Total CPU in milli-cores
	MaxDisk         int64             `json:"max_disk"`        // Total disk in bytes
	Status          WorkerStatus      `json:"status"`
	LastHeartbeat   time.Time         `json:"last_heartbeat"`
	RegisteredAt    time.Time         `json:"registered_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Metadata        map[string]string `json:"metadata"`        // Additional metadata
}

// WorkerStatus represents the health/status of a worker.
type WorkerStatus string

const (
	WorkerStatusHealthy     WorkerStatus = "healthy"
	WorkerStatusDegraded    WorkerStatus = "degraded"    // Some resources exhausted
	WorkerStatusUnhealthy   WorkerStatus = "unhealthy"   // Failed health checks
	WorkerStatusDraining    WorkerStatus = "draining"    // Not accepting new work
	WorkerStatusOffline     WorkerStatus = "offline"     // No heartbeat
)

// WorkerCapacity represents available resources on a worker.
type WorkerCapacity struct {
	WorkerID       uuid.UUID `json:"worker_id"`
	AvailableMemory int64    `json:"available_memory"`   // Bytes
	AvailableCPU    int64    `json:"available_cpu"`     // Milli-cores
	AvailableDisk   int64    `json:"available_disk"`    // Bytes
	ActiveSandboxes int      `json:"active_sandboxes"`  // Current running sandboxes
}

// WorkerRepository defines the interface for worker persistence.
type WorkerRepository interface {
	Create(ctx context.Context, w *Worker) error
	GetByID(ctx context.Context, id uuid.UUID) (*Worker, error)
	GetByHostname(ctx context.Context, hostname string) (*Worker, error)
	Update(ctx context.Context, w *Worker) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status WorkerStatus) error
	UpdateHeartbeat(ctx context.Context, id uuid.UUID, capacity *WorkerCapacity) error
	List(ctx context.Context) ([]*Worker, error)
	ListHealthy(ctx context.Context) ([]*Worker, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// Scheduler orchestrates sandbox placement across workers.
type Scheduler interface {
	// Schedule finds the best worker for a sandbox spec.
	Schedule(ctx context.Context, spec *SandboxSpec, tenant *Tenant) (*Worker, error)
	
	// Reschedule moves a sandbox to a different worker.
	Reschedule(ctx context.Context, sandboxID uuid.UUID, targetWorkerID uuid.UUID) error
	
	// Evict evicts sandboxes to make room for higher priority work.
	Evict(ctx context.Context, workerID uuid.UUID, neededMemory, neededCPU int64) ([]uuid.UUID, error)
}

// PlacementStrategy defines how sandboxes are placed on workers.
type PlacementStrategy string

const (
	PlacementStrategyBinPack    PlacementStrategy = "binpack"    // Pack tightly to minimize fragmentation
	PlacementStrategySpread     PlacementStrategy = "spread"     // Spread across workers for HA
	PlacementStrategyLeastUsed  PlacementStrategy = "least_used" // Pick worker with least load
	PlacementStrategyCustom     PlacementStrategy = "custom"     // Custom scoring function
)

// SchedulerConfig contains scheduler configuration.
type SchedulerConfig struct {
	Strategy           PlacementStrategy `json:"strategy"`
	EnablePreemption   bool              `json:"enable_preemption"`
	ReservedMemory     int64             `json:"reserved_memory"`     // Memory reserved per worker (bytes)
	ReservedCPU        int64             `json:"reserved_cpu"`        // CPU reserved per worker (milli-cores)
	MaxSandboxesPerWorker int            `json:"max_sandboxes_per_worker"`
	HealthCheckInterval time.Duration   `json:"health_check_interval"`
	HeartbeatTimeout   time.Duration    `json:"heartbeat_timeout"`
}