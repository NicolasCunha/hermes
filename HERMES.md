# 🏛️ Hermes - API Gateway & Service Discovery

**A lightweight, high-performance API Gateway with built-in service discovery, user management, and Aegis authentication integration.**

Hermes acts as the entry point for all client requests, routing traffic to backend services, managing service discovery, and providing a dashboard for monitoring and administration.

---

## 📋 Project Overview

### Purpose
Hermes is an API Gateway that:
- Routes incoming HTTP requests to appropriate backend services
- Discovers and tracks service instances dynamically (check-in/check-out)
- Monitors service health with periodic checks (configurable interval/timeout)
- Provides a web dashboard for service and user management
- Integrates with Aegis for dashboard authentication only
- Forwards requests to backends without authentication (passes headers through)
- Manages dashboard users with role-based access (admin/viewer)

### Architecture Position
```
Client → Hermes (Gateway) → Backend Services (unauthenticated by Hermes)
           ↓
Dashboard Users → Hermes UI → Hermes API (authenticated via Aegis)
                                    ↓
                                 Aegis (Auth for dashboard only)
```

### Key Architectural Decisions
1. **No Authentication of Proxied Requests** - Hermes forwards Authorization headers to backends as-is
2. **Aegis for Dashboard Only** - User authentication for dashboard access uses Aegis JWT tokens
3. **Local User Management** - SQLite database stores dashboard users (admin/viewer roles)
4. **Path Grouping** - All Hermes management routes under `/hermes` context path
5. **Separate UI** - React application in hermes-ui/, communicates via REST API
6. **In-Memory Service Registry** - No external dependencies for service discovery

### Design Principles
1. **Simplicity First** - SQLite database, in-memory registry, no complex dependencies
2. **Performance** - Low latency overhead, efficient routing
3. **Reliability** - Health checks, automatic failover, graceful degradation
4. **Observability** - Comprehensive logging throughout the system
5. **Security** - Aegis JWT integration for dashboard, role-based access control
6. **Modularity** - Clear separation: api/, domain/, services/, utils/

---

## 🎯 Core Features

### Phase 1: Foundation (Week 1-2)
**Goal:** Basic reverse proxy with static routing

#### 1.1 HTTP Reverse Proxy
- Forward HTTP requests to backend services
- Preserve headers, query parameters, and request body
- Handle request/response streaming
- Timeout configuration per route
- Error handling and proper status codes

**Implementation Details:**
```go
// Core proxy handler
type Proxy struct {
    client *http.Client
    timeout time.Duration
}

func (p *Proxy) Forward(w http.ResponseWriter, r *http.Request, target *url.URL) error {
    // Create proxy request
    proxyReq := http.Request{
        Method: r.Method,
        URL:    target,
        Header: r.Header.Clone(),
        Body:   r.Body,
    }
    
    // Forward request
    resp, err := p.client.Do(&proxyReq)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // Copy response
    copyHeaders(w.Header(), resp.Header)
    w.WriteHeader(resp.StatusCode)
    io.Copy(w, resp.Body)
    
    return nil
}
```

#### 1.2 Route Configuration
- YAML-based configuration file
- Route matching: prefix, exact, regex
- Per-route configuration (timeout, retry, auth)
- Hot reload on config change

**Configuration Format:**
```yaml
# config/hermes.yaml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

routes:
  - name: user-service
    match:
      type: prefix  # prefix, exact, regex
      path: /api/users
    target:
      service: user-service  # Service name from registry
      timeout: 10s
      retry:
        attempts: 3
        backoff: exponential
    auth:
      required: true
      aegis_endpoint: http://localhost/api/auth/validate
    strip_prefix: false  # Remove matched prefix before forwarding

  - name: public-api
    match:
      type: prefix
      path: /api/public
    target:
      service: public-service
      timeout: 5s
    auth:
      required: false

  - name: health-check
    match:
      type: exact
      path: /health
    target:
      handler: builtin  # Use built-in handler
```

#### 1.3 Request Logging
- Structured logging (JSON format)
- Request ID generation and propagation
- Log levels: DEBUG, INFO, WARN, ERROR
- Request/response logging with sanitization

**Log Format:**
```json
{
  "timestamp": "2025-11-28T10:00:00Z",
  "level": "info",
  "request_id": "req-uuid-123",
  "method": "GET",
  "path": "/api/users/123",
  "remote_addr": "192.168.1.1",
  "service": "user-service",
  "instance": "localhost:8081",
  "status": 200,
  "duration_ms": 45,
  "error": null
}
```

---

### Phase 2: Service Discovery (Week 3)
**Goal:** Dynamic service registration and health monitoring

#### 2.1 Service Registry (In-Memory)
- Register/deregister services via HTTP API
- Store multiple instances per service
- Service metadata (version, tags, weight)
- Thread-safe concurrent access

