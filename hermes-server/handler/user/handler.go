// Package user provides HTTP handlers for user management.
// All operations are proxied to the Aegis authentication service.
package user

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"nfcunha/hermes/hermes-server/core"
	"nfcunha/hermes/hermes-server/handler/middleware"
)

// Handler manages user-related HTTP requests.
// It acts as a reverse proxy to the Aegis authentication service,
// forwarding all user operations and returning responses transparently.
type Handler struct {
	aegisClient *core.AegisClient
	aegisURL    string
	httpClient  *http.Client
}

// NewHandler creates a new user handler with the given Aegis client and URL.
func NewHandler(client *core.AegisClient, aegisURL string) *Handler {
	return &Handler{
		aegisClient: client,
		aegisURL:    aegisURL,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// RegisterRoutes registers all user management routes with the given router.
// Routes:
//   - POST   /users/login                   (public) - Authenticate and get JWT token
//   - POST   /users/register                (admin)  - Create a new user
//   - GET    /users                         (admin)  - List all users
//   - GET    /users/:id                     (admin)  - Get user details
//   - PUT    /users/:id                     (admin)  - Update user
//   - DELETE /users/:id                     (admin)  - Delete user
//   - POST   /users/:id/roles               (admin)  - Add role to user
//   - DELETE /users/:id/roles/:roleId       (admin)  - Remove role from user
//   - POST   /users/:id/permissions         (admin)  - Add permission to user
//   - DELETE /users/:id/permissions/:permId (admin)  - Remove permission from user
//   - PUT    /users/:id/password            (auth)   - Change password (own or admin)
func (h *Handler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	// User management endpoints
	users := router.Group("/users")
	{
		// Login endpoint (public - no auth required)
		users.POST("/login", h.handleLogin)

		// Authenticated endpoints
		authenticated := users.Group("")
		authenticated.Use(authMiddleware)
		{
			// Admin-only endpoints
			adminOnly := authenticated.Group("")
			adminOnly.Use(middleware.RequireAdmin())
			{
				adminOnly.POST("/register", h.handleRegisterUser)
				adminOnly.GET("", h.handleListUsers)
				adminOnly.GET("/:id", h.handleGetUser)
				adminOnly.PUT("/:id", h.handleUpdateUser)
				adminOnly.DELETE("/:id", h.handleDeleteUser)
				adminOnly.POST("/:id/roles", h.handleAddRole)
				adminOnly.DELETE("/:id/roles/:roleId", h.handleRemoveRole)
				adminOnly.POST("/:id/permissions", h.handleAddPermission)
				adminOnly.DELETE("/:id/permissions/:permissionId", h.handleRemovePermission)
			}

			// Self-service endpoint: any user can change their own password
			authenticated.PUT("/:id/password", h.handleChangePassword)
		}
	}
}

// handleLogin processes user login requests by forwarding credentials to Aegis.
// Returns JWT tokens on successful authentication.
func (h *Handler) handleLogin(c *gin.Context) {
	body, err := readRequestBody(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request"})
		return
	}

	respBody, statusCode, err := h.proxyToAegis("POST", "/aegis/users/login", body)
	if err != nil {
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.Data(statusCode, "application/json", respBody)
}

// handleRegisterUser creates a new user by forwarding the request to Aegis.
// Only admin users can register new users.
func (h *Handler) handleRegisterUser(c *gin.Context) {
	body, err := readRequestBody(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request"})
		return
	}

	respBody, statusCode, err := h.proxyToAegis("POST", "/aegis/users/register", body)
	if err != nil {
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.Data(statusCode, "application/json", respBody)
}

// handleListUsers retrieves all users from Aegis.
// Only admin users can list users.
func (h *Handler) handleListUsers(c *gin.Context) {
	respBody, statusCode, err := h.proxyToAegis("GET", "/aegis/users", nil)
	if err != nil {
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.Data(statusCode, "application/json", respBody)
}

// handleGetUser retrieves a specific user by ID from Aegis.
// Only admin users can view user details.
func (h *Handler) handleGetUser(c *gin.Context) {
	userID := c.Param("id")
	path := fmt.Sprintf("/aegis/users/%s", userID)

	respBody, statusCode, err := h.proxyToAegis("GET", path, nil)
	if err != nil {
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.Data(statusCode, "application/json", respBody)
}

// handleUpdateUser updates user information by forwarding to Aegis.
// Only admin users can update user details.
func (h *Handler) handleUpdateUser(c *gin.Context) {
	userID := c.Param("id")
	path := fmt.Sprintf("/aegis/users/%s", userID)

	body, err := readRequestBody(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request"})
		return
	}

	respBody, statusCode, err := h.proxyToAegis("PUT", path, body)
	if err != nil {
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.Data(statusCode, "application/json", respBody)
}

// handleDeleteUser removes a user by forwarding the request to Aegis.
// Only admin users can delete users.
func (h *Handler) handleDeleteUser(c *gin.Context) {
	userID := c.Param("id")
	path := fmt.Sprintf("/aegis/users/%s", userID)

	respBody, statusCode, err := h.proxyToAegis("DELETE", path, nil)
	if err != nil {
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.Data(statusCode, "application/json", respBody)
}

// handleAddRole adds a role to a user by forwarding to Aegis.
// Only admin users can manage roles.
func (h *Handler) handleAddRole(c *gin.Context) {
	userID := c.Param("id")
	path := fmt.Sprintf("/aegis/users/%s/roles", userID)

	body, err := readRequestBody(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request"})
		return
	}

	respBody, statusCode, err := h.proxyToAegis("POST", path, body)
	if err != nil {
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.Data(statusCode, "application/json", respBody)
}

// handleRemoveRole removes a role from a user by forwarding to Aegis.
// Only admin users can manage roles.
func (h *Handler) handleRemoveRole(c *gin.Context) {
	userID := c.Param("id")
	roleID := c.Param("roleId")
	path := fmt.Sprintf("/aegis/users/%s/roles/%s", userID, roleID)

	respBody, statusCode, err := h.proxyToAegis("DELETE", path, nil)
	if err != nil {
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.Data(statusCode, "application/json", respBody)
}

// handleAddPermission adds a permission to a user by forwarding to Aegis.
// Only admin users can manage permissions.
func (h *Handler) handleAddPermission(c *gin.Context) {
	userID := c.Param("id")
	path := fmt.Sprintf("/aegis/users/%s/permissions", userID)

	body, err := readRequestBody(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request"})
		return
	}

	respBody, statusCode, err := h.proxyToAegis("POST", path, body)
	if err != nil {
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.Data(statusCode, "application/json", respBody)
}

// handleRemovePermission removes a permission from a user by forwarding to Aegis.
// Only admin users can manage permissions.
func (h *Handler) handleRemovePermission(c *gin.Context) {
	userID := c.Param("id")
	permissionID := c.Param("permissionId")
	path := fmt.Sprintf("/aegis/users/%s/permissions/%s", userID, permissionID)

	respBody, statusCode, err := h.proxyToAegis("DELETE", path, nil)
	if err != nil {
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.Data(statusCode, "application/json", respBody)
}

// handleChangePassword allows users to change passwords.
// Users can change their own password, admins can change any password.
func (h *Handler) handleChangePassword(c *gin.Context) {
	userID := c.Param("id")
	authenticatedUserID, _ := c.Get("user_id")

	// Validate authorization: users can only change their own password (unless admin)
	roles, _ := c.Get("user_roles")
	userRoles, ok := roles.([]string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid roles format"})
		return
	}

	authUserIDStr, ok := authenticatedUserID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID format"})
		return
	}

	// Check if user is changing own password or is admin
	if authUserIDStr != userID {
		isAdmin := false
		for _, role := range userRoles {
			if role == "admin" {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			log.Printf("User %s attempted to change password for user %s", authUserIDStr, userID)
			c.JSON(http.StatusForbidden, gin.H{"error": "can only change your own password"})
			return
		}
	}

	// Read and forward password change request
	body, err := readRequestBody(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request"})
		return
	}

	path := fmt.Sprintf("/aegis/users/%s/password", userID)
	respBody, statusCode, err := h.proxyToAegis("POST", path, body)
	if err != nil {
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.Data(statusCode, "application/json", respBody)
}

// readRequestBody reads and returns the request body.
// Returns the body bytes and an error if reading fails.
func readRequestBody(c *gin.Context) ([]byte, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// proxyToAegis forwards HTTP requests to the Aegis service.
// It handles request creation, execution, and response reading.
func (h *Handler) proxyToAegis(method, path string, body []byte) ([]byte, int, error) {
	targetURL := h.aegisURL + path
	log.Printf("Proxying %s request to Aegis: %s", method, targetURL)

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, targetURL, bodyReader)
	if err != nil {
		log.Printf("Failed to create HTTP request: %v", err)
		return nil, http.StatusInternalServerError, errors.New("failed to create request")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		log.Printf("Aegis request failed: %v", err)
		return nil, http.StatusBadGateway, errors.New("authentication service unavailable")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read Aegis response: %v", err)
		return nil, http.StatusInternalServerError, errors.New("failed to read response")
	}

	return respBody, resp.StatusCode, nil
}
