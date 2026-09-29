package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/AyushCN/berth/internal/usecase"
)

// FileHandler handles workspace file HTTP requests.
//
// Authorization lives in FileUsecase, not here, so a new file route cannot
// accidentally ship without it. These endpoints previously performed no
// ownership check at all, which let any authenticated user read, overwrite or
// delete the files of any environment by UUID.
type FileHandler struct {
	fileUC *usecase.FileUsecase
}

func NewFileHandler(uc *usecase.FileUsecase) *FileHandler {
	return &FileHandler{fileUC: uc}
}

// fileRequest resolves the :id and userId route values.
//
// A missing or unknown environment is reported as 404 rather than 403 so that
// a caller cannot use the status code to discover which environment ids exist.
func fileRequest(c *gin.Context) (environmentID, userID uuid.UUID, ok bool) {
	environmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid environment id"})
		return uuid.Nil, uuid.Nil, false
	}
	raw, _ := c.Get("userId")
	parsed, err := uuid.Parse(raw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, uuid.Nil, false
	}
	return environmentID, parsed, true
}

func fileError(c *gin.Context, err error) {
	// Access failures and unknown ids are indistinguishable on purpose.
	if err.Error() == "environment not found" || err.Error() == "workspace not found" {
		c.JSON(http.StatusNotFound, gin.H{"error": "environment not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

// ListFiles returns the file tree.
func (h *FileHandler) ListFiles(c *gin.Context) {
	environmentID, userID, ok := fileRequest(c)
	if !ok {
		return
	}

	path := c.Query("path")
	if path == "" {
		path = "."
	}

	files, err := h.fileUC.ListFiles(c.Request.Context(), environmentID, userID, path)
	if err != nil {
		fileError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"files": files})
}

// GetFileContent returns file content.
func (h *FileHandler) GetFileContent(c *gin.Context) {
	environmentID, userID, ok := fileRequest(c)
	if !ok {
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}

	content, err := h.fileUC.GetFileContent(c.Request.Context(), environmentID, userID, path)
	if err != nil {
		fileError(c, err)
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", content)
}

// UpdateFileContent writes file content.
func (h *FileHandler) UpdateFileContent(c *gin.Context) {
	environmentID, userID, ok := fileRequest(c)
	if !ok {
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

	result, err := h.fileUC.UpdateFileContent(c.Request.Context(), environmentID, userID, path, content)
	if err != nil {
		fileError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":        "Saved",
		"reloadSignaled": result.ReloadSignaled,
	})
}

// CreateFile handles creating files or directories.
func (h *FileHandler) CreateFile(c *gin.Context) {
	environmentID, userID, ok := fileRequest(c)
	if !ok {
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

	if err := h.fileUC.CreateFile(c.Request.Context(), environmentID, userID, req.Path, req.IsDir); err != nil {
		fileError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "created successfully"})
}

// DeleteFile handles deleting files or directories.
func (h *FileHandler) DeleteFile(c *gin.Context) {
	environmentID, userID, ok := fileRequest(c)
	if !ok {
		return
	}

	var req struct {
		Path string `json:"path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.fileUC.DeleteFile(c.Request.Context(), environmentID, userID, req.Path); err != nil {
		fileError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}
