package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	infradis "github.com/swordrookie/berth/internal/infrastructure/redis"
	"log/slog"
)

func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		client := infradis.Client()
		if client == nil {
			c.Next()
			return
		}

		key := fmt.Sprintf("rate_limit:%s:%s", c.ClientIP(), c.Request.URL.Path)
		ctx := context.Background()

		count, err := client.Incr(ctx, key).Result()
		if err != nil {
			slog.Error("redis rate limit error", "error", err)
			c.Next()
			return
		}

		if count == 1 {
			client.Expire(ctx, key, time.Minute)
		}

		if count > 200 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		c.Next()
	}
}
