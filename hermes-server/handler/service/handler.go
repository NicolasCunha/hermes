// Package service provides HTTP handlers for service registration and management.
package service

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"nfcunha/hermes/hermes-server/core"
	"nfcunha/hermes/hermes-server/core/domain/healthlog"
	"nfcunha/hermes/hermes-server/core/domain/service"
)

// Handler manages service registration and lifecycle.
// It handles HTTP requests for service registration, deregistration, and health checks.
type Handler struct {
	registry      *core.ServiceRegistry
	healthClient  *http.Client
	healthLogRepo *healthlog.Repository
}

// NewHandler creates a new service handler with the given registry and health log repository.
func NewHandler(reg *core.ServiceRegistry, healthLogRepo *healthlog.Repository) *Handler {
	return &Handler{
		registry:      reg,
		healthClient:  &http.Client{Timeout: 5 * time.Second},
		healthLogRepo: healthLogRepo,
	}
}

// RegisterRoutes registers all service management routes with the given router.
// Routes:
//   - POST   /register                  (public) - Self-registration endpoint
//   - POST   /services                  (admin)  - Register a service
//   - DELETE /services/:id              (admin)  - Deregister a service
//   - GET    /services                  (admin)  - List all services
//   - GET    /services/:id              (admin)  - Get service details
//   - GET    /services/:id/health-logs  (admin)  - Get health check history
func RegisterRoutes(router gin.IRouter, reg *core.ServiceRegistry, healthLogRepo *healthlog.Repository, authMiddleware, adminMiddleware gin.HandlerFunc) {
	handler := NewHandler(reg, healthLogRepo)

	// Public self-registration endpoint (no auth required)
	router.POST("/register", handler.handleSelfRegister)

	services := router.Group("/services")
	// All service management endpoints require authentication and admin privileges
	services.Use(authMiddleware, adminMiddleware)
	{
		services.POST("", handler.handleRegisterService)
		services.DELETE("/:id", handler.handleDeregisterService)
		services.GET("", handler.handleListServices)
		services.GET("/:id", handler.handleGetService)
		services.GET("/:id/health-logs", handler.handleGetHealthLogs)
	}
}

// RegisterRequest represents the payload for registering a new service.
type RegisterRequest struct {
	Name            string            `json:"name" binding:"required"`
	Host            string            `json:"host" binding:"required"`
	Port            int               `json:"port" binding:"required"`
	HealthCheckPath string            `json:"health_check_path" binding:"required"`
	Protocol        string            `json:"protocol"`
	Metadata        map[string]string `json:"metadata"`
}

// SelfRegisterRequest represents the payload for self-registration by external services.
// Host and Port are optional - if not provided, they will be auto-detected from the request.
type SelfRegisterRequest struct {
	Name            string            `json:"name" binding:"required"`
	Host            string            `json:"host"`
	Port            int               `json:"port"`
	HealthCheckPath string            `json:"health_check_path" binding:"required"`
	Protocol        string            `json:"protocol"`
	Metadata        map[string]string `json:"metadata"`
}

// handleRegisterService processes service registration requests.
// It validates the health check endpoint before registering the service.
func (h *Handler) handleRegisterService(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create service domain object
	svc := service.NewService(req.Name, req.Host, req.Port, req.HealthCheckPath)
	if req.Protocol != "" {
		svc.Protocol = req.Protocol
	}
	if req.Metadata != nil {
		svc.Metadata = req.Metadata
	}

	// Perform initial health check but allow registration even if unhealthy
	if err := h.checkServiceHealth(svc); err != nil {
		log.Printf("Initial health check failed for %s, registering as unhealthy: %v", svc.Name, err)
		svc.Status = "unhealthy"
	}

	// Register service in the registry
	if err := h.registry.Register(svc); err != nil {
		// Check if it's a duplicate service error
		if err.Error() == "service with this name, host, and port already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Service registered: %s at %s", svc.Name, svc.BaseURL())
	c.JSON(http.StatusCreated, svc)
}

