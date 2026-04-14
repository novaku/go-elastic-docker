package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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

// JWTClaims holds the standard JWT claims plus a custom subject field.
type JWTClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// JWTAuth validates the Bearer token from the Authorization header.
// On success, the username claim is stored in the gin context as "jwt_username".
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "authorization header required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "authorization header format must be: Bearer <token>"})
			return
		}

		tokenStr := parts[1]
		claims := &JWTClaims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid or expired token"})
			return
		}

		c.Set("jwt_username", claims.Username)
		c.Next()
	}
}

// GenerateJWT creates a signed JWT token for the given username.
func GenerateJWT(secret, username string, expiry time.Duration) (string, error) {
	claims := JWTClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
