package main

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

func recoveryLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			panicValue := recover()
			if panicValue == nil {
				return
			}
			logger.ErrorContext(c.Request.Context(), "http.panic",
				slog.Any("panic", panicValue),
				slog.String("stack", string(debug.Stack())),
			)
			c.AbortWithStatus(http.StatusInternalServerError)
		}()
		c.Next()
	}
}

func accessLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", status),
			slog.Int("response_bytes", c.Writer.Size()),
			slog.Duration("duration", time.Since(start)),
			slog.String("client_ip", c.ClientIP()),
		}

		switch {
		case status >= http.StatusInternalServerError:
			logger.LogAttrs(c.Request.Context(), slog.LevelError, "http.request", attrs...)
		case status >= http.StatusBadRequest:
			logger.LogAttrs(c.Request.Context(), slog.LevelWarn, "http.request", attrs...)
		default:
			logger.LogAttrs(c.Request.Context(), slog.LevelInfo, "http.request", attrs...)
		}
	}
}
