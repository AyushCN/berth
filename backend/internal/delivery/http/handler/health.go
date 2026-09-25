package handler

import (
	"net/http"

	"github.com/AyushCN/berth/internal/infrastructure/db"
	"github.com/AyushCN/berth/internal/infrastructure/redis"
	"github.com/gin-gonic/gin"
)

func HealthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	dbOK := true
	if err := db.Pool().Ping(ctx); err != nil {
		dbOK = false
	}

	redisOK := false
	if redis.Client() != nil {
		redisOK = true
		if err := redis.Client().Ping(ctx).Err(); err != nil {
			redisOK = false
		}
	}

	status := http.StatusOK
	health := "ok"
	if !dbOK || !redisOK {
		status = http.StatusServiceUnavailable
		health = "unhealthy"
	}

	c.JSON(status, gin.H{
		"status": health,
		"db":     dbOK,
		"redis":  redisOK,
	})
}
