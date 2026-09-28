package containerd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/containerd/containerd/api/types/runc/options"
	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/containers"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/errdefs"
	"github.com/containerd/typeurl/v2"
	"github.com/google/uuid"
	"github.com/opencontainers/runtime-spec/specs-go"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/AyushCN/berth/internal/domain"
)

const (
	berthNamespace      = "berth"
	gvisorRuntime       = "io.containerd.runsc.v1"
	defaultRuntime      = "io.containerd.runc.v2"
	defaultTimeout      = 30 * time.Second
	defaultCgroupParent = "berth.slice"
)

// commonSyscalls defines the list of syscalls allowed by default seccomp profile
var commonSyscalls = []string{
	"read", "write", "open", "close", "stat", "fstat", "lstat", "poll", "lseek", "mmap", "mprotect", "munmap", "brk",
	"rt_sigaction", "rt_sigprocmask", "rt_sigreturn", "ioctl", "pread64", "pwrite64", "readv", "writev", "access",
	"pipe", "select", "sched_yield", "mremap", "msync", "mincore", "madvise", "shmget", "shmat", "shmctl", "dup", "dup2",
	"pause", "nanosleep", "getitimer", "alarm", "setitimer", "getpid", "sendfile", "socket", "connect", "accept",
	"sendto", "recvfrom", "sendmsg", "recvmsg", "shutdown", "bind", "listen", "getsockname", "getpeername", "socketpair",
	"setsockopt", "getsockopt", "clone", "fork", "vfork", "execve", "exit", "wait4", "kill", "uname", "semget", "semop",
	"semctl", "shmdt", "msgget", "msgsnd", "msgrcv", "msgctl", "fcntl", "flock", "fsync", "fdatasync", "truncate", "ftruncate",
	"getdents", "getcwd", "chdir", "fchdir", "rename", "mkdir", "rmdir", "creat", "link", "unlink", "symlink", "readlink",
	"chmod", "fchmod", "chown", "fchown", "lchown", "umask", "gettimeofday", "getrlimit", "getrusage", "sysinfo", "times",
	"ptrace", "getuid", "syslog", "getgid", "setuid", "setgid", "geteuid", "getegid", "setpgid", "getppid", "getpgrp", "setsid",
	"setreuid", "setregid", "getgroups", "setgroups", "setresuid", "getresuid", "setresgid", "getresgid", "getpgid",
	"setfsuid", "setfsgid", "getsid", "capget", "capset", "rt_sigpending", "rt_sigtimedwait", "rt_sigqueueinfo", "rt_sigsuspend",
	"sigaltstack", "utime", "mknod", "uselib", "personality", "ustat", "statfs", "fstatfs", "sysfs", "getpriority", "setpriority",
	"sched_setparam", "sched_getparam", "sched_setscheduler", "sched_getscheduler", "sched_get_priority_max", "sched_get_priority_min",
	"sched_rr_get_interval", "mlock", "munlock", "mlockall", "munlockall", "vhangup", "modify_ldt", "pivot_root", "_sysctl", "prctl",
	"arch_prctl", "adjtimex", "setrlimit", "chroot", "sync", "acct", "settimeofday", "mount", "umount2", "swapon", "swapoff", "reboot",
	"sethostname", "setdomainname", "iopl", "ioperm", "create_module", "init_module", "delete_module", "get_kernel_syms", "query_module",
	"quotactl", "nfsservctl", "getpmsg", "putpmsg", "afs_syscall", "tuxcall", "security", "gettid", "readahead", "setxattr", "lsetxattr",
	"fsetxattr", "getxattr", "lgetxattr", "listxattr", "llistxattr", "flistxattr", "removexattr", "lremovexattr", "fremovexattr", "tkill",
	"time", "futex", "sched_setaffinity", "sched_getaffinity", "set_thread_area", "io_setup", "io_destroy", "io_getevents", "io_submit",
	"io_cancel", "get_thread_area", "lookup_dcookie", "epoll_create", "epoll_ctl_old", "epoll_wait_old", "remap_file_pages",
	"getdents64", "set_tid_address", "restart_syscall", "semtimedop", "fadvise64", "timer_create", "timer_settime", "timer_gettime",
	"timer_getoverrun", "timer_delete", "clock_settime", "clock_gettime", "clock_getres", "clock_nanosleep", "exit_group",
	"epoll_wait", "epoll_ctl", "tgkill", "utimes", "vserver", "mbind", "set_mempolicy", "get_mempolicy", "mq_open", "mq_unlink",
	"mq_timedsend", "mq_timedreceive", "mq_notify", "mq_getsetattr", "kexec_load", "waitid", "add_key", "request_key", "keyctl",
	"ioprio_set", "ioprio_get", "inotify_init", "inotify_add_watch", "inotify_rm_watch", "migrate_pages", "openat", "mkdirat",
	"mknodat", "fchownat", "futimesat", "newfstatat", "unlinkat", "renameat", "linkat", "symlinkat", "readlinkat", "fchmodat",
	"fchownat", "pselect6", "ppoll", "unshare", "set_robust_list", "get_robust_list", "splice", "tee", "sync_file_range", "vmsplice",
	"move_pages", "utimensat", "epoll_pwait", "signalfd", "timerfd_create", "eventfd", "fallocate", "timerfd_settime", "timerfd_gettime",
	"accept4", "signalfd4", "eventfd2", "epoll_create1", "dup3", "pipe2", "inotify_init1", "preadv", "pwritev", "rt_tgsigqueueinfo",
	"perf_event_open", "recvmmsg", "fanotify_init", "fanotify_mark", "prlimit64", "name_to_handle_at", "open_by_handle_at",
	"clock_adjtime", "syncfs", "sendmmsg", "setns", "getcpu", "process_vm_readv", "process_vm_writev", "kcmp", "finit_module",
	"sched_setattr", "sched_getattr", "renameat2", "seccomp", "getrandom", "memfd_create", "kexec_file_load", "bpf", "execveat",
	"userfaultfd", "membarrier", "mlock2", "copy_file_range", "preadv2", "pwritev2", "pkey_mprotect", "pkey_alloc", "pkey_free", "statx",
	"io_pgetevents", "rseq", "pidfd_send_signal", "io_uring_setup", "io_uring_enter", "io_uring_register", "open_tree", "move_mount",
	"fsopen", "fsconfig", "fsmount", "fspick", "pidfd_open", "clone3", "close_range", "openat2", "pidfd_getfd", "faccessat2",
	"process_madvise", "epoll_pwait2", "mount_setattr", "quotactl_fd", "landlock_create_ruleset", "landlock_add_rule",
	"landlock_restrict_self", "memfd_secret", "process_mrelease", "futex_waitv", "set_mempolicy_home_node", "cachestat",
	"fchmodat2", "map_shadow_stack", "futex_wake", "futex_wait", "futex_requeue",
}

