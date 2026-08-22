package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GithubLogin(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "github login not yet implemented"})
}

func GithubCallback(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "github callback not yet implemented"})
}
