package user

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"nfcunha/hermes/hermes-server/middleware"
	"nfcunha/hermes/hermes-server/services/aegis"
)

// API handles user management
type API struct {
	aegisClient *aegis.Client
	aegisURL    string
}

// NewAPI creates user management API
func NewAPI(client *aegis.Client, aegisURL string) *API {
	return &API{
		aegisClient: client,
		aegisURL:    aegisURL,
	}
}

// RegisterRoutes registers user management routes
func (a *API) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	// Login endpoint (public - no auth required)
	router.POST("/login", a.login)
	
	// User management endpoints (authenticated)
	users := router.Group("/users")
	users.Use(authMiddleware) // All endpoints require authentication
	{
		// Admin-only endpoints
		adminOnly := users.Group("")
		adminOnly.Use(middleware.RequireAdmin())
		{
			adminOnly.POST("/register", a.registerUser)
			adminOnly.GET("", a.listUsers)
			adminOnly.GET("/:id", a.getUser)
			adminOnly.PUT("/:id", a.updateUser)
			adminOnly.DELETE("/:id", a.deleteUser)
			adminOnly.POST("/:id/roles", a.addRole)
			adminOnly.DELETE("/:id/roles/:roleId", a.removeRole)
			adminOnly.POST("/:id/permissions", a.addPermission)
			adminOnly.DELETE("/:id/permissions/:permissionId", a.removePermission)
		}
		
		// Self-service endpoint: any user can change their own password
		users.POST("/:id/password", a.changePassword)
	}
}

// login forwards login request to Aegis
func (a *API) login(c *gin.Context) {
	targetURL := a.aegisURL + "/aegis/users/login"
	log.Printf("Login request - forwarding to Aegis: %s", targetURL)
	
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request"})
		return
	}
	
	req, err := http.NewRequest("POST", targetURL, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Aegis login request failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "authentication service unavailable"})
		return
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}
	
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// registerUser forwards registration to Aegis
func (a *API) registerUser(c *gin.Context) {
	targetURL := a.aegisURL + "/aegis/users/register"
	log.Printf("Register user request - forwarding to Aegis: %s", targetURL)
	
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request"})
		return
	}
	
	req, err := http.NewRequest("POST", targetURL, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Aegis registration failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "authentication service unavailable"})
		return
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}
	
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// listUsers fetches all users from Aegis
func (a *API) listUsers(c *gin.Context) {
	targetURL := a.aegisURL + "/aegis/users"
	log.Printf("List users request - fetching from Aegis")
	
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to fetch users from Aegis: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "authentication service unavailable"})
		return
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}
	
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// getUser fetches specific user from Aegis
func (a *API) getUser(c *gin.Context) {
	userID := c.Param("id")
	targetURL := fmt.Sprintf("%s/aegis/users/%s", a.aegisURL, userID)
	log.Printf("Get user %s - fetching from Aegis", userID)
	
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to fetch user from Aegis: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "authentication service unavailable"})
		return
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}
	
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// updateUser forwards update to Aegis
func (a *API) updateUser(c *gin.Context) {
	userID := c.Param("id")
	targetURL := fmt.Sprintf("%s/aegis/users/%s", a.aegisURL, userID)
	log.Printf("Update user %s - forwarding to Aegis", userID)
	
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request"})
		return
	}
	
	req, err := http.NewRequest("PUT", targetURL, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to update user in Aegis: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "authentication service unavailable"})
		return
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}
	
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// deleteUser forwards deletion to Aegis
func (a *API) deleteUser(c *gin.Context) {
	userID := c.Param("id")
	targetURL := fmt.Sprintf("%s/aegis/users/%s", a.aegisURL, userID)
	log.Printf("Delete user %s - forwarding to Aegis", userID)
	
	req, err := http.NewRequest("DELETE", targetURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to delete user in Aegis: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "authentication service unavailable"})
		return
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}
	
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// addRole forwards role addition to Aegis
func (a *API) addRole(c *gin.Context) {
	userID := c.Param("id")
	targetURL := fmt.Sprintf("%s/aegis/users/%s/roles", a.aegisURL, userID)
	log.Printf("Add role to user %s - forwarding to Aegis", userID)
	
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request"})
		return
	}
	
	req, err := http.NewRequest("POST", targetURL, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to add role in Aegis: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "authentication service unavailable"})
		return
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}
	
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// removeRole forwards role removal to Aegis
func (a *API) removeRole(c *gin.Context) {
	userID := c.Param("id")
	roleID := c.Param("roleId")
	targetURL := fmt.Sprintf("%s/aegis/users/%s/roles/%s", a.aegisURL, userID, roleID)
	log.Printf("Remove role %s from user %s - forwarding to Aegis", roleID, userID)
	
	req, err := http.NewRequest("DELETE", targetURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to remove role in Aegis: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "authentication service unavailable"})
		return
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}
	
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// addPermission forwards permission addition to Aegis
func (a *API) addPermission(c *gin.Context) {
	userID := c.Param("id")
	targetURL := fmt.Sprintf("%s/aegis/users/%s/permissions", a.aegisURL, userID)
	log.Printf("Add permission to user %s - forwarding to Aegis", userID)
	
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request"})
		return
	}
	
	req, err := http.NewRequest("POST", targetURL, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to add permission in Aegis: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "authentication service unavailable"})
		return
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}
	
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// removePermission forwards permission removal to Aegis
func (a *API) removePermission(c *gin.Context) {
	userID := c.Param("id")
	permissionID := c.Param("permissionId")
	targetURL := fmt.Sprintf("%s/aegis/users/%s/permissions/%s", a.aegisURL, userID, permissionID)
	log.Printf("Remove permission %s from user %s - forwarding to Aegis", permissionID, userID)
	
	req, err := http.NewRequest("DELETE", targetURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to remove permission in Aegis: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "authentication service unavailable"})
		return
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}
	
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// changePassword allows users to change their own password (or admins to change any password)
func (a *API) changePassword(c *gin.Context) {
	userID := c.Param("id")
	authenticatedUserID, _ := c.Get("user_id")

	// Validate authorization: users can only change their own password (unless admin)
	roles, _ := c.Get("user_roles")
	userRoles, ok := roles.([]string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid roles format"})
		return
	}

	isAdmin := false
	for _, role := range userRoles {
		if role == "admin" {
			isAdmin = true
			break
		}
	}

	if !isAdmin && userID != authenticatedUserID {
		log.Printf("User %s attempted to change password for user %s", authenticatedUserID, userID)
		c.JSON(http.StatusForbidden, gin.H{"error": "can only change your own password"})
		return
	}

	// Forward to Aegis
	targetURL := fmt.Sprintf("%s/aegis/users/%s/password", a.aegisURL, userID)
	log.Printf("Change password for user %s - forwarding to Aegis", userID)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request"})
		return
	}

	req, err := http.NewRequest("POST", targetURL, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to change password in Aegis: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "authentication service unavailable"})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}
