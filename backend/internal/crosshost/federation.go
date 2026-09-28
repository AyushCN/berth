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

// ClusterFederation manages multi-cluster orchestration
type ClusterFederation struct {
	localCluster   *Cluster
	remoteClusters map[string]*RemoteCluster
	globalScheduler *GlobalScheduler
	mu             sync.RWMutex
}

type Cluster struct {
	ID              uuid.UUID              `json:"id"`
	Name            string                 `json:"name"`
	Region          string                 `json:"region"`
	Zone            string                 `json:"zone"`
	Provider        string                 `json:"provider"` // aws, gcp, azure, bare-metal
	Endpoint        string                 `json:"endpoint"` // API endpoint
	CA              string                 `json:"ca"`       // CA cert for TLS
	ClientCert      string                 `json:"client_cert"`
	ClientKey       string                 `json:"client_key"`
	Capacity        ClusterCapacity        `json:"capacity"`
	Status          ClusterStatus          `json:"status"`
	LastHeartbeat   time.Time              `json:"last_heartbeat"`
	RegisteredAt    time.Time              `json:"registered_at"`
	Labels          map[string]string      `json:"labels"`      // gpu=true, spot=true, etc.
	Taints          []Taint                `json:"taints"`      // Scheduling taints
}

type ClusterCapacity struct {
	TotalMemory   int64 `json:"total_memory"`   // Bytes
	TotalCPU      int64 `json:"total_cpu"`      // Milli-cores
	TotalDisk     int64 `json:"total_disk"`     // Bytes
	AvailableMemory int64 `json:"available_memory"`
	AvailableCPU  int64 `json:"available_cpu"`
	AvailableDisk int64 `json:"available_disk"`
	WorkerCount   int   `json:"worker_count"`
}

type Taint struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"` // NoSchedule, PreferNoSchedule, NoExecute
}

type ClusterStatus string

const (
	ClusterStatusHealthy   ClusterStatus = "healthy"
	ClusterStatusDegraded  ClusterStatus = "degraded"
	ClusterStatusUnhealthy ClusterStatus = "unhealthy"
	ClusterStatusOffline   ClusterStatus = "offline"
)

type RemoteCluster struct {
	Cluster
	Client  ClusterClient `json:"-"` // gRPC/HTTP client
	Latency time.Duration `json:"latency"`
	Cost    float64       `json:"cost_per_hour"` // For cost-aware scheduling
}

type ClusterClient interface {
	Schedule(ctx context.Context, req *ScheduleRequest) (*ScheduleResponse, error)
	GetCapacity(ctx context.Context) (*ClusterCapacity, error)
	MigrateIn(ctx context.Context, req *MigrationRequest) (*MigrationResponse, error)
	MigrateOut(ctx context.Context, req *MigrationRequest) (*MigrationResponse, error)
	GetClusterInfo(ctx context.Context) (*ClusterInfo, error)
	HealthCheck(ctx context.Context) error
}

type ClusterInfo struct {
	Cluster
	Workers []WorkerSummary `json:"workers"`
}

type WorkerSummary struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	Capacity       ClusterCapacity `json:"capacity"`
	ActiveSandboxes int      `json:"active_sandboxes"`
	Labels         map[string]string `json:"labels"`
}

type ScheduleRequest struct {
	SandboxSpec   domain.SandboxSpec     `json:"sandbox_spec"`
	Tenant        *domain.Tenant         `json:"tenant"`
	Preferences   SchedulePreferences    `json:"preferences"`
}

type SchedulePreferences struct {
	Region          string            `json:"region,omitempty"`
	Zone            string            `json:"zone,omitempty"`
	Provider        string            `json:"provider,omitempty"`
	RequireGPU      bool              `json:"require_gpu"`
	SpotInstance    bool              `json:"spot_instance"`
	MaxPricePerHour float64           `json:"max_price_per_hour"`
	Labels          map[string]string `json:"labels"`
	Affinity        *AffinityRules    `json:"affinity"`
}

type AffinityRules struct {
	RequiredDuringScheduling  []AffinityTerm `json:"required_during_scheduling"`
	PreferredDuringScheduling []WeightedAffinityTerm `json:"preferred_during_scheduling"`
}

type AffinityTerm struct {
	TopologyKey string   `json:"topology_key"`
	Values      []string `json:"values"`
}

