package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	infradis "github.com/AyushCN/berth/internal/infrastructure/redis"
	"log/slog"
)

func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		client := infradis.Client()
		if client == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "rate limiter unavailable"})
			return
		}

		key := fmt.Sprintf("rate_limit:%s:%s", c.ClientIP(), c.Request.URL.Path)
		ctx := context.Background()

		count, err := client.Incr(ctx, key).Result()
		if err != nil {
			slog.Error("redis rate limit error", "error", err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "rate limiter unavailable"})
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

// RateLimitUser applies a stricter rate limit per user bucket for authenticated routes.
func RateLimitUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		client := infradis.Client()
		if client == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "rate limiter unavailable"})
			return
		}

		userID, exists := c.Get("userId")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		key := fmt.Sprintf("ratelimit:user:%s", userID)
		ctx := context.Background()

		count, err := client.Incr(ctx, key).Result()
		if err != nil {
			slog.Error("redis rate limit error", "error", err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "rate limiter unavailable"})
			return
		}

		if count == 1 {
			client.Expire(ctx, key, time.Minute)
		}

		if count > 30 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		c.Next()
	}
}
