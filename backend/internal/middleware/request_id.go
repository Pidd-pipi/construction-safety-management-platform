package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-Id"

// RequestID 为每个请求生成 request_id，响应头与日志统一携带。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		c.Header(requestIDHeader, id)
		c.Set("request_id", id)
		c.Next()
	}
}

func newRequestID() string {
	return uuid.NewString()
}
