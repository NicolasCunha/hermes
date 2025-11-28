package health

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"nfcunha/hermes/hermes-server/domain/service"
	"nfcunha/hermes/hermes-server/services/registry"
)

// Checker performs periodic health checks on registered services
type Checker struct {
	registry         *registry.Registry
	client           *http.Client
	interval         time.Duration
	timeout          time.Duration
	failureThreshold int
	maxFailures      int // Remove service after this many failures
	stopChan         chan struct{}
}

// New creates a new health checker
func New(reg *registry.Registry) *Checker {
	return &Checker{
		registry:         reg,
		client:           &http.Client{Timeout: getTimeout()},
		interval:         getInterval(),
		timeout:          getTimeout(),
		failureThreshold: getFailureThreshold(),
		maxFailures:      getMaxFailures(),
		stopChan:         make(chan struct{}),
	}
}

// Start begins periodic health checking
func (c *Checker) Start() {
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

// Stop stops the health checker
func (c *Checker) Stop() {
	close(c.stopChan)
}

// checkAll checks health of all registered services
func (c *Checker) checkAll() {
	services := c.registry.List()

	log.Printf("Running health checks for %d services", len(services))

	for _, svc := range services {
		go c.check(svc)
	}
}

// check performs health check on a single service
func (c *Checker) check(svc *service.Service) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", svc.HealthCheckURL(), nil)
	if err != nil {
		log.Printf("Failed to create health check request for %s: %v", svc.Name, err)
		c.handleFailure(svc)
		return
	}

	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("Health check failed for %s (%s): %v", svc.Name, svc.ID, err)
		c.handleFailure(svc)
		return
	}
	defer resp.Body.Close()

	// Consider 2xx status codes as healthy
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		svc.MarkHealthy()
		log.Printf("Health check passed for %s (%s): status=%d", svc.Name, svc.ID, resp.StatusCode)
	} else {
		log.Printf("Health check failed for %s (%s): status=%d", svc.Name, svc.ID, resp.StatusCode)
		c.handleFailure(svc)
	}
}

// handleFailure handles a failed health check
func (c *Checker) handleFailure(svc *service.Service) {
	svc.MarkUnhealthy(c.failureThreshold)

	// Remove service if exceeded max failures
	if svc.FailureCount >= c.maxFailures {
		log.Printf("Service %s (%s) exceeded max failures, removing from registry", svc.Name, svc.ID)
		c.registry.Deregister(svc.ID)
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

func getMaxFailures() int {
	val := os.Getenv("HERMES_HEALTH_CHECK_MAX_FAILURES")
	if val == "" {
		return 10
	}
	maxFail, err := strconv.Atoi(val)
	if err != nil {
		return 10
	}
	return maxFail
}
