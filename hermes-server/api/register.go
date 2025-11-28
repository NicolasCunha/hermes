package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"nfcunha/hermes/hermes-server/api/route"
	"nfcunha/hermes/hermes-server/api/service"
	"nfcunha/hermes/hermes-server/api/user"
	"nfcunha/hermes/hermes-server/middleware"
	"nfcunha/hermes/hermes-server/services/aegis"
	"nfcunha/hermes/hermes-server/services/proxy"
	"nfcunha/hermes/hermes-server/services/registry"
	"nfcunha/hermes/hermes-server/services/router"
	"nfcunha/hermes/hermes-server/services/routing"
)

// RegisterRoutes sets up all API routes under /hermes context path
func RegisterRoutes(engine *gin.Engine, rtr *router.Router, prx *proxy.Proxy, reg *registry.Registry, aegisClient *aegis.Client, aegisURL string) {
	// Create routing service
	routingService := routing.New(reg, prx)

	// All management routes under /hermes context path
	hermes := engine.Group("/hermes")
	{
		// Health check endpoint (public)
		hermes.GET("/health", handleHealth)

		// Authentication middleware (used for protected routes)
		authMiddleware := middleware.AuthMiddleware(aegisClient)
		adminMiddleware := middleware.RequireAdmin()

		// User management API (protected - Phase 2)
		// Proxies requests to Aegis for all user operations
		userAPI := user.NewAPI(aegisClient, aegisURL)
		userAPI.RegisterRoutes(hermes, authMiddleware)

		// Service management API (Phase 3)
		service.RegisterRoutes(hermes, reg, authMiddleware, adminMiddleware)

		// Service routing API (Phase 3)
		routeAPI := route.NewAPI(routingService)
		routeAPI.RegisterRoutes(hermes)
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