**Data Structures:**
```go
// Service registry
type ServiceRegistry struct {
    services map[string]*Service
    mu       sync.RWMutex
}

type Service struct {
    Name      string
    Instances []*ServiceInstance
    mu        sync.RWMutex
}

type ServiceInstance struct {
    ID          string    // UUID
    ServiceName string
    Host        string
    Port        int
    Protocol    string    // http, https, grpc
    HealthCheck HealthCheckConfig
    Status      InstanceStatus  // healthy, unhealthy, draining
    Metadata    map[string]string
    RegisteredAt time.Time
    LastSeen     time.Time
    FailureCount int
}

type HealthCheckConfig struct {
    Path     string        // /health
    Interval time.Duration // 30s
    Timeout  time.Duration // 5s
    Threshold int          // Failures before marking unhealthy
}

type InstanceStatus string

const (
    StatusHealthy   InstanceStatus = "healthy"
    StatusUnhealthy InstanceStatus = "unhealthy"
    StatusDraining  InstanceStatus = "draining"  // For graceful shutdown
)
```

**Registry API:**
```go
// POST /registry/register
type RegisterRequest struct {
    ServiceName string                 `json:"service_name" binding:"required"`
    Host        string                 `json:"host" binding:"required"`
    Port        int                    `json:"port" binding:"required"`
    Protocol    string                 `json:"protocol"`  // Default: http
    HealthCheck HealthCheckConfig      `json:"health_check"`
    Metadata    map[string]string      `json:"metadata"`
}

// POST /registry/deregister
type DeregisterRequest struct {
    ServiceName string `json:"service_name" binding:"required"`
    InstanceID  string `json:"instance_id" binding:"required"`
}

// POST /registry/heartbeat
type HeartbeatRequest struct {
    ServiceName string `json:"service_name" binding:"required"`
    InstanceID  string `json:"instance_id" binding:"required"`
}

// GET /registry/services
// Returns list of all services with their instances

// GET /registry/services/:name
// Returns specific service with all instances
```

#### 2.2 Health Checking
- Active health checks (HTTP GET to health endpoint)
- Passive health checks (monitor request failures)
- Configurable check interval and timeout
- Automatic marking unhealthy/healthy based on threshold
- Remove dead instances after extended failure

**Health Check Implementation:**
```go
type HealthChecker struct {
    registry *ServiceRegistry
    client   *http.Client
    interval time.Duration
    quit     chan struct{}
}

func (h *HealthChecker) Start() {
    ticker := time.NewTicker(h.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            h.checkAll()
        case <-h.quit:
            return
        }
    }
}

func (h *HealthChecker) checkAll() {
    services := h.registry.GetAllServices()
    
    for _, service := range services {
        for _, instance := range service.Instances {
            // Skip if already unhealthy and exceeded max failures
            if instance.Status == StatusUnhealthy && 
               instance.FailureCount > 10 {
                // Remove instance
                h.registry.Remove(service.Name, instance.ID)
                continue
            }
            
            // Perform health check
            healthy := h.check(instance)
            
            if healthy {
                instance.Status = StatusHealthy
                instance.FailureCount = 0
                instance.LastSeen = time.Now()
            } else {
                instance.FailureCount++
                if instance.FailureCount >= instance.HealthCheck.Threshold {
                    instance.Status = StatusUnhealthy
                }
            }
        }
    }
}

func (h *HealthChecker) check(instance *ServiceInstance) bool {
    url := fmt.Sprintf("%s://%s:%d%s", 
        instance.Protocol, instance.Host, instance.Port, 
        instance.HealthCheck.Path)
    
    ctx, cancel := context.WithTimeout(context.Background(), 
        instance.HealthCheck.Timeout)
    defer cancel()
    
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := h.client.Do(req)
    
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    
    return resp.StatusCode >= 200 && resp.StatusCode < 300
}
```

#### 2.3 Service Discovery Integration
- Combine static config + dynamic registration
- Priority: dynamic registry → static config → error
- Fallback mechanism if all instances fail

```go
type ServiceResolver struct {
    registry *ServiceRegistry
    static   map[string][]*ServiceInstance  // From config
}

func (r *ServiceResolver) Resolve(serviceName string) ([]*ServiceInstance, error) {
    // Try dynamic registry first
    if instances := r.registry.GetHealthy(serviceName); len(instances) > 0 {
        return instances, nil
    }
    
    // Fallback to static config
    if instances, ok := r.static[serviceName]; ok && len(instances) > 0 {
        return instances, nil
    }
    
    return nil, ErrServiceNotFound
}
```

---

### Phase 3: Load Balancing (Week 4)
**Goal:** Distribute traffic across service instances

#### 3.1 Load Balancing Algorithms
Implement multiple strategies:

**Round Robin:**
```go
type RoundRobinBalancer struct {
    counter uint64
}

func (b *RoundRobinBalancer) Next(instances []*ServiceInstance) *ServiceInstance {
    n := atomic.AddUint64(&b.counter, 1)
    return instances[n % uint64(len(instances))]
}
```

**Least Connections:**
```go
type LeastConnectionsBalancer struct {
    connections map[string]int  // instanceID -> active connections
    mu          sync.RWMutex
}

func (b *LeastConnectionsBalancer) Next(instances []*ServiceInstance) *ServiceInstance {
    b.mu.RLock()
    defer b.mu.RUnlock()
    
    var selected *ServiceInstance
    minConns := int(^uint(0) >> 1)  // Max int
    
    for _, instance := range instances {
        conns := b.connections[instance.ID]
        if conns < minConns {
            minConns = conns
            selected = instance
        }
    }
    
    return selected
}

func (b *LeastConnectionsBalancer) Acquire(instanceID string) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.connections[instanceID]++
}

func (b *LeastConnectionsBalancer) Release(instanceID string) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.connections[instanceID]--
}
```

