package core

import (
	"errors"
	"log"

	"github.com/gin-gonic/gin"
)

// RoutingService handles routing requests to registered backend services.
// It uses the service registry to discover healthy instances and forwards
// requests using the proxy service. Currently uses first-available routing
// strategy (future: implement load balancing).
type RoutingService struct {
	registry *ServiceRegistry
	proxy    *ProxyService
}

// NewRoutingService creates a new routing service with the given registry and proxy.
func NewRoutingService(reg *ServiceRegistry, prx *ProxyService) *RoutingService {
	return &RoutingService{
		registry: reg,
		proxy:    prx,
	}
}

// RouteToService routes a request to a registered service by name.
// It looks up healthy instances of the service and forwards the request.
// Currently uses the first healthy instance found.
// TODO: Implement load balancing across multiple healthy instances.
//
// Parameters:
//   - c: Gin context containing the request
//   - serviceName: name of the registered service to route to
//   - path: path to append to the service base URL
//
// Returns an error if no healthy instances are available or if forwarding fails.
func (s *RoutingService) RouteToService(c *gin.Context, serviceName string, path string) error {
	log.Printf("Routing request to service '%s' with path '%s'", serviceName, path)

	// Get healthy instances of the service
	instances := s.registry.GetHealthy(serviceName)
	if len(instances) == 0 {
		log.Printf("No healthy instances found for service: %s", serviceName)
		return errors.New("no healthy instances available")
	}

	// Use first healthy instance (TODO: implement load balancing)
	target := instances[0]
	targetURL := target.BaseURL() + path

	log.Printf("Forwarding request to: %s", targetURL)

	// Forward the request using the proxy
	return s.proxy.ForwardToURL(c, targetURL)
}
