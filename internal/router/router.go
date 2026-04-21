package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/novaku/go-elastic-search/config"
	"github.com/novaku/go-elastic-search/internal/handler"
	"github.com/novaku/go-elastic-search/internal/middleware"
	"github.com/novaku/go-elastic-search/internal/service"
	"github.com/novaku/go-elastic-search/internal/ui"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

type statusResponse struct {
	Status string `json:"status"`
}

type healthUnhealthyResponse struct {
	Status        string `json:"status"`
	Elasticsearch string `json:"elasticsearch"`
}

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

	r.GET("/health", healthHandler(healthChecker))
	r.GET("/ready", readyHandler())
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

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

// healthHandler godoc
// @Summary Health check
// @Description Check API and Elasticsearch health.
// @Tags system
// @Produce json
// @Success 200 {object} statusResponse
// @Failure 503 {object} healthUnhealthyResponse
// @Router /health [get]
func healthHandler(healthChecker service.HealthChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := healthChecker.Check(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, healthUnhealthyResponse{
				Status:        "unhealthy",
				Elasticsearch: err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, statusResponse{Status: "ok"})
	}
}

// readyHandler godoc
// @Summary Readiness check
// @Description Check if API process is ready to serve requests.
// @Tags system
// @Produce json
// @Success 200 {object} statusResponse
// @Router /ready [get]
func readyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, statusResponse{Status: "ready"})
	}
}