**Random:**
```go
type RandomBalancer struct {
    rand *rand.Rand
    mu   sync.Mutex
}

func (b *RandomBalancer) Next(instances []*ServiceInstance) *ServiceInstance {
    b.mu.Lock()
    defer b.mu.Unlock()
    return instances[b.rand.Intn(len(instances))]
}
```

**Weighted Random:**
```go
type WeightedRandomBalancer struct {
    rand *rand.Rand
    mu   sync.Mutex
}

func (b *WeightedRandomBalancer) Next(instances []*ServiceInstance) *ServiceInstance {
    // Calculate total weight
    totalWeight := 0
    for _, inst := range instances {
        weight, _ := strconv.Atoi(inst.Metadata["weight"])
        if weight <= 0 {
            weight = 100  // Default weight
        }
        totalWeight += weight
    }
    
    // Pick random weight
    b.mu.Lock()
    random := b.rand.Intn(totalWeight)
    b.mu.Unlock()
    
    // Find instance
    for _, inst := range instances {
        weight, _ := strconv.Atoi(inst.Metadata["weight"])
        if weight <= 0 {
            weight = 100
        }
        random -= weight
        if random < 0 {
            return inst
        }
    }
    
    return instances[0]  // Fallback
}
```

#### 3.2 Sticky Sessions (Optional)
- Session affinity based on cookie or header
- Consistent hashing for distribution
- Fallback if preferred instance is down

```go
type StickySessionBalancer struct {
    underlying LoadBalancer
    sessions   map[string]string  // sessionID -> instanceID
    mu         sync.RWMutex
}

func (b *StickySessionBalancer) Next(r *http.Request, instances []*ServiceInstance) *ServiceInstance {
    // Extract session ID from cookie/header
    sessionID := extractSessionID(r)
    
    if sessionID != "" {
        b.mu.RLock()
        instanceID, exists := b.sessions[sessionID]
        b.mu.RUnlock()
        
        if exists {
            // Find instance
            for _, inst := range instances {
                if inst.ID == instanceID && inst.Status == StatusHealthy {
                    return inst
                }
            }
        }
    }
    
    // No session or instance unavailable, use underlying balancer
    selected := b.underlying.Next(instances)
    
    // Store session mapping
    if sessionID != "" {
        b.mu.Lock()
        b.sessions[sessionID] = selected.ID
        b.mu.Unlock()
    }
    
    return selected
}
```

#### 3.3 Configurable Per Route
```yaml
routes:
  - name: user-service
    match:
      path: /api/users
    target:
      service: user-service
    load_balancing:
      strategy: least_connections  # round_robin, random, weighted_random, least_connections
      sticky_sessions:
        enabled: true
        cookie_name: HERMES_SESSION
```

---

### Phase 4: Aegis Integration (Week 5)
**Goal:** Seamless authentication with Aegis

#### 4.1 Token Validation Middleware
- Extract JWT token from Authorization header
- Call Aegis `/api/auth/validate` endpoint
- Cache validation results (short TTL: 60s)
- Forward user claims to backend services

**Implementation:**
```go
type AegisAuthenticator struct {
    aegisURL string
    client   *http.Client
    cache    *TokenCache  // Simple in-memory cache
}

type TokenCache struct {
    entries map[string]*CacheEntry
    mu      sync.RWMutex
    ttl     time.Duration
}

type CacheEntry struct {
    Claims    *jwt.Claims
    ExpiresAt time.Time
}

func (a *AegisAuthenticator) Validate(r *http.Request) (*jwt.Claims, error) {
    // Extract token
    token := extractToken(r)
    if token == "" {
        return nil, ErrMissingToken
    }
    
    // Check cache
    if claims := a.cache.Get(token); claims != nil {
        return claims, nil
    }
    
    // Call Aegis
    validateReq := map[string]string{"token": token}
    body, _ := json.Marshal(validateReq)
    
    resp, err := a.client.Post(
        a.aegisURL+"/api/auth/validate",
        "application/json",
        bytes.NewBuffer(body),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Valid bool        `json:"valid"`
        User  *jwt.Claims `json:"user"`
        Error string      `json:"error"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    if !result.Valid {
        return nil, ErrInvalidToken
    }
    
    // Cache result
    a.cache.Set(token, result.User)
    
    return result.User, nil
}

// Middleware
func (g *Gateway) AuthMiddleware(route *Route) gin.HandlerFunc {
    return func(c *gin.Context) {
        if !route.AuthRequired {
            c.Next()
            return
        }
        
        claims, err := g.authenticator.Validate(c.Request)
        if err != nil {
            c.JSON(401, gin.H{"error": "unauthorized"})
            c.Abort()
            return
        }
        
        // Add claims to context for downstream handlers
        c.Set("user_id", claims.ID)
        c.Set("user_email", claims.Subject)
        c.Set("user_roles", claims.Roles)
        c.Set("user_permissions", claims.Permissions)
        
        // Forward claims to backend as headers
        c.Request.Header.Set("X-User-ID", claims.ID)
        c.Request.Header.Set("X-User-Email", claims.Subject)
        c.Request.Header.Set("X-User-Roles", strings.Join(claims.Roles, ","))
        c.Request.Header.Set("X-User-Permissions", strings.Join(claims.Permissions, ","))
        
        c.Next()
    }
}
```

#### 4.2 Token Caching Strategy
- Cache validated tokens for 60 seconds
- LRU eviction policy
- Automatic cleanup of expired entries
- Configurable TTL per route

```go
type TokenCache struct {
    entries map[string]*CacheEntry
    mu      sync.RWMutex
    ttl     time.Duration
    maxSize int
    lru     *list.List  // For LRU eviction
}

