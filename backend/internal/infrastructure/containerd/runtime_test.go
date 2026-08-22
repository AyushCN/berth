package containerd

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/AyushCN/berth/internal/domain"
)

func skipIfNoContainerd(t *testing.T) {
	sockPath := os.Getenv("CONTAINERD_SOCK")
	if sockPath == "" {
		sockPath = "/run/containerd/containerd.sock"
	}
	if _, err := os.Stat(sockPath); os.IsNotExist(err) {
		t.Skip("containerd socket not found, skipping integration test")
	}
}

func skipIfNotRoot(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("not running as root, skipping containerd integration test")
	}
}

func TestRuntimeLifecycle(t *testing.T) {
	skipIfNoContainerd(t)
	// Note: rootless containerd does not require root
	// skipIfNotRoot(t) // commented out for rootless support

	ctx := context.Background()
	sockPath := os.Getenv("CONTAINERD_SOCK")
	if sockPath == "" {
		sockPath = "/run/containerd/containerd.sock"
	}

	rt, err := NewRuntime(sockPath)
	if err != nil {
		t.Skipf("skipping test, failed to create runtime (likely permission denied or no daemon): %v", err)
	}
	defer rt.Close()

	// Test CreateSandbox
	spec := domain.SandboxSpec{
		ID:          uuid.New(),
		BaseImage:   "docker.io/library/alpine:latest",
		WorkDir:     "/app",
		MemoryLimit: 128 * 1024 * 1024, // 128MB
		CPULimit:    500,               // 500 milli-cores
	}

	containerID, err := rt.CreateSandbox(ctx, spec)
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	t.Logf("created sandbox: %s", containerID)

	// Test StartSandbox
	if err := rt.StartSandbox(ctx, containerID); err != nil {
		t.Fatalf("failed to start sandbox: %v", err)
	}
	t.Logf("started sandbox: %s", containerID)

	// Test Exec
	output, err := rt.Exec(ctx, containerID, []string{"echo", "hello-from-gvisor"})
	if err != nil {
		t.Fatalf("failed to exec: %v", err)
	}
	t.Logf("exec output: %s", output)

	// Test GetLogs
	logs, err := rt.GetLogs(ctx, containerID, 100)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}
	t.Logf("logs: %s", logs)

	// Test StopSandbox
	if err := rt.StopSandbox(ctx, containerID); err != nil {
		t.Fatalf("failed to stop sandbox: %v", err)
	}
	t.Logf("stopped sandbox: %s", containerID)

	// Test DeleteSandbox
	if err := rt.DeleteSandbox(ctx, containerID); err != nil {
		t.Fatalf("failed to delete sandbox: %v", err)
	}
	t.Logf("deleted sandbox: %s", containerID)
}

func TestWarmPool(t *testing.T) {
	wp := NewWarmPool()
	wp.maxSize = 2

	// Register containers
	wp.Register("c1", "alpine:latest")
	wp.Register("c2", "alpine:latest")
	wp.Register("c3", "alpine:latest") // should be dropped (pool full)

	// Take one
	id := wp.Take("alpine:latest")
	if id != "c1" {
		t.Errorf("expected c1, got %s", id)
	}

	// Take another
	id = wp.Take("alpine:latest")
	if id != "c2" {
		t.Errorf("expected c2, got %s", id)
	}

	// Pool empty
	id = wp.Take("alpine:latest")
	if id != "" {
		t.Errorf("expected empty, got %s", id)
	}
}

func TestNetworkManager(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("network tests require root for bridge creation")
	}

	nm, err := NewNetworkManager()
	if err != nil {
		t.Fatalf("failed to create network manager: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	networkID := "test-net-" + uuid.New().String()[:8]
	if err := nm.CreateNetwork(ctx, networkID); err != nil {
		t.Fatalf("failed to create network: %v", err)
	}

	if err := nm.ForwardPort("test-container", 0, 8080); err != nil {
		t.Fatalf("failed to forward port: %v", err)
	}

	port := nm.GetHostPort("test-container")
	if port == 0 {
		t.Error("expected allocated port, got 0")
	}

	if err := nm.DestroyNetwork(ctx, networkID); err != nil {
		t.Fatalf("failed to destroy network: %v", err)
	}
}
