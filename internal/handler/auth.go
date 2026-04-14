package handler

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/novaku/go-elastic-search/config"
	"github.com/novaku/go-elastic-search/internal/middleware"
)

type AuthHandler struct {
	cfg *config.JWTConfig
}

func NewAuthHandler(cfg *config.JWTConfig) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login validates credentials and returns a signed JWT token.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "username and password are required"})
		return
	}

	usernameMatch := subtle.ConstantTimeCompare([]byte(req.Username), []byte(h.cfg.AdminUsername)) == 1
	passwordMatch := subtle.ConstantTimeCompare([]byte(req.Password), []byte(h.cfg.AdminPassword)) == 1

	if !usernameMatch || !passwordMatch {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid credentials"})
		return
	}

	token, err := middleware.GenerateJWT(h.cfg.Secret, req.Username, h.cfg.ExpiryDuration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_in": int(h.cfg.ExpiryDuration.Seconds()),
	})
}
