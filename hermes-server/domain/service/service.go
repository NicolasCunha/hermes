package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Status represents service instance status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDraining  Status = "draining" // For graceful shutdown
)

// Service represents a registered backend service
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

// NewService creates a new service instance
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

// BaseURL returns the full base URL of the service
func (s *Service) BaseURL() string {
	return fmt.Sprintf("%s://%s:%d", s.Protocol, s.Host, s.Port)
}

// HealthCheckURL returns the full health check URL
func (s *Service) HealthCheckURL() string {
	return fmt.Sprintf("%s%s", s.BaseURL(), s.HealthCheckPath)
}

// MarkHealthy marks service as healthy and resets failure count
func (s *Service) MarkHealthy() {
	s.Status = StatusHealthy
	s.FailureCount = 0
	s.LastCheckedAt = time.Now()
}

// MarkUnhealthy increments failure count and marks as unhealthy if threshold reached
func (s *Service) MarkUnhealthy(threshold int) {
	s.FailureCount++
	s.LastCheckedAt = time.Now()

	if s.FailureCount >= threshold {
		s.Status = StatusUnhealthy
	}
}