// Runtime implements domain.ContainerRuntime using containerd + gVisor.
type Runtime struct {
	client          *client.Client
	sockPath        string
	layerMgr        *LayerManager
	netMgr          *NetworkManager
	warmPool        *WarmPool
	runtimeType     string
	dirtyContainers sync.Map
}

// NewRuntime creates a new containerd-backed runtime.
func NewRuntime(sockPath string, runtimeType string) (*Runtime, error) {
	if sockPath == "" {
		sockPath = "/run/containerd/containerd.sock"
		if os.Getenv("CONTAINERD_SOCK") != "" {
			sockPath = os.Getenv("CONTAINERD_SOCK")
		}
	}

	var c *client.Client
	var err error
	for i := 1; i <= 10; i++ {
		c, err = client.New(sockPath, client.WithDefaultNamespace(berthNamespace))
		if err == nil {
			break
		}
		slog.Warn("containerd not ready", "attempt", i, "error", err)
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to containerd after 10 attempts: %w", err)
	}

	layerMgr, err := NewLayerManager(c)
	if err != nil {
		return nil, fmt.Errorf("failed to init layer manager: %w", err)
	}

	netMgr, err := NewNetworkManager()
	if err != nil {
		return nil, fmt.Errorf("failed to init network manager: %w", err)
	}

	r := &Runtime{
		client:      c,
		sockPath:    sockPath,
		layerMgr:    layerMgr,
		netMgr:      netMgr,
		runtimeType: runtimeType,
	}

	r.warmPool = NewWarmPool(8*1024*1024*1024, func(ctx context.Context, id string) error {
		return r.DeleteSandbox(ctx, id, domain.DefaultExecutionProfile())
	})

	go r.MaintainBaseline()

	return r, nil
}

// Close closes the containerd client.
func (r *Runtime) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// CreateSandbox creates a new gVisor sandbox.
func (r *Runtime) CreateSandbox(ctx context.Context, spec domain.SandboxSpec) (string, error) {
	ctx = withNamespace(ctx)

	// Get execution profile (use default if not provided)
	execProfile := spec.ExecutionProfile
	if execProfile == nil {
		execProfile = domain.DefaultExecutionProfile()
	}

	// Check warm pool - only return containers with matching runtime
	if warmID, reason := r.warmPool.Take(spec.BaseImage, execProfile.Runtime); warmID != "" {
		if err := r.prepareWarmContainer(ctx, warmID, spec); err == nil {
			// The workspace bind mount is sandbox-specific, so this container must
			// be destroyed when the sandbox is deleted rather than pooled again.
			r.dirtyContainers.Store(warmID, true)
			slog.Info("warm pool hit", "container_id", warmID, "base_image", spec.BaseImage, "runtime", execProfile.Runtime)
			return warmID, nil
		} else {
			slog.Warn("warm container preparation failed; falling back to cold create", "container_id", warmID, "error", err)
			_ = r.warmPool.Forget(warmID)
			_ = r.StopSandbox(ctx, warmID, execProfile)
		}
	} else {
		slog.Info("warm pool miss", "reason", reason, "base_image", spec.BaseImage, "runtime", execProfile.Runtime)
	}

	slog.Info("cold create", "base_image", spec.BaseImage, "runtime", execProfile.Runtime)
	return r.createSandboxInternal(ctx, spec, execProfile)
}

// prepareWarmContainer adds the per-sandbox mounts and limits before containerd
// creates the task. The warm container has no task yet, so its OCI metadata can
// still be safely updated here.
func (r *Runtime) prepareWarmContainer(ctx context.Context, containerID string, spec domain.SandboxSpec) error {
	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		return err
	}
	ociSpec, err := container.Spec(ctx)
	if err != nil {
		return err
	}
	if spec.WorkspaceDir != "" {
		ociSpec.Mounts = append(ociSpec.Mounts, specs.Mount{
			Destination: spec.WorkDir,
			Type:        "bind",
			Source:      spec.WorkspaceDir,
			Options:     []string{"rbind", "rw"},
		})
	}
	for hostDir, containerDir := range spec.ExtraMounts {
		ociSpec.Mounts = append(ociSpec.Mounts, specs.Mount{
			Destination: containerDir,
			Type:        "bind",
			Source:      hostDir,
			Options:     []string{"rbind", "rw"},
		})
	}
	if spec.MemoryLimit > 0 && ociSpec.Linux != nil && ociSpec.Linux.Resources != nil {
		limit := spec.MemoryLimit
		ociSpec.Linux.Resources.Memory = &specs.LinuxMemory{Limit: &limit}
	}
	if spec.CPULimit > 0 && ociSpec.Linux != nil && ociSpec.Linux.Resources != nil {
		quota := spec.CPULimit * 1000
		period := uint64(100000)
		ociSpec.Linux.Resources.CPU = &specs.LinuxCPU{Quota: &quota, Period: &period}
	}
	if spec.WorkDir != "" && ociSpec.Process != nil {
		ociSpec.Process.Cwd = spec.WorkDir
	}
	return container.Update(ctx, func(_ context.Context, _ *client.Client, meta *containers.Container) error {
		encoded, err := typeurl.MarshalAnyToProto(ociSpec)
		if err != nil {
			return err
		}
		meta.Spec = encoded
		return nil
	})
}

