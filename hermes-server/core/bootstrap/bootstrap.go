// Package bootstrap handles initialization of the admin user on Hermes startup.
// It ensures that a default admin user exists in Aegis for initial system access.
package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// AdminBootstrapper handles the creation and setup of the initial admin user.
// It communicates with Aegis to ensure roles, permissions, and the admin user exist.
type AdminBootstrapper struct {
	aegisURL   string
	httpClient *http.Client
	adminUser  string
	adminPass  string
}

// NewAdminBootstrapper creates a new admin bootstrapper instance.
// Parameters:
//   - aegisURL: base URL of the Aegis API (e.g., "http://aegis:3100/api")
//   - adminUser: username for the admin account
//   - adminPassword: password for the admin account
func NewAdminBootstrapper(aegisURL, adminUser, adminPassword string) *AdminBootstrapper {
	return &AdminBootstrapper{
		aegisURL:   aegisURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		adminUser:  adminUser,
		adminPass:  adminPassword,
	}
}

// EnsureAdminUser ensures the admin user exists with proper roles and permissions.
// This is called on Hermes startup to bootstrap the system. It performs the following:
//  1. Checks if the admin user already exists (skips if found)
//  2. Creates the "admin" role if it doesn't exist
//  3. Creates the "manage:system" permission if it doesn't exist
//  4. Registers the admin user with the role and permission
//
// Returns an error if any step fails, except for 409 Conflict (already exists).
func (b *AdminBootstrapper) EnsureAdminUser() error {
	log.Printf("Bootstrapping admin user: %s", b.adminUser)

	// Step 1: Check if admin user already exists
	exists, userID, err := b.checkUserExists()
	if err != nil {
		log.Printf("Error checking if admin user exists: %v", err)
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if exists {
		log.Printf("Admin user already exists (ID: %s), skipping bootstrap", userID)
		// Optionally verify roles/permissions here
		return nil
	}

	// Step 2: Ensure admin role exists
	if err := b.ensureRole("admin", "System administrator"); err != nil {
		log.Printf("Failed to ensure admin role: %v", err)
		return fmt.Errorf("failed to create admin role: %w", err)
	}

	// Step 3: Ensure manage:system permission exists
	if err := b.ensurePermission("manage:system", "Full system access"); err != nil {
		log.Printf("Failed to ensure manage:system permission: %v", err)
		return fmt.Errorf("failed to create permission: %w", err)
	}

	// Step 4: Register admin user with roles and permissions
	userID, err = b.registerUser()
	if err != nil {
		log.Printf("Failed to register admin user: %v", err)
		return fmt.Errorf("failed to register admin user: %w", err)
	}

	log.Printf("Admin user bootstrapped successfully (ID: %s)", userID)
	return nil
}

// checkUserExists checks if a user with the admin subject exists.
func (b *AdminBootstrapper) checkUserExists() (bool, string, error) {
	url := fmt.Sprintf("%s/aegis/users", b.aegisURL)

	resp, err := b.httpClient.Get(url)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", err
	}

	var users []map[string]interface{}
	if err := json.Unmarshal(body, &users); err != nil {
		return false, "", err
	}

	// Look for user with matching subject
	for _, user := range users {
		if subject, ok := user["subject"].(string); ok && subject == b.adminUser {
			if id, ok := user["id"].(string); ok {
				return true, id, nil
			}
		}
	}

	return false, "", nil
}

// ensureRole creates the admin role if it doesn't exist.
func (b *AdminBootstrapper) ensureRole(name, description string) error {
	url := fmt.Sprintf("%s/aegis/roles", b.aegisURL)

	payload := map[string]string{
		"name":        name,
		"description": description,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := b.httpClient.Post(url, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 201 Created or 409 Conflict (already exists) are both acceptable
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	if resp.StatusCode == http.StatusCreated {
		log.Printf("Created role: %s", name)
	} else {
		log.Printf("Role already exists: %s", name)
	}

	return nil
}

// ensurePermission creates a permission if it doesn't exist.
func (b *AdminBootstrapper) ensurePermission(name, description string) error {
	url := fmt.Sprintf("%s/aegis/permissions", b.aegisURL)

	payload := map[string]string{
		"name":        name,
		"description": description,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := b.httpClient.Post(url, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 201 Created or 409 Conflict (already exists) are both acceptable
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	if resp.StatusCode == http.StatusCreated {
		log.Printf("Created permission: %s", name)
	} else {
		log.Printf("Permission already exists: %s", name)
	}

	return nil
}

// registerUser creates the admin user with roles and permissions.
func (b *AdminBootstrapper) registerUser() (string, error) {
	url := fmt.Sprintf("%s/aegis/users/register", b.aegisURL)

	payload := map[string]interface{}{
		"subject":     b.adminUser,
		"password":    b.adminPass,
		"roles":       []string{"admin"},
		"permissions": []string{"manage:system"},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	resp, err := b.httpClient.Post(url, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	userID, ok := result["id"].(string)
	if !ok {
		return "", errors.New("user ID not found in response")
	}

	log.Printf("Registered admin user: %s", b.adminUser)
	return userID, nil
}