func (c *TokenCache) Set(token string, claims *jwt.Claims) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Check size limit
    if len(c.entries) >= c.maxSize {
        // Evict oldest
        oldest := c.lru.Back()
        if oldest != nil {
            delete(c.entries, oldest.Value.(string))
            c.lru.Remove(oldest)
        }
    }
    
    c.entries[token] = &CacheEntry{
        Claims:    claims,
        ExpiresAt: time.Now().Add(c.ttl),
    }
    c.lru.PushFront(token)
}

func (c *TokenCache) Get(token string) *jwt.Claims {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    entry, exists := c.entries[token]
    if !exists {
        return nil
    }
    
    if time.Now().After(entry.ExpiresAt) {
        delete(c.entries, token)
        return nil
    }
    
    return entry.Claims
}

// Cleanup goroutine
func (c *TokenCache) StartCleanup() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        c.mu.Lock()
        now := time.Now()
        for token, entry := range c.entries {
            if now.After(entry.ExpiresAt) {
                delete(c.entries, token)
            }
        }
        c.mu.Unlock()
    }
}
```

#### 4.3 Authorization Rules (Optional)
- Per-route role/permission requirements
- Claims-based access control

```yaml
routes:
  - name: admin-api
    match:
      path: /api/admin
    target:
      service: admin-service
    auth:
      required: true
      roles: [admin]  # Require admin role
      permissions: [manage:system]  # OR require specific permission
```

---

### Phase 5: Resilience (Week 6)
**Goal:** Handle failures gracefully

#### 5.1 Circuit Breaker
- Prevent cascading failures
- Open circuit after N consecutive failures
- Half-open state for recovery testing
- Per-service circuit breaker

**Implementation:**
```go
type CircuitBreaker struct {
    state         CircuitState
    failureCount  int
    successCount  int
    lastFailTime  time.Time
    threshold     int           // Failures to open circuit
    timeout       time.Duration // Time in open state
    halfOpenLimit int           // Successes to close circuit
    mu            sync.RWMutex
}

type CircuitState string

const (
    StateClosed   CircuitState = "closed"
    StateOpen     CircuitState = "open"
    StateHalfOpen CircuitState = "half_open"
)

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mu.Lock()
    state := cb.state
    cb.mu.Unlock()
    
    switch state {
    case StateOpen:
        // Check if timeout elapsed
        if time.Since(cb.lastFailTime) > cb.timeout {
            cb.setState(StateHalfOpen)
            return cb.Call(fn)
        }
        return ErrCircuitOpen
        
    case StateHalfOpen:
        err := fn()
        if err != nil {
            cb.onFailure()
            return err
        }
        cb.onSuccess()
        return nil
        
    case StateClosed:
        err := fn()
        if err != nil {
            cb.onFailure()
            return err
        }
        cb.onSuccess()
        return nil
    }
    
    return nil
}

func (cb *CircuitBreaker) onFailure() {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    
    cb.failureCount++
    cb.successCount = 0
    cb.lastFailTime = time.Now()
    
    if cb.state == StateHalfOpen {
        cb.state = StateOpen
    } else if cb.failureCount >= cb.threshold {
        cb.state = StateOpen
    }
}

func (cb *CircuitBreaker) onSuccess() {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    
    cb.failureCount = 0
    cb.successCount++
    
    if cb.state == StateHalfOpen && cb.successCount >= cb.halfOpenLimit {
        cb.state = StateClosed
    }
}
```

#### 5.2 Retry Logic
- Automatic retry on transient failures
- Exponential backoff
- Maximum retry attempts
- Only retry idempotent methods (GET, PUT, DELETE)

```go
type RetryConfig struct {
    Attempts int
    Backoff  BackoffStrategy  // fixed, linear, exponential
    InitialDelay time.Duration
}

type BackoffStrategy string

const (
    BackoffFixed       BackoffStrategy = "fixed"
    BackoffLinear      BackoffStrategy = "linear"
    BackoffExponential BackoffStrategy = "exponential"
)

func (p *Proxy) ForwardWithRetry(w http.ResponseWriter, r *http.Request, 
    target *url.URL, config RetryConfig) error {
    
    var lastErr error
    
    for attempt := 0; attempt < config.Attempts; attempt++ {
        if attempt > 0 {
            // Wait before retry
            delay := p.calculateBackoff(config, attempt)
            time.Sleep(delay)
        }
        
        // Try request
        err := p.Forward(w, r, target)
        if err == nil {
            return nil
        }
        
        // Check if retryable
        if !isRetryable(err, r.Method) {
            return err
        }
        
        lastErr = err
    }
    
    return lastErr
}

