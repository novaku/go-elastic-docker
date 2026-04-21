package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestID_GenerateAndPreserve(t *testing.T) {
	t.Run("generate when missing", func(t *testing.T) {
		r := gin.New()
		r.Use(RequestID())
		r.GET("/", func(c *gin.Context) {
			v, ok := c.Get("request_id")
			if !ok || v.(string) == "" {
				t.Fatalf("expected request_id in context")
			}
			c.Status(http.StatusOK)
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		r.ServeHTTP(rec, req)

		if rec.Header().Get("X-Request-ID") == "" {
			t.Fatalf("expected response header X-Request-ID")
		}
	})

	t.Run("preserve incoming id", func(t *testing.T) {
		r := gin.New()
		r.Use(RequestID())
		r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Request-ID", "req-123")
		r.ServeHTTP(rec, req)

		if got := rec.Header().Get("X-Request-ID"); got != "req-123" {
			t.Fatalf("expected req-123, got %q", got)
		}
	})
}

func TestCORS(t *testing.T) {
	r := gin.New()
	r.Use(CORS(""))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected wildcard origin")
	}
}

func TestJWTAuth(t *testing.T) {
	const secret = "secret"

	t.Run("missing header", func(t *testing.T) {
		r := gin.New()
		r.Use(JWTAuth(secret))
		r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		r := gin.New()
		r.Use(JWTAuth(secret))
		r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Token abc")
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		r := gin.New()
		r.Use(JWTAuth(secret))
		r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer bad.token")
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("valid token", func(t *testing.T) {
		token, err := GenerateJWT(secret, "alice", time.Hour)
		if err != nil {
			t.Fatalf("generate token: %v", err)
		}

		r := gin.New()
		r.Use(JWTAuth(secret))
		r.GET("/", func(c *gin.Context) {
			v, _ := c.Get("jwt_username")
			c.JSON(http.StatusOK, gin.H{"u": v})
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "alice") {
			t.Fatalf("expected username in response body")
		}
	})
}

func TestGenerateJWT(t *testing.T) {
	token, err := GenerateJWT("k", "bob", time.Minute)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	claims := &JWTClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (interface{}, error) {
		return []byte("k"), nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("expected valid token, err=%v", err)
	}
	if claims.Username != "bob" {
		t.Fatalf("expected bob, got %q", claims.Username)
	}
}

func TestRecoveryAndLogger(t *testing.T) {
	logger := zap.NewNop()
	r := gin.New()
	r.Use(RequestID(), Recovery(logger), Logger(logger))
	r.GET("/panic", func(c *gin.Context) { panic("boom") })
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/err", func(c *gin.Context) {
		c.Error(http.ErrAbortHandler) //nolint:errcheck
		c.Status(http.StatusInternalServerError)
	})

	for _, path := range []string{"/panic", "/ok", "/err"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		r.ServeHTTP(rec, req)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for panic, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["message"] != "internal server error" {
		t.Fatalf("unexpected message: %q", body["message"])
	}
}
