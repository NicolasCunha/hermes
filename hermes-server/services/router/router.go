// Package router handles request routing and matching for registered services.
package router

import (
	"strings"
)

// Router matches incoming requests to registered services.
// This is a placeholder for Phase 3 when service registry is implemented.
type Router struct {
	// Will be integrated with service registry in Phase 3
}

// New creates a new Router instance.
func New() *Router {
	return &Router{}
}

// FindServiceForPath attempts to find a registered service for the given path.
// Returns service name and matched prefix, or empty strings if no match.
// This will be implemented properly in Phase 3 with the service registry.
func (r *Router) FindServiceForPath(path string) (serviceName string, matchedPrefix string) {
	// For now, return empty - will be implemented with service registry
	// Example future logic:
	// - Check if path starts with /api/service-name/
	// - Look up service in registry
	// - Return service instance and prefix to strip
	return "", ""
}

// IsManagementPath checks if a path is for Hermes management API.
func IsManagementPath(path string) bool {
	// All Hermes management routes are under /hermes
	return strings.HasPrefix(path, "/hermes")
}
