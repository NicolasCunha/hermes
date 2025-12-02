package route

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"nfcunha/hermes/hermes-server/core"
)

// Handler manages dynamic routing to registered services
type Handler struct {
	routingService *core.RoutingService
}

// NewHandler creates a new routing handler
func NewHandler(routingService *core.RoutingService) *Handler {
	return &Handler{
		routingService: routingService,
	}
}

// RegisterRoutes registers routing endpoints
// Routes all requests matching /route/{serviceName}/*path to registered services
func (h *Handler) RegisterRoutes(router gin.IRouter) {
	// Service routing proxy - /route/{serviceName}/*path
	router.Any("/route/:serviceName/*path", h.handleRouteToService)
}

// handleRouteToService proxies requests to registered services
// Pattern: /route/{serviceName}/{path}
// Example: /route/aegis/api/aegis/health -> http://aegis-host:port/api/aegis/health
//
// This handler enables dynamic service discovery and routing without hardcoding URLs.
// It looks up the service by name in the registry and forwards the request,
// preserving HTTP method, headers, body, and query parameters.
func (h *Handler) handleRouteToService(c *gin.Context) {
	serviceName := c.Param("serviceName")
	path := c.Param("path") // This includes the leading slash

	// Route request through the routing service
	err := h.routingService.RouteToService(c, serviceName, path)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "service unavailable",
			"service": serviceName,
			"message": err.Error(),
		})
		return
	}
}
