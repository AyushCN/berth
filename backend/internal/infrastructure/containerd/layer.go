package containerd

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/platforms"
	"github.com/containerd/errdefs"
	"github.com/google/uuid"
	"github.com/opencontainers/runtime-spec/specs-go"
	"syscall"

	"github.com/AyushCN/berth/internal/domain"
)

// LayerManager handles base images, dependency layers, and overlayfs composition.
type LayerManager struct {
	client   *client.Client
	layerDir string
	cacheDir string
}

// NewLayerManager creates a layer manager backed by containerd.
func NewLayerManager(c *client.Client) (*LayerManager, error) {
	home, _ := os.UserHomeDir()
	layerDir := filepath.Join(home, ".local", "state", "berth", "layers")
	cacheDir := filepath.Join(home, ".local", "state", "berth", "cache")

	for _, dir := range []string{layerDir, cacheDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create layer dir %s: %w", dir, err)
		}
	}

	return &LayerManager{
		client:   c,
		layerDir: layerDir,
		cacheDir: cacheDir,
	}, nil
}

// ResolveBaseImage pulls an image if not present and returns it.
func (lm *LayerManager) ResolveBaseImage(ctx context.Context, ref string) (client.Image, error) {
	ctx = namespaces.WithNamespace(ctx, berthNamespace)

	img, err := lm.client.GetImage(ctx, ref)
	if err != nil {
		if !errdefs.IsNotFound(err) {
			return nil, fmt.Errorf("failed to check image: %w", err)
		}
		// Pull image
		slog.Info("pulling image", "ref", ref)
		img, err = lm.client.Pull(ctx, ref,
			client.WithPullUnpack,
			client.WithPlatform(platforms.DefaultString()),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to pull image %s: %w", ref, err)
		}
	}

	return img, nil
}

// BuildDependencyLayer creates a reusable layer with pre-installed dependencies.
// It creates a temp container from baseImage, runs the install command, and
// commits the resulting filesystem as a new image reference.
//
// TODO(Phase 2): Currently, this builds the layer but does not commit the snapshot
// as a new cached image. The cache hit check at the top will always miss until
// Phase 2 implements proper snapshot export/import. For Phase 1, dependency
// installation happens on every cold start, but the warm pool mitigates this.
func (lm *LayerManager) BuildDependencyLayer(ctx context.Context, baseImage string, profile domain.RuntimeProfile) (string, error) {
	ctx = namespaces.WithNamespace(ctx, berthNamespace)

	// Check cache first
	cacheRef := fmt.Sprintf("berth-deps:%s-%s", profile.Language, hashString(baseImage+profile.InstallCmd))
	if _, err := lm.client.GetImage(ctx, cacheRef); err == nil {
		slog.Info("dependency layer cache hit", "ref", cacheRef)
		return cacheRef, nil
	}

	// Pull base image
	baseImg, err := lm.ResolveBaseImage(ctx, baseImage)
	if err != nil {
		return "", err
	}

	// Create temp container
	tempID := "build-" + uuid.New().String()
	container, err := lm.client.NewContainer(ctx, tempID,
		client.WithImage(baseImg),
		client.WithNewSnapshot(tempID+"-snap", baseImg),
		client.WithRuntime(gvisorRuntime, nil),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create build container: %w", err)
	}
	defer func() {
		_ = container.Delete(ctx, client.WithSnapshotCleanup)
	}()

	// Create task and run install
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return "", fmt.Errorf("failed to create build task: %w", err)
	}

	if err := task.Start(ctx); err != nil {
		task.Delete(ctx, client.WithProcessKill)
		return "", fmt.Errorf("failed to start build task: %w", err)
	}

	// For install, we exec into the running task
	processSpec := &specs.Process{
		Terminal: false,
		Args:     []string{"sh", "-c", profile.InstallCmd},
		Cwd:      profile.WorkDir,
	}

	process, err := task.Exec(ctx, "install", processSpec, cio.NewCreator(cio.WithStdio))
	if err != nil {
		_ = task.Kill(ctx, syscall.SIGKILL)
		task.Delete(ctx, client.WithProcessKill)
		return "", fmt.Errorf("failed to exec install: %w", err)
	}

	if err := process.Start(ctx); err != nil {
		_ = task.Kill(ctx, syscall.SIGKILL)
		task.Delete(ctx, client.WithProcessKill)
		return "", fmt.Errorf("failed to start install: %w", err)
	}

	statusC, _ := process.Wait(ctx)
	status := <-statusC
	if status.ExitCode() != 0 {
		_ = task.Kill(ctx, syscall.SIGKILL)
		task.Delete(ctx, client.WithProcessKill)
		return "", fmt.Errorf("install command failed with exit code %d", status.ExitCode())
	}

	process.Delete(ctx)

	// Stop the task
	_ = task.Kill(ctx, syscall.SIGTERM)
	exitCh, _ := task.Wait(ctx)
	<-exitCh
	task.Delete(ctx, client.WithProcessKill)

	// TODO(Phase 2): Commit the container's snapshot as a new image.
	// In containerd v2, this requires:
	//  1. Get the container's snapshot info via container.Info(ctx)
	//  2. Export the snapshot filesystem using client.DiffService().Diff()
	//  3. Build an OCI image config + manifest
	//  4. Write to content store via client.ContentStore()
	//  5. Create image reference via client.ImageService().Create()
	// Until then, the cache hit check above will always miss on first use,
	// but the warm pool (runtime.go) mitigates repeated cold starts.

	slog.Info("dependency layer built", "ref", cacheRef, "language", profile.Language,
		"note", "snapshot commit deferred to Phase 2")
	return cacheRef, nil
}

// ComposeLayers mounts an overlayfs from base + deps + writable upper.
// Returns the merged mount point path.
func (lm *LayerManager) ComposeLayers(baseRef, depsRef, sandboxID string) (string, error) {
	mergedDir := filepath.Join(lm.layerDir, sandboxID, "merged")
	upperDir := filepath.Join(lm.layerDir, sandboxID, "upper")
	workDir := filepath.Join(lm.layerDir, sandboxID, "work")

	for _, dir := range []string{mergedDir, upperDir, workDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create overlay dir: %w", err)
		}
	}

	// Phase 1: containerd handles overlayfs internally via snapshotter.
	// We return the merged path for documentation; actual mount is managed by containerd.
	_ = baseRef
	_ = depsRef
	return mergedDir, nil
}

// SnapshotLayer exports a directory as a reusable layer tarball.
func (lm *LayerManager) SnapshotLayer(path string) (string, error) {
	outPath := filepath.Join(lm.cacheDir, filepath.Base(path)+".tar.gz")
	// TODO: implement tar export
	_ = outPath
	return outPath, nil
}

// CleanupSandbox removes the per-sandbox overlay directories.
func (lm *LayerManager) CleanupSandbox(sandboxID string) error {
	sandboxDir := filepath.Join(lm.layerDir, sandboxID)
	if err := os.RemoveAll(sandboxDir); err != nil {
		return fmt.Errorf("failed to cleanup sandbox layers: %w", err)
	}
	return nil
}

func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))[:12]
}
