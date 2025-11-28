package service

import (
	"testing"
	"time"
)

func TestNewService(t *testing.T) {
	svc := NewService("test-service", "localhost", 8080, "/health")

	if svc.Name != "test-service" {
		t.Errorf("Expected name 'test-service', got %s", svc.Name)
	}
	if svc.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got %s", svc.Host)
	}
	if svc.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", svc.Port)
	}
	if svc.HealthCheckPath != "/health" {
		t.Errorf("Expected health check path '/health', got %s", svc.HealthCheckPath)
	}
	if svc.Status != StatusHealthy {
		t.Errorf("Expected status %s, got %s", StatusHealthy, svc.Status)
	}
	if svc.Protocol != "http" {
		t.Errorf("Expected protocol 'http', got %s", svc.Protocol)
	}
	if svc.FailureCount != 0 {
		t.Errorf("Expected failure count 0, got %d", svc.FailureCount)
	}
	if svc.ID == "" {
		t.Error("Expected non-empty ID")
	}
}

func TestService_BaseURL(t *testing.T) {
	svc := NewService("test-service", "localhost", 8080, "/health")
	expected := "http://localhost:8080"

	if svc.BaseURL() != expected {
		t.Errorf("Expected base URL %s, got %s", expected, svc.BaseURL())
	}
}

func TestService_BaseURL_HTTPS(t *testing.T) {
	svc := NewService("test-service", "localhost", 8443, "/health")
	svc.Protocol = "https"
	expected := "https://localhost:8443"

	if svc.BaseURL() != expected {
		t.Errorf("Expected base URL %s, got %s", expected, svc.BaseURL())
	}
}

func TestService_HealthCheckURL(t *testing.T) {
	svc := NewService("test-service", "localhost", 8080, "/health")
	expected := "http://localhost:8080/health"

	if svc.HealthCheckURL() != expected {
		t.Errorf("Expected health check URL %s, got %s", expected, svc.HealthCheckURL())
	}
}

func TestService_MarkHealthy(t *testing.T) {
	svc := NewService("test-service", "localhost", 8080, "/health")
	svc.FailureCount = 5
	svc.Status = StatusUnhealthy
	beforeTime := time.Now()

	svc.MarkHealthy()

	if svc.Status != StatusHealthy {
		t.Errorf("Expected status %s, got %s", StatusHealthy, svc.Status)
	}
	if svc.FailureCount != 0 {
		t.Errorf("Expected failure count 0, got %d", svc.FailureCount)
	}
	if svc.LastCheckedAt.Before(beforeTime) {
		t.Error("Expected LastCheckedAt to be updated")
	}
}

func TestService_MarkUnhealthy(t *testing.T) {
	svc := NewService("test-service", "localhost", 8080, "/health")
	beforeTime := time.Now()

	// First failure - still healthy
	svc.MarkUnhealthy(3)
	if svc.Status != StatusHealthy {
		t.Errorf("Expected status %s after 1 failure, got %s", StatusHealthy, svc.Status)
	}
	if svc.FailureCount != 1 {
		t.Errorf("Expected failure count 1, got %d", svc.FailureCount)
	}

	// Second failure - still healthy
	svc.MarkUnhealthy(3)
	if svc.Status != StatusHealthy {
		t.Errorf("Expected status %s after 2 failures, got %s", StatusHealthy, svc.Status)
	}
	if svc.FailureCount != 2 {
		t.Errorf("Expected failure count 2, got %d", svc.FailureCount)
	}

	// Third failure - now unhealthy
	svc.MarkUnhealthy(3)
	if svc.Status != StatusUnhealthy {
		t.Errorf("Expected status %s after 3 failures, got %s", StatusUnhealthy, svc.Status)
	}
	if svc.FailureCount != 3 {
		t.Errorf("Expected failure count 3, got %d", svc.FailureCount)
	}
	if svc.LastCheckedAt.Before(beforeTime) {
		t.Error("Expected LastCheckedAt to be updated")
	}
}

func TestService_MarkUnhealthy_ThresholdOne(t *testing.T) {
	svc := NewService("test-service", "localhost", 8080, "/health")

	svc.MarkUnhealthy(1)
	if svc.Status != StatusUnhealthy {
		t.Errorf("Expected status %s with threshold 1, got %s", StatusUnhealthy, svc.Status)
	}
}

func TestService_Metadata(t *testing.T) {
	svc := NewService("test-service", "localhost", 8080, "/health")
	svc.Metadata["version"] = "1.0.0"
	svc.Metadata["environment"] = "production"

	if svc.Metadata["version"] != "1.0.0" {
		t.Errorf("Expected metadata version '1.0.0', got %s", svc.Metadata["version"])
	}
	if svc.Metadata["environment"] != "production" {
		t.Errorf("Expected metadata environment 'production', got %s", svc.Metadata["environment"])
	}
}
