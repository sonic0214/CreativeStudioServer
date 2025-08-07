package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiter *rate.Limiter
}

func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(r, b),
	}
}

func (rl *RateLimiter) Allow() bool {
	return rl.limiter.Allow()
}

var limiters = make(map[string]*RateLimiter)

func RateLimit(requestsPerMinute int, burst int) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		limiter, exists := limiters[clientIP]
		if !exists {
			limiter = NewRateLimiter(rate.Every(time.Minute/time.Duration(requestsPerMinute)), burst)
			limiters[clientIP] = limiter
		}

		if !limiter.Allow() {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", requestsPerMinute))
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func AuthRateLimit() gin.HandlerFunc {
	return RateLimit(5, 10) // 5 requests per minute with burst of 10
}

func APIRateLimit() gin.HandlerFunc {
	return RateLimit(100, 200) // 100 requests per minute with burst of 200
}