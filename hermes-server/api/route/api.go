package route

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"nfcunha/hermes/hermes-server/services/routing"
)

// API handles routing endpoint
type API struct {
	routingService *routing.Service
}

// NewAPI creates a new routing API handler
func NewAPI(routingService *routing.Service) *API {
	return &API{
		routingService: routingService,
	}
}

// RegisterRoutes registers routing endpoints
func (a *API) RegisterRoutes(router gin.IRouter) {
	// Service routing proxy - /route/{serviceName}/*path
	router.Any("/route/:serviceName/*path", a.routeToService)
}

// routeToService handles routing requests to registered services
// Pattern: /route/{serviceName}/{path}
// Example: /route/aegis/api/aegis/health -> http://aegis-host:port/api/aegis/health
func (a *API) routeToService(c *gin.Context) {
	serviceName := c.Param("serviceName")
	path := c.Param("path") // This includes the leading slash

	// Route request through the routing service
	err := a.routingService.RouteToService(c, serviceName, path)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "service unavailable",
			"service": serviceName,
			"message": err.Error(),
		})
		return
	}
}