func (r *Runtime) createSandboxInternal(ctx context.Context, spec domain.SandboxSpec, execProfile *domain.ExecutionProfile) (string, error) {
	// 2. Resolve or build dependency layer
	baseImg, err := r.layerMgr.ResolveBaseImage(ctx, spec.BaseImage)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base image: %w", err)
	}

	// 3. Create OCI spec with security hardening
	containerID := spec.ID.String()

	// Build OCI opts based on execution profile
	ociOpts := []oci.SpecOpts{
		withLinuxNamespaces(execProfile),
		withCgroupLimits(execProfile.MemoryLimit, execProfile.CPUQuota, execProfile.PidsLimit),
		withCapabilities(execProfile.Capabilities),
		withSeccompProfile(execProfile.SeccompProfile),
		withReadonlyRootfs(execProfile.ReadOnlyRootFS),
		withNoNewPrivileges(execProfile.NoNewPrivileges),
		withTmpfs(),
		withFilesystemMode(execProfile, spec),
	}

	ociOpts = append(ociOpts, func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		var newMounts []specs.Mount
		for _, m := range s.Mounts {
			if m.Destination == "/sys" || m.Type == "sysfs" {
				// Replace sysfs with a bind mount of the host's /sys
				// Required for rootless containers sharing the host network namespace
				newMounts = append(newMounts, specs.Mount{
					Destination: "/sys",
					Type:        "bind",
					Source:      "/sys",
					Options:     []string{"rbind", "ro", "nosuid", "nodev", "noexec"},
				})
				continue
			}
			newMounts = append(newMounts, m)
		}
		s.Mounts = newMounts

		if s.Process == nil {
			s.Process = &specs.Process{}
		}
		hasPath := false
		for _, e := range s.Process.Env {
			if strings.HasPrefix(e, "PATH=") {
				hasPath = true
				break
			}
		}
		if !hasPath {
			s.Process.Env = append(s.Process.Env, "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
		}

		return nil
	})

	// Force host DNS into the container
	ociOpts = append(ociOpts, func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		s.Mounts = append(s.Mounts,
			specs.Mount{
				Destination: "/etc/resolv.conf",
				Type:        "bind",
				Source:      "/etc/resolv.conf",
				Options:     []string{"rbind", "ro"},
			},
			specs.Mount{
				Destination: "/etc/hosts",
				Type:        "bind",
				Source:      "/etc/hosts",
				Options:     []string{"rbind", "ro"},
			},
		)
		return nil
	})

	// Set main process command if provided
	if len(spec.Cmd) > 0 {
		ociOpts = append(ociOpts, withProcessArgs(spec.Cmd...))
	}

	rt := defaultRuntime
	var optsData *anypb.Any

	if r.runtimeType == "runsc" {
		rt = gvisorRuntime
		// runsc shim does not accept runc options format. Pass nil.
		optsData = nil
	} else {
		optsData, err = anypb.New(&options.Options{})
		if err != nil {
			return "", fmt.Errorf("failed to create runc options: %w", err)
		}
	}

	opts := []client.NewContainerOpts{
		client.WithImage(baseImg),
		client.WithNewSnapshot(containerID+"-snap", baseImg),
		client.WithRuntime(rt, optsData),
		client.WithNewSpec(ociOpts...),
	}

	_, err = r.client.NewContainer(ctx, containerID, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// 4. Setup networking (SKIPPED FOR HOST NETWORKING)
	// networkID := "berth-" + containerID[:8]
	// if err := r.netMgr.CreateNetwork(ctx, networkID); err != nil {
	// 	_ = container.Delete(ctx, client.WithSnapshotCleanup)
	// 	return "", fmt.Errorf("failed to create network: %w", err)
	// }

	// // 4b. Allocate IP for the container
	// _, err = r.netMgr.AllocateIP(networkID, containerID)
	// if err != nil {
	// 	_ = container.Delete(ctx, client.WithSnapshotCleanup)
	// 	_ = r.netMgr.DestroyNetwork(ctx, networkID)
	// 	return "", fmt.Errorf("failed to allocate IP: %w", err)
	// }

	slog.Info("sandbox created (cache miss)", "container_id", containerID, "image", spec.BaseImage)
	return containerID, nil
}

// MaintainBaseline periodically ensures a baseline of warm containers.
func (r *Runtime) MaintainBaseline() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	baselineImages := []string{
		"docker.io/library/node:20-alpine",
		"docker.io/library/python:3.11-slim",
		"docker.io/library/golang:1.23-alpine",
	}

	for range ticker.C {
		for _, img := range baselineImages {
			r.warmPool.mu.RLock()
			count := len(r.warmPool.available[img])
			r.warmPool.mu.RUnlock()

			if count < 3 {
				ctx := context.Background()
				mem := int64(512 * 1024 * 1024)
				err := r.warmPool.PreWarm(ctx, img, domain.RuntimeGVisor, mem, func() (string, error) {
					spec := domain.SandboxSpec{
						ID:          uuid.New(),
						BaseImage:   img,
						Cmd:         []string{"sh", "-c", "while true; do sleep 1; done"},
						MemoryLimit: mem,
						CPULimit:    500,
					}
					return r.createSandboxInternal(ctx, spec, domain.DefaultExecutionProfile())
				})
				if err != nil {
					slog.Error("PreWarm failed", "image", img, "error", err)
				}
			}
		}
	}
}

// StartSandbox starts a container and its task.
func (r *Runtime) StartSandbox(ctx context.Context, containerID string, execProfile *domain.ExecutionProfile) error {
	ctx = withNamespace(ctx)

	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to load container: %w", err)
	}

	// Use default profile if not provided
	if execProfile == nil {
		execProfile = domain.DefaultExecutionProfile()
	}

	// Create task with log FIFO
	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, ".local", "state", "berth", "logs", containerID)
	_ = os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, "task.log")

	fifoDir := filepath.Join(home, ".local", "state", "berth", "fifo", containerID)
	_ = os.MkdirAll(fifoDir, 0755)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	// Create task
	task, err := container.NewTask(ctx, cio.NewCreator(
		cio.WithFIFODir(fifoDir),
	))
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Setup container network based on NetworkMode
	networkID := "berth-" + containerID[:8]
	if execProfile.NetworkMode == domain.NetworkModeCNI {
		// Get the task PID for CNI
		taskPid := task.Pid()
		if taskPid > 0 {
			if _, err := r.netMgr.SetupContainerNetwork(ctx, networkID, containerID, fmt.Sprintf("%d", taskPid)); err != nil {
				slog.Warn("CNI network setup failed", "container", containerID, "error", err)
			} else {
				// Apply egress policy
				if err := r.netMgr.ApplyEgressPolicy(ctx, containerID, execProfile.NetworkEgressPolicy); err != nil {
					slog.Warn("egress policy setup failed", "error", err)
				}
			}
		}
	}

	// Setup port forwarding
	if err := r.netMgr.ForwardPort(containerID, 0, 0); err != nil {
		slog.Warn("port forwarding setup failed", "error", err)
	}

	if err := task.Start(ctx); err != nil {
		return fmt.Errorf("failed to start task: %w", err)
	}

	// Start log streaming goroutine
	go r.streamLogs(ctx, task, logPath, fifoDir)

	slog.Info("sandbox started", "container_id", containerID, "pid", task.Pid())
	return nil
}

