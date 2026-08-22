package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swordrookie/berth/internal/infrastructure/db"
	"github.com/swordrookie/berth/internal/infrastructure/redis"
)

func HealthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	dbOK := true
	if err := db.Pool().Ping(ctx); err != nil {
		dbOK = false
	}

	redisOK := true
	if redis.Client() != nil {
		if err := redis.Client().Ping(ctx).Err(); err != nil {
			redisOK = false
		}
	}

	status := http.StatusOK
	if !dbOK || !redisOK {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status": "ok",
		"db":     dbOK,
		"redis":  redisOK,
	})
}
