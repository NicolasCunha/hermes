package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"nfcunha/hermes/hermes-server/services/aegis"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	// Setup mock Aegis server
	aegisServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(aegis.ValidateTokenResponse{
			Valid: true,
			User: &aegis.User{
				ID:          "123",
				Subject:     "test@test.com",
				Roles:       []string{"admin"},
				Permissions: []string{"read:all"},
			},
		})
	}))
	defer aegisServer.Close()

	// Create middleware
	client := aegis.NewClient(aegisServer.URL, 5*time.Second)
	middleware := AuthMiddleware(client)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware)
	router.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		subject, _ := c.Get("user_subject")
		roles, _ := c.Get("user_roles")
		permissions, _ := c.Get("user_permissions")
		
		c.JSON(http.StatusOK, gin.H{
			"user_id":     userID,
			"subject":     subject,
			"roles":       roles,
			"permissions": permissions,
		})
	})

	// Test request
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["user_id"] != "123" {
		t.Errorf("Expected user_id 123, got %v", response["user_id"])
	}
	if response["subject"] != "test@test.com" {
		t.Errorf("Expected subject test@test.com, got %v", response["subject"])
	}
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	client := aegis.NewClient("http://localhost", 5*time.Second)
	middleware := AuthMiddleware(client)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware)
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)
	if response["error"] != "missing authorization token" {
		t.Errorf("Expected error message, got %v", response)
	}
}

func TestAuthMiddleware_InvalidHeaderFormat(t *testing.T) {
	client := aegis.NewClient("http://localhost", 5*time.Second)
	middleware := AuthMiddleware(client)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware)
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	aegisServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(aegis.ValidateTokenResponse{
			Valid: false,
			Error: "token expired",
		})
	}))
	defer aegisServer.Close()

	client := aegis.NewClient(aegisServer.URL, 5*time.Second)
	middleware := AuthMiddleware(client)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware)
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_AegisDown(t *testing.T) {
	// Use invalid URL to simulate Aegis being down
	client := aegis.NewClient("http://invalid-host:9999", 1*time.Second)
	middleware := AuthMiddleware(client)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware)
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestRequireAdmin_WithAdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Simulate authenticated user with admin role
	router.Use(func(c *gin.Context) {
		c.Set("user_roles", []string{"admin"})
		c.Next()
	})
	router.Use(RequireAdmin())
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRequireAdmin_WithoutAdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("user_roles", []string{"viewer"})
		c.Next()
	})
	router.Use(RequireAdmin())
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestRequireAdmin_NoRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(RequireAdmin())
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestRequireAdmin_MultipleRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("user_roles", []string{"viewer", "admin", "manager"})
		c.Next()
	})
	router.Use(RequireAdmin())
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRequirePermission_WithPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("user_permissions", []string{"read:all", "write:all"})
		c.Next()
	})
	router.Use(RequirePermission("read:all"))
	router.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/data", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRequirePermission_WithoutPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("user_permissions", []string{"read:some"})
		c.Next()
	})
	router.Use(RequirePermission("write:all"))
	router.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/data", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestRequirePermission_NoPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(RequirePermission("read:all"))
	router.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/data", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}