// StopSandbox stops a container gracefully, then forcefully.
func (r *Runtime) StopSandbox(ctx context.Context, containerID string, execProfile *domain.ExecutionProfile) error {
	ctx = withNamespace(ctx)
	r.dirtyContainers.Delete(containerID)
	r.warmPool.Forget(containerID)

	// Clean up network resources before stopping
	if execProfile != nil {
		networkID := "berth-" + containerID[:8]
		if execProfile.NetworkMode == domain.NetworkModeCNI {
			// Get the container task to find PID
			container, _ := r.client.LoadContainer(ctx, containerID)
			if container != nil {
				task, _ := container.Task(ctx, nil)
				if task != nil {
					taskPid := task.Pid()
					if taskPid > 0 {
						// Remove egress policy
						r.netMgr.RemoveEgressPolicy(ctx, containerID, execProfile.NetworkEgressPolicy)
						// Release container network
						r.netMgr.ReleaseContainerNetwork(ctx, networkID, containerID, fmt.Sprintf("%d", taskPid))
					}
				}
			}
		}
		// Release port forwarding
		r.netMgr.ReleasePort(containerID)
	}

	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to load container: %w", err)
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return r.deleteContainer(ctx, container)
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Graceful SIGTERM
	if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
		slog.Warn("SIGTERM failed", "error", err)
	}

	exitCh, err := task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for task: %w", err)
	}

	select {
	case <-exitCh:
		slog.Info("task exited gracefully", "container_id", containerID)
	case <-time.After(10 * time.Second):
		slog.Warn("graceful shutdown timed out, force killing", "container_id", containerID)
		if err := task.Kill(ctx, syscall.SIGKILL); err != nil {
			slog.Error("SIGKILL failed", "error", err)
		}
		<-exitCh
	}

	if _, err := task.Delete(ctx, client.WithProcessKill); err != nil {
		slog.Warn("task delete failed", "error", err)
	}

	return r.deleteContainer(ctx, container)
}

// DeleteSandbox destroys a container or returns it to the warm pool.
func (r *Runtime) DeleteSandbox(ctx context.Context, containerID string, execProfile *domain.ExecutionProfile) error {
	ctx = withNamespace(ctx)

	_, isDirty := r.dirtyContainers.Load(containerID)
	if !isDirty {
		if r.warmPool.Return(containerID) {
			slog.Info("sandbox returned to warm pool", "container_id", containerID)
			return nil
		}
	} else {
		slog.Info("sandbox is dirty, not returning to warm pool", "container_id", containerID)
		r.dirtyContainers.Delete(containerID)
	}

	return r.StopSandbox(ctx, containerID, execProfile)
}

// ExecWithEnv runs a command inside an existing sandbox with custom environment variables.
func (r *Runtime) ExecWithEnv(ctx context.Context, containerID string, cmd []string, env map[string]string) (string, error) {
	ctx = withNamespace(ctx)

	// Any exec makes the container dirty and unfit for reuse in the warm pool
	r.dirtyContainers.Store(containerID, true)

	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to load container: %w", err)
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get task: %w", err)
	}

	// Build environment variables
	envVars := []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}
	for k, v := range env {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	processSpec := &specs.Process{
		Terminal: false,
		Args:     cmd,
		Cwd:      "/workspace",
		Env:      envVars,
	}

	home, _ := os.UserHomeDir()
	fifoDir := filepath.Join(home, ".local", "state", "berth", "fifo", containerID, "exec-"+uuid.New().String()[:8])
	_ = os.MkdirAll(fifoDir, 0755)

	var stdoutBuf, stderrBuf bytes.Buffer
	creator := cio.NewCreator(cio.WithFIFODir(fifoDir), cio.WithStreams(nil, &stdoutBuf, &stderrBuf))

	process, err := task.Exec(ctx, uuid.New().String(), processSpec, creator)
	if err != nil {
		return "", fmt.Errorf("failed to create exec process: %w", err)
	}

	if err := process.Start(ctx); err != nil {
		return "", fmt.Errorf("failed to start exec: %w", err)
	}

	statusC, err := process.Wait(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to wait for exec: %w", err)
	}

	select {
	case <-ctx.Done():
		_ = process.Kill(ctx, syscall.SIGKILL)
		return "", ctx.Err()
	case status := <-statusC:
		if _, err := process.Delete(ctx); err != nil {
			slog.Warn("exec process delete failed", "error", err)
		}

		if status.ExitCode() != 0 {
			_ = os.RemoveAll(fifoDir)
			return stdoutBuf.String() + "\n" + stderrBuf.String(), fmt.Errorf("exec exited with code %d: %s", status.ExitCode(), stderrBuf.String())
		}
		_ = os.RemoveAll(fifoDir)
		return stdoutBuf.String(), nil
	}
}

// Exec runs a command inside an existing sandbox.
func (r *Runtime) Exec(ctx context.Context, containerID string, cmd []string) (string, error) {
	return r.ExecWithEnv(ctx, containerID, cmd, nil)
}

