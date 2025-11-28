package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"nfcunha/hermes/hermes-server/services/aegis"
)

// AuthMiddleware validates JWT tokens using Aegis
func AuthMiddleware(aegisClient *aegis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Println("Missing Authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization token"})
			c.Abort()
			return
		}

		// Extract Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Println("Invalid Authorization header format")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token with Aegis
		resp, err := aegisClient.ValidateToken(token)
		if err != nil {
			log.Printf("Aegis validation error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication service unavailable"})
			c.Abort()
			return
		}

		if !resp.Valid {
			log.Printf("Invalid token: %s", resp.Error)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// Store user info in context for handlers
		c.Set("user_id", resp.User.ID)
		c.Set("user_subject", resp.User.Subject)
		c.Set("user_roles", resp.User.Roles)
		c.Set("user_permissions", resp.User.Permissions)

		log.Printf("Authenticated user: %s (%s)", resp.User.Subject, resp.User.ID)
		c.Next()
	}
}

// RequireAdmin middleware ensures user has admin role
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, exists := c.Get("user_roles")
		if !exists {
			log.Println("No roles found in context")
			c.JSON(http.StatusForbidden, gin.H{"error": "no roles found"})
			c.Abort()
			return
		}

		userRoles, ok := roles.([]string)
		if !ok {
			log.Println("Invalid roles format in context")
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid roles format"})
			c.Abort()
			return
		}

		isAdmin := false
		for _, role := range userRoles {
			if role == "admin" {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			log.Println("Access denied: admin role required")
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission middleware checks for specific permission
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		permissions, exists := c.Get("user_permissions")
		if !exists {
			log.Printf("No permissions found in context")
			c.JSON(http.StatusForbidden, gin.H{"error": "no permissions found"})
			c.Abort()
			return
		}

		userPerms, ok := permissions.([]string)
		if !ok {
			log.Println("Invalid permissions format in context")
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid permissions format"})
			c.Abort()
			return
		}

		hasPermission := false
		for _, perm := range userPerms {
			if perm == permission {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			log.Printf("Access denied: permission '%s' required", permission)
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}
