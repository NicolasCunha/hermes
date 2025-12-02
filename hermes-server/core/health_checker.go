package core

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"nfcunha/hermes/hermes-server/core/domain/healthlog"
	"nfcunha/hermes/hermes-server/core/domain/service"
)

// HealthChecker performs periodic health checks on registered services.
// It runs in a background goroutine and updates service status based on
// health check results. Health check logs are persisted to the database
// for historical analysis and debugging.
type HealthChecker struct {
	registry         *ServiceRegistry
	client           *http.Client
	interval         time.Duration
	timeout          time.Duration
	failureThreshold int
	stopChan         chan struct{}
	healthLogRepo    *healthlog.Repository
}

// NewHealthChecker creates a new health checker with the given registry and health log repository.
// Configuration is loaded from environment variables:
//   - HERMES_HEALTH_CHECK_INTERVAL: how often to check (default: 30s)
//   - HERMES_HEALTH_CHECK_TIMEOUT: HTTP timeout for checks (default: 5s)
//   - HERMES_HEALTH_CHECK_THRESHOLD: failures before marking unhealthy (default: 3)
func NewHealthChecker(reg *ServiceRegistry, healthLogRepo *healthlog.Repository) *HealthChecker {
	return &HealthChecker{
		registry:         reg,
		client:           &http.Client{Timeout: getTimeout()},
		interval:         getInterval(),
		timeout:          getTimeout(),
		failureThreshold: getFailureThreshold(),
		stopChan:         make(chan struct{}),
		healthLogRepo:    healthLogRepo,
	}
}

// Start begins periodic health checking in the current goroutine.
// This method blocks until Stop() is called, so it should typically be
// run in a separate goroutine using: go checker.Start()
func (c *HealthChecker) Start() {
	log.Printf("Starting health checker: interval=%v, timeout=%v, threshold=%d",
		c.interval, c.timeout, c.failureThreshold)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.checkAll()
		case <-c.stopChan:
			log.Println("Health checker stopped")
			return
		}
	}
}

// Stop signals the health checker to stop and waits for it to complete.
// This method is safe to call multiple times.
func (c *HealthChecker) Stop() {
	close(c.stopChan)
}

// checkAll checks health of all registered services
func (c *HealthChecker) checkAll() {
	services := c.registry.List()

	log.Printf("Running health checks for %d services", len(services))

	for _, svc := range services {
		go c.check(svc)
	}
}

// check performs health check on a single service
func (c *HealthChecker) check(svc *service.Service) {
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", svc.HealthCheckURL(), nil)
	if err != nil {
		log.Printf("Failed to create health check request for %s: %v", svc.Name, err)
		c.logHealthCheck(svc.ID, "error", err.Error(), "", 0)
		c.handleFailure(svc)
		return
	}

	resp, err := c.client.Do(req)
	responseTime := time.Since(startTime).Milliseconds()

	if err != nil {
		log.Printf("Health check failed for %s (%s): %v", svc.Name, svc.ID, err)
		c.logHealthCheck(svc.ID, "unhealthy", err.Error(), "", responseTime)
		c.handleFailure(svc)
		return
	}
	defer resp.Body.Close()

	// Read response body (limit to 10KB to avoid memory issues)
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024))
	responseBody := ""
	if err == nil {
		responseBody = string(bodyBytes)
	}

	// Consider 2xx status codes as healthy
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		svc.MarkHealthy()
		// Persist status change to database
		if err := c.registry.UpdateStatus(svc.ID, svc.Status); err != nil {
			log.Printf("Failed to persist healthy status for %s: %v", svc.Name, err)
		}
		c.logHealthCheck(svc.ID, "healthy", "", responseBody, responseTime)
		log.Printf("Health check passed for %s (%s): status=%d, time=%dms", svc.Name, svc.ID, resp.StatusCode, responseTime)
	} else {
		errorMsg := "HTTP " + strconv.Itoa(resp.StatusCode)
		log.Printf("Health check failed for %s (%s): status=%d", svc.Name, svc.ID, resp.StatusCode)
		c.logHealthCheck(svc.ID, "unhealthy", errorMsg, responseBody, responseTime)
		c.handleFailure(svc)
	}
}

// handleFailure handles a failed health check
func (c *HealthChecker) handleFailure(svc *service.Service) {
	svc.MarkUnhealthy(c.failureThreshold)
	// Persist status change to database
	if err := c.registry.UpdateStatus(svc.ID, svc.Status); err != nil {
		log.Printf("Failed to persist unhealthy status for %s: %v", svc.Name, err)
	}
}

// Configuration helpers with defaults
func getInterval() time.Duration {
	val := os.Getenv("HERMES_HEALTH_CHECK_INTERVAL")
	if val == "" {
		return 30 * time.Second
	}
	duration, err := time.ParseDuration(val)
	if err != nil {
		return 30 * time.Second
	}
	return duration
}

func getTimeout() time.Duration {
	val := os.Getenv("HERMES_HEALTH_CHECK_TIMEOUT")
	if val == "" {
		return 5 * time.Second
	}
	duration, err := time.ParseDuration(val)
	if err != nil {
		return 5 * time.Second
	}
	return duration
}

func getFailureThreshold() int {
	val := os.Getenv("HERMES_HEALTH_CHECK_THRESHOLD")
	if val == "" {
		return 3
	}
	threshold, err := strconv.Atoi(val)
	if err != nil {
		return 3
	}
	return threshold
}

// logHealthCheck stores health check result in the database
func (c *HealthChecker) logHealthCheck(serviceID, status, errorMsg, responseBody string, responseTimeMs int64) {
	if c.healthLogRepo == nil {
		return
	}

	if err := c.healthLogRepo.Create(serviceID, status, errorMsg, responseBody, responseTimeMs); err != nil {
		log.Printf("Failed to log health check for service %s: %v", serviceID, err)
	}
}