type WeightedAffinityTerm struct {
	Weight int          `json:"weight"`
	Term   AffinityTerm `json:"term"`
}

type ScheduleResponse struct {
	ClusterID   uuid.UUID `json:"cluster_id"`
	WorkerID    uuid.UUID `json:"worker_id"`
	Score       float64   `json:"score"`
	Reason      string    `json:"reason"`
}

type MigrationRequest struct {
	ContainerID   string `json:"container_id"`
	SourceCluster string `json:"source_cluster"`
	TargetCluster string `json:"target_cluster"`
	Reason        string `json:"reason"`
	Priority      int    `json:"priority"`
}

type MigrationResponse struct {
	MigrationID string `json:"migration_id"`
	Status      string `json:"status"`
	TargetNode  string `json:"target_node"`
}

type GlobalScheduler struct {
	federation       *ClusterFederation
	mu               sync.RWMutex
	clusterScores    map[uuid.UUID]float64
	lastScoreUpdate  time.Time
}

func NewClusterFederation(localCluster *Cluster) *ClusterFederation {
	return &ClusterFederation{
		localCluster:    localCluster,
		remoteClusters:  make(map[string]*RemoteCluster),
		globalScheduler: NewGlobalScheduler(),
	}
}

func NewGlobalScheduler() *GlobalScheduler {
	return &GlobalScheduler{
		clusterScores:   make(map[uuid.UUID]float64),
		lastScoreUpdate: time.Now(),
	}
}

func (cf *ClusterFederation) RegisterRemoteCluster(ctx context.Context, cluster *Cluster, client ClusterClient) error {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	remote := &RemoteCluster{
		Cluster: *cluster,
		Client:  client,
	}

	if err := remote.Client.HealthCheck(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	cf.remoteClusters[cluster.ID.String()] = remote
	slog.Info("Remote cluster registered", "cluster_id", cluster.ID, "name", cluster.Name)
	return nil
}

func (cf *ClusterFederation) UnregisterRemoteCluster(clusterID uuid.UUID) {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	delete(cf.remoteClusters, clusterID.String())
	slog.Info("Remote cluster unregistered", "cluster_id", clusterID)
}

func (cf *ClusterFederation) GetCluster(clusterID uuid.UUID) (*RemoteCluster, bool) {
	cf.mu.RLock()
	defer cf.mu.RUnlock()
	cluster, ok := cf.remoteClusters[clusterID.String()]
	return cluster, ok
}

func (cf *ClusterFederation) ListClusters() []*RemoteCluster {
	cf.mu.RLock()
	defer cf.mu.RUnlock()

	clusters := make([]*RemoteCluster, 0, len(cf.remoteClusters)+1)
	clusters = append(clusters, &RemoteCluster{Cluster: *cf.localCluster})
	for _, c := range cf.remoteClusters {
		clusters = append(clusters, c)
	}
	return clusters
}

func (cf *ClusterFederation) GetHealthyClusters() []*RemoteCluster {
	cf.mu.RLock()
	defer cf.mu.RUnlock()

	clusters := []*RemoteCluster{}
	if cf.localCluster.Status == ClusterStatusHealthy {
		clusters = append(clusters, &RemoteCluster{Cluster: *cf.localCluster})
	}
	for _, c := range cf.remoteClusters {
		if c.Status == ClusterStatusHealthy {
			clusters = append(clusters, c)
		}
	}
	return clusters
}

// ScheduleGlobally finds the best cluster and worker for a sandbox
func (cf *ClusterFederation) ScheduleGlobally(ctx context.Context, req *ScheduleRequest) (*GlobalScheduleResult, error) {
	clusters := cf.GetHealthyClusters()
	if len(clusters) == 0 {
		return nil, fmt.Errorf("no healthy clusters available")
	}

	// Filter clusters by preferences
	candidates := cf.filterClusters(clusters, req.Preferences)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no clusters match preferences")
	}

	// Score each cluster
	type scoredCluster struct {
		cluster *RemoteCluster
		score   float64
		worker  *WorkerSummary
	}

	var best *scoredCluster
	for _, cluster := range candidates {
		// Get cluster capacity
		capacity, err := cluster.Client.GetCapacity(ctx)
		if err != nil {
			slog.Warn("failed to get cluster capacity", "cluster", cluster.Name, "error", err)
			continue
		}

		// Check if cluster has capacity
		if !cf.hasCapacity(capacity, req.SandboxSpec) {
			continue
		}

		// Ask cluster to schedule
		resp, err := cluster.Client.Schedule(ctx, req)
		if err != nil {
			slog.Warn("cluster schedule failed", "cluster", cluster.Name, "error", err)
			continue
		}

		// Calculate score based on preferences
		score := cf.calculateScore(cluster, req.Preferences, capacity, resp)

		if best == nil || score > best.score {
			best = &scoredCluster{
				cluster: cluster,
				score:   score,
				worker:  &WorkerSummary{ID: resp.WorkerID},
			}
		}
	}

	if best == nil {
		return nil, fmt.Errorf("no suitable cluster found")
	}

	return &GlobalScheduleResult{
		ClusterID:   best.cluster.ID,
		ClusterName: best.cluster.Name,
		WorkerID:    best.worker.ID,
		Score:       best.score,
		Reason:      fmt.Sprintf("Selected cluster %s with score %.2f", best.cluster.Name, best.score),
	}, nil
}

