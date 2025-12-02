package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"nfcunha/hermes/hermes-server/core"
	"nfcunha/hermes/hermes-server/core/domain/healthlog"
	"nfcunha/hermes/hermes-server/database"
	"nfcunha/hermes/hermes-server/handler/middleware"
	"nfcunha/hermes/hermes-server/handler/route"
	"nfcunha/hermes/hermes-server/handler/service"
	"nfcunha/hermes/hermes-server/handler/user"
)

// CORSMiddleware exposes the CORS middleware from the middleware package.
func CORSMiddleware() gin.HandlerFunc {
	return middleware.CORSMiddleware()
}

// RegisterRoutes sets up all API routes under /hermes context path.
// It creates handlers for user management, service management, and routing.
func RegisterRoutes(engine *gin.Engine, prx *core.ProxyService, reg *core.ServiceRegistry, aegisClient *core.AegisClient, aegisURL string) {
	// Create routing service
	routingService := core.NewRoutingService(reg, prx)

	// Create health log repository
	healthLogRepo := healthlog.NewRepository(database.GetDB())

	// All management routes under /hermes context path
	hermes := engine.Group("/hermes")
	{
		// Health check endpoint (public)
		hermes.GET("/health", handleHealth)

		// Authentication middleware (used for protected routes)
		authMiddleware := middleware.AuthMiddleware(aegisClient)
		adminMiddleware := middleware.RequireAdmin()

		// User management handler (Phase 5)
		// Proxies requests to Aegis for all user operations
		userHandler := user.NewHandler(aegisClient, aegisURL)
		userHandler.RegisterRoutes(hermes, authMiddleware)

		// Service management handler (Phase 4)
		// Handles service registration, health checks, and lifecycle
		service.RegisterRoutes(hermes, reg, healthLogRepo, authMiddleware, adminMiddleware)

		// Service routing handler (Phase 3)
		// Handles dynamic request routing to registered services
		routeHandler := route.NewHandler(routingService)
		routeHandler.RegisterRoutes(hermes)
	}
}

// handleHealth handles health check requests.
func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "hermes",
		"timestamp": time.Now().UTC(),
	})
}