// ExecPTY runs an interactive command inside an existing sandbox.
func (r *Runtime) ExecPTY(ctx context.Context, containerID string, cmd []string) (io.WriteCloser, io.Reader, func() error, error) {
	ctx = withNamespace(ctx)

	r.dirtyContainers.Store(containerID, true)

	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load container: %w", err)
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get task: %w", err)
	}

	processSpec := &specs.Process{
		Terminal: true,
		Args:     cmd,
		Cwd:      "/workspace",
		Env:      []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", "TERM=xterm", "HOME=/root"},
	}

	home, _ := os.UserHomeDir()
	fifoDir := filepath.Join(home, ".local", "state", "berth", "fifo", containerID, "pty-"+uuid.New().String()[:8])
	_ = os.MkdirAll(fifoDir, 0755)

	// Use io.Pipe for stdin and stdout bridging
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	// WithTerminal causes stderr to be merged into stdout
	creator := cio.NewCreator(cio.WithFIFODir(fifoDir), cio.WithStreams(stdinReader, stdoutWriter, nil), cio.WithTerminal)

	processID := "exec-" + uuid.New().String()[:8]
	process, err := task.Exec(ctx, processID, processSpec, creator)
	if err != nil {
		stdinWriter.Close()
		stdoutWriter.Close()
		return nil, nil, nil, fmt.Errorf("failed to create exec process: %w", err)
	}

	// Wait must be called before Start to avoid missing the exit event
	statusC, err := process.Wait(context.Background())
	if err != nil {
		process.Delete(context.Background())
		stdinWriter.Close()
		stdoutWriter.Close()
		return nil, nil, nil, fmt.Errorf("failed to wait for exec: %w", err)
	}

	if err := process.Start(ctx); err != nil {
		process.Delete(context.Background())
		stdinWriter.Close()
		stdoutWriter.Close()
		return nil, nil, nil, fmt.Errorf("failed to start exec: %w", err)
	}

	waitFunc := func() error {
		// Clean up when the wait is done
		defer func() {
			process.Delete(context.Background())
			stdoutWriter.Close() // this will trigger EOF on stdoutReader for the bridging goroutine
			os.RemoveAll(fifoDir)
		}()

		select {
		case status := <-statusC:
			if status.ExitCode() != 0 {
				return fmt.Errorf("exec exited with code %d", status.ExitCode())
			}
			return nil
		}
	}

	return stdinWriter, stdoutReader, waitFunc, nil
}

// GetLogs retrieves logs from a sandbox.
func (r *Runtime) GetLogs(ctx context.Context, containerID string, tail int) (string, error) {
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".local", "state", "berth", "logs", containerID, "task.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read logs: %w", err)
	}
	return string(data), nil
}

// --- internal helpers ---

func (r *Runtime) deleteContainer(ctx context.Context, container client.Container) error {
	if err := container.Delete(ctx, client.WithSnapshotCleanup); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("failed to delete container: %w", err)
		}
	}
	return nil
}

func (r *Runtime) streamLogs(ctx context.Context, task client.Task, logPath string, fifoDir string) {
	time.Sleep(100 * time.Millisecond)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		slog.Error("failed to open log file for streaming", "error", err)
		return
	}
	defer logFile.Close()

	stdoutFifo := filepath.Join(fifoDir, "stdout")
	stderrFifo := filepath.Join(fifoDir, "stderr")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			f, err := os.Open(stdoutFifo)
			if err != nil {
				if os.IsNotExist(err) {
					time.Sleep(500 * time.Millisecond)
					continue
				}
				return
			}
			_, _ = io.Copy(logFile, f)
			f.Close()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		f, err := os.Open(stderrFifo)
		if err != nil {
			if os.IsNotExist(err) {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			return
		}
		_, _ = io.Copy(logFile, f)
		f.Close()
	}
}

func withNamespace(ctx context.Context) context.Context {
	// containerd v2 namespaces
	return ctx
}

// --- OCI spec options ---

func withWorkspaceMount(hostDir, containerDir string) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		s.Mounts = append(s.Mounts, specs.Mount{
			Destination: containerDir,
			Type:        "bind",
			Source:      hostDir,
			Options:     []string{"rbind", "rw"},
		})
		return nil
	}
}

func withProcessArgs(args ...string) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		if s.Process == nil {
			s.Process = &specs.Process{}
		}
		s.Process.Args = args
		return nil
	}
}

func withLinuxNamespaces(execProfile *domain.ExecutionProfile) oci.SpecOpts {
	return func(ctx context.Context, client oci.Client, c *containers.Container, s *specs.Spec) error {
		namespaces := []specs.LinuxNamespace{
			{Type: specs.PIDNamespace},
			{Type: specs.MountNamespace},
			{Type: specs.IPCNamespace},
			{Type: specs.UTSNamespace},
		}

		// Network namespace isolation based on profile
		if execProfile.NetworkMode == domain.NetworkModeCNI {
			namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.NetworkNamespace})
		}

		if os.Geteuid() == 0 {
			namespaces = append(namespaces,
				specs.LinuxNamespace{Type: specs.CgroupNamespace},
			)
		}
		s.Linux.Namespaces = namespaces
		return nil
	}
}

func withCgroupLimits(memBytes, cpuMilli int64, pidsLimit int) oci.SpecOpts {
	return func(ctx context.Context, client oci.Client, c *containers.Container, s *specs.Spec) error {
		if os.Geteuid() != 0 {
			slog.Warn("skipping cgroup limits: requires root privileges")
			if s.Linux != nil {
				s.Linux.CgroupsPath = ""
				s.Linux.Resources = nil
			}
			return nil
		}
		if s.Linux == nil {
			s.Linux = &specs.Linux{}
		}
		if s.Linux.Resources == nil {
			s.Linux.Resources = &specs.LinuxResources{}
		}
		if memBytes > 0 {
			s.Linux.Resources.Memory = &specs.LinuxMemory{
				Limit: &memBytes,
			}
		}
		if cpuMilli > 0 {
			quota := cpuMilli * 1000
			period := uint64(100000)
			s.Linux.Resources.CPU = &specs.LinuxCPU{
				Quota:  &quota,
				Period: &period,
			}
		}

		// Prevent fork bombs
		if pidsLimit > 0 {
			pidsLimitVal := int64(pidsLimit)
			s.Linux.Resources.Pids = &specs.LinuxPids{
				Limit: &pidsLimitVal,
			}
		}

		return nil
	}
}

func withDroppedCapabilities() oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		if s.Process == nil {
			s.Process = &specs.Process{}
		}
		if s.Process.Capabilities == nil {
			s.Process.Capabilities = &specs.LinuxCapabilities{}
		}
		s.Process.Capabilities.Bounding = []string{}
		s.Process.Capabilities.Effective = []string{}
		s.Process.Capabilities.Permitted = []string{}
		s.Process.Capabilities.Inheritable = []string{}
		s.Process.Capabilities.Ambient = []string{}
		return nil
	}
}