type GlobalScheduleResult struct {
	ClusterID   uuid.UUID `json:"cluster_id"`
	ClusterName string    `json:"cluster_name"`
	WorkerID    uuid.UUID `json:"worker_id"`
	Score       float64   `json:"score"`
	Reason      string    `json:"reason"`
}

func (cf *ClusterFederation) filterClusters(clusters []*RemoteCluster, prefs SchedulePreferences) []*RemoteCluster {
	filtered := []*RemoteCluster{}

	for _, c := range clusters {
		// Region filter
		if prefs.Region != "" && c.Region != prefs.Region {
			continue
		}
		if prefs.Zone != "" && c.Zone != prefs.Zone {
			continue
		}
		if prefs.Provider != "" && c.Provider != prefs.Provider {
			continue
		}

		// GPU requirement
		if prefs.RequireGPU {
			hasGPU := false
			for k := range c.Labels {
				if k == "gpu" || k == "nvidia-gpu" {
					hasGPU = true
					break
				}
			}
			if !hasGPU {
				continue
			}
		}

		// Spot instance preference
		if prefs.SpotInstance {
			hasSpot := false
			for k, v := range c.Labels {
				if k == "spot" && v == "true" {
					hasSpot = true
					break
				}
			}
			if !hasSpot {
				continue
			}
		}

		// Max price check
		if prefs.MaxPricePerHour > 0 && c.Cost > prefs.MaxPricePerHour {
			continue
		}

		// Label matching
		labelMatch := true
		for k, v := range prefs.Labels {
			if c.Labels[k] != v {
				labelMatch = false
				break
			}
		}
		if !labelMatch {
			continue
		}

		filtered = append(filtered, c)
	}

	return filtered
}

func (cf *ClusterFederation) hasCapacity(capacity *ClusterCapacity, spec domain.SandboxSpec) bool {
	return capacity.AvailableMemory >= spec.MemoryLimit &&
		capacity.AvailableCPU >= spec.CPULimit &&
		capacity.AvailableDisk >= spec.DiskLimit
}

func (cf *ClusterFederation) calculateScore(cluster *RemoteCluster, prefs SchedulePreferences, capacity *ClusterCapacity, resp *ScheduleResponse) float64 {
	score := 0.0

	// Base score from cluster scheduler
	score += resp.Score * 0.4

	// Resource availability score (0-0.3)
	memAvail := float64(capacity.AvailableMemory) / float64(capacity.TotalMemory)
	cpuAvail := float64(capacity.AvailableCPU) / float64(capacity.TotalCPU)
	score += (memAvail + cpuAvail) / 2 * 0.3

	// Latency score (lower latency = higher score) (0-0.2)
	latencyScore := 1.0 - (float64(cluster.Latency.Milliseconds()) / 1000.0)
	if latencyScore < 0 {
		latencyScore = 0
	}
	score += latencyScore * 0.2

	// Cost score (0-0.1) - lower cost = higher score
	if cluster.Cost > 0 {
		costScore := 1.0 / (1.0 + cluster.Cost/100.0) // Normalize
		score += costScore * 0.1
	}

	return score
}

