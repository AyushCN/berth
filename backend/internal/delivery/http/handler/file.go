package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/AyushCN/berth/internal/usecase"
)

// FileHandler handles workspace file HTTP requests.
type FileHandler struct {
	fileUC *usecase.FileUsecase
}

func NewFileHandler(uc *usecase.FileUsecase) *FileHandler {
	return &FileHandler{fileUC: uc}
}

// ListFiles returns the file tree.
func (h *FileHandler) ListFiles(c *gin.Context) {
	sandboxID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sandbox id"})
		return
	}

	path := c.Query("path")
	if path == "" {
		path = "."
	}

	files, err := h.fileUC.ListFiles(c.Request.Context(), sandboxID, path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"files": files})
}

// GetFileContent returns file content.
func (h *FileHandler) GetFileContent(c *gin.Context) {
	sandboxID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sandbox id"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}

	content, err := h.fileUC.GetFileContent(c.Request.Context(), sandboxID, path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", content)
}

// UpdateFileContent writes file content.
func (h *FileHandler) UpdateFileContent(c *gin.Context) {
	sandboxID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sandbox id"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}

	const maxFileSize = 10 * 1024 * 1024 // 10MB
	if c.Request.ContentLength > maxFileSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file exceeds 10MB limit"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFileSize)

	content, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read file content (possibly exceeds 10MB limit): " + err.Error()})
		return
	}

	result, err := h.fileUC.UpdateFileContent(c.Request.Context(), sandboxID, path, content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":        "Saved",
		"reloadSignaled": result.ReloadSignaled,
	})
}

// CreateFile handles creating files or directories.
func (h *FileHandler) CreateFile(c *gin.Context) {
	sandboxID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sandbox id"})
		return
	}

	var req struct {
		Path  string `json:"path" binding:"required"`
		IsDir bool   `json:"is_dir"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.fileUC.CreateFile(c.Request.Context(), sandboxID, req.Path, req.IsDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "created successfully"})
}

// DeleteFile handles deleting files or directories.
func (h *FileHandler) DeleteFile(c *gin.Context) {
	sandboxID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sandbox id"})
		return
	}

	var req struct {
		Path string `json:"path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.fileUC.DeleteFile(c.Request.Context(), sandboxID, req.Path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}