// handleSelfRegister allows external services to register themselves without authentication.
// Host and Port are auto-detected from the request if not provided.
func (h *Handler) handleSelfRegister(c *gin.Context) {
	var req SelfRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Auto-detect host and port from the request if not provided
	if req.Host == "" || req.Port == 0 {
		clientIP := c.ClientIP()

		// Try to get the original host from headers (in case of proxy/forwarding)
		if forwardedFor := c.GetHeader("X-Forwarded-For"); forwardedFor != "" {
			// X-Forwarded-For can contain multiple IPs, take the first one
			ips := strings.Split(forwardedFor, ",")
			clientIP = strings.TrimSpace(ips[0])
		} else if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
			clientIP = realIP
		}

		if req.Host == "" {
			req.Host = clientIP
			log.Printf("Auto-detected host for %s: %s", req.Name, req.Host)
		}

		// If port is not provided, we can't auto-detect it reliably
		// Services should provide their actual service port, not the source port
		if req.Port == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "port must be provided (cannot auto-detect service port)",
			})
			return
		}
	}

	// Set default protocol if not provided
	if req.Protocol == "" {
		req.Protocol = "http"
	}

	// Create service domain object
	svc := service.NewService(req.Name, req.Host, req.Port, req.HealthCheckPath)
	svc.Protocol = req.Protocol
	if req.Metadata != nil {
		svc.Metadata = req.Metadata
	}

	// Perform initial health check but allow registration even if unhealthy
	if err := h.checkServiceHealth(svc); err != nil {
		log.Printf("Initial health check failed for %s (self-registered), registering as unhealthy: %v", svc.Name, err)
		svc.Status = "unhealthy"
	}

	// Register service in the registry
	if err := h.registry.Register(svc); err != nil {
		// Check if it's a duplicate service error
		if err.Error() == "service with this name, host, and port already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Service self-registered: %s at %s (from %s)", svc.Name, svc.BaseURL(), c.ClientIP())
	c.JSON(http.StatusCreated, svc)
}

// checkServiceHealth verifies that a service's health check endpoint is accessible.
// Returns an error if the health check fails or returns a non-2xx status code.
func (h *Handler) checkServiceHealth(svc *service.Service) error {
	startTime := time.Now()
	resp, err := h.healthClient.Get(svc.HealthCheckURL())
	responseTime := time.Since(startTime).Milliseconds()

	if err != nil {
		log.Printf("Health check failed for %s: %v", svc.Name, err)
		// Log the failed health check
		if h.healthLogRepo != nil {
			h.healthLogRepo.Create(svc.ID, "unhealthy", err.Error(), "", responseTime)
		}
		return err
	}
	defer resp.Body.Close()

	// Read response body (limit to 10KB)
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024))
	responseBody := ""
	if err == nil {
		responseBody = string(bodyBytes)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("Health check returned non-2xx status for %s: %d", svc.Name, resp.StatusCode)
		// Log the unhealthy status with response body
		if h.healthLogRepo != nil {
			errorMsg := "HTTP " + strconv.Itoa(resp.StatusCode)
			h.healthLogRepo.Create(svc.ID, "unhealthy", errorMsg, responseBody, responseTime)
		}
		return err
	}

	// Log successful health check with response body
	if h.healthLogRepo != nil {
		h.healthLogRepo.Create(svc.ID, "healthy", "", responseBody, responseTime)
	}

	return nil
}

// handleDeregisterService removes a service from the registry by ID.
func (h *Handler) handleDeregisterService(c *gin.Context) {
	id := c.Param("id")

	if err := h.registry.Deregister(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Service deregistered: %s", id)
	c.JSON(http.StatusOK, gin.H{"message": "service deregistered"})
}

// handleListServices returns all registered services with their current status.
func (h *Handler) handleListServices(c *gin.Context) {
	services := h.registry.List()
	c.JSON(http.StatusOK, gin.H{
		"services": services,
		"count":    len(services),
	})
}

// handleGetService retrieves detailed information about a specific service by ID.
func (h *Handler) handleGetService(c *gin.Context) {
	id := c.Param("id")

	svc, err := h.registry.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, svc)
}

// handleGetHealthLogs retrieves health check logs for a specific service.
func (h *Handler) handleGetHealthLogs(c *gin.Context) {
	id := c.Param("id")

	// Verify service exists
	_, err := h.registry.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}

	// Get limit from query parameter, default to 50
	limit := 50
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	logs, err := h.healthLogRepo.GetByServiceID(id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve health logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"service_id": id,
		"logs":       logs,
		"count":      len(logs),
	})
}
