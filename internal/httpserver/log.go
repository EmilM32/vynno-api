package httpserver

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

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
		}
		if status >= httpStatusServerError {
			slog.Error("request", args...)
			return
		}
		slog.Info("request", args...)
	}
}

const httpStatusServerError = 500
