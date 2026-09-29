package worker

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// provisionRequest is everything the provisioning pipeline needs, independent
// of whether it is provisioning a legacy sandbox or an environment. Keeping it
// free of domain types is what lets both callers share one implementation.
type provisionRequest struct {
	// ID drives the container name, the committed image tag and the preview
	// hostname, so it is the identity the container will be known by.
	ID uuid.UUID

	// WorkspaceDir is the host directory bind-mounted at WorkDir.
	WorkspaceDir string
	WorkDir      string

	GitURL    string
	GitBranch string
	OwnerID   uuid.UUID

	MemoryLimit int64
	CPULimit    int64
	Env         map[string]string
}

type provisionResult struct {
	ContainerID string
	Port        int
	PublicURL   string
	Profile     *domain.RuntimeProfile
	Timings     map[string]time.Duration
	// Ready reports whether the application accepted a connection inside the
	// readiness window. False is a real outcome for a repository with no
	// server to run, not a provisioning failure, so the container is still
	// returned and the caller decides which state to record.
	Ready        bool
	ReadinessErr string
}

// validateGitURL restricts clones to https github.com URLs and rejects
// anything that could be read as a git flag or escape the path.
func validateGitURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("git url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("only https scheme allowed, got %s", u.Scheme)
	}
	if u.Host != "github.com" {
		return fmt.Errorf("only github.com allowed, got %s", u.Host)
	}
	if strings.Contains(u.Path, "..") {
		return fmt.Errorf("path traversal detected")
	}
	if strings.HasPrefix(u.Path, "-") {
		return fmt.Errorf("path looks like a flag")
	}
	return nil
}

// validateGitBranch rejects branch names that git would read as an option.
func validateGitBranch(branch string) error {
	if strings.HasPrefix(branch, "-") {
		return fmt.Errorf("branch looks like a flag: %q", branch)
	}
	return nil
}

// gitAuthArgs returns -c arguments that authenticate to github.com without
// ever writing the token to disk.
//
// The previous implementation rewrote the clone URL to
// https://<token>@github.com/... and then cloned --bare into a *shared* cache
// directory. That persisted the plaintext token into <cache>/config, where it
// outlived the sandbox and was readable by every other sandbox using the same
// repo. Passing an Authorization header on the command line keeps it in the
// process arguments only, and git does not write -c values to any config file.
func gitAuthArgs(token string) []string {
	if token == "" {
		return nil
	}
	auth := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token))
	return []string{
		"-c", "credential.helper=",
		"-c", "http.https://github.com/.extraHeader=AUTHORIZATION: basic " + auth,
	}
}

// previewURL builds the advertised preview URL for a provisioned container.
//
// Traefik routes on Host(`<id>.<TRAEFIK_DOMAIN>`); when no domain is
// configured the api's own path proxy (/p/<id>/) is used instead.
func previewURL(id uuid.UUID) string {
	host := strings.TrimSuffix(os.Getenv("API_PUBLIC_HOST"), "/")
	if host == "" {
		host = "http://localhost:8080"
	}
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		scheme, bare, _ := strings.Cut(host, "://")
		return fmt.Sprintf("%s://%s.%s/", scheme, id, bare)
	}
	return fmt.Sprintf("%s/p/%s/", host, id)
}