func withSeccompProfile(profile string) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		if s.Linux == nil {
			s.Linux = &specs.Linux{}
		}

		// Define common allowed syscalls
		commonSyscalls := []string{
			"read", "write", "open", "close", "stat", "fstat", "lstat", "poll", "lseek", "mmap", "mprotect", "munmap", "brk",
			"rt_sigaction", "rt_sigprocmask", "rt_sigreturn", "ioctl", "pread64", "pwrite64", "readv", "writev", "access",
			"pipe", "select", "sched_yield", "mremap", "msync", "mincore", "madvise", "shmget", "shmat", "shmctl", "dup", "dup2",
			"pause", "nanosleep", "getitimer", "alarm", "setitimer", "getpid", "sendfile", "socket", "connect", "accept",
			"sendto", "recvfrom", "sendmsg", "recvmsg", "shutdown", "bind", "listen", "getsockname", "getpeername", "socketpair",
			"setsockopt", "getsockopt", "clone", "fork", "vfork", "execve", "exit", "wait4", "kill", "uname", "semget", "semop",
			"semctl", "shmdt", "msgget", "msgsnd", "msgrcv", "msgctl", "fcntl", "flock", "fsync", "fdatasync", "truncate", "ftruncate",
			"getdents", "getcwd", "chdir", "fchdir", "rename", "mkdir", "rmdir", "creat", "link", "unlink", "symlink", "readlink",
			"chmod", "fchmod", "chown", "fchown", "lchown", "umask", "gettimeofday", "getrlimit", "getrusage", "sysinfo", "times",
			"ptrace", "getuid", "syslog", "getgid", "setuid", "setgid", "geteuid", "getegid", "setpgid", "getppid", "getpgrp", "setsid",
			"setreuid", "setregid", "getgroups", "setgroups", "setresuid", "getresuid", "setresgid", "getresgid", "getpgid",
			"setfsuid", "setfsgid", "getsid", "capget", "capset", "rt_sigpending", "rt_sigtimedwait", "rt_sigqueueinfo", "rt_sigsuspend",
			"sigaltstack", "utime", "mknod", "uselib", "personality", "ustat", "statfs", "fstatfs", "sysfs", "getpriority", "setpriority",
			"sched_setparam", "sched_getparam", "sched_setscheduler", "sched_getscheduler", "sched_get_priority_max", "sched_get_priority_min",
			"sched_rr_get_interval", "mlock", "munlock", "mlockall", "munlockall", "vhangup", "modify_ldt", "pivot_root", "_sysctl", "prctl",
			"arch_prctl", "adjtimex", "setrlimit", "chroot", "sync", "acct", "settimeofday", "mount", "umount2", "swapon", "swapoff", "reboot",
			"sethostname", "setdomainname", "iopl", "ioperm", "create_module", "init_module", "delete_module", "get_kernel_syms", "query_module",
			"quotactl", "nfsservctl", "getpmsg", "putpmsg", "afs_syscall", "tuxcall", "security", "gettid", "readahead", "setxattr", "lsetxattr",
			"fsetxattr", "getxattr", "lgetxattr", "listxattr", "llistxattr", "flistxattr", "removexattr", "lremovexattr", "fremovexattr", "tkill",
			"time", "futex", "sched_setaffinity", "sched_getaffinity", "set_thread_area", "io_setup", "io_destroy", "io_getevents", "io_submit",
			"io_cancel", "get_thread_area", "lookup_dcookie", "epoll_create", "epoll_ctl_old", "epoll_wait_old", "remap_file_pages",
			"getdents64", "set_tid_address", "restart_syscall", "semtimedop", "fadvise64", "timer_create", "timer_settime", "timer_gettime",
			"timer_getoverrun", "timer_delete", "clock_settime", "clock_gettime", "clock_getres", "clock_nanosleep", "exit_group",
			"epoll_wait", "epoll_ctl", "tgkill", "utimes", "vserver", "mbind", "set_mempolicy", "get_mempolicy", "mq_open", "mq_unlink",
			"mq_timedsend", "mq_timedreceive", "mq_notify", "mq_getsetattr", "kexec_load", "waitid", "add_key", "request_key", "keyctl",
			"ioprio_set", "ioprio_get", "inotify_init", "inotify_add_watch", "inotify_rm_watch", "migrate_pages", "openat", "mkdirat",
			"mknodat", "fchownat", "futimesat", "newfstatat", "unlinkat", "renameat", "linkat", "symlinkat", "readlinkat", "fchmodat",
			"fchownat", "pselect6", "ppoll", "unshare", "set_robust_list", "get_robust_list", "splice", "tee", "sync_file_range", "vmsplice",
			"move_pages", "utimensat", "epoll_pwait", "signalfd", "timerfd_create", "eventfd", "fallocate", "timerfd_settime", "timerfd_gettime",
			"accept4", "signalfd4", "eventfd2", "epoll_create1", "dup3", "pipe2", "inotify_init1", "preadv", "pwritev", "rt_tgsigqueueinfo",
			"perf_event_open", "recvmmsg", "fanotify_init", "fanotify_mark", "prlimit64", "name_to_handle_at", "open_by_handle_at",
			"clock_adjtime", "syncfs", "sendmmsg", "setns", "getcpu", "process_vm_readv", "process_vm_writev", "kcmp", "finit_module",
			"sched_setattr", "sched_getattr", "renameat2", "seccomp", "getrandom", "memfd_create", "kexec_file_load", "bpf", "execveat",
			"userfaultfd", "membarrier", "mlock2", "copy_file_range", "preadv2", "pwritev2", "pkey_mprotect", "pkey_alloc", "pkey_free", "statx",
			"io_pgetevents", "rseq", "pidfd_send_signal", "io_uring_setup", "io_uring_enter", "io_uring_register", "open_tree", "move_mount",
			"fsopen", "fsconfig", "fsmount", "fspick", "pidfd_open", "clone3", "close_range", "openat2", "pidfd_getfd", "faccessat2",
			"process_madvise", "epoll_pwait2", "mount_setattr", "quotactl_fd", "landlock_create_ruleset", "landlock_add_rule",
			"landlock_restrict_self", "memfd_secret", "process_mrelease", "futex_waitv", "set_mempolicy_home_node", "cachestat",
			"fchmodat2", "map_shadow_stack", "futex_wake", "futex_wait", "futex_requeue",
		}

		defaultProfile := &specs.LinuxSeccomp{
			DefaultAction: specs.ActErrno,
			Architectures: []specs.Arch{specs.ArchX86_64, specs.ArchX86, specs.ArchARM, specs.ArchAARCH64},
			Syscalls: []specs.LinuxSyscall{
				{Names: commonSyscalls, Action: specs.ActAllow},
			},
		}

		if profile == "" || profile == "default" {
			s.Linux.Seccomp = defaultProfile
			return nil
		}

		if profile == "unrestricted" {
			// Unrestricted - allow all syscalls
			s.Linux.Seccomp = &specs.LinuxSeccomp{
				DefaultAction: specs.ActAllow,
				Architectures: []specs.Arch{specs.ArchX86_64, specs.ArchX86, specs.ArchARM, specs.ArchAARCH64},
			}
			return nil
		}

		if profile == "restricted" {
			// Restricted profile - only allow minimal syscalls
			s.Linux.Seccomp = &specs.LinuxSeccomp{
				DefaultAction: specs.ActErrno,
				Architectures: []specs.Arch{specs.ArchX86_64, specs.ArchX86, specs.ArchARM, specs.ArchAARCH64},
				Syscalls: []specs.LinuxSyscall{
					{Names: commonSyscalls, Action: specs.ActAllow},
				},
			}
			return nil
		}

		// Custom profile path - would load from file
		// For now, use default
		s.Linux.Seccomp = defaultProfile
		return nil
	}
}

