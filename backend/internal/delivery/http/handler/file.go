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

	content, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.fileUC.UpdateFileContent(c.Request.Context(), sandboxID, path, content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
