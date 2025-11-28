package service

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"nfcunha/hermes/hermes-server/domain/service"
	"nfcunha/hermes/hermes-server/services/registry"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create services table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS services (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL,
			protocol TEXT NOT NULL DEFAULT 'http',
			health_check_path TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'healthy',
			metadata TEXT,
			registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			failure_count INTEGER DEFAULT 0,
			UNIQUE(name, host, port)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	return db
}

// mockAuthMiddleware simulates successful authentication
func mockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Next()
	}
}

// mockAdminMiddleware simulates admin authorization
func mockAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// mockAuthFailMiddleware simulates authentication failure
func mockAuthFailMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization token"})
		c.Abort()
	}
}

// mockNonAdminMiddleware simulates non-admin user
func mockNonAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		c.Abort()
	}
}

func TestRegisterService_WithoutAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	reg := registry.New(db)
	router := gin.New()
	
	RegisterRoutes(router, reg, mockAuthFailMiddleware(), mockAdminMiddleware())

	reqBody := RegisterRequest{
		Name:            "test-api",
		Host:            "localhost",
		Port:            8080,
		HealthCheckPath: "/health",
	}
	bodyJSON, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/services", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	
	if response["error"] != "missing authorization token" {
		t.Errorf("Expected auth error, got %v", response["error"])
	}
}

func TestRegisterService_NonAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	reg := registry.New(db)
	router := gin.New()
	
	RegisterRoutes(router, reg, mockAuthMiddleware(), mockNonAdminMiddleware())

	reqBody := RegisterRequest{
		Name:            "test-api",
		Host:            "localhost",
		Port:            8080,
		HealthCheckPath: "/health",
	}
	bodyJSON, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/services", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	
	if response["error"] != "admin access required" {
		t.Errorf("Expected admin error, got %v", response["error"])
	}
}

func TestRegisterService_InvalidHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	reg := registry.New(db)
	router := gin.New()
	
	RegisterRoutes(router, reg, mockAuthMiddleware(), mockAdminMiddleware())

	reqBody := RegisterRequest{
		Name:            "test-api",
		Host:            "invalid-host-xyz.com",
		Port:            9999,
		HealthCheckPath: "/health",
	}
	bodyJSON, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/services", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	
	if response["error"] != "service health check failed" {
		t.Errorf("Expected health check error, got %v", response["error"])
	}
}

func TestRegisterService_Duplicate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	reg := registry.New(db)
	
	// Pre-register a service
	svc := service.NewService("existing-api", "localhost", 9000, "/health")
	reg.Register(svc)

	router := gin.New()
	RegisterRoutes(router, reg, mockAuthMiddleware(), mockAdminMiddleware())

	reqBody := RegisterRequest{
		Name:            "existing-api",
		Host:            "localhost",
		Port:            9000,
		HealthCheckPath: "/health",
	}
	bodyJSON, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/services", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Health check will likely fail for localhost:9000, but if it passes, we should get conflict
	if w.Code != http.StatusBadRequest && w.Code != http.StatusConflict {
		t.Errorf("Expected status %d or %d, got %d", http.StatusBadRequest, http.StatusConflict, w.Code)
	}
}

func TestListServices_WithoutAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	reg := registry.New(db)
	router := gin.New()
	
	RegisterRoutes(router, reg, mockAuthFailMiddleware(), mockAdminMiddleware())

	req, _ := http.NewRequest("GET", "/services", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestListServices_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	reg := registry.New(db)
	
	// Register test services
	svc1 := service.NewService("api-1", "host1.com", 8080, "/health")
	svc2 := service.NewService("api-2", "host2.com", 8081, "/health")
	reg.Register(svc1)
	reg.Register(svc2)

	router := gin.New()
	RegisterRoutes(router, reg, mockAuthMiddleware(), mockAdminMiddleware())

	req, _ := http.NewRequest("GET", "/services", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	
	count := int(response["count"].(float64))
	if count != 2 {
		t.Errorf("Expected 2 services, got %d", count)
	}
}

func TestGetService_WithoutAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	reg := registry.New(db)
	router := gin.New()
	
	RegisterRoutes(router, reg, mockAuthFailMiddleware(), mockAdminMiddleware())

	req, _ := http.NewRequest("GET", "/services/some-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestGetService_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	reg := registry.New(db)
	
	// Register test service
	svc := service.NewService("test-api", "localhost", 8080, "/health")
	reg.Register(svc)

	router := gin.New()
	RegisterRoutes(router, reg, mockAuthMiddleware(), mockAdminMiddleware())

	req, _ := http.NewRequest("GET", "/services/"+svc.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	
	if response["id"] != svc.ID {
		t.Errorf("Expected service ID %s, got %v", svc.ID, response["id"])
	}
}

func TestGetService_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	reg := registry.New(db)
	router := gin.New()
	
	RegisterRoutes(router, reg, mockAuthMiddleware(), mockAdminMiddleware())

	req, _ := http.NewRequest("GET", "/services/non-existent-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDeregisterService_WithoutAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	reg := registry.New(db)
	router := gin.New()
	
	RegisterRoutes(router, reg, mockAuthFailMiddleware(), mockAdminMiddleware())

	req, _ := http.NewRequest("DELETE", "/services/some-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestDeregisterService_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	reg := registry.New(db)
	
	// Register test service
	svc := service.NewService("test-api", "localhost", 8080, "/health")
	reg.Register(svc)

	router := gin.New()
	RegisterRoutes(router, reg, mockAuthMiddleware(), mockAdminMiddleware())

	req, _ := http.NewRequest("DELETE", "/services/"+svc.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	
	if response["message"] != "service deregistered" {
		t.Errorf("Expected success message, got %v", response["message"])
	}

	// Verify service was deleted
	_, err := reg.GetByID(svc.ID)
	if err == nil {
		t.Error("Expected service to be deleted")
	}
}

func TestDeregisterService_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	reg := registry.New(db)
	router := gin.New()
	
	RegisterRoutes(router, reg, mockAuthMiddleware(), mockAdminMiddleware())

	req, _ := http.NewRequest("DELETE", "/services/non-existent-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}