func (p *Proxy) calculateBackoff(config RetryConfig, attempt int) time.Duration {
    switch config.Backoff {
    case BackoffFixed:
        return config.InitialDelay
    case BackoffLinear:
        return config.InitialDelay * time.Duration(attempt)
    case BackoffExponential:
        return config.InitialDelay * time.Duration(1<<uint(attempt))
    default:
        return config.InitialDelay
    }
}
```

#### 5.3 Timeout Management
- Request timeout
- Connection timeout
- Per-route timeout configuration
- Timeout propagation to backends

```go
type TimeoutConfig struct {
    Request    time.Duration  // Total request timeout
    Connection time.Duration  // TCP connection timeout
    TLS        time.Duration  // TLS handshake timeout
}

// Apply timeout to request
func (p *Proxy) applyTimeout(r *http.Request, timeout time.Duration) (*http.Request, context.CancelFunc) {
    ctx, cancel := context.WithTimeout(r.Context(), timeout)
    return r.WithContext(ctx), cancel
}
```

---

### Phase 6: Observability (Week 7)
**Goal:** Monitor and debug effectively

#### 6.1 Metrics Collection
- Request count, duration, errors
- Service health status
- Circuit breaker states
- Per-route metrics

**Metrics to Track:**
```go
type Metrics struct {
    // Request metrics
    RequestTotal      *prometheus.CounterVec    // By service, route, status
    RequestDuration   *prometheus.HistogramVec  // By service, route
    RequestSize       *prometheus.HistogramVec
    ResponseSize      *prometheus.HistogramVec
    
    // Service metrics
    ServiceHealthy    *prometheus.GaugeVec      // By service
    ServiceInstances  *prometheus.GaugeVec      // By service
    
    // Circuit breaker metrics
    CircuitBreakerState *prometheus.GaugeVec    // By service (0=closed, 1=open, 2=half-open)
    
    // Auth metrics
    AuthAttempts      *prometheus.CounterVec    // By status (success, failure)
    AuthCacheHits     *prometheus.Counter
    AuthCacheMisses   *prometheus.Counter
}

// Instrument request
func (g *Gateway) instrumentRequest(route *Route, duration time.Duration, status int) {
    g.metrics.RequestTotal.WithLabelValues(
        route.ServiceName,
        route.Name,
        strconv.Itoa(status),
    ).Inc()
    
    g.metrics.RequestDuration.WithLabelValues(
        route.ServiceName,
        route.Name,
    ).Observe(duration.Seconds())
}
```

#### 6.2 Distributed Tracing
- Generate trace ID for each request
- Propagate trace context (W3C Trace Context)
- Log trace ID in all log messages
- Optional: OpenTelemetry integration

```go
type TraceContext struct {
    TraceID  string
    SpanID   string
    ParentID string
}

func (g *Gateway) createTraceContext(r *http.Request) *TraceContext {
    // Check for existing trace context
    if traceParent := r.Header.Get("traceparent"); traceParent != "" {
        return parseTraceParent(traceParent)
    }
    
    // Create new trace
    return &TraceContext{
        TraceID: generateTraceID(),
        SpanID:  generateSpanID(),
    }
}

func (g *Gateway) propagateTrace(r *http.Request, trace *TraceContext) {
    // Set W3C Trace Context headers
    traceParent := fmt.Sprintf("00-%s-%s-01", trace.TraceID, trace.SpanID)
    r.Header.Set("traceparent", traceParent)
    
    // Add custom headers
    r.Header.Set("X-Request-ID", trace.TraceID)
}
```

#### 6.3 Access Logging
- Log all requests with full context
- Sanitize sensitive data
- Configurable log level and format
- Rotation and archival

**Log Entry:**
```json
{
  "timestamp": "2025-11-28T10:00:00Z",
  "level": "info",
  "trace_id": "abc123",
  "request": {
    "method": "GET",
    "path": "/api/users/123",
    "remote_addr": "192.168.1.1",
    "user_agent": "curl/7.0"
  },
  "route": {
    "name": "user-service",
    "service": "user-service"
  },
  "backend": {
    "instance_id": "inst-123",
    "host": "localhost:8081",
    "attempt": 1
  },
  "response": {
    "status": 200,
    "duration_ms": 45,
    "size_bytes": 1024
  },
  "user": {
    "id": "user-uuid",
    "email": "user@example.com"
  }
}
```

#### 6.4 Health Check Endpoint
```go
// GET /health
func (g *Gateway) HealthCheck(c *gin.Context) {
    health := map[string]interface{}{
        "status": "healthy",
        "timestamp": time.Now(),
        "services": g.getServicesHealth(),
        "aegis": g.checkAegisHealth(),
    }
    
    c.JSON(200, health)
}

