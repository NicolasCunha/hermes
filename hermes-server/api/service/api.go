package service

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"nfcunha/hermes/hermes-server/domain/service"
	"nfcunha/hermes/hermes-server/services/registry"
)

type API struct {
	registry *registry.Registry
}

// RegisterRoutes registers service management routes
func RegisterRoutes(router gin.IRouter, reg *registry.Registry, authMiddleware, adminMiddleware gin.HandlerFunc) {
	api := &API{registry: reg}

	services := router.Group("/services")
	// All service management endpoints require authentication and admin privileges
	services.Use(authMiddleware, adminMiddleware)
	{
		services.POST("", api.registerService)
		services.DELETE("/:id", api.deregisterService)
		services.GET("", api.listServices)
		services.GET("/:id", api.getService)
	}
}

type RegisterRequest struct {
	Name            string            `json:"name" binding:"required"`
	Host            string            `json:"host" binding:"required"`
	Port            int               `json:"port" binding:"required"`
	HealthCheckPath string            `json:"health_check_path" binding:"required"`
	Protocol        string            `json:"protocol"`
	Metadata        map[string]string `json:"metadata"`
}

// registerService handles service registration (manual or auto check-in)
func (a *API) registerService(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create service
	svc := service.NewService(req.Name, req.Host, req.Port, req.HealthCheckPath)
	if req.Protocol != "" {
		svc.Protocol = req.Protocol
	}
	if req.Metadata != nil {
		svc.Metadata = req.Metadata
	}

	// Perform initial health check
	if !a.checkServiceHealth(svc) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "service health check failed",
			"url":   svc.HealthCheckURL(),
		})
		return
	}

	// Register service
	if err := a.registry.Register(svc); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Service registered: %s at %s", svc.Name, svc.BaseURL())
	c.JSON(http.StatusCreated, svc)
}

// checkServiceHealth performs a simple health check
func (a *API) checkServiceHealth(svc *service.Service) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(svc.HealthCheckURL())
	if err != nil {
		log.Printf("Health check failed for %s: %v", svc.Name, err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// deregisterService handles service deregistration
func (a *API) deregisterService(c *gin.Context) {
	id := c.Param("id")

	if err := a.registry.Deregister(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Service deregistered: %s", id)
	c.JSON(http.StatusOK, gin.H{"message": "service deregistered"})
}

// listServices lists all registered services
func (a *API) listServices(c *gin.Context) {
	services := a.registry.List()
	c.JSON(http.StatusOK, gin.H{
		"services": services,
		"count":    len(services),
	})
}

// getService retrieves a specific service
func (a *API) getService(c *gin.Context) {
	id := c.Param("id")

	svc, err := a.registry.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, svc)
}
