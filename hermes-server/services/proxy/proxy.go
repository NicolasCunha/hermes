// Package proxy implements HTTP reverse proxy functionality.
package proxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Proxy handles forwarding requests to backend services.
type Proxy struct {
	client *http.Client
}

// New creates a new Proxy instance.
func New() *Proxy {
	return &Proxy{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		},
	}
}

// Forward forwards a request to the target backend URL.
func (p *Proxy) Forward(c *gin.Context, targetBaseURL string, stripPrefix string, timeout time.Duration) error {
	// Build target URL
	targetURL, err := p.buildTargetURL(c.Request, targetBaseURL, stripPrefix)
	if err != nil {
		return fmt.Errorf("failed to build target URL: %w", err)
	}

	log.Printf("Forwarding request to: %s", targetURL.String())

	// Create proxy request
	proxyReq, err := p.createProxyRequest(c.Request, targetURL)
	if err != nil {
		return fmt.Errorf("failed to create proxy request: %w", err)
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

// ForwardToURL forwards a request to a specific target URL
// This is a simpler version that takes a complete URL string
func (p *Proxy) ForwardToURL(c *gin.Context, targetURL string) error {
	log.Printf("Forwarding request to: %s", targetURL)

	// Parse the target URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
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
		return fmt.Errorf("failed to create proxy request: %w", err)
	}

	return p.doRequest(c, proxyReq, p.client)
}

// buildTargetURL constructs the target URL for the backend request.
func (p *Proxy) buildTargetURL(req *http.Request, targetBaseURL string, stripPrefix string) (*url.URL, error) {
	targetURL, err := url.Parse(targetBaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
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
func (p *Proxy) createProxyRequest(original *http.Request, targetURL *url.URL) (*http.Request, error) {
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
func (p *Proxy) doRequest(c *gin.Context, proxyReq *http.Request, client *http.Client) error {
	// Execute request
	resp, err := client.Do(proxyReq)
	if err != nil {
		return fmt.Errorf("backend request failed: %w", err)
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
		return fmt.Errorf("failed to copy response body: %w", err)
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