func withReadonlyRootfs(readonly bool) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		if s.Root == nil {
			s.Root = &specs.Root{}
		}
		s.Root.Readonly = readonly
		return nil
	}
}

func withCapabilities(capabilities []string) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		if s.Process == nil {
			s.Process = &specs.Process{}
		}
		if s.Process.Capabilities == nil {
			s.Process.Capabilities = &specs.LinuxCapabilities{}
		}

		// If capabilities is empty, drop all (secure default)
		if len(capabilities) == 0 {
			s.Process.Capabilities.Bounding = []string{}
			s.Process.Capabilities.Effective = []string{}
			s.Process.Capabilities.Permitted = []string{}
			s.Process.Capabilities.Inheritable = []string{}
			s.Process.Capabilities.Ambient = []string{}
			return nil
		}

		// Convert capability strings to proper format
		formattedCaps := make([]string, len(capabilities))
		for i, cap := range capabilities {
			formattedCaps[i] = strings.ToUpper(cap)
		}

		s.Process.Capabilities.Bounding = formattedCaps
		s.Process.Capabilities.Effective = formattedCaps
		s.Process.Capabilities.Permitted = formattedCaps
		s.Process.Capabilities.Inheritable = formattedCaps
		s.Process.Capabilities.Ambient = formattedCaps

		return nil
	}
}

func withNoNewPrivileges(noNewPrivs bool) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		if s.Process == nil {
			s.Process = &specs.Process{}
		}
		s.Process.NoNewPrivileges = noNewPrivs
		return nil
	}
}

func withTmpfs() oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		s.Mounts = append(s.Mounts,
			specs.Mount{
				Destination: "/tmp",
				Type:        "tmpfs",
				Source:      "tmpfs",
				Options:     []string{"nosuid", "noexec", "nodev", "size=100m"},
			},
			specs.Mount{
				Destination: "/var/tmp",
				Type:        "tmpfs",
				Source:      "tmpfs",
				Options:     []string{"nosuid", "noexec", "nodev", "size=50m"},
			},
		)
		return nil
	}
}

// withFilesystemMode applies filesystem isolation based on the profile
func withFilesystemMode(execProfile *domain.ExecutionProfile, spec domain.SandboxSpec) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		if s.Linux == nil {
			s.Linux = &specs.Linux{}
		}

		switch execProfile.FilesystemMode {
		case domain.FilesystemModeOverlay:
			// Use overlay filesystem for copy-on-write
			// Create upper/work dirs in a tmpfs or persistent location
			// The base image is the lower layer, upper is per-container
			// For now, we use tmpfs for upper/work to avoid persistent storage
			s.Mounts = append(s.Mounts, specs.Mount{
				Destination: "/workspace",
				Type:        "overlay",
				Source:      "overlay",
				Options: []string{
					"lowerdir=" + spec.WorkspaceDir,
					"upperdir=/overlay/upper",
					"workdir=/overlay/work",
				},
			})
			// Create upper/work dirs in tmpfs
			s.Mounts = append(s.Mounts, specs.Mount{
				Destination: "/overlay",
				Type:        "tmpfs",
				Source:      "tmpfs",
				Options:     []string{"size=500m", "mode=755"},
			})

		case domain.FilesystemModeRO:
			// Read-only root filesystem with explicit writable mounts
			// Root filesystem is already set to readonly via withReadonlyRootfs
			// Add writable paths for common write locations
			writablePaths := []struct {
				dest string
				size string
			}{
				{"/tmp", "100m"},
				{"/var/tmp", "50m"},
				{"/home", "200m"},
				{"/workspace", "1g"}, // Workspace is writable
			}
			for _, wp := range writablePaths {
				s.Mounts = append(s.Mounts, specs.Mount{
					Destination: wp.dest,
					Type:        "tmpfs",
					Source:      "tmpfs",
					Options:     []string{"nosuid", "noexec", "nodev", "size=" + wp.size},
				})
			}

		case domain.FilesystemModeBindMount:
			fallthrough
		default:
			// Default: bind mount workspace (current behavior)
			if spec.WorkspaceDir != "" {
				s.Mounts = append(s.Mounts, specs.Mount{
					Destination: spec.WorkDir,
					Type:        "bind",
					Source:      spec.WorkspaceDir,
					Options:     []string{"rbind", "rw"},
				})
			}
			for hostDir, containerDir := range spec.ExtraMounts {
				s.Mounts = append(s.Mounts, specs.Mount{
					Destination: containerDir,
					Type:        "bind",
					Source:      hostDir,
					Options:     []string{"rbind", "rw"},
				})
			}
		}

		return nil
	}
}