func (g *Gateway) getServicesHealth() map[string]interface{} {
    services := g.registry.GetAllServices()
    result := make(map[string]interface{})
    
    for name, service := range services {
        healthy := 0
        total := len(service.Instances)
        
        for _, inst := range service.Instances {
            if inst.Status == StatusHealthy {
                healthy++
            }
        }
        
        result[name] = map[string]interface{}{
            "healthy_instances": healthy,
            "total_instances":   total,
            "status": func() string {
                if healthy == 0 {
                    return "down"
                } else if healthy < total {
                    return "degraded"
                }
                return "healthy"
            }(),
        }
    }
    
    return result
}
```

---

## 🏗️ Project Structure

```
hermes/
├── cmd/
│   └── hermes/
│       └── main.go              # Application entry point
├── internal/
│   ├── proxy/
│   │   ├── proxy.go             # HTTP reverse proxy
│   │   ├── balancer.go          # Load balancer interface
│   │   └── balancers/           # LB implementations
│   │       ├── roundrobin.go
│   │       ├── leastconn.go
│   │       └── random.go
│   ├── registry/
│   │   ├── registry.go          # Service registry
│   │   ├── instance.go          # Service instance
│   │   └── health.go            # Health checker
│   ├── router/
│   │   ├── router.go            # Route matcher
│   │   ├── route.go             # Route definition
│   │   └── matcher.go           # Path matching logic
│   ├── auth/
│   │   ├── authenticator.go     # Aegis integration
│   │   ├── cache.go             # Token cache
│   │   └── middleware.go        # Auth middleware
│   ├── resilience/
│   │   ├── circuitbreaker.go    # Circuit breaker
│   │   └── retry.go             # Retry logic
│   ├── observability/
│   │   ├── metrics.go           # Prometheus metrics
│   │   ├── logging.go           # Structured logging
│   │   └── tracing.go           # Distributed tracing
│   ├── config/
│   │   ├── config.go            # Configuration structs
│   │   └── loader.go            # Config file loader
│   └── api/
│       ├── registry.go          # Registry API handlers
│       └── admin.go             # Admin API handlers
├── pkg/
│   └── client/
│       └── client.go            # Go client library for services
├── config/
│   └── hermes.yaml              # Example configuration
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
├── docs/
│   ├── API.md                   # API documentation
│   ├── CONFIGURATION.md         # Config reference
│   └── INTEGRATION.md           # Integration guide
├── examples/
│   ├── simple-service/          # Example backend service
│   └── aegis-integration/       # Aegis + Hermes example
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 🔧 Configuration Reference

### Complete Example Configuration

```yaml
# config/hermes.yaml

# Server configuration
server:
  host: 0.0.0.0
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s
  max_header_bytes: 1048576  # 1MB

# Service registry
registry:
  type: memory  # memory, consul, etcd (future)
  health_check:
    interval: 30s
    timeout: 5s
    threshold: 3  # Failures before marking unhealthy
  cleanup:
    interval: 5m
    max_failures: 10  # Remove instance after this many failures

# Authentication
auth:
  aegis:
    url: http://localhost/api/auth/validate
    timeout: 5s
  cache:
    enabled: true
    ttl: 60s
    max_size: 10000

# Load balancing
load_balancing:
  default_strategy: round_robin  # round_robin, least_connections, random, weighted_random

# Resilience
resilience:
  circuit_breaker:
    enabled: true
    threshold: 5          # Failures to open circuit
    timeout: 30s          # Time in open state
    half_open_requests: 3 # Successes to close circuit
  retry:
    enabled: true
    max_attempts: 3
    backoff: exponential
    initial_delay: 100ms

# Observability
observability:
  logging:
    level: info  # debug, info, warn, error
    format: json # json, text
    output: stdout
  metrics:
    enabled: true
    port: 9090
    path: /metrics
  tracing:
    enabled: false
    provider: opentelemetry  # opentelemetry, jaeger (future)

# Static service configuration (fallback)
services:
  - name: user-service
    instances:
      - host: localhost
        port: 8081
        weight: 100
      - host: localhost
        port: 8082
        weight: 100
    health_check:
      path: /health
      interval: 30s
      timeout: 5s

# Routes
routes:
  - name: user-api
    match:
      type: prefix
      path: /api/users
    target:
      service: user-service
      timeout: 10s
      strip_prefix: false
      retry:
        attempts: 3
        backoff: exponential
    auth:
      required: true
      roles: []
      permissions: []
    load_balancing:
      strategy: round_robin

  - name: public-api
    match:
      type: prefix
      path: /api/public
    target:
      service: public-service
      timeout: 5s
    auth:
      required: false

  - name: admin-api
    match:
      type: prefix
      path: /api/admin
    target:
      service: admin-service
      timeout: 15s
    auth:
      required: true
      roles: [admin]
      permissions: [manage:system]
    rate_limit:  # Future feature
      requests_per_second: 10
      burst: 20

  - name: health
    match:
      type: exact
      path: /health
    target:
      handler: builtin
```

---

## 📡 API Reference

### Admin API

#### Service Registry Management

**Register Service:**
```http
POST /admin/registry/register
Content-Type: application/json

{
  "service_name": "user-service",
  "host": "localhost",
  "port": 8081,
  "protocol": "http",
  "health_check": {
    "path": "/health",
    "interval": "30s",
    "timeout": "5s",
    "threshold": 3
  },
  "metadata": {
    "version": "1.0.0",
    "environment": "production",
    "weight": "100"
  }
}

Response: 201 Created
{
  "instance_id": "inst-uuid-123",
  "service_name": "user-service",
  "registered_at": "2025-11-28T10:00:00Z"
}
```

