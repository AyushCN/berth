package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"github.com/google/uuid"
)

type FileUsecase struct {
	workspaceDir string
}

func NewFileUsecase(dir string) *FileUsecase {
	return &FileUsecase{workspaceDir: dir}
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

func (uc *FileUsecase) UpdateFileContent(ctx context.Context, sandboxID uuid.UUID, path string, content []byte) error {
	target, err := uc.resolvePath(sandboxID, path)
	if err != nil {
		return err
	}
	
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(target, content, 0644)
}
