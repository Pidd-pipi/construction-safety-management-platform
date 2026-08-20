package middleware

import (
	"net/http"
	"sync"
	"time"

	"safetyplatform/internal/constants"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	tokens   float64
	lastFill time.Time
}

// RateLimiter 基于 IP 的令牌桶限流。
type RateLimiter struct {
	mu       sync.Mutex
	perMin   int
	buckets  map[string]*bucket
	capacity float64
}

// NewRateLimiter 构造限流器。
func NewRateLimiter(perMin int) *RateLimiter {
	if perMin <= 0 {
		perMin = 120
	}
	return &RateLimiter{perMin: perMin, buckets: make(map[string]*bucket), capacity: float64(perMin)}
}

// Limit 返回限流中间件。
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		allowed := rl.allow(ip, time.Now())
		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": constants.CodeTooManyRequests, "message": constants.MsgTooManyRequests, "data": nil})
			return
		}
		c.Next()
	}
}

// allow 在锁保护下完成取桶、补令牌、判定与扣减的整段读改写，避免并发同一 IP 的数据竞争。
func (rl *RateLimiter) allow(ip string, now time.Time) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[ip]
	if !ok {
		b = &bucket{tokens: rl.capacity, lastFill: now}
		rl.buckets[ip] = b
	}

	// 按毫秒粒度补充令牌，保证短时间窗口内也能恢复额度。
	elapsed := now.Sub(b.lastFill).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * float64(rl.perMin) / 60.0
		if b.tokens > rl.capacity {
			b.tokens = rl.capacity
		}
		b.lastFill = now
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Stats 返回各 IP 当前令牌数快照。
func (rl *RateLimiter) Stats() map[string]int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	out := make(map[string]int, len(rl.buckets))
	for ip, b := range rl.buckets {
		out[ip] = int(b.tokens)
	}
	return out
}
