package containerd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/platforms"
	"github.com/containerd/errdefs"
	"github.com/google/uuid"
	"github.com/opencontainers/runtime-spec/specs-go"
	"syscall"

	"github.com/AyushCN/berth/internal/domain"
)

// Constants are defined in runtime.go
// const (
// 	berthNamespace = "berth"
// 	gvisorRuntime  = "io.containerd.runsc.v1"
// )

type LayerManager struct {
	client   *client.Client
	layerDir string
	cacheDir string
}

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

func (lm *LayerManager) ResolveBaseImage(ctx context.Context, ref string) (client.Image, error) {
	ctx = namespaces.WithNamespace(ctx, berthNamespace)

	img, err := lm.client.GetImage(ctx, ref)
	if err != nil {
		if !errdefs.IsNotFound(err) {
			return nil, fmt.Errorf("failed to check image: %w", err)
		}
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

func (lm *LayerManager) BuildDependencyLayer(ctx context.Context, baseImage string, profile domain.RuntimeProfile) (string, error) {
	ctx = namespaces.WithNamespace(ctx, berthNamespace)

	cacheRef := fmt.Sprintf("berth-deps:%s-%s", profile.Language, hashString(baseImage+profile.InstallCmd))
	if _, err := lm.client.GetImage(ctx, cacheRef); err == nil {
		slog.Info("dependency layer cache hit", "ref", cacheRef)
		return cacheRef, nil
	}

	baseImg, err := lm.ResolveBaseImage(ctx, baseImage)
	if err != nil {
		return "", err
	}

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

	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return "", fmt.Errorf("failed to create build task: %w", err)
	}

	if err := task.Start(ctx); err != nil {
		task.Delete(ctx, client.WithProcessKill)
		return "", fmt.Errorf("failed to start build task: %w", err)
	}

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

	_ = task.Kill(ctx, syscall.SIGTERM)
	exitCh, _ := task.Wait(ctx)
	<-exitCh
	task.Delete(ctx, client.WithProcessKill)


	newImage := images.Image{
		Name: cacheRef,
		Labels: map[string]string{
			"berth.layer":      "deps",
			"berth.language":   profile.Language,
			"berth.base_image": baseImage,
		},
	}

	img, err := lm.client.ImageService().Create(ctx, newImage)
	if err != nil {
		return "", fmt.Errorf("failed to create cached image: %w", err)
	}

	slog.Info("dependency layer built and cached", "ref", img.Name, "language", profile.Language)
	return img.Name, nil
}

func (lm *LayerManager) ComposeLayers(baseRef, depsRef, sandboxID string) (string, error) {
	mergedDir := filepath.Join(lm.layerDir, sandboxID, "merged")
	upperDir := filepath.Join(lm.layerDir, sandboxID, "upper")
	workDir := filepath.Join(lm.layerDir, sandboxID, "work")

	for _, dir := range []string{mergedDir, upperDir, workDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create overlay dir: %w", err)
		}
	}

	_ = baseRef
	_ = depsRef
	return mergedDir, nil
}

func (lm *LayerManager) SnapshotLayer(path string) (string, error) {
	outPath := filepath.Join(lm.cacheDir, filepath.Base(path)+".tar.gz")
	f, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("failed to create tar.gz: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	_ = filepath.Walk(path, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}

		header.Name = filepath.ToSlash(strings.TrimPrefix(file, path))
		if fi.IsDir() {
			header.Name += "/"
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			defer data.Close()
			if _, err := io.Copy(tw, data); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	if err := tw.Close(); err != nil {
		return "", err
	}
	if err := gw.Close(); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}

	return outPath, nil
}

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