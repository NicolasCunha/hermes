// Package core provides integration with the Aegis authentication service.
package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)

// AegisClient handles communication with the Aegis authentication service.
// It provides methods for token validation and health checking.
type AegisClient struct {
	baseURL    string
	httpClient *http.Client
}

// ValidateTokenRequest represents a token validation request sent to Aegis.
type ValidateTokenRequest struct {
	Token string `json:"token"`
}

// ValidateTokenResponse represents the response from Aegis token validation.
// It contains the validation result and user information if the token is valid.
type ValidateTokenResponse struct {
	Valid     bool       `json:"valid"`
	Error     string     `json:"error,omitempty"`
	User      *AegisUser `json:"user,omitempty"`
	ExpiresAt time.Time  `json:"expires_at,omitempty"`
}

// AegisUser represents authenticated user information returned from Aegis.
// It includes the user's identity, roles, and permissions.
type AegisUser struct {
	ID          string   `json:"id"`
	Subject     string   `json:"subject"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

// NewAegisClient creates a new Aegis HTTP client with the specified base URL and timeout.
// The baseURL should be the full API base path (e.g., "http://aegis:3100/api").
func NewAegisClient(baseURL string, timeout time.Duration) *AegisClient {
	return &AegisClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ValidateToken calls Aegis to validate a JWT token.
// Returns the validation response containing user information if valid,
// or an error if the request fails or the token is invalid.
func (c *AegisClient) ValidateToken(token string) (*ValidateTokenResponse, error) {
	reqBody := ValidateTokenRequest{Token: token}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("Failed to marshal validation request: %v", err)
		return nil, errors.New("failed to marshal request")
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/aegis/api/auth/validate",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		log.Printf("Aegis validation call failed: %v", err)
		return nil, errors.New("Aegis call failed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read Aegis response body: %v", err)
		return nil, errors.New("failed to read response")
	}

	var result ValidateTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Failed to unmarshal Aegis response: %v", err)
		return nil, errors.New("failed to unmarshal response")
	}

	return &result, nil
}

// Health checks if the Aegis service is available and responding.
// Returns an error if Aegis is unreachable or returns a non-200 status.
func (c *AegisClient) Health() error {
	resp, err := c.httpClient.Get(c.baseURL + "/aegis/health")
	if err != nil {
		log.Printf("Aegis health check request failed: %v", err)
		return errors.New("health check failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Aegis unhealthy: status %d", resp.StatusCode)
		return errors.New("Aegis unhealthy")
	}
	return nil
}
