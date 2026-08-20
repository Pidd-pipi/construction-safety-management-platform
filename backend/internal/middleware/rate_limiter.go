package middleware

import (
	"net/http"
	"sync"
	"time"

	"safetyplatform/internal/constants"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	tokens   int
	lastFill time.Time
}

// RateLimiter 基于 IP 的令牌桶限流。
type RateLimiter struct {
	mu       sync.Mutex
	perMin   int
	buckets  map[string]*bucket
	capacity int
}

// NewRateLimiter 构造限流器。
func NewRateLimiter(perMin int) *RateLimiter {
	if perMin <= 0 {
		perMin = 120
	}
	return &RateLimiter{perMin: perMin, buckets: make(map[string]*bucket), capacity: perMin}
}

// Limit 返回限流中间件。
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		rl.mu.Lock()
		b, ok := rl.buckets[ip]
		now := time.Now()
		if !ok {
			b = &bucket{tokens: rl.capacity, lastFill: now}
			rl.buckets[ip] = b
		}
		rl.mu.Unlock()
		elapsed := now.Sub(b.lastFill)
		b.tokens += int(elapsed.Minutes()) * rl.perMin
		if b.tokens > rl.capacity {
			b.tokens = rl.capacity
		}
		b.lastFill = now
		if b.tokens <= 0 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": constants.CodeTooManyRequests, "message": constants.MsgTooManyRequests, "data": nil})
			return
		}
		b.tokens--
		c.Next()
	}
}

// Stats 返回各 IP 当前令牌数快照。
func (rl *RateLimiter) Stats() map[string]int {
	out := make(map[string]int, len(rl.buckets))
	for ip, b := range rl.buckets {
		out[ip] = b.tokens
	}
	return out
}
