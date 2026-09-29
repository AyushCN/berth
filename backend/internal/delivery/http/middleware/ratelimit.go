package middleware

import (
	"fmt"
	"net/http"
	"time"

	infradis "github.com/AyushCN/berth/internal/infrastructure/redis"
	"github.com/gin-gonic/gin"
	"log/slog"
)

// RateLimit applies a per-client-IP, per-path limit to the whole /api group.
func RateLimit(requestsPerMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		client := infradis.Client()
		if client == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "rate limiter unavailable"})
			return
		}

		key := fmt.Sprintf("rate_limit:%s:%s", c.ClientIP(), c.Request.URL.Path)
		ctx := c.Request.Context()

		count, err := client.Incr(ctx, key).Result()
		if err != nil {
			slog.Error("redis rate limit error", "error", err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "rate limiter unavailable"})
			return
		}

		if count == 1 {
			client.Expire(ctx, key, time.Minute)
		}

		if count > int64(requestsPerMinute) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		c.Next()
	}
}

// RateLimitUser applies a stricter per-user limit to authenticated routes.
//
// Every authenticated route shares one bucket per user, so the ceiling has to
// accommodate the client's own polling or a user merely watching a log stream
// locks themselves out of the whole API. At the previous hardcoded 30/min the
// frontend generated roughly 36 requests a minute (the log tail alone is
// 20/min), so opening the Build Logs tab reliably produced 429s everywhere.
// The value now comes from configuration.
func RateLimitUser(requestsPerMinute int) gin.HandlerFunc {
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
		ctx := c.Request.Context()

		count, err := client.Incr(ctx, key).Result()
		if err != nil {
			slog.Error("redis rate limit error", "error", err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "rate limiter unavailable"})
			return
		}

		if count == 1 {
			client.Expire(ctx, key, time.Minute)
		}

		if count > int64(requestsPerMinute) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		c.Next()
	}
}
