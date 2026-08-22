package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ListEnvironments(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"sandboxes": []any{}})
}

func CreateEnvironment(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "create environment not yet implemented"})
}
