package aegis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles communication with Aegis authentication service
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// ValidateTokenRequest represents token validation request
type ValidateTokenRequest struct {
	Token string `json:"token"`
}

// ValidateTokenResponse represents token validation response
type ValidateTokenResponse struct {
	Valid     bool      `json:"valid"`
	Error     string    `json:"error,omitempty"`
	User      *User     `json:"user,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// User represents authenticated user information
type User struct {
	ID          string   `json:"id"`
	Subject     string   `json:"subject"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

// NewClient creates a new Aegis HTTP client
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ValidateToken calls Aegis to validate a JWT token
func (c *Client) ValidateToken(token string) (*ValidateTokenResponse, error) {
	reqBody := ValidateTokenRequest{Token: token}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/aegis/api/auth/validate",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("Aegis call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result ValidateTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// Health checks if Aegis service is available
func (c *Client) Health() error {
	resp, err := c.httpClient.Get(c.baseURL + "/aegis/health")
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Aegis unhealthy: status %d", resp.StatusCode)
	}
	return nil
}
