package user

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"nfcunha/hermes/hermes-server/core"
)

func TestProxyToAegis_ListUsers(t *testing.T) {
	// Mock Aegis server
	aegisServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/aegis/users" {
			t.Errorf("Expected /aegis/users, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{
			{"id": "1", "subject": "test@test.com"},
		})
	}))
	defer aegisServer.Close()

	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	client := core.NewAegisClient(aegisServer.URL, 5*time.Second)
	handler := NewHandler(client, aegisServer.URL)

	// Mock auth middleware
	mockAuth := func(c *gin.Context) {
		c.Set("user_id", "admin-id")
		c.Set("user_roles", []string{"admin"})
		c.Next()
	}

	handler.RegisterRoutes(router.Group("/hermes"), mockAuth)

	// Test
	req := httptest.NewRequest("GET", "/hermes/users", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProxyToAegis_CreateUser(t *testing.T) {
	aegisServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/aegis/users/register" {
			t.Errorf("Expected /aegis/users/register, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["subject"] != "new@test.com" {
			t.Error("Expected subject in request")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "new-user-id"})
	}))
	defer aegisServer.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	client := core.NewAegisClient(aegisServer.URL, 5*time.Second)
	handler := NewHandler(client, aegisServer.URL)

	mockAuth := func(c *gin.Context) {
		c.Set("user_id", "admin-id")
		c.Set("user_roles", []string{"admin"})
		c.Next()
	}

	handler.RegisterRoutes(router.Group("/hermes"), mockAuth)

	reqBody := map[string]string{
		"subject":  "new@test.com",
		"password": "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/hermes/users/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProxyToAegis_GetUser(t *testing.T) {
	aegisServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/aegis/users/user-123" {
			t.Errorf("Expected /aegis/users/user-123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "user-123", "subject": "test@test.com"})
	}))
	defer aegisServer.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	client := core.NewAegisClient(aegisServer.URL, 5*time.Second)
	handler := NewHandler(client, aegisServer.URL)

	mockAuth := func(c *gin.Context) {
		c.Set("user_id", "admin-id")
		c.Set("user_roles", []string{"admin"})
		c.Next()
	}

	handler.RegisterRoutes(router.Group("/hermes"), mockAuth)

	req := httptest.NewRequest("GET", "/hermes/users/user-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestChangePassword_OwnPassword(t *testing.T) {
	aegisServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/aegis/users/user-123/password" {
			t.Errorf("Expected /aegis/users/user-123/password, got %s", r.URL.Path)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)

		if body["old_password"] != "old123" || body["new_password"] != "new123" {
			t.Error("Expected password fields in request")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "password changed"})
	}))
	defer aegisServer.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	client := core.NewAegisClient(aegisServer.URL, 5*time.Second)
	handler := NewHandler(client, aegisServer.URL)

	mockAuth := func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.Set("user_roles", []string{"viewer"})
		c.Next()
	}

	handler.RegisterRoutes(router.Group("/hermes"), mockAuth)

	reqBody := map[string]string{
		"old_password": "old123",
		"new_password": "new123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/hermes/users/user-123/password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChangePassword_OtherUserPassword_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	client := core.NewAegisClient("http://localhost", 5*time.Second)
	handler := NewHandler(client, "http://localhost")

	mockAuth := func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.Set("user_roles", []string{"viewer"}) // Not admin
		c.Next()
	}

	handler.RegisterRoutes(router.Group("/hermes"), mockAuth)

	req := httptest.NewRequest("POST", "/hermes/users/other-user-id/password", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestChangePassword_AdminCanChangeAnyPassword(t *testing.T) {
	aegisServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "password changed"})
	}))
	defer aegisServer.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	client := core.NewAegisClient(aegisServer.URL, 5*time.Second)
	handler := NewHandler(client, aegisServer.URL)

	mockAuth := func(c *gin.Context) {
		c.Set("user_id", "admin-123")
		c.Set("user_roles", []string{"admin"})
		c.Next()
	}

	handler.RegisterRoutes(router.Group("/hermes"), mockAuth)

	reqBody := map[string]string{
		"old_password": "old123",
		"new_password": "new123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/hermes/users/other-user-id/password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProxyToAegis_AddRole(t *testing.T) {
	aegisServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/aegis/users/user-123/roles" {
			t.Errorf("Expected /aegis/users/user-123/roles, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer aegisServer.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	client := core.NewAegisClient(aegisServer.URL, 5*time.Second)
	handler := NewHandler(client, aegisServer.URL)

	mockAuth := func(c *gin.Context) {
		c.Set("user_id", "admin-id")
		c.Set("user_roles", []string{"admin"})
		c.Next()
	}

	handler.RegisterRoutes(router.Group("/hermes"), mockAuth)

	reqBody := map[string]string{"role": "manager"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/hermes/users/user-123/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestProxyToAegis_AegisDown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Use invalid URL
	client := core.NewAegisClient("http://invalid-host:9999", 1*time.Second)
	handler := NewHandler(client, "http://invalid-host:9999")

	mockAuth := func(c *gin.Context) {
		c.Set("user_id", "admin-id")
		c.Set("user_roles", []string{"admin"})
		c.Next()
	}

	handler.RegisterRoutes(router.Group("/hermes"), mockAuth)

	req := httptest.NewRequest("GET", "/hermes/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected status 502, got %d", w.Code)
	}
}
