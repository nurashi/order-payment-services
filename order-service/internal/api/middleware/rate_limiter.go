package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RateLimiter(client *redis.Client, maxRequests int, windowSeconds int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := fmt.Sprintf("rate_limit:%s", ip)
		window := time.Duration(windowSeconds) * time.Second

		ctx := context.Background()

		count, err := client.Incr(ctx, key).Result()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "rate limiter error"})
			return
		}

		if count == 1 {
			client.Expire(ctx, key, window)
		}

		if int(count) > maxRequests {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": fmt.Sprintf("rate limit exceeded: max %d requests per %d seconds", maxRequests, windowSeconds),
			})
			return
		}

		c.Next()
	}
}
