package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/novaku/go-elastic-search/config"
	"github.com/novaku/go-elastic-search/internal/handler"
	"github.com/novaku/go-elastic-search/internal/middleware"
	"github.com/novaku/go-elastic-search/internal/service"
	"github.com/novaku/go-elastic-search/internal/ui"
	"go.uber.org/zap"
)

func New(cfg *config.Config, productSvc service.ProductUseCase, healthChecker service.HealthChecker, logger *zap.Logger) *gin.Engine {
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.Recovery(logger),
		middleware.Logger(logger),
		middleware.CORS(cfg.CORSAllowedOrigins),
	)

	r.GET("/health", func(c *gin.Context) {
		if err := healthChecker.Check(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":        "unhealthy",
				"elasticsearch": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Auth — no JWT required
	authHandler := handler.NewAuthHandler(&cfg.JWT)
	r.POST("/auth/login", authHandler.Login)

	qaAssets := gin.WrapH(http.StripPrefix("/qa", ui.Handler()))
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/qa/")
	})
	r.GET("/qa", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/qa/")
	})
	r.GET("/qa/*filepath", qaAssets)

	// Protected API routes — require valid JWT
	v1 := r.Group("/v1", middleware.JWTAuth(cfg.JWT.Secret))
	handler.NewProductHandler(productSvc, logger).RegisterRoutes(v1)

	return r
}
