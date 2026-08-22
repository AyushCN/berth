package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetMe(c *gin.Context) {
	userID, _ := c.Get("userId")
	c.JSON(http.StatusOK, gin.H{
		"id": userID,
		"message": "user profile endpoint stub",
	})
}