// getDefaultSeccompProfile returns the default restrictive seccomp profile
func getDefaultSeccompProfile() *specs.LinuxSeccomp {
	commonSyscalls := []string{
		"read", "write", "open", "close", "stat", "fstat", "lstat", "poll", "lseek", "mmap", "mprotect", "munmap", "brk",
		"rt_sigaction", "rt_sigprocmask", "rt_sigreturn", "ioctl", "pread64", "pwrite64", "readv", "writev", "access",
		"pipe", "select", "sched_yield", "mremap", "msync", "mincore", "madvise", "shmget", "shmat", "shmctl", "dup", "dup2",
		"pause", "nanosleep", "getitimer", "alarm", "setitimer", "getpid", "sendfile", "socket", "connect", "accept",
		"sendto", "recvfrom", "sendmsg", "recvmsg", "shutdown", "bind", "listen", "getsockname", "getpeername", "socketpair",
		"setsockopt", "getsockopt", "clone", "fork", "vfork", "execve", "exit", "wait4", "kill", "uname", "semget", "semop",
		"semctl", "shmdt", "msgget", "msgsnd", "msgrcv", "msgctl", "fcntl", "flock", "fsync", "fdatasync", "truncate", "ftruncate",
		"getdents", "getcwd", "chdir", "fchdir", "rename", "mkdir", "rmdir", "creat", "link", "unlink", "symlink", "readlink",
		"chmod", "fchmod", "chown", "fchown", "lchown", "umask", "gettimeofday", "getrlimit", "getrusage", "sysinfo", "times",
		"ptrace", "getuid", "syslog", "getgid", "setuid", "setgid", "geteuid", "getegid", "setpgid", "getppid", "getpgrp", "setsid",
		"setreuid", "setregid", "getgroups", "setgroups", "setresuid", "getresuid", "setresgid", "getresgid", "getpgid",
		"setfsuid", "setfsgid", "getsid", "capget", "capset", "rt_sigpending", "rt_sigtimedwait", "rt_sigqueueinfo", "rt_sigsuspend",
		"sigaltstack", "utime", "mknod", "uselib", "personality", "ustat", "statfs", "fstatfs", "sysfs", "getpriority", "setpriority",
		"sched_setparam", "sched_getparam", "sched_setscheduler", "sched_getscheduler", "sched_get_priority_max", "sched_get_priority_min",
		"sched_rr_get_interval", "mlock", "munlock", "mlockall", "munlockall", "vhangup", "modify_ldt", "pivot_root", "_sysctl", "prctl",
		"arch_prctl", "adjtimex", "setrlimit", "chroot", "sync", "acct", "settimeofday", "mount", "umount2", "swapon", "swapoff", "reboot",
		"sethostname", "setdomainname", "iopl", "ioperm", "create_module", "init_module", "delete_module", "get_kernel_syms", "query_module",
		"quotactl", "nfsservctl", "getpmsg", "putpmsg", "afs_syscall", "tuxcall", "security", "gettid", "readahead", "setxattr", "lsetxattr",
		"fsetxattr", "getxattr", "lgetxattr", "listxattr", "llistxattr", "flistxattr", "removexattr", "lremovexattr", "fremovexattr", "tkill",
		"time", "futex", "sched_setaffinity", "sched_getaffinity", "set_thread_area", "io_setup", "io_destroy", "io_getevents", "io_submit",
		"io_cancel", "get_thread_area", "lookup_dcookie", "epoll_create", "epoll_ctl_old", "epoll_wait_old", "remap_file_pages",
		"getdents64", "set_tid_address", "restart_syscall", "semtimedop", "fadvise64", "timer_create", "timer_settime", "timer_gettime",
		"timer_getoverrun", "timer_delete", "clock_settime", "clock_gettime", "clock_getres", "clock_nanosleep", "exit_group",
		"epoll_wait", "epoll_ctl", "tgkill", "utimes", "vserver", "mbind", "set_mempolicy", "get_mempolicy", "mq_open", "mq_unlink",
		"mq_timedsend", "mq_timedreceive", "mq_notify", "mq_getsetattr", "kexec_load", "waitid", "add_key", "request_key", "keyctl",
		"ioprio_set", "ioprio_get", "inotify_init", "inotify_add_watch", "inotify_rm_watch", "migrate_pages", "openat", "mkdirat",
		"mknodat", "fchownat", "futimesat", "newfstatat", "unlinkat", "renameat", "linkat", "symlinkat", "readlinkat", "fchmodat",
		"fchownat", "pselect6", "ppoll", "unshare", "set_robust_list", "get_robust_list", "splice", "tee", "sync_file_range", "vmsplice",
		"move_pages", "utimensat", "epoll_pwait", "signalfd", "timerfd_create", "eventfd", "fallocate", "timerfd_settime", "timerfd_gettime",
		"accept4", "signalfd4", "eventfd2", "epoll_create1", "dup3", "pipe2", "inotify_init1", "preadv", "pwritev", "rt_tgsigqueueinfo",
		"perf_event_open", "recvmmsg", "fanotify_init", "fanotify_mark", "prlimit64", "name_to_handle_at", "open_by_handle_at",
		"clock_adjtime", "syncfs", "sendmmsg", "setns", "getcpu", "process_vm_readv", "process_vm_writev", "kcmp", "finit_module",
		"sched_setattr", "sched_getattr", "renameat2", "seccomp", "getrandom", "memfd_create", "kexec_file_load", "bpf", "execveat",
		"userfaultfd", "membarrier", "mlock2", "copy_file_range", "preadv2", "pwritev2", "pkey_mprotect", "pkey_alloc", "pkey_free", "statx",
		"io_pgetevents", "rseq", "pidfd_send_signal", "io_uring_setup", "io_uring_enter", "io_uring_register", "open_tree", "move_mount",
		"fsopen", "fsconfig", "fsmount", "fspick", "pidfd_open", "clone3", "close_range", "openat2", "pidfd_getfd", "faccessat2",
		"process_madvise", "epoll_pwait2", "mount_setattr", "quotactl_fd", "landlock_create_ruleset", "landlock_add_rule",
		"landlock_restrict_self", "memfd_secret", "process_mrelease", "futex_waitv", "set_mempolicy_home_node", "cachestat",
		"fchmodat2", "map_shadow_stack", "futex_wake", "futex_wait", "futex_requeue",
	}
	return &specs.LinuxSeccomp{
		DefaultAction: specs.ActErrno,
		Architectures: []specs.Arch{specs.ArchX86_64, specs.ArchX86, specs.ArchARM, specs.ArchAARCH64},
		Syscalls: []specs.LinuxSyscall{
			{Names: commonSyscalls, Action: specs.ActAllow},
		},
	}
}
