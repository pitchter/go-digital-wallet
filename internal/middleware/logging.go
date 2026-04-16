package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger logs request metadata after each response.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		requestID, _ := c.Get(RequestIDKey)
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		attrs := []any{
			"request_id", requestID,
			"method", c.Request.Method,
			"route", route,
			"status_code", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		if c.Writer.Status() >= 500 {
			logger.Error("http_request", attrs...)
			return
		}

		logger.Info("http_request", attrs...)
	}
}
