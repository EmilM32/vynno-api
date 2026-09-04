package httpserver

import (
	"log/slog"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	requestIDHeader       = "X-Request-ID"
	ctxRequestID          = "requestID"
	httpStatusServerError = 500
)

var requestIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if !requestIDRe.MatchString(id) {
			id = uuid.NewString()
		}
		c.Set(ctxRequestID, id)
		c.Writer.Header().Set(requestIDHeader, id)
		c.Next()
	}
}

func requestIDFrom(c *gin.Context) string {
	v, ok := c.Get(ctxRequestID)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func requestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.Request.URL.Path
		if path == "/healthz" || path == "/readyz" {
			return
		}

		status := c.Writer.Status()
		args := []any{
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"ms", time.Since(start).Milliseconds(),
			"request_id", requestIDFrom(c),
		}
		if status >= httpStatusServerError {
			slog.Error("request", args...)
			return
		}
		slog.Info("request", args...)
	}
}