**Deregister Service:**
```http
POST /admin/registry/deregister
Content-Type: application/json

{
  "service_name": "user-service",
  "instance_id": "inst-uuid-123"
}

Response: 200 OK
{
  "success": true,
  "message": "Instance deregistered successfully"
}
```

**Heartbeat:**
```http
POST /admin/registry/heartbeat
Content-Type: application/json

{
  "service_name": "user-service",
  "instance_id": "inst-uuid-123"
}

Response: 200 OK
{
  "success": true,
  "next_heartbeat": "2025-11-28T10:00:30Z"
}
```

**List Services:**
```http
GET /admin/registry/services

Response: 200 OK
{
  "services": [
    {
      "name": "user-service",
      "instance_count": 2,
      "healthy_instances": 2,
      "instances": [
        {
          "id": "inst-1",
          "host": "localhost",
          "port": 8081,
          "status": "healthy",
          "last_seen": "2025-11-28T10:00:00Z"
        }
      ]
    }
  ]
}
```

**Get Service Details:**
```http
GET /admin/registry/services/{service_name}

Response: 200 OK
{
  "name": "user-service",
  "instances": [...],
  "health": {
    "healthy": 2,
    "unhealthy": 0,
    "draining": 0
  }
}
```

#### Route Management

**List Routes:**
```http
GET /admin/routes

Response: 200 OK
{
  "routes": [
    {
      "name": "user-api",
      "path": "/api/users",
      "service": "user-service",
      "auth_required": true
    }
  ]
}
```

**Reload Configuration:**
```http
POST /admin/config/reload

Response: 200 OK
{
  "success": true,
  "message": "Configuration reloaded successfully",
  "routes_loaded": 5,
  "services_loaded": 3
}
```

#### Health & Metrics

**Gateway Health:**
```http
GET /health

Response: 200 OK
{
  "status": "healthy",
  "timestamp": "2025-11-28T10:00:00Z",
  "uptime": "24h30m",
  "services": {
    "user-service": {
      "status": "healthy",
      "healthy_instances": 2,
      "total_instances": 2
    }
  },
  "aegis": {
    "status": "healthy",
    "response_time_ms": 5
  }
}
```

**Metrics (Prometheus):**
```http
GET /metrics

Response: 200 OK (Prometheus format)
# HELP hermes_requests_total Total number of requests
# TYPE hermes_requests_total counter
hermes_requests_total{service="user-service",route="user-api",status="200"} 1000

# HELP hermes_request_duration_seconds Request duration
# TYPE hermes_request_duration_seconds histogram
hermes_request_duration_seconds_bucket{service="user-service",route="user-api",le="0.005"} 800
...
```

---

## 🚀 Getting Started

### Prerequisites
- Go 1.21+
- Aegis authentication service running
- Backend services to proxy

### Quick Start

**1. Clone and Build:**
```bash
git clone <repo>
cd hermes
go build -o hermes cmd/hermes/main.go
```

**2. Create Configuration:**
```bash
cp config/hermes.example.yaml config/hermes.yaml
# Edit config/hermes.yaml with your settings
```

**3. Run Hermes:**
```bash
./hermes --config config/hermes.yaml
```

**4. Register a Service:**
```bash
curl -X POST http://localhost:8080/admin/registry/register \
  -H "Content-Type: application/json" \
  -d '{
    "service_name": "my-service",
    "host": "localhost",
    "port": 9000,
    "health_check": {
      "path": "/health",
      "interval": "30s"
    }
  }'
```

**5. Send Requests:**
```bash
# Request will be routed to your service
curl http://localhost:8080/api/my-service/endpoint
```

---

## 🔌 Client Library

### Go Service Integration

```go
package main

import (
    "github.com/yourusername/hermes/pkg/client"
    "log"
)

func main() {
    // Create Hermes client
    hermesClient := client.NewClient("http://gateway:8080")
    
    // Register service on startup
    err := hermesClient.Register(client.RegistrationConfig{
        ServiceName: "my-service",
        Host:        "localhost",
        Port:        9000,
        HealthCheck: "/health",
        Metadata: map[string]string{
            "version": "1.0.0",
        },
    })
    if err != nil {
        log.Fatalf("Failed to register: %v", err)
    }
    
    // Send heartbeats
    go hermesClient.StartHeartbeat(30 * time.Second)
    
    // Deregister on shutdown
    defer hermesClient.Deregister()
    
    // Start your service
    startServer()
}
```

---

## 🧪 Testing Strategy

### Unit Tests
- Each component isolated
- Mock dependencies
- Test edge cases
- Aim for >85% coverage

### Integration Tests
- Full request/response cycle
- Service registry operations
- Health check behavior
- Auth integration with Aegis

### Load Tests
- Performance benchmarks
- Latency measurements (p50, p95, p99)
- Throughput testing
- Memory/CPU profiling

### Test Structure
```
internal/
├── proxy/
│   ├── proxy_test.go
│   └── balancer_test.go
├── registry/
│   ├── registry_test.go
│   └── health_test.go
└── auth/
    ├── authenticator_test.go
    └── cache_test.go
```

---

## 📊 Performance Targets

### Latency
- p50: <2ms overhead
- p95: <5ms overhead
- p99: <10ms overhead

### Throughput
- 10,000+ requests/second per instance
- Linear scaling with CPU cores

