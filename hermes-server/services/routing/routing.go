package routing

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"nfcunha/hermes/hermes-server/services/proxy"
	"nfcunha/hermes/hermes-server/services/registry"
)

// Service handles routing requests to registered backend services
type Service struct {
	registry *registry.Registry
	proxy    *proxy.Proxy
}

// New creates a new routing service
func New(reg *registry.Registry, prx *proxy.Proxy) *Service {
	return &Service{
		registry: reg,
		proxy:    prx,
	}
}

// RouteToService routes a request to a registered service by name
// serviceName: the name of the registered service
// path: the path to append to the service base URL
func (s *Service) RouteToService(c *gin.Context, serviceName string, path string) error {
	log.Printf("Routing request to service '%s' with path '%s'", serviceName, path)

	// Get healthy instances of the service
	instances := s.registry.GetHealthy(serviceName)
	if len(instances) == 0 {
		log.Printf("No healthy instances found for service: %s", serviceName)
		return fmt.Errorf("no healthy instances available for service: %s", serviceName)
	}

	// Use first healthy instance (TODO: implement load balancing)
	target := instances[0]
	targetURL := target.BaseURL() + path

	log.Printf("Forwarding request to: %s", targetURL)

	// Forward the request using the proxy
	return s.proxy.ForwardToURL(c, targetURL)
}
