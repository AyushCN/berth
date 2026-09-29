package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

type FileUsecase struct {
	workspaceDir string
	envRepo      domain.EnvironmentRepository
	wsRepo       domain.WorkspaceRepository
	projRepo     domain.ProjectRepository
	envUC        *EnvironmentUsecase // optional, used to signal reload via Exec
}

func NewFileUsecase(dir string, envRepo domain.EnvironmentRepository, wsRepo domain.WorkspaceRepository, projRepo domain.ProjectRepository, envUC *EnvironmentUsecase) *FileUsecase {
	return &FileUsecase{workspaceDir: dir, envRepo: envRepo, wsRepo: wsRepo, projRepo: projRepo, envUC: envUC}
}

// authorize is called before every file operation. The endpoints previously
// performed no ownership check at all, so any authenticated user could read,
// overwrite or delete any environment's files by guessing its UUID.
func (uc *FileUsecase) authorize(ctx context.Context, environmentID, userID uuid.UUID, want accessLevel) error {
	return requireAccess(ctx, uc.envRepo, uc.wsRepo, uc.projRepo, environmentID, userID, want, "file operation")
}

// workspaceDirFor resolves the host checkout directory for an environment.
// The directory is keyed by workspace id, which is what the worker provisions;
// keying it by environment id only coincided for rows backfilled by migration
// 000006 and pointed at a nonexistent directory for anything created through
// the api.
func (uc *FileUsecase) workspaceDirFor(ctx context.Context, environmentID uuid.UUID) (string, error) {
	wsID, err := workspaceIDFor(ctx, uc.envRepo, environmentID)
	if err != nil {
		return "", err
	}
	return filepath.Join(uc.workspaceDir, wsID.String()), nil
}

func (uc *FileUsecase) resolvePath(ctx context.Context, environmentID uuid.UUID, reqPath string) (string, error) {
	baseDir, err := uc.workspaceDirFor(ctx, environmentID)
	if err != nil {
		return "", err
	}
	cleanPath := filepath.Clean(filepath.Join(baseDir, reqPath))
	if !strings.HasPrefix(cleanPath, baseDir) {
		return "", fmt.Errorf("path traversal denied")
	}
	return cleanPath, nil
}

type FileInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

func (uc *FileUsecase) ListFiles(ctx context.Context, environmentID, userID uuid.UUID, reqPath string) (any, error) {
	if err := uc.authorize(ctx, environmentID, userID, accessRead); err != nil {
		return nil, err
	}
	baseDir, err := uc.workspaceDirFor(ctx, environmentID)
	if err != nil {
		return nil, err
	}
	target, err := uc.resolvePath(ctx, environmentID, reqPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		if os.IsNotExist(err) {
			return []FileInfo{}, nil
		}
		return nil, err
	}

	var result []FileInfo
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		// Paths are relative to the workspace root, not to the directory being
		// listed, so the client can navigate without re-joining prefixes.
		relPath, _ := filepath.Rel(baseDir, filepath.Join(target, e.Name()))
		result = append(result, FileInfo{
			Name:    e.Name(),
			Path:    relPath,
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	return result, nil
}

func (uc *FileUsecase) GetFileContent(ctx context.Context, environmentID, userID uuid.UUID, path string) ([]byte, error) {
	if err := uc.authorize(ctx, environmentID, userID, accessRead); err != nil {
		return nil, err
	}
	target, err := uc.resolvePath(ctx, environmentID, path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(target)
}

// SaveResult is returned by UpdateFileContent.
type SaveResult struct {
	ReloadSignaled bool
}

func (uc *FileUsecase) UpdateFileContent(ctx context.Context, environmentID, userID uuid.UUID, path string, content []byte) (*SaveResult, error) {
	if err := uc.authorize(ctx, environmentID, userID, accessWrite); err != nil {
		return nil, err
	}
	target, err := uc.resolvePath(ctx, environmentID, path)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	if err := os.WriteFile(target, content, 0644); err != nil {
		return nil, err
	}

	// Signal a hot-reload by touching the file inside the container via exec
	reloadSignaled := false
	if uc.envUC != nil && uc.envUC.runtime != nil {
		env, err := uc.envUC.envRepo.GetByID(ctx, environmentID)
		if err == nil && env.State == domain.EnvironmentStateRunning && env.ContainerID != "" {
			inContainerPath := "/workspace/" + strings.TrimPrefix(path, "/")
			touchCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, touchErr := uc.envUC.runtime.Exec(touchCtx, env.ContainerID, []string{"touch", inContainerPath})
			if touchErr == nil {
				reloadSignaled = true
			} else {
				slog.Warn("touch-on-save failed", "environment_id", environmentID, "path", inContainerPath, "err", touchErr)
			}
		}
	}

	// Async: stage file in git and update git tracking in DB
	sandboxDir, dirErr := uc.workspaceDirFor(ctx, environmentID)
	if dirErr != nil {
		return nil, dirErr
	}
	go func() {
		backgroundCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(backgroundCtx, "git", "add", path)
		cmd.Dir = sandboxDir
		if err := cmd.Run(); err != nil {
			slog.Warn("git add failed on save", "environment_id", environmentID, "path", path, "err", err)
		}
		// Git tracking is updated via workspace - skip for now
	}()

	return &SaveResult{ReloadSignaled: reloadSignaled}, nil
}

func (uc *FileUsecase) CreateFile(ctx context.Context, environmentID, userID uuid.UUID, path string, isDir bool) error {
	if err := uc.authorize(ctx, environmentID, userID, accessWrite); err != nil {
		return err
	}
	target, err := uc.resolvePath(ctx, environmentID, path)
	if err != nil {
		return err
	}

	if isDir {
		return os.MkdirAll(target, 0755)
	}

	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(target, []byte{}, 0644)
}

func (uc *FileUsecase) DeleteFile(ctx context.Context, environmentID, userID uuid.UUID, path string) error {
	if err := uc.authorize(ctx, environmentID, userID, accessWrite); err != nil {
		return err
	}
	target, err := uc.resolvePath(ctx, environmentID, path)
	if err != nil {
		return err
	}

	// Prevent deleting the root workspace directory
	baseDir, err := uc.workspaceDirFor(ctx, environmentID)
	if err != nil {
		return err
	}
	if target == baseDir || target == filepath.Clean(baseDir) {
		return fmt.Errorf("cannot delete workspace root")
	}

	return os.RemoveAll(target)
}