### Resource Usage
- Memory: <100MB baseline
- CPU: <5% idle, <50% under load

### Availability
- 99.9% uptime
- Automatic failover <1s
- Zero-downtime config reload

---

## 🛣️ Development Roadmap

### Phase 1: Foundation (Week 1-2) ✅
- [ ] Project setup and structure
- [ ] HTTP reverse proxy
- [ ] Static route configuration
- [ ] Basic logging
- [ ] Health check endpoint

### Phase 2: Service Discovery (Week 3) ✅
- [ ] In-memory service registry
- [ ] Registration API
- [ ] Health checker
- [ ] Service resolver

### Phase 3: Load Balancing (Week 4) ✅
- [ ] Round robin balancer
- [ ] Least connections balancer
- [ ] Random balancer
- [ ] Weighted balancer
- [ ] Per-route LB config

### Phase 4: Aegis Integration (Week 5) ✅
- [ ] Token validation middleware
- [ ] Token caching
- [ ] Claims forwarding
- [ ] Authorization rules

### Phase 5: Resilience (Week 6) ✅
- [ ] Circuit breaker
- [ ] Retry logic
- [ ] Timeout management
- [ ] Graceful degradation

### Phase 6: Observability (Week 7) ✅
- [ ] Prometheus metrics
- [ ] Distributed tracing
- [ ] Access logging
- [ ] Performance monitoring

### Phase 7: Advanced Features (Future)
- [ ] WebSocket proxying
- [ ] gRPC support
- [ ] Request/response transformation
- [ ] Rate limiting
- [ ] API versioning
- [ ] Blue/green deployments
- [ ] Canary releases
- [ ] Consul/etcd integration
- [ ] Admin web UI

---

## 🤝 Integration with Aegis

### Authentication Flow
```
1. Client → Hermes: GET /api/users (with Authorization: Bearer <token>)
2. Hermes → Aegis: POST /api/auth/validate {"token": "<token>"}
3. Aegis → Hermes: {"valid": true, "user": {...}}
4. Hermes → Backend: GET /api/users (with X-User-* headers)
5. Backend → Hermes: Response
6. Hermes → Client: Response
```

### Header Forwarding
Hermes forwards user claims as headers to backend services:
- `X-User-ID`: User UUID
- `X-User-Email`: User email
- `X-User-Roles`: Comma-separated roles
- `X-User-Permissions`: Comma-separated permissions
- `X-Request-ID`: Trace ID for distributed tracing

Backend services can trust these headers for authorization without calling Aegis directly.

---

## 🔒 Security Considerations

1. **HTTPS Only in Production** - Terminate TLS at gateway
2. **Aegis Token Validation** - Always validate tokens, cache carefully
3. **Header Sanitization** - Remove X-User-* headers from client requests
4. **Rate Limiting** - Protect backend services from abuse
5. **Input Validation** - Validate all configuration and API inputs
6. **Secret Management** - Don't hardcode Aegis URLs/credentials
7. **Audit Logging** - Log all admin API operations

---

## 📝 Coding Standards

### Go Best Practices
- Follow standard Go project layout
- Use interfaces for testability
- Prefer composition over inheritance
- Handle errors explicitly, never ignore
- Use context for cancellation and timeouts
- Document exported functions and types

### Error Handling
```go
// Good: Specific error types
var (
    ErrServiceNotFound = errors.New("service not found")
    ErrNoHealthyInstances = errors.New("no healthy instances available")
    ErrCircuitOpen = errors.New("circuit breaker is open")
)

// Good: Wrap errors with context
return fmt.Errorf("failed to register service %s: %w", name, err)
```

### Logging
```go
// Use structured logging
log.WithFields(log.Fields{
    "service": serviceName,
    "instance": instanceID,
    "duration": duration,
}).Info("Request completed")
```

### Testing
```go
// Table-driven tests
func TestLoadBalancer(t *testing.T) {
    tests := []struct {
        name      string
        instances []*ServiceInstance
        want      string
        wantErr   bool
    }{
        // Test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic...
        })
    }
}
```

---

## 🎯 Success Criteria

### Functional Requirements
- ✅ Routes requests to backend services
- ✅ Discovers and tracks service instances
- ✅ Validates tokens via Aegis
- ✅ Load balances across instances
- ✅ Monitors service health
- ✅ Handles failures gracefully

### Non-Functional Requirements
- ✅ Latency overhead <5ms (p95)
- ✅ Handles 10k+ req/s per instance
- ✅ Test coverage >85%
- ✅ Zero-downtime config reload
- ✅ Comprehensive observability
- ✅ Production-ready documentation

### Quality Gates
- All tests passing
- No race conditions (go test -race)
- No memory leaks
- Benchmark results meet targets
- Security review completed
- Documentation complete

---

## 📚 References

- [Reverse Proxy Pattern](https://microservices.io/patterns/apigateway.html)
- [Service Discovery](https://microservices.io/patterns/service-registry.html)
- [Circuit Breaker](https://martinfowler.com/bliki/CircuitBreaker.html)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/)
- [Go HTTP Reverse Proxy](https://pkg.go.dev/net/http/httputil#ReverseProxy)

---

**This document will evolve as we build Hermes. Keep it updated with learnings and design decisions!**
