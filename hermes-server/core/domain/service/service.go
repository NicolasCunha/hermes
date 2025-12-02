// Package service defines the domain model for registered backend services.
package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Status represents the health status of a service instance.
type Status string

const (
	// StatusHealthy indicates the service is responding to health checks.
	StatusHealthy   Status = "healthy"
	// StatusUnhealthy indicates the service has failed health checks.
	StatusUnhealthy Status = "unhealthy"
	// StatusDraining indicates the service is being gracefully shut down.
	StatusDraining  Status = "draining"
)

// Service represents a registered backend service instance.
// It contains connection details, health status, and metadata.
type Service struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Host            string            `json:"host"`
	Port            int               `json:"port"`
	Protocol        string            `json:"protocol"` // http, https
	HealthCheckPath string            `json:"health_check_path"`
	Status          Status            `json:"status"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	RegisteredAt    time.Time         `json:"registered_at"`
	LastCheckedAt   time.Time         `json:"last_checked_at"`
	FailureCount    int               `json:"failure_count"`
}

// NewService creates a new service instance with the given parameters.
// It generates a unique ID and initializes the service in healthy status.
// The protocol defaults to "http" and can be changed after creation.
func NewService(name, host string, port int, healthCheckPath string) *Service {
	return &Service{
		ID:              uuid.New().String(),
		Name:            name,
		Host:            host,
		Port:            port,
		Protocol:        "http", // Default
		HealthCheckPath: healthCheckPath,
		Status:          StatusHealthy,
		Metadata:        make(map[string]string),
		RegisteredAt:    time.Now(),
		LastCheckedAt:   time.Now(),
		FailureCount:    0,
	}
}

// BaseURL returns the full base URL of the service.
// Example: "http://api-server:8080"
func (s *Service) BaseURL() string {
	return fmt.Sprintf("%s://%s:%d", s.Protocol, s.Host, s.Port)
}

// HealthCheckURL returns the full health check URL.
// Example: "http://api-server:8080/health"
func (s *Service) HealthCheckURL() string {
	return fmt.Sprintf("%s%s", s.BaseURL(), s.HealthCheckPath)
}

// MarkHealthy marks the service as healthy and resets the failure count.
// This should be called when a health check succeeds.
func (s *Service) MarkHealthy() {
	s.Status = StatusHealthy
	s.FailureCount = 0
	s.LastCheckedAt = time.Now()
}

// MarkUnhealthy increments the failure count and marks as unhealthy if threshold reached.
// The threshold parameter specifies how many consecutive failures trigger unhealthy status.
// This should be called when a health check fails.
func (s *Service) MarkUnhealthy(threshold int) {
	s.FailureCount++
	s.LastCheckedAt = time.Now()

	if s.FailureCount >= threshold {
		s.Status = StatusUnhealthy
	}
}