// MigrateAcrossClusters migrates a container from one cluster to another
func (cf *ClusterFederation) MigrateAcrossClusters(ctx context.Context, req *MigrationRequest) (*CrossClusterMigrationResult, error) {
	sourceCluster, ok := cf.GetCluster(uuid.MustParse(req.SourceCluster))
	if !ok {
		return nil, fmt.Errorf("source cluster not found")
	}

	targetCluster, ok := cf.GetCluster(uuid.MustParse(req.TargetCluster))
	if !ok {
		return nil, fmt.Errorf("target cluster not found")
	}

	// Initiate migration out from source
	migrateOutResp, err := sourceCluster.Client.MigrateOut(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("migrate out failed: %w", err)
	}

	// Initiate migration in to target
	migrateInReq := &MigrationRequest{
		ContainerID:   req.ContainerID,
		SourceCluster: req.SourceCluster,
		TargetCluster: req.TargetCluster,
		Reason:        req.Reason,
		Priority:      req.Priority,
	}

	migrateInResp, err := targetCluster.Client.MigrateIn(ctx, migrateInReq)
	if err != nil {
		return nil, fmt.Errorf("migrate in failed: %w", err)
	}

	slog.Info("Cross-cluster migration initiated",
		"container", req.ContainerID,
		"source", req.SourceCluster,
		"target", req.TargetCluster,
		"migration_id", migrateOutResp.MigrationID)

	return &CrossClusterMigrationResult{
		MigrationID:      migrateOutResp.MigrationID,
		SourceCluster:    req.SourceCluster,
		TargetCluster:    req.TargetCluster,
		Status:           "initiated",
		TargetNode:       migrateInResp.TargetNode,
		StartedAt:        time.Now(),
	}, nil
}

type CrossClusterMigrationResult struct {
	MigrationID   string    `json:"migration_id"`
	SourceCluster string    `json:"source_cluster"`
	TargetCluster string    `json:"target_cluster"`
	Status        string    `json:"status"`
	TargetNode    string    `json:"target_node"`
	StartedAt     time.Time `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// HeartbeatLoop periodically checks remote cluster health
func (cf *ClusterFederation) HeartbeatLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cf.checkClusterHealth(ctx)
		}
	}
}

func (cf *ClusterFederation) checkClusterHealth(ctx context.Context) {
	cf.mu.RLock()
	clusters := make([]*RemoteCluster, 0, len(cf.remoteClusters))
	for _, c := range cf.remoteClusters {
		clusters = append(clusters, c)
	}
	cf.mu.RUnlock()

	for _, cluster := range clusters {
		go func(c *RemoteCluster) {
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			if err := c.Client.HealthCheck(ctx); err != nil {
				slog.Warn("Cluster health check failed", "cluster", c.Name, "error", err)
				cf.updateClusterStatus(c.ID, ClusterStatusUnhealthy)
			} else {
				c.Latency = time.Since(time.Now().Add(-c.Latency)) // Approximate
				c.LastHeartbeat = time.Now()
				if c.Status != ClusterStatusHealthy {
					cf.updateClusterStatus(c.ID, ClusterStatusHealthy)
				}
			}
		}(cluster)
	}
}

func (cf *ClusterFederation) updateClusterStatus(clusterID uuid.UUID, status ClusterStatus) {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	if clusterID == cf.localCluster.ID {
		cf.localCluster.Status = status
	} else if cluster, ok := cf.remoteClusters[clusterID.String()]; ok {
		cluster.Status = status
	}
}

// GetClusterCapacity returns aggregated capacity across all clusters
func (cf *ClusterFederation) GetClusterCapacity(ctx context.Context) (map[uuid.UUID]*ClusterCapacity, error) {
	capacities := make(map[uuid.UUID]*ClusterCapacity)

	// Local cluster
	capacities[cf.localCluster.ID] = &cf.localCluster.Capacity

	// Remote clusters
	cf.mu.RLock()
	clusters := make([]*RemoteCluster, 0, len(cf.remoteClusters))
	for _, c := range cf.remoteClusters {
		clusters = append(clusters, c)
	}
	cf.mu.RUnlock()

	for _, cluster := range clusters {
		capacity, err := cluster.Client.GetCapacity(ctx)
		if err != nil {
			slog.Warn("failed to get capacity", "cluster", cluster.Name, "error", err)
			continue
		}
		capacities[cluster.ID] = capacity
	}

	return capacities, nil
}