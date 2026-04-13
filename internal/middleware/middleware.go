package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const headerRequestID = "X-Request-ID"

// RequestID injects a unique request ID into each request and response.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(headerRequestID)
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header(headerRequestID, requestID)
		c.Next()
	}
}

// Logger logs each HTTP request using structured zap logging.
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)

		fields := []zap.Field{
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		if reqID, exists := c.Get("request_id"); exists {
			fields = append(fields, zap.String("request_id", reqID.(string)))
		}

		if len(c.Errors) > 0 {
			for _, e := range c.Errors.Errors() {
				logger.Error(e, fields...)
			}
		} else {
			switch {
			case c.Writer.Status() >= 500:
				logger.Error("server error", fields...)
			case c.Writer.Status() >= 400:
				logger.Warn("client error", fields...)
			default:
				logger.Info("request", fields...)
			}
		}
	}
}

// Recovery recovers from panics and returns a 500.
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.Error("panic recovered",
			zap.Any("error", recovered),
			zap.String("path", c.Request.URL.Path),
		)
		c.AbortWithStatusJSON(500, gin.H{"message": "internal server error"})
	})
}

// CORS adds permissive CORS headers. Adjust AllowOrigins for production.
func CORS(allowedOrigins string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if allowedOrigins == "" {
			allowedOrigins = "*"
		}
		c.Header("Access-Control-Allow-Origin", allowedOrigins)
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin,Content-Type,Authorization,X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
