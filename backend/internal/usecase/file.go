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
	"github.com/google/uuid"
)

type FileUsecase struct {
	workspaceDir string
	sandboxUC    *SandboxUsecase // optional, used to signal reload via Exec
}

func NewFileUsecase(dir string, sandboxUC *SandboxUsecase) *FileUsecase {
	return &FileUsecase{workspaceDir: dir, sandboxUC: sandboxUC}
}

func (uc *FileUsecase) getSandboxDir(sandboxID uuid.UUID) string {
	return filepath.Join(uc.workspaceDir, sandboxID.String())
}

func (uc *FileUsecase) resolvePath(sandboxID uuid.UUID, reqPath string) (string, error) {
	baseDir := uc.getSandboxDir(sandboxID)
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

func (uc *FileUsecase) ListFiles(ctx context.Context, sandboxID uuid.UUID, reqPath string) (any, error) {
	target, err := uc.resolvePath(sandboxID, reqPath)
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
		relPath, _ := filepath.Rel(uc.getSandboxDir(sandboxID), filepath.Join(target, e.Name()))
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

func (uc *FileUsecase) GetFileContent(ctx context.Context, sandboxID uuid.UUID, path string) ([]byte, error) {
	target, err := uc.resolvePath(sandboxID, path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(target)
}

// SaveResult is returned by UpdateFileContent.
type SaveResult struct {
	ReloadSignaled bool
}

func (uc *FileUsecase) UpdateFileContent(ctx context.Context, sandboxID uuid.UUID, path string, content []byte) (*SaveResult, error) {
	target, err := uc.resolvePath(sandboxID, path)
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
	if uc.sandboxUC != nil && uc.sandboxUC.runtime != nil {
		sandbox, err := uc.sandboxUC.repo.GetByID(ctx, sandboxID)
		if err == nil && sandbox.State == "RUNNING" && sandbox.ContainerID != nil {
			inContainerPath := "/workspace/" + strings.TrimPrefix(path, "/")
			touchCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, touchErr := uc.sandboxUC.runtime.Exec(touchCtx, *sandbox.ContainerID, []string{"touch", inContainerPath})
			if touchErr == nil {
				reloadSignaled = true
			} else {
				slog.Warn("touch-on-save failed", "sandbox_id", sandboxID, "path", inContainerPath, "err", touchErr)
			}
		}
	}

	// Async: stage file in git and update git tracking in DB
	sandboxDir := uc.getSandboxDir(sandboxID)
	go func() {
		backgroundCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(backgroundCtx, "git", "add", path)
		cmd.Dir = sandboxDir
		if err := cmd.Run(); err != nil {
			slog.Warn("git add failed on save", "sandbox_id", sandboxID, "path", path, "err", err)
		}
		if uc.sandboxUC != nil {
			if err := uc.sandboxUC.repo.UpdateGitTracking(backgroundCtx, sandboxID, true, nil, nil); err != nil {
				slog.Warn("failed to update git tracking", "sandbox_id", sandboxID, "err", err)
			}
		}
	}()

	return &SaveResult{ReloadSignaled: reloadSignaled}, nil
}

func (uc *FileUsecase) CreateFile(ctx context.Context, sandboxID uuid.UUID, path string, isDir bool) error {
	target, err := uc.resolvePath(sandboxID, path)
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

func (uc *FileUsecase) DeleteFile(ctx context.Context, sandboxID uuid.UUID, path string) error {
	target, err := uc.resolvePath(sandboxID, path)
	if err != nil {
		return err
	}
	
	// Prevent deleting the root workspace directory
	baseDir := uc.getSandboxDir(sandboxID)
	if target == baseDir || target == filepath.Clean(baseDir) {
		return fmt.Errorf("cannot delete workspace root")
	}

	return os.RemoveAll(target)
}
