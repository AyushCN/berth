package docker

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/AyushCN/berth/internal/domain"
)

type DockerRuntime struct {
	networkName   string
	traefikDomain string
}

func NewDockerRuntime(dockerHost, networkName, traefikDomain string) (*DockerRuntime, error) {
	// Verify Docker CLI is available
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "json")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker CLI not available: %w", err)
	}

	// Ensure network exists
	if networkName != "" {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd = exec.CommandContext(ctx, "docker", "network", "inspect", networkName)
		if err := cmd.Run(); err != nil {
			// Network doesn't exist, create it
			cmd = exec.CommandContext(ctx, "docker", "network", "create", "--driver", "bridge", networkName)
			if err := cmd.Run(); err != nil {
				return nil, fmt.Errorf("failed to create docker network: %w", err)
			}
			slog.Info("created docker network", "network", networkName)
		}
	}

	return &DockerRuntime{
		networkName:   networkName,
		traefikDomain: traefikDomain,
	}, nil
}

func (d *DockerRuntime) Close() error {
	return nil
}

func (d *DockerRuntime) CreateSandbox(ctx context.Context, spec domain.SandboxSpec) (string, error) {
	// Pull image if not present
	if err := d.ensureImage(ctx, spec.BaseImage); err != nil {
		return "", fmt.Errorf("failed to ensure image: %w", err)
	}

	// Build docker create command
	args := []string{"create"}

	// Name
	args = append(args, "--name", spec.ID.String())

	// Network
	if d.networkName != "" {
		args = append(args, "--network", d.networkName)
	}

	// Labels
	args = append(args, "--label", "berth.sandbox.id="+spec.ID.String())

	// Traefik labels
	if d.traefikDomain != "" && spec.ExposedPort != nil && *spec.ExposedPort > 0 {
		routerName := "sandbox-" + spec.ID.String()
		port := *spec.ExposedPort
		args = append(args,
			"--label", "traefik.enable=true",
			"--label", fmt.Sprintf("traefik.http.routers.%s.rule=Host(`%s.%s`)", routerName, spec.ID.String(), d.traefikDomain),
			"--label", fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port=%d", routerName, port),
		)
	}

	// User-provided labels
	for k, v := range spec.Labels {
		args = append(args, "--label", k+"="+v)
	}

	// Mounts
	args = append(args, "--mount", fmt.Sprintf("type=bind,source=%s,target=%s", spec.WorkspaceDir, spec.WorkDir))
	for hostDir, containerDir := range spec.ExtraMounts {
		args = append(args, "--mount", fmt.Sprintf("type=bind,source=%s,target=%s", hostDir, containerDir))
	}

	// Resources
	args = append(args,
		"--memory", fmt.Sprintf("%db", spec.MemoryLimit),
		"--cpus", fmt.Sprintf("%.3f", float64(spec.CPULimit)/1000.0),
		"--pids-limit", "256",
		"--init",
	)

	// Working directory
	args = append(args, "--workdir", spec.WorkDir)

	// Environment
	for k, v := range spec.Env {
		args = append(args, "-e", k+"="+v)
	}
	args = append(args, "-e", "HOME=/workspace", "-e", "TERM=xterm-256color")

	// Exposed port and port publishing (for direct access when not using Traefik)
	if spec.ExposedPort != nil && *spec.ExposedPort > 0 {
		args = append(args, "--expose", strconv.Itoa(*spec.ExposedPort))
		// Publish port to host for direct access when not using Traefik
		if d.traefikDomain == "" {
			args = append(args, "-p", fmt.Sprintf("127.0.0.1:%d:%d", *spec.ExposedPort, *spec.ExposedPort))
		}
	}

	// TTY and stdin
	args = append(args, "-t", "-i")

	// Image and command
	args = append(args, spec.BaseImage)
	args = append(args, spec.Cmd...)

	// Execute create
	slog.Info("docker create command", "args", strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker create failed: %w, output: %s", err, string(output))
	}

	containerID := strings.TrimSpace(string(output))
	slog.Info("created sandbox container", "sandbox_id", spec.ID, "container_id", containerID[:12], "image", spec.BaseImage)

	return containerID, nil
}

func (d *DockerRuntime) ensureImage(ctx context.Context, image string) error {
	// Check if image exists locally
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", image)
	if err := cmd.Run(); err == nil {
		return nil // Image already present
	}

	// Pull image
	slog.Info("pulling image", "image", image)
	cmd = exec.CommandContext(ctx, "docker", "pull", image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker pull failed: %w, output: %s", err, string(output))
	}

	slog.Info("image pulled successfully", "image", image)
	return nil
}

func (d *DockerRuntime) StartSandbox(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, "docker", "start", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker start failed: %w, output: %s", err, string(output))
	}
	return nil
}

func (d *DockerRuntime) StopSandbox(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, "docker", "stop", "-t", "10", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker stop failed: %w, output: %s", err, string(output))
	}
	return nil
}

func (d *DockerRuntime) DeleteSandbox(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", "-v", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker rm failed: %w, output: %s", err, string(output))
	}
	return nil
}

func (d *DockerRuntime) Exec(ctx context.Context, containerID string, cmdArgs []string) (string, error) {
	return d.ExecWithEnv(ctx, containerID, cmdArgs, nil)
}

func (d *DockerRuntime) ExecWithEnv(ctx context.Context, containerID string, cmdArgs []string, env map[string]string) (string, error) {
	args := []string{"exec", "-w", "/workspace"}
	for k, v := range env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	args = append(args, containerID)
	args = append(args, cmdArgs...)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(output), fmt.Errorf("command exited with code %d", exitErr.ExitCode())
		}
		return string(output), fmt.Errorf("docker exec failed: %w", err)
	}

	return string(output), nil
}

func (d *DockerRuntime) ExecPTY(ctx context.Context, containerID string, cmdArgs []string) (io.WriteCloser, io.Reader, func() error, error) {
	// For PTY, we need interactive mode with a TTY
	// Using docker exec -i -t with a pipe for stdin/stdout
	args := []string{"exec", "-i", "-t", "-w", "/workspace", containerID}
	args = append(args, cmdArgs...)

	cmd := exec.CommandContext(ctx, "docker", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to start docker exec: %w", err)
	}

	// Combine stdout and stderr for PTY mode
	// In PTY mode, Docker merges them anyway
	combinedReader := io.MultiReader(stdout, stderr)

	waitFn := func() error {
		return cmd.Wait()
	}

	return stdin, combinedReader, waitFn, nil
}

func (d *DockerRuntime) GetLogs(ctx context.Context, containerID string, tail int) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "logs", "--tail", strconv.Itoa(tail), containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker logs failed: %w", err)
	}

	return string(output), nil
}

// Ensure DockerRuntime implements domain.ContainerRuntime
var _ domain.ContainerRuntime = (*DockerRuntime)(nil)