// Package core implements HTTP reverse proxy functionality.
// It provides request forwarding, header manipulation, and response streaming
// for proxying requests to backend services.
package core

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ProxyService handles forwarding HTTP requests to backend services.
// It preserves HTTP methods, headers, query parameters, and request bodies
// while adding standard forwarding headers (X-Forwarded-*).
type ProxyService struct {
	client *http.Client
}

// NewProxyService creates a new ProxyService instance with sensible defaults.
// The default HTTP client has a 30-second timeout and does not follow redirects.
func NewProxyService() *ProxyService {
	return &ProxyService{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		},
	}
}

// Forward forwards a request to the target backend URL.
// Parameters:
//   - c: Gin context containing the original request
//   - targetBaseURL: base URL of the backend service (e.g., "http://api:8080")
//   - stripPrefix: path prefix to remove before forwarding (e.g., "/api/v1")
//   - timeout: request timeout (0 means use default client timeout)
//
// The method preserves the HTTP method, headers, body, and query parameters.
// Standard forwarding headers (X-Forwarded-For, X-Forwarded-Proto) are added.
func (p *ProxyService) Forward(c *gin.Context, targetBaseURL string, stripPrefix string, timeout time.Duration) error {
	// Build target URL
	targetURL, err := p.buildTargetURL(c.Request, targetBaseURL, stripPrefix)
	if err != nil {
		log.Printf("Failed to build target URL: %v", err)
		return errors.New("failed to build target URL")
	}

	log.Printf("Forwarding request to: %s", targetURL.String())

	// Create proxy request
	proxyReq, err := p.createProxyRequest(c.Request, targetURL)
	if err != nil {
		log.Printf("Failed to create proxy request: %v", err)
		return errors.New("failed to create proxy request")
	}

	// Apply timeout if specified
	if timeout > 0 {
		client := &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		return p.doRequest(c, proxyReq, client)
	}

	return p.doRequest(c, proxyReq, p.client)
}

// ForwardToURL forwards a request to a specific target URL.
// This is a simpler version of Forward that takes a complete URL string.
// Query parameters from the original request are appended to the target URL.
func (p *ProxyService) ForwardToURL(c *gin.Context, targetURL string) error {
	log.Printf("Forwarding request to: %s", targetURL)

	// Parse the target URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		log.Printf("Invalid target URL %s: %v", targetURL, err)
		return errors.New("invalid target URL")
	}

	// Copy query parameters from original request
	if c.Request.URL.RawQuery != "" {
		if parsedURL.RawQuery != "" {
			parsedURL.RawQuery += "&" + c.Request.URL.RawQuery
		} else {
			parsedURL.RawQuery = c.Request.URL.RawQuery
		}
	}

	// Create proxy request
	proxyReq, err := p.createProxyRequest(c.Request, parsedURL)
	if err != nil {
		log.Printf("Failed to create proxy request: %v", err)
		return errors.New("failed to create proxy request")
	}

	return p.doRequest(c, proxyReq, p.client)
}

// buildTargetURL constructs the target URL for the backend request.
func (p *ProxyService) buildTargetURL(req *http.Request, targetBaseURL string, stripPrefix string) (*url.URL, error) {
	targetURL, err := url.Parse(targetBaseURL)
	if err != nil {
		log.Printf("Failed to parse target URL %s: %v", targetBaseURL, err)
		return nil, errors.New("invalid target URL")
	}

	// Handle path
	path := req.URL.Path
	if stripPrefix != "" {
		path = strings.TrimPrefix(path, stripPrefix)
	}

	// Ensure path starts with /
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Combine paths
	targetURL.Path = strings.TrimSuffix(targetURL.Path, "/") + path

	// Copy query parameters
	targetURL.RawQuery = req.URL.RawQuery

	return targetURL, nil
}

// createProxyRequest creates a new HTTP request for the backend.
func (p *ProxyService) createProxyRequest(original *http.Request, targetURL *url.URL) (*http.Request, error) {
	// Create new request
	proxyReq, err := http.NewRequest(original.Method, targetURL.String(), original.Body)
	if err != nil {
		return nil, err
	}

	// Copy headers
	for key, values := range original.Header {
		// Skip hop-by-hop headers
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Set forwarding headers
	if original.RemoteAddr != "" {
		proxyReq.Header.Set("X-Forwarded-For", original.RemoteAddr)
	}
	proxyReq.Header.Set("X-Forwarded-Proto", original.URL.Scheme)
	if original.Host != "" {
		proxyReq.Header.Set("X-Forwarded-Host", original.Host)
	}

	return proxyReq, nil
}

// doRequest executes the proxy request and copies the response.
func (p *ProxyService) doRequest(c *gin.Context, proxyReq *http.Request, client *http.Client) error {
	// Execute request
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Printf("Backend request failed: %v", err)
		return errors.New("backend request failed")
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// Copy status code
	c.Status(resp.StatusCode)

	// Copy response body
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		log.Printf("Failed to copy response body: %v", err)
		return errors.New("failed to copy response body")
	}

	return nil
}

// isHopByHopHeader returns true if the header is a hop-by-hop header.
// These headers are meaningful only for a single transport-level connection.
func isHopByHopHeader(header string) bool {
	hopByHop := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}

	header = strings.ToLower(header)
	for _, h := range hopByHop {
		if strings.ToLower(h) == header {
			return true
		}
	}
	return false
}