// provision runs the full pipeline: clone (with a shared bare cache), detect
// the stack, create a keepalive container, install dependencies, commit the
// result to an image, then recreate from that image with the file watcher.
//
// On any error it removes the workspace directory and the container it
// created, so a failed provision leaves nothing behind.
func (w *Worker) provision(ctx context.Context, req provisionRequest) (_ *provisionResult, err error) {
	totalStart := time.Now()

	if err := validateGitURL(req.GitURL); err != nil {
		return nil, fmt.Errorf("invalid git url: %w", err)
	}
	if err := validateGitBranch(req.GitBranch); err != nil {
		return nil, fmt.Errorf("invalid git branch: %w", err)
	}

	gitToken, err := w.resolveGitToken(ctx, req.OwnerID)
	if err != nil {
		slog.Warn("could not resolve git token, falling back to unauthenticated clone",
			"id", req.ID, "error", err)
	}

	timings := map[string]time.Duration{}

	// Clone, then clean up on any later failure.
	var containerID string
	success := false
	defer func() {
		if success {
			return
		}
		_ = os.RemoveAll(req.WorkspaceDir)
		if containerID != "" {
			go func(id string) {
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				_ = w.runtime.Remove(cleanupCtx, id)
			}(containerID)
		}
	}()

	bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// --- clone -------------------------------------------------------------
	cloneStart := time.Now()
	branchPart := req.GitBranch
	if branchPart == "" {
		branchPart = "HEAD"
	}
	sum := sha256.Sum256([]byte(req.GitURL + "@" + branchPart))
	cacheKey := fmt.Sprintf("%x", sum)[:16]
	cacheDir := filepath.Join(gitCacheRoot(), cacheKey)

	if _, statErr := os.Stat(cacheDir); os.IsNotExist(statErr) {
		slog.Info("git cache miss, cold clone", "id", req.ID)
		if err := os.MkdirAll(filepath.Dir(cacheDir), 0755); err != nil {
			return nil, fmt.Errorf("failed to create git cache dir: %w", err)
		}
		cloneArgs := gitAuthArgs(gitToken)
		cloneArgs = append(cloneArgs, "clone", "--bare")
		if req.GitBranch != "" {
			cloneArgs = append(cloneArgs, "-b", req.GitBranch)
		}
		cloneArgs = append(cloneArgs, req.GitURL, cacheDir)
		if out, cmdErr := exec.CommandContext(bgCtx, "git", cloneArgs...).CombinedOutput(); cmdErr != nil {
			return nil, fmt.Errorf("git cold clone failed: %w: %s", cmdErr, truncate(string(out), 500))
		}
	} else {
		slog.Info("git cache hit", "id", req.ID)
	}

	_ = os.RemoveAll(req.WorkspaceDir)
	localArgs := []string{"clone", "--local", "--shared", cacheDir, req.WorkspaceDir}
	if out, cmdErr := exec.CommandContext(bgCtx, "git", localArgs...).CombinedOutput(); cmdErr != nil {
		return nil, fmt.Errorf("git local clone failed: %w: %s", cmdErr, truncate(string(out), 500))
	}
	entries, err := os.ReadDir(req.WorkspaceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace after clone: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("workspace is empty after clone")
	}
	timings["clone"] = time.Since(cloneStart)

	// --- detect ------------------------------------------------------------
	analyzeStart := time.Now()
	profile := analyzer.Analyze(req.WorkspaceDir)
	timings["analyze"] = time.Since(analyzeStart)
	slog.Info("static analysis completed", "id", req.ID, "language", profile.Language)

	// --- port --------------------------------------------------------------
	port, err := getFreePort()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate port: %w", err)
	}
	containerEnv := map[string]string{"PORT": strconv.Itoa(port)}
	for k, v := range req.Env {
		containerEnv[k] = v
	}

	memoryLimit := req.MemoryLimit
	if memoryLimit == 0 {
		memoryLimit = 512 * 1024 * 1024
	}
	// ContainerSpec.CPULimit is milli-cores (the docker runtime divides by
	// 1000 to get a CPU count). Callers holding nanocpus must convert with
	// cpuMilliCores first, otherwise 1e9 nanocores becomes 1,000,000 CPUs and
	// docker refuses with "range of CPUs is from 0.01 to 12.00".
	cpuLimit := req.CPULimit
	if cpuLimit <= 0 {
		cpuLimit = defaultCPUMilliCores
	}
	execProfile := domain.DefaultExecutionProfile()

	spec := func(baseImage string, cmd []string) domain.ContainerSpec {
		return domain.ContainerSpec{
			ID:               req.ID,
			BaseImage:        baseImage,
			WorkDir:          req.WorkDir,
			WorkspaceDir:     req.WorkspaceDir,
			Cmd:              cmd,
			MemoryLimit:      memoryLimit,
			CPULimit:         cpuLimit,
			ExposedPort:      &port,
			Env:              containerEnv,
			ExecutionProfile: execProfile,
			Labels:           map[string]string{"berth.language": profile.Language},
		}
	}

	// --- phase 1: keepalive container, so dependencies can be installed ----
	createStart := time.Now()
	containerID, err = w.runtime.Create(bgCtx, spec(profile.BaseImage, []string{"sleep", "infinity"}))
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}
	timings["create"] = time.Since(createStart)

	startStart := time.Now()
	if err := w.runtime.Start(bgCtx, containerID); err != nil {
		return nil, fmt.Errorf("failed to start keepalive container: %w", err)
	}
	timings["start"] = time.Since(startStart)

	// --- phase 2: install dependencies in the running container ------------
	if profile.InstallCmd != "" {
		installStart := time.Now()
		args := []string{"sh", "-c", "cd " + req.WorkDir + " && " + profile.InstallCmd}
		if out, err := w.runtime.Exec(bgCtx, containerID, args); err != nil {
			return nil, fmt.Errorf("dependency install failed: %w: %s", err, truncate(out, 500))
		}
		timings["install"] = time.Since(installStart)
	}

	// --- phase 3: commit the installed state to an image -------------------
	if err := w.runtime.Stop(bgCtx, containerID); err != nil {
		return nil, fmt.Errorf("failed to stop container after install: %w", err)
	}
	image := fmt.Sprintf("berth-%s:latest", req.ID.String()[:8])
	commitStart := time.Now()
	if err := w.runtime.CommitContainer(bgCtx, containerID, image); err != nil {
		return nil, fmt.Errorf("failed to commit container: %w", err)
	}
	timings["commit"] = time.Since(commitStart)
	if err := w.runtime.Remove(bgCtx, containerID); err != nil {
		slog.Warn("failed to delete keepalive container", "id", req.ID, "error", err)
	}

	// --- phase 4: recreate from the image with the watcher -----------------
	createStart = time.Now()
	containerID, err = w.runtime.Create(bgCtx, spec(image, WatcherCommand(profile)))
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher container: %w", err)
	}
	timings["create_watcher"] = time.Since(createStart)

	startStart = time.Now()
	if err := w.runtime.Start(bgCtx, containerID); err != nil {
		return nil, fmt.Errorf("failed to start watcher container: %w", err)
	}
	timings["start_watcher"] = time.Since(startStart)

	// --- readiness --------------------------------------------------------
	// The container is started but the application inside it is not listening
	// yet, and a repository with no server never will be. Without this the
	// environment is reported RUNNING with nothing on its port, which is how a
	// static site or a broken start command looked identical to a working one.
	ready, readyErr := waitForListener(ctx, port, readinessTimeout())

	// total is recorded whether or not the app came up, so the timings still
	// describe the work that was actually done.
	timings["total"] = time.Since(totalStart)
	success = true

	return &provisionResult{
		ContainerID:  containerID,
		Port:         port,
		PublicURL:    previewURL(req.ID),
		Profile:      profile,
		Timings:      timings,
		Ready:        ready,
		ReadinessErr: readyErr,
	}, nil
}

// readinessTimeout is how long to wait for the application to start listening.
func readinessTimeout() time.Duration {
	if v := os.Getenv("READINESS_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
		slog.Warn("ignoring invalid READINESS_TIMEOUT", "value", v)
	}
	return 60 * time.Second
}

// waitForListener polls until something accepts a TCP connection on port, or the
// timeout expires.
//
// Containers run with host networking, so the application's port is bound
// directly on the host and can be dialled from here. A TCP accept is used
// rather than an HTTP request because the app may not speak HTTP, and a
// connection refused is the signal we care about, not the response.
func waitForListener(ctx context.Context, port int, timeout time.Duration) (bool, string) {
	deadline := time.Now().Add(timeout)
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))

	for {
		dialCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		conn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", addr)
		cancel()
		if err == nil {
			_ = conn.Close()
			return true, ""
		}
		if time.Now().After(deadline) {
			return false, fmt.Sprintf(
				"the application did not start listening on port %d within %s: %v", port, timeout, err)
		}
		select {
		case <-ctx.Done():
			return false, "cancelled while waiting for the application to start"
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// truncate keeps log output and error messages bounded.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
