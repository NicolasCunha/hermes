package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidateToken_ValidToken(t *testing.T) {
	// Mock Aegis server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/aegis/api/auth/validate" {
			t.Errorf("Expected /aegis/api/auth/validate, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ValidateTokenResponse{
			Valid: true,
			User: &AegisUser{
				ID:          "123",
				Subject:     "test@test.com",
				Roles:       []string{"admin"},
				Permissions: []string{"read:all"},
			},
			ExpiresAt: time.Now().Add(1 * time.Hour),
		})
	}))
	defer server.Close()

	client := NewAegisClient(server.URL, 5*time.Second)
	resp, err := client.ValidateToken("valid-token")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !resp.Valid {
		t.Error("Expected valid=true")
	}
	if resp.User == nil || resp.User.Subject != "test@test.com" {
		t.Error("Expected user data")
	}
	if len(resp.User.Roles) != 1 || resp.User.Roles[0] != "admin" {
		t.Error("Expected admin role")
	}
	if len(resp.User.Permissions) != 1 || resp.User.Permissions[0] != "read:all" {
		t.Error("Expected read:all permission")
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ValidateTokenResponse{
			Valid: false,
			Error: "token expired",
		})
	}))
	defer server.Close()

	client := NewAegisClient(server.URL, 5*time.Second)
	resp, err := client.ValidateToken("invalid-token")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if resp.Valid {
		t.Error("Expected valid=false")
	}
	if resp.Error != "token expired" {
		t.Errorf("Expected error message, got %s", resp.Error)
	}
}

func TestValidateToken_MalformedToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ValidateTokenResponse{
			Valid: false,
			Error: "malformed token",
		})
	}))
	defer server.Close()

	client := NewAegisClient(server.URL, 5*time.Second)
	resp, err := client.ValidateToken("malformed.token")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if resp.Valid {
		t.Error("Expected valid=false")
	}
}

func TestValidateToken_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewAegisClient(server.URL, 5*time.Second)
	_, err := client.ValidateToken("any-token")

	if err == nil {
		t.Error("Expected error for server error")
	}
}

func TestValidateToken_NetworkError(t *testing.T) {
	// Use invalid URL to trigger network error
	client := NewAegisClient("http://invalid-host-that-does-not-exist:9999", 1*time.Second)
	_, err := client.ValidateToken("any-token")

	if err == nil {
		t.Error("Expected error for network failure")
	}
}

func TestHealth_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/aegis/health" {
			t.Errorf("Expected /aegis/health, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()

	client := NewAegisClient(server.URL, 5*time.Second)
	err := client.Health()

	if err != nil {
		t.Errorf("Expected health check to pass, got %v", err)
	}
}

func TestHealth_ServiceDown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewAegisClient(server.URL, 5*time.Second)
	err := client.Health()

	if err == nil {
		t.Error("Expected error for unhealthy service")
	}
}

func TestHealth_NetworkError(t *testing.T) {
	client := NewAegisClient("http://invalid-host-that-does-not-exist:9999", 1*time.Second)
	err := client.Health()

	if err == nil {
		t.Error("Expected error for network failure")
	}
}

func TestNewAegisClient(t *testing.T) {
	baseURL := "http://localhost:8080"
	timeout := 10 * time.Second

	client := NewAegisClient(baseURL, timeout)

	if client.baseURL != baseURL {
		t.Errorf("Expected baseURL %s, got %s", baseURL, client.baseURL)
	}
	if client.httpClient.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, client.httpClient.Timeout)
	}
}
