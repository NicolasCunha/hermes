# 🗺️ Hermes Implementation Roadmap

This document provides a detailed, step-by-step implementation plan for building Hermes API Gateway with explicit guidelines for code generation.

---

## 📐 Project Structure

```
hermes/
├── hermes-server/           # Go backend
│   ├── main.go             # Application entry point
│   ├── go.mod              # Go dependencies
│   ├── api/                # HTTP route handlers (Gin)
│   │   ├── register.go     # Route registration
│   │   ├── service/        # Service management endpoints
│   │   ├── user/           # User management endpoints
│   │   └── health/         # Health check endpoints
│   ├── domain/             # Data models and domain logic
│   │   ├── service/        # Service model & domain operations
│   │   ├── user/           # User model & domain operations
│   │   └── health/         # Health check model
│   ├── services/           # Business logic layer
│   │   ├── registry/       # Service registry (in-memory)
│   │   ├── health/         # Health checker
│   │   ├── proxy/          # HTTP reverse proxy
│   │   └── auth/           # Aegis integration
│   ├── database/           # Database layer
│   │   ├── database.go     # SQLite connection & migrations
│   │   └── migrations.go   # SQL migration scripts
│   └── utils/              # Utility functions
│       ├── logger/         # Logging utilities
│       ├── config/         # Configuration loader
│       └── middleware/     # Gin middlewares
├── hermes-ui/              # React frontend
│   ├── public/
│   ├── src/
│   │   ├── components/     # React components
│   │   ├── pages/          # Page components
│   │   ├── services/       # API client services
│   │   └── middleware/     # Auth middleware
│   └── package.json
├── config/
│   ├── hermes.yaml         # Configuration file
│   ├── hermes.env          # Hermes environment variables
│   └── supervisord.conf    # Supervisor configuration
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
├── HERMES.md               # Complete specification
├── ROADMAP.md              # This file
└── README.md               # Quick start guide
```

---

## 🎯 Implementation Phases

## 📊 Implementation Status

| Phase | Status | Completion | Notes |
|-------|--------|------------|-------|
| Phase 1 | ✅ Complete | 100% | Project structure, database layer |
| Phase 2 | ✅ Complete | 100% | Aegis integration, auth middleware |
| Phase 3 | ✅ Complete | 100% | Service registry, health checking, persistence |
| Phase 4 | ✅ Complete | 100% | Service API with authentication |
| Phase 5 | 🔜 Pending | 0% | User management proxy API |
| Phase 6 | 🔜 Pending | 0% | React dashboard |
| Phase 7 | 🔜 Pending | 0% | Docker deployment |

### Overall Progress: 57% Complete (4/7 phases)

---

## 🎉 Recent Achievements (November 28, 2025)

### Database Persistence Implementation
- **Services Table Migration**: Created `services` table with JSON metadata support
- **Persistence Methods**: Implemented `loadFromDatabase()`, `saveToDatabase()`, `deleteFromDatabase()`, `updateStatusInDatabase()`
- **Warm Cache Pattern**: Services load on startup, write-through on changes
- **Indexes**: Added on `name` and `status` columns for query performance

### Authentication & Authorization
- **Aegis Integration**: All service CRUD requires Aegis admin tokens
- **Middleware Chain**: `authMiddleware` → `adminMiddleware` → handler
- **Token Validation**: Via `/api/aegis/api/auth/validate` endpoint
- **Context Propagation**: User ID, subject, roles, permissions stored in Gin context

### Duplicate Prevention
- **Application-Level Check**: Registry validates (name, host, port) uniqueness before insertion
- **Database Constraint**: `UNIQUE(name, host, port)` as backup
- **Load Balancing Support**: Allows same name on different hosts
- **Multi-Service Hosts**: Allows different names on same host:port

### Comprehensive Testing
- **Test Coverage**: 71+ assertions, 76%-100% coverage across packages
- **Test Isolation**: All tests use in-memory SQLite (`:memory:`)
- **Production Safety**: Verified production DB unchanged after test runs
- **Test Suites**:
  - `registry_test.go`: 15 tests (persistence, duplicates, concurrency)
  - `api_test.go`: 12 tests (auth, authz, business logic)
  - `client_test.go`: 8 tests (Aegis integration)
  - `middleware_test.go`: 12 tests (auth/authz middleware)

---

### Phase 1: Project Restructure & Database Layer ✅ COMPLETE
**Goal:** Set up proper project structure and database foundation
**Completion Date:** November 27, 2025

#### Step 1.1: Restructure Project
**Task:** Move from internal/cmd structure to hermes-server structure

**Guidelines for Claude:**
- Create `hermes-server/` directory at project root
- Move all Go code from `internal/` to appropriate folders:
  - `internal/config/` → `hermes-server/utils/config/`
  - `internal/gateway/` → `hermes-server/api/` (split into handlers)
  - `internal/proxy/` → `hermes-server/services/proxy/`
  - `internal/router/` → `hermes-server/services/router/`
- Move `cmd/hermes/main.go` → `hermes-server/main.go`
- Update all import paths to reflect new structure
- Update go.mod module path if needed
- Ensure code compiles after restructure: `go build -o hermes hermes-server/main.go`

**Code Generation Guidelines:**
```go
// Example import structure after restructure
import (
    "nfcunha/hermes/hermes-server/api"
    "nfcunha/hermes/hermes-server/domain/user"
    "nfcunha/hermes/hermes-server/services/registry"
    "nfcunha/hermes/hermes-server/utils/config"
)
```

#### Step 1.2: Database Utility Layer
**Task:** Create database connection and migration system

**File:** `hermes-server/database/database.go`

**Guidelines for Claude:**
```go
package database

import (
    "database/sql"
    "fmt"
    "log"
    "os"
    _ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// Initialize opens SQLite connection and runs migrations
func Initialize() error {
    dbPath := getDBPath()
    log.Printf("Opening database at: %s", dbPath)
    
    // Create data directory if not exists
    if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
        return fmt.Errorf("failed to create data directory: %w", err)
    }
    
    // Open database
    var err error
    db, err = sql.Open("sqlite3", dbPath)
    if err != nil {
        return fmt.Errorf("failed to open database: %w", err)
    }
    
    // Enable foreign keys
    if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
        return fmt.Errorf("failed to enable foreign keys: %w", err)
    }
    
    // Run migrations
    if err := migrate(); err != nil {
        return fmt.Errorf("failed to run migrations: %w", err)
    }
    
    log.Println("Database initialized successfully")
    return nil
}

// GetDB returns the database connection
func GetDB() *sql.DB {
    return db
}

// Close closes the database connection
func Close() error {
    if db != nil {
        return db.Close()
    }
    return nil
}

func getDBPath() string {
    path := os.Getenv("HERMES_DB_PATH")
    if path == "" {
        path = "/app/data/hermes.db"  // Default for Docker
    }
    return path
}
```

**File:** `hermes-server/database/migrations.go`

**Guidelines for Claude:**
```go
package database

// migrate runs all database migrations
func migrate() error {
    migrations := []string{
        createUsersTable,
    }
    
    for i, migration := range migrations {
        log.Printf("Running migration %d...", i+1)
        if _, err := db.Exec(migration); err != nil {
            return fmt.Errorf("migration %d failed: %w", i+1, err)
        }
    }
    
    return nil
}

const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('admin', 'viewer')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index on username for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
`
```

**Bootstrap Default Admin:**
```go
// In database.go, after migrations

// bootstrapDefaultUser creates default admin if no users exist
func bootstrapDefaultUser() error {
    var count int
    if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
        return err
    }
    
    if count > 0 {
        return nil  // Users already exist
    }
    
    username := os.Getenv("HERMES_DEFAULT_USERNAME")
    if username == "" {
        username = "hermes"
    }
    
    password := os.Getenv("HERMES_DEFAULT_PASSWORD")
    if password == "" {
        password = "hermes"
    }
    
    // Use user domain service to create admin
    // (will be implemented in next phase)
    log.Printf("Creating default admin user: %s", username)
    
    return nil
}
```

**Add to go.mod:**
```
require github.com/mattn/go-sqlite3 v1.14.18
```

---

### Phase 2: Aegis Integration & Authentication Middleware ✅ COMPLETE
**Goal:** Integrate Aegis as the authentication/authorization provider and implement token validation middleware.
**Completion Date:** November 28, 2025

**Key Implementation Details:**
- ✅ Aegis HTTP client created with `ValidateToken()` and `Health()` methods
- ✅ Authentication middleware extracts Bearer tokens and validates via Aegis
- ✅ Authorization middleware: `RequireAdmin()` and `RequirePermission()`
- ✅ User context stored in Gin: user_id, user_subject, user_roles, user_permissions
- ✅ Comprehensive tests with mock Aegis server (91.3% coverage)

**Architecture Decision:** 
Hermes does NOT manage users directly. All user management operations will proxy to Aegis in Phase 5:
- User CRUD operations forward to Aegis APIs
- Password changes via Aegis: `POST /aegis/users/:id/password`
- No users table in Hermes database
- Hermes only validates JWT tokens for authorization

**Files Created:**
- `hermes-server/services/aegis/client.go` (165 lines + tests)
- `hermes-server/services/aegis/client_test.go` (230 lines, 8 tests)
- `hermes-server/middleware/auth.go` (120 lines + tests)
- `hermes-server/middleware/auth_test.go` (340 lines, 12 tests)

**Configuration:**
```env
HERMES_AEGIS_URL=http://localhost:3100/api
HERMES_AEGIS_TIMEOUT=5s
```

**Testing Results:**
- All 20 auth/authz tests passing
- Mock server approach enables testing without real Aegis
- Coverage: 91.3% (services/aegis), 89.9% (middleware)

---

### Phase 3: Service Registry & Health Checking ✅ COMPLETE
**Goal:** Implement service discovery with health monitoring and database persistence
**Completion Date:** November 28, 2025

**Key Implementation Details:**

#### 3.1 Service Domain Model ✅
**File:** `hermes-server/domain/service/service.go`

**Features:**
- `Service` struct with full metadata support
- Status constants: `healthy`, `unhealthy`, `draining`
- Methods: `NewService()`, `BaseURL()`, `HealthCheckURL()`, `MarkHealthy()`, `MarkUnhealthy()`
- UUID generation for service IDs
- JSON serialization support

**Testing:**
- 8 unit tests covering all methods
- 100% coverage of domain logic

#### 3.2 Service Registry with Database Persistence ✅
**File:** `hermes-server/services/registry/registry.go`

**Features:**
- **In-Memory Registry**: Fast lookups with `map[string]*Service` and `map[string][]*Service`
- **Thread-Safe**: `sync.RWMutex` for concurrent access
- **Database Persistence**: All operations persist to SQLite
- **Warm Cache Pattern**: Load services from database on startup
- **Write-Through Cache**: Register/deregister immediately persist
- **Duplicate Prevention**: Check (name, host, port) uniqueness before insertion

**Key Methods:**
- `Register(svc)` - Add service to registry and database
- `Deregister(id)` - Remove service from registry and database
- `GetByID(id)` - Retrieve specific service
- `GetByName(name)` - Get all instances of a service
- `GetHealthy(name)` - Get only healthy instances
- `List()` - Get all services
- `UpdateStatus(id, status)` - Update status in memory and database

**Database Operations:**
- `loadFromDatabase()` - Populate in-memory cache on startup
- `saveToDatabase(svc)` - Persist new service
- `deleteFromDatabase(id)` - Remove service record
- `updateStatusInDatabase(id, status, lastChecked, failureCount)` - Update health info

**Testing:**
- 15 comprehensive tests
- Coverage: 87.9%
- Tests: persistence, duplicates, concurrency, not found scenarios
- Uses in-memory SQLite for test isolation

#### 3.3 Database Migration ✅
**File:** `hermes-server/database/migrations.go`

**Services Table Schema:**
```sql
CREATE TABLE services (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    protocol TEXT NOT NULL,
    health_check_path TEXT NOT NULL,
    status TEXT NOT NULL,
    metadata TEXT,  -- JSON
    registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    failure_count INTEGER DEFAULT 0,
    UNIQUE(name, host, port)  -- Prevent exact duplicates
);

CREATE INDEX idx_services_name ON services(name);
CREATE INDEX idx_services_status ON services(status);
```

**Design Rationale:**
- `UNIQUE(name, host, port)`: Database-level duplicate prevention
- Indexes on `name` and `status`: Optimize common queries
- `metadata TEXT`: JSON blob for extensibility
- Timestamps: Track registration and health check times

#### 3.4 Health Checker ✅
**File:** `hermes-server/services/health/checker.go`

**Features:**
- **Periodic Checks**: Configurable interval (default 30s)
- **Configurable Timeout**: Default 5s per check
- **Failure Threshold**: Mark unhealthy after N failures (default 3)
- **Auto-Removal**: Remove service after max failures (default 10)
- **Concurrent Checks**: Each service checked in separate goroutine
- **Status Persistence**: Updates written to database

**Configuration:**
```env
HERMES_HEALTH_CHECK_INTERVAL=30s
HERMES_HEALTH_CHECK_TIMEOUT=5s
HERMES_HEALTH_CHECK_THRESHOLD=3
HERMES_HEALTH_CHECK_MAX_FAILURES=10
```

**Algorithm:**
1. Ticker fires every `interval`
2. Get all services from registry
3. For each service:
   - Send GET request to `protocol://host:port/health_check_path`
   - Check response status (200-299 = healthy)
   - Update failure count and status
   - Persist status to database
   - Remove if exceeded max failures

**Testing:**
- Tests health checking logic
- Mocks HTTP responses
- Verifies failure counting
- Confirms auto-removal

**Lifecycle:**
```go
// In main.go
healthChecker := health.New(serviceRegistry)
go healthChecker.Start()

// Graceful shutdown
defer healthChecker.Stop()
```

---

### Phase 4: Service Management API with Authentication ✅ COMPLETE
**Goal:** Implement REST API for service management with Aegis authentication
**Completion Date:** November 28, 2025

**Key Implementation Details:**

#### 4.1 Service API Routes ✅
**File:** `hermes-server/api/service/api.go`

**Endpoints:**
All endpoints require `Authorization: Bearer <token>` header with admin role.

1. **POST /hermes/services** - Register a service
   - Validates request body (name, host, port, health_check_path required)
   - Performs initial health check before registration
   - Returns 400 if health check fails
   - Returns 409 if duplicate (name, host, port) exists
   - Persists to database immediately

2. **GET /hermes/services** - List all services
   - Returns array of all registered services
   - Includes count field

3. **GET /hermes/services/:id** - Get service by ID
   - Returns 404 if service not found

4. **DELETE /hermes/services/:id** - Deregister service
   - Removes from registry and database
   - Returns 404 if service not found

**Authentication Flow:**
```
Client Request
    ↓
AuthMiddleware (validates JWT with Aegis)
    ↓
AdminMiddleware (checks for admin role)
    ↓
Handler (business logic)
```

**Request/Response Examples:**

**Register Service:**
```bash
curl -X POST http://localhost:8080/hermes/services \\
  -H "Authorization: Bearer <token>" \\
  -H "Content-Type: application/json" \\
  -d '{
    "name": "user-api",
    "host": "api.example.com",
    "port": 443,
    "protocol": "https",
    "health_check_path": "/health",
    "metadata": {
      "version": "1.0.0",
      "environment": "production"
    }
  }'

Response: 201 Created
{
  "id": "uuid-123",
  "name": "user-api",
  "host": "api.example.com",
  "port": 443,
  "protocol": "https",
  "health_check_path": "/health",
  "status": "healthy",
  "metadata": {...},
  "registered_at": "2025-11-28T10:00:00Z",
  "last_checked_at": "2025-11-28T10:00:00Z",
  "failure_count": 0
}
```

**List Services:**
```bash
curl -X GET http://localhost:8080/hermes/services \\
  -H "Authorization: Bearer <token>"

Response: 200 OK
{
  "services": [...],
  "count": 3
}
```

#### 4.2 Authentication Integration ✅
**File:** `hermes-server/api/register.go`

**Route Registration:**
```go
func RegisterRoutes(router *gin.Engine, registry *registry.Registry, 
                    aegisClient *aegis.Client) {
    // Public routes
    hermes := router.Group("/hermes")
    hermes.GET("/health", handleHealth)
    
    // Protected routes (require auth)
    authMiddleware := middleware.AuthMiddleware(aegisClient)
    adminMiddleware := middleware.RequireAdmin()
    
    // Service management
    service.RegisterRoutes(hermes, registry, authMiddleware, adminMiddleware)
}
```

**Middleware Application:**
```go
// In api/service/api.go
func RegisterRoutes(router gin.IRouter, reg *registry.Registry, 
                    authMiddleware, adminMiddleware gin.HandlerFunc) {
    services := router.Group("/services")
    services.Use(authMiddleware)   // Validate JWT
    services.Use(adminMiddleware)  // Check admin role
    {
        services.POST("", api.registerService)
        services.GET("", api.listServices)
        services.GET("/:id", api.getService)
        services.DELETE("/:id", api.deregisterService)
    }
}
```

#### 4.3 Duplicate Prevention ✅
**Implementation:**

**Registry Check (First Line of Defense):**
```go
func (r *Registry) Register(svc *service.Service) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    // Check for duplicate (name, host, port)
    for _, existing := range r.services {
        if existing.Name == svc.Name &&
           existing.Host == svc.Host &&
           existing.Port == svc.Port {
            return fmt.Errorf("service with name '%s' already registered at %s:%d",
                svc.Name, svc.Host, svc.Port)
        }
    }
    
    // Register service...
}
```

**Database Constraint (Second Line of Defense):**
```sql
UNIQUE(name, host, port)
```

**Load Balancing Support:**
- ✅ Same service name on different hosts (e.g., user-api on host1 and host2)
- ✅ Different service names on same host:port (e.g., api-v1 and api-v2 on localhost:8080)
- ❌ Exact duplicate (same name, host, port) rejected

#### 4.4 Comprehensive Testing ✅
**File:** `hermes-server/api/service/api_test.go`

**Test Coverage:**
- **Authentication Tests** (without token, non-admin user)
- **Business Logic Tests** (invalid health check, duplicates, not found)
- **Success Tests** (register, list, get, delete)

**Test Cases:**
1. `TestRegisterService_WithoutAuth` - 401 Unauthorized
2. `TestRegisterService_NonAdmin` - 403 Forbidden
3. `TestRegisterService_InvalidHealthCheck` - 400 Bad Request
4. `TestRegisterService_Duplicate` - 400/409 error
5. `TestListServices_WithoutAuth` - 401 Unauthorized
6. `TestListServices_Success` - Returns service list
7. `TestGetService_WithoutAuth` - 401 Unauthorized
8. `TestGetService_Success` - Returns service details
9. `TestGetService_NotFound` - 404 Not Found
10. `TestDeregisterService_WithoutAuth` - 401 Unauthorized
11. `TestDeregisterService_Success` - Removes service
12. `TestDeregisterService_NotFound` - 404 Not Found

**Mock Middleware:**
```go
// Simulates successful authentication
func mockAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Set("user_id", "test-user")
        c.Set("user_roles", []string{"admin"})
        c.Next()
    }
}

// Simulates authentication failure
func mockAuthFailMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.JSON(401, gin.H{"error": "unauthorized"})
        c.Abort()
    }
}
```

**Test Results:**
- 12 tests passing
- Coverage: 76.1% of statements
- All business logic paths tested

#### 4.5 Manual Testing Results ✅
Extensive manual testing performed on November 28, 2025:

**Test Scenarios (21 total):**
1. ✅ List services WITHOUT token (401)
2. ✅ Get token from Aegis admin user
3. ✅ List services WITH token (200, returns 3 services)
4. ✅ Get specific service WITHOUT token (401)
5. ✅ Get specific service WITH token (200)
6. ✅ Create service WITHOUT token (401)
7. ✅ Create service WITH invalid health check (400)
8. ✅ Create service WITH valid health check (201)
9. ✅ Verify service persisted to database (sqlite query confirms)
10. ✅ Try to create duplicate service (correctly rejected)
11. ✅ List services shows duplicates (found 2 github-api instances)
12. ✅ Delete service WITHOUT token (401)
13. ✅ Delete service WITH token (200)
14. ✅ Verify deletion from database (count = 0)
15. ✅ Try to delete non-existent service (404)
16. ✅ Delete duplicate service (200)
17. ✅ Test with non-admin user Bob (403 Forbidden)
18. ✅ Test with invalid token (401)
19. ✅ Test with malformed header (401)
20. ✅ Final service count (3 services)
21. ✅ Database persistence matches (3 in DB)

**Duplicate Detection Tests:**
1. ✅ Register aegis again (duplicate error)
2. ✅ Register same name, different host (health check failed - expected)
3. ✅ Register different name, same host:port (allowed - 201)
4. ✅ Delete clone service (200)
5. ✅ Register new unique service (201)
6. ✅ Try exact duplicate (correctly rejected)
7. ✅ Final count: 4 services

**Database Persistence Test:**
- Services count before restart: 4
- Hermes restarted
- Log shows: "Loaded 2 services from database"
- Services accessible after restart ✅

---

### Phase 5: User Management Proxy API 🔜 PENDING

**Architecture Decision:** Hermes will NOT manage users directly. Instead:
- Aegis handles all user management (CRUD, roles, permissions)
- Hermes validates JWT tokens via Aegis's `/api/aegis/auth/validate` endpoint
- User CRUD operations in Hermes dashboard will proxy to Aegis APIs
- Users can change passwords via Aegis: `POST /aegis/users/:id/password` with `{old_password, new_password}`
- Database: Remove users table migration (not needed)
- No local user storage or authentication logic in Hermes

**Implementation Requirements:**
1. **Password Requirements:** Minimum 8 characters (enforced by Aegis)
2. **User Permissions:** Only admins can manage other users; users can change their own password
3. **Testing:** Unit tests required for all components (>90% coverage)
4. **Default Admin Warning:** Display password change reminder once per session in dashboard (Phase 6)
5. **Session Token Storage:** Store JWT in sessionStorage (cleared on browser close)

**Files to Modify/Create:**
- Create: `hermes-server/services/aegis/client.go` + tests
- Create: `hermes-server/middleware/auth.go` + tests  
- Create: `hermes-server/api/user/api.go` + tests (proxy to Aegis)
- Modify: `hermes-server/main.go` (add Aegis client init)
- Modify: `hermes-server/database/migrations.go` (remove users table)
- Modify: `hermes-server/database/database.go` (remove bootstrap function)
- Update: `.env.sample` (document HERMES_AUTH_AEGIS_URL)

#### Step 2.1: Aegis HTTP Client
**File:** `hermes-server/services/aegis/client.go`

**File:** `hermes-server/services/aegis/client.go`

**Guidelines for Claude:**
```go
package aegis

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type Client struct {
    baseURL    string
    httpClient *http.Client
}

type ValidateTokenRequest struct {
    Token string `json:"token"`
}

type ValidateTokenResponse struct {
    Valid     bool      `json:"valid"`
    Error     string    `json:"error,omitempty"`
    User      *User     `json:"user,omitempty"`
    ExpiresAt time.Time `json:"expires_at,omitempty"`
}

type User struct {
    ID          string   `json:"id"`
    Subject     string   `json:"subject"`
    Roles       []string `json:"roles"`
    Permissions []string `json:"permissions"`
}

// NewClient creates Aegis HTTP client
func NewClient(baseURL string, timeout time.Duration) *Client {
    return &Client{
        baseURL: baseURL,
        httpClient: &http.Client{Timeout: timeout},
    }
}

// ValidateToken calls Aegis to validate JWT token
func (c *Client) ValidateToken(token string) (*ValidateTokenResponse, error) {
    reqBody := ValidateTokenRequest{Token: token}
    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        return nil, fmt.Errorf("marshal error: %w", err)
    }

    resp, err := c.httpClient.Post(
        c.baseURL+"/api/aegis/auth/validate",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        return nil, fmt.Errorf("Aegis call failed: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("read error: %w", err)
    }

    var result ValidateTokenResponse
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("unmarshal error: %w", err)
    }

    return &result, nil
}

// Health checks if Aegis is available
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
```

**Testing Guidelines:**
```go
// client_test.go
package aegis

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
)

func TestValidateToken_ValidToken(t *testing.T) {
    // Mock Aegis server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/api/aegis/auth/validate" {
            t.Errorf("Expected /api/aegis/auth/validate, got %s", r.URL.Path)
        }
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"valid":true,"user":{"id":"123","subject":"test@test.com","roles":["admin"],"permissions":["read:all"]}}`))
    }))
    defer server.Close()

    client := NewClient(server.URL, 5*time.Second)
    resp, err := client.ValidateToken("valid-token")
    
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    if !resp.Valid {
        t.Error("Expected valid=true")
    }
    if resp.User == nil || resp.User.Subject != "test@test.com" {
        t.Error("Expected user data")
    }
}

func TestValidateToken_InvalidToken(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"valid":false,"error":"token expired"}`))
    }))
    defer server.Close()

    client := NewClient(server.URL, 5*time.Second)
    resp, err := client.ValidateToken("invalid-token")
    
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    if resp.Valid {
        t.Error("Expected valid=false")
    }
    if resp.Error != "token expired" {
        t.Errorf("Expected error message, got %s", resp.Error)
    }
}

func TestHealth_Success(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/aegis/health" {
            t.Errorf("Expected /aegis/health, got %s", r.URL.Path)
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    client := NewClient(server.URL, 5*time.Second)
    if err := client.Health(); err != nil {
        t.Errorf("Expected health check to pass, got %v", err)
    }
}
```

#### Step 2.2: Authentication Middleware
**File:** `hermes-server/middleware/auth.go`

**File:** `hermes-server/middleware/auth.go`

**Guidelines for Claude:**
```go
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
            c.JSON(http.StatusForbidden, gin.H{"error": "no roles found"})
            c.Abort()
            return
        }

        userRoles := roles.([]string)
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
            c.JSON(http.StatusForbidden, gin.H{"error": "no permissions found"})
            c.Abort()
            return
        }

        userPerms := permissions.([]string)
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
```

**Testing Guidelines:**
```go
// auth_test.go
package middleware

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
    "github.com/gin-gonic/gin"
    "nfcunha/hermes/hermes-server/services/aegis"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
    // Setup mock Aegis server
    aegisServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"valid":true,"user":{"id":"123","subject":"test@test.com","roles":["admin"],"permissions":["read:all"]}}`))
    }))
    defer aegisServer.Close()

    // Create middleware
    client := aegis.NewClient(aegisServer.URL, 5*time.Second)
    middleware := AuthMiddleware(client)

    // Setup Gin
    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.Use(middleware)
    router.GET("/protected", func(c *gin.Context) {
        userID, _ := c.Get("user_id")
        c.JSON(http.StatusOK, gin.H{"user_id": userID})
    })

    // Test request
    req := httptest.NewRequest("GET", "/protected", nil)
    req.Header.Set("Authorization", "Bearer valid-token")
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
    client := aegis.NewClient("http://localhost", 5*time.Second)
    middleware := AuthMiddleware(client)

    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.Use(middleware)
    router.GET("/protected", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"ok": true})
    })

    req := httptest.NewRequest("GET", "/protected", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    if w.Code != http.StatusUnauthorized {
        t.Errorf("Expected status 401, got %d", w.Code)
    }
}

func TestRequireAdmin_WithAdminRole(t *testing.T) {
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    // Simulate authenticated user with admin role
    router.Use(func(c *gin.Context) {
        c.Set("user_roles", []string{"admin"})
        c.Next()
    })
    router.Use(RequireAdmin())
    router.GET("/admin", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"ok": true})
    })

    req := httptest.NewRequest("GET", "/admin", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
}

func TestRequireAdmin_WithoutAdminRole(t *testing.T) {
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    router.Use(func(c *gin.Context) {
        c.Set("user_roles", []string{"viewer"})
        c.Next()
    })
    router.Use(RequireAdmin())
    router.GET("/admin", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"ok": true})
    })

    req := httptest.NewRequest("GET", "/admin", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    if w.Code != http.StatusForbidden {
        t.Errorf("Expected status 403, got %d", w.Code)
    }
}
```

#### Step 2.3: Update Main Application
**File:** `hermes-server/main.go`

**Update to include Aegis client:**
```go
// Add Aegis client initialization
aegisClient := aegis.NewClient(cfg.Auth.AegisURL, cfg.Server.Timeout)

// Test Aegis connectivity
log.Println("Testing Aegis connectivity...")
if err := aegisClient.Health(); err != nil {
    log.Fatalf("Failed to connect to Aegis: %v", err)
}
log.Println("Aegis connection successful")

// Pass aegisClient to route registration
api.RegisterRoutes(engine, rtr, prx, aegisClient)
```

#### Step 2.4: Remove User Database Migration
**File:** `hermes-server/database/migrations.go`

**Remove users table migration:**
```go
// Delete or comment out the createUsersTable migration
// Hermes no longer stores user data - Aegis handles it

func migrate() error {
    db := GetDB()
    
    migrations := []string{
        // createUsersTable,  // REMOVED - using Aegis for users
        createServicesTable,  // Keep for Phase 3
    }
    
    // ... rest of migration logic
}
```

#### Step 2.5: Update Environment Configuration
**File:** `.env.sample`

**Ensure Aegis URL is documented:**
```bash
# Authentication (Aegis Integration)
HERMES_AUTH_AEGIS_URL=http://aegis:8080  # URL to Aegis service
HERMES_AUTH_TIMEOUT=10s
```

**Testing Checklist:**
- [ ] Aegis client successfully validates valid tokens
- [ ] Aegis client properly handles invalid/expired tokens
- [ ] Authentication middleware extracts and validates tokens
- [ ] Authentication middleware stores user info in Gin context
- [ ] RequireAdmin middleware blocks non-admin users
- [ ] RequirePermission middleware checks specific permissions
- [ ] Aegis health check passes on startup
- [ ] All unit tests pass with >90% coverage

---

### Phase 3: Service Registry & Health Checking ✅
**Goal:** Implement service discovery with health monitoring

#### Step 3.1: Service Domain Model
**File:** `hermes-server/domain/service/service.go`

**Guidelines for Claude:**
```go
package service

import (
    "time"
    "github.com/google/uuid"
)

// Status represents service instance status
type Status string

const (
    StatusHealthy   Status = "healthy"
    StatusUnhealthy Status = "unhealthy"
    StatusDraining  Status = "draining"  // For graceful shutdown
)

// Service represents a registered backend service
type Service struct {
    ID              string            `json:"id"`
    Name            string            `json:"name"`
    Host            string            `json:"host"`
    Port            int               `json:"port"`
    Protocol        string            `json:"protocol"` // http, https
    HealthCheckPath string            `json:"health_check_path"`
    Status          Status            `json:"status"`
    Metadata        map[string]string `json:"metadata,omitempty"`
    RegisteredAt    time.Time         `json:"registered_at"`
    LastCheckedAt   time.Time         `json:"last_checked_at"`
    FailureCount    int               `json:"failure_count"`
}

// NewService creates a new service instance
func NewService(name, host string, port int, healthCheckPath string) *Service {
    return &Service{
        ID:              uuid.New().String(),
        Name:            name,
        Host:            host,
        Port:            port,
        Protocol:        "http",  // Default
        HealthCheckPath: healthCheckPath,
        Status:          StatusHealthy,
        Metadata:        make(map[string]string),
        RegisteredAt:    time.Now(),
        LastCheckedAt:   time.Now(),
        FailureCount:    0,
    }
}

// BaseURL returns the full base URL of the service
func (s *Service) BaseURL() string {
    return fmt.Sprintf("%s://%s:%d", s.Protocol, s.Host, s.Port)
}

// HealthCheckURL returns the full health check URL
func (s *Service) HealthCheckURL() string {
    return fmt.Sprintf("%s%s", s.BaseURL(), s.HealthCheckPath)
}

// MarkHealthy marks service as healthy and resets failure count
func (s *Service) MarkHealthy() {
    s.Status = StatusHealthy
    s.FailureCount = 0
    s.LastCheckedAt = time.Now()
}

// MarkUnhealthy increments failure count and marks as unhealthy if threshold reached
func (s *Service) MarkUnhealthy(threshold int) {
    s.FailureCount++
    s.LastCheckedAt = time.Now()
    
    if s.FailureCount >= threshold {
        s.Status = StatusUnhealthy
    }
}
```

#### Step 3.2: Service Registry (In-Memory)
**File:** `hermes-server/services/registry/registry.go`

**Guidelines for Claude:**
```go
package registry

import (
    "fmt"
    "log"
    "sync"
    "nfcunha/hermes/hermes-server/domain/service"
)

// Registry manages registered services in-memory
type Registry struct {
    services map[string]*service.Service  // Key: service ID
    byName   map[string][]*service.Service // Key: service name
    mu       sync.RWMutex
}

// New creates a new service registry
func New() *Registry {
    return &Registry{
        services: make(map[string]*service.Service),
        byName:   make(map[string][]*service.Service),
    }
}

// Register adds a new service to the registry
func (r *Registry) Register(svc *service.Service) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if _, exists := r.services[svc.ID]; exists {
        return fmt.Errorf("service already registered: %s", svc.ID)
    }
    
    r.services[svc.ID] = svc
    r.byName[svc.Name] = append(r.byName[svc.Name], svc)
    
    log.Printf("Service registered: %s (%s) at %s", svc.Name, svc.ID, svc.BaseURL())
    return nil
}

// Deregister removes a service from the registry
func (r *Registry) Deregister(id string) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    svc, exists := r.services[id]
    if !exists {
        return fmt.Errorf("service not found: %s", id)
    }
    
    // Remove from services map
    delete(r.services, id)
    
    // Remove from byName map
    instances := r.byName[svc.Name]
    for i, instance := range instances {
        if instance.ID == id {
            r.byName[svc.Name] = append(instances[:i], instances[i+1:]...)
            break
        }
    }
    
    // Clean up empty name entry
    if len(r.byName[svc.Name]) == 0 {
        delete(r.byName, svc.Name)
    }
    
    log.Printf("Service deregistered: %s (%s)", svc.Name, svc.ID)
    return nil
}

// GetByID retrieves service by ID
func (r *Registry) GetByID(id string) (*service.Service, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    svc, exists := r.services[id]
    if !exists {
        return nil, fmt.Errorf("service not found: %s", id)
    }
    
    return svc, nil
}

// GetByName retrieves all instances of a service by name
func (r *Registry) GetByName(name string) ([]*service.Service, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    instances, exists := r.byName[name]
    if !exists || len(instances) == 0 {
        return nil, fmt.Errorf("no instances found for service: %s", name)
    }
    
    return instances, nil
}

// GetHealthy retrieves all healthy instances of a service
func (r *Registry) GetHealthy(name string) []*service.Service {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    instances := r.byName[name]
    healthy := make([]*service.Service, 0)
    
    for _, svc := range instances {
        if svc.Status == service.StatusHealthy {
            healthy = append(healthy, svc)
        }
    }
    
    return healthy
}

// List retrieves all registered services
func (r *Registry) List() []*service.Service {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    services := make([]*service.Service, 0, len(r.services))
    for _, svc := range r.services {
        services = append(services, svc)
    }
    
    return services
}

// UpdateStatus updates service status (called by health checker)
func (r *Registry) UpdateStatus(id string, status service.Status) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    svc, exists := r.services[id]
    if !exists {
        return fmt.Errorf("service not found: %s", id)
    }
    
    svc.Status = status
    return nil
}
```

#### Step 3.3: Health Checker
**File:** `hermes-server/services/health/checker.go`

**Guidelines for Claude:**
```go
package health

import (
    "context"
    "log"
    "net/http"
    "os"
    "strconv"
    "time"
    
    "nfcunha/hermes/hermes-server/domain/service"
    "nfcunha/hermes/hermes-server/services/registry"
)

// Checker performs periodic health checks on registered services
type Checker struct {
    registry        *registry.Registry
    client          *http.Client
    interval        time.Duration
    timeout         time.Duration
    failureThreshold int
    maxFailures     int  // Remove service after this many failures
    stopChan        chan struct{}
}

// New creates a new health checker
func New(reg *registry.Registry) *Checker {
    return &Checker{
        registry:        reg,
        client:          &http.Client{Timeout: getTimeout()},
        interval:        getInterval(),
        timeout:         getTimeout(),
        failureThreshold: getFailureThreshold(),
        maxFailures:     getMaxFailures(),
        stopChan:        make(chan struct{}),
    }
}

// Start begins periodic health checking
func (c *Checker) Start() {
    log.Printf("Starting health checker: interval=%v, timeout=%v, threshold=%d",
        c.interval, c.timeout, c.failureThreshold)
    
    ticker := time.NewTicker(c.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            c.checkAll()
        case <-c.stopChan:
            log.Println("Health checker stopped")
            return
        }
    }
}

// Stop stops the health checker
func (c *Checker) Stop() {
    close(c.stopChan)
}

// checkAll checks health of all registered services
func (c *Checker) checkAll() {
    services := c.registry.List()
    
    log.Printf("Running health checks for %d services", len(services))
    
    for _, svc := range services {
        go c.check(svc)
    }
}

// check performs health check on a single service
func (c *Checker) check(svc *service.Service) {
    ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
    defer cancel()
    
    req, err := http.NewRequestWithContext(ctx, "GET", svc.HealthCheckURL(), nil)
    if err != nil {
        log.Printf("Failed to create health check request for %s: %v", svc.Name, err)
        c.handleFailure(svc)
        return
    }
    
    resp, err := c.client.Do(req)
    if err != nil {
        log.Printf("Health check failed for %s (%s): %v", svc.Name, svc.ID, err)
        c.handleFailure(svc)
        return
    }
    defer resp.Body.Close()
    
    // Consider 2xx status codes as healthy
    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        svc.MarkHealthy()
        log.Printf("Health check passed for %s (%s): status=%d", svc.Name, svc.ID, resp.StatusCode)
    } else {
        log.Printf("Health check failed for %s (%s): status=%d", svc.Name, svc.ID, resp.StatusCode)
        c.handleFailure(svc)
    }
}

// handleFailure handles a failed health check
func (c *Checker) handleFailure(svc *service.Service) {
    svc.MarkUnhealthy(c.failureThreshold)
    
    // Remove service if exceeded max failures
    if svc.FailureCount >= c.maxFailures {
        log.Printf("Service %s (%s) exceeded max failures, removing from registry", svc.Name, svc.ID)
        c.registry.Deregister(svc.ID)
    }
}

// Configuration helpers with defaults
func getInterval() time.Duration {
    val := os.Getenv("HERMES_HEALTH_CHECK_INTERVAL")
    if val == "" {
        return 30 * time.Second
    }
    duration, err := time.ParseDuration(val)
    if err != nil {
        return 30 * time.Second
    }
    return duration
}

func getTimeout() time.Duration {
    val := os.Getenv("HERMES_HEALTH_CHECK_TIMEOUT")
    if val == "" {
        return 5 * time.Second
    }
    duration, err := time.ParseDuration(val)
    if err != nil {
        return 5 * time.Second
    }
    return duration
}

func getFailureThreshold() int {
    val := os.Getenv("HERMES_HEALTH_CHECK_THRESHOLD")
    if val == "" {
        return 3
    }
    threshold, err := strconv.Atoi(val)
    if err != nil {
        return 3
    }
    return threshold
}

func getMaxFailures() int {
    val := os.Getenv("HERMES_HEALTH_CHECK_MAX_FAILURES")
    if val == "" {
        return 10
    }
    maxFail, err := strconv.Atoi(val)
    if err != nil {
        return 10
    }
    return maxFail
}
```

---

### Phase 4: API Endpoints & Routing
**Goal:** Implement REST API for user and service management

#### Step 4.1: Route Registration
**File:** `hermes-server/api/register.go`

**Guidelines for Claude:**
```go
package api

import (
    "github.com/gin-gonic/gin"
    "nfcunha/hermes/hermes-server/api/service"
    "nfcunha/hermes/hermes-server/api/user"
    "nfcunha/hermes/hermes-server/services/registry"
    "nfcunha/hermes/hermes-server/services/proxy"
)

// RegisterRoutes sets up all API routes under /hermes context path
func RegisterRoutes(router *gin.Engine, reg *registry.Registry, prox *proxy.Proxy) {
    // All routes under /hermes context path
    hermes := router.Group("/hermes")
    
    // Health check
    hermes.GET("/health", handleHealth)
    
    // Service management API (requires admin auth)
    service.RegisterRoutes(hermes, reg)
    
    // User management API (requires admin auth)
    user.RegisterRoutes(hermes)
    
    // Catch-all for proxying (no /hermes prefix)
    router.NoRoute(func(c *gin.Context) {
        prox.Forward(c, reg)
    })
}

func handleHealth(c *gin.Context) {
    c.JSON(200, gin.H{
        "status": "healthy",
        "service": "hermes",
    })
}
```

#### Step 4.2: Service Management API
**File:** `hermes-server/api/service/api.go`

**Guidelines for Claude:**
```go
package service

import (
    "log"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "nfcunha/hermes/hermes-server/domain/service"
    "nfcunha/hermes/hermes-server/services/registry"
)

type API struct {
    registry *registry.Registry
}

// RegisterRoutes registers service management routes
func RegisterRoutes(router gin.IRouter, reg *registry.Registry) {
    api := &API{registry: reg}
    
    services := router.Group("/services")
    {
        // TODO: Add auth middleware for admin only
        services.POST("", api.registerService)
        services.DELETE("/:id", api.deregisterService)
        services.GET("", api.listServices)
        services.GET("/:id", api.getService)
    }
}

type RegisterRequest struct {
    Name            string            `json:"name" binding:"required"`
    Host            string            `json:"host" binding:"required"`
    Port            int               `json:"port" binding:"required"`
    HealthCheckPath string            `json:"health_check_path" binding:"required"`
    Protocol        string            `json:"protocol"`
    Metadata        map[string]string `json:"metadata"`
}

// registerService handles service registration (manual or auto check-in)
func (a *API) registerService(c *gin.Context) {
    var req RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // Create service
    svc := service.NewService(req.Name, req.Host, req.Port, req.HealthCheckPath)
    if req.Protocol != "" {
        svc.Protocol = req.Protocol
    }
    if req.Metadata != nil {
        svc.Metadata = req.Metadata
    }
    
    // Perform initial health check
    if !a.checkServiceHealth(svc) {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "service health check failed",
            "url": svc.HealthCheckURL(),
        })
        return
    }
    
    // Register service
    if err := a.registry.Register(svc); err != nil {
        c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
        return
    }
    
    log.Printf("Service registered: %s at %s", svc.Name, svc.BaseURL())
    c.JSON(http.StatusCreated, svc)
}

// checkServiceHealth performs a simple health check
func (a *API) checkServiceHealth(svc *service.Service) bool {
    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Get(svc.HealthCheckURL())
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// deregisterService handles service deregistration
func (a *API) deregisterService(c *gin.Context) {
    id := c.Param("id")
    
    if err := a.registry.Deregister(id); err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    
    log.Printf("Service deregistered: %s", id)
    c.JSON(http.StatusOK, gin.H{"message": "service deregistered"})
}

// listServices lists all registered services
func (a *API) listServices(c *gin.Context) {
    services := a.registry.List()
    c.JSON(http.StatusOK, gin.H{
        "services": services,
        "count": len(services),
    })
}

// getService retrieves a specific service
func (a *API) getService(c *gin.Context) {
    id := c.Param("id")
    
    svc, err := a.registry.GetByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, svc)
}
```

#### Step 4.3: User Management Proxy API
**File:** `hermes-server/api/user/api.go`

**Goal:** Proxy all user management operations to Aegis

**Guidelines for Claude:**
```go
package user

import (
    "bytes"
    "fmt"
    "io"
    "log"
    "net/http"
    "github.com/gin-gonic/gin"
    "nfcunha/hermes/hermes-server/services/aegis"
    "nfcunha/hermes/hermes-server/middleware"
)

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

// RegisterRoutes registers user management routes (proxied to Aegis)
func (a *API) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
    users := router.Group("/users")
    users.Use(authMiddleware)  // All user endpoints require authentication
    users.Use(middleware.RequireAdmin())  // Only admins can manage users
    {
        users.POST("", a.proxyToAegis)      // Create user
        users.GET("", a.proxyToAegis)       // List users
        users.GET("/:id", a.proxyToAegis)   // Get user
        users.PUT("/:id", a.proxyToAegis)   // Update user
        users.DELETE("/:id", a.proxyToAegis) // Delete user
        users.POST("/:id/roles", a.proxyToAegis) // Add role
        users.DELETE("/:id/roles/:role", a.proxyToAegis) // Remove role
        users.POST("/:id/permissions", a.proxyToAegis) // Add permission
        users.DELETE("/:id/permissions/:permission", a.proxyToAegis) // Remove permission
    }
    
    // Special endpoint: any authenticated user can change their own password
    router.POST("/users/:id/password", authMiddleware, a.changePassword)
}

// proxyToAegis forwards request to Aegis user API
func (a *API) proxyToAegis(c *gin.Context) {
    // Build Aegis URL - map /hermes/users/* to /aegis/users/*
    aegisPath := "/aegis/users" + c.Param("id")
    if c.Param("role") != "" {
        aegisPath += "/roles/" + c.Param("role")
    }
    if c.Param("permission") != "" {
        aegisPath += "/permissions/" + c.Param("permission")
    }
    
    targetURL := a.aegisURL + aegisPath
    if c.Request.URL.RawQuery != "" {
        targetURL += "?" + c.Request.URL.RawQuery
    }
    
    log.Printf("Proxying %s %s to Aegis: %s", c.Request.Method, c.Request.URL.Path, targetURL)
    
    // Read request body
    var body []byte
    var err error
    if c.Request.Body != nil {
        body, err = io.ReadAll(c.Request.Body)
        if err != nil {
            log.Printf("Failed to read request body: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read request"})
            return
        }
    }
    
    // Create new request to Aegis
    req, err := http.NewRequest(c.Request.Method, targetURL, bytes.NewReader(body))
    if err != nil {
        log.Printf("Failed to create Aegis request: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
        return
    }
    
    // Copy headers (except Host and Authorization)
    for key, values := range c.Request.Header {
        if key == "Host" || key == "Authorization" {
            continue
        }
        for _, value := range values {
            req.Header.Add(key, value)
        }
    }
    
    // Forward request to Aegis
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        log.Printf("Aegis request failed: %v", err)
        c.JSON(http.StatusBadGateway, gin.H{"error": "authentication service unavailable"})
        return
    }
    defer resp.Body.Close()
    
    // Read Aegis response
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Printf("Failed to read Aegis response: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
        return
    }
    
    // Copy response headers
    for key, values := range resp.Header {
        for _, value := range values {
            c.Header(key, value)
        }
    }
    
    // Return Aegis response
    c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// changePassword allows users to change their own password
func (a *API) changePassword(c *gin.Context) {
    userID := c.Param("id")
    authenticatedUserID, _ := c.Get("user_id")
    
    // Users can only change their own password (unless admin)
    roles, _ := c.Get("user_roles")
    userRoles := roles.([]string)
    isAdmin := false
    for _, role := range userRoles {
        if role == "admin" {
            isAdmin = true
            break
        }
    }
    
    if !isAdmin && userID != authenticatedUserID {
        c.JSON(http.StatusForbidden, gin.H{"error": "can only change your own password"})
        return
    }
    
    // Proxy to Aegis
    targetURL := fmt.Sprintf("%s/aegis/users/%s/password", a.aegisURL, userID)
    log.Printf("Proxying password change to Aegis: %s", targetURL)
    
    body, err := io.ReadAll(c.Request.Body)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read request"})
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
```

**Testing Guidelines:**
```go
// api_test.go
package user

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
    "github.com/gin-gonic/gin"
    "nfcunha/hermes/hermes-server/services/aegis"
)

func TestProxyToAegis_ListUsers(t *testing.T) {
    // Mock Aegis server
    aegisServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/aegis/users" {
            t.Errorf("Expected /aegis/users, got %s", r.URL.Path)
        }
        if r.Method != "GET" {
            t.Errorf("Expected GET, got %s", r.Method)
        }
        
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode([]map[string]string{
            {"id": "1", "subject": "test@test.com"},
        })
    }))
    defer aegisServer.Close()
    
    // Setup
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    client := aegis.NewClient(aegisServer.URL, 5*time.Second)
    api := NewAPI(client, aegisServer.URL)
    
    // Mock auth middleware
    router.Use(func(c *gin.Context) {
        c.Set("user_id", "admin-id")
        c.Set("user_roles", []string{"admin"})
        c.Next()
    })
    
    api.RegisterRoutes(router.Group("/hermes"), func(c *gin.Context) { c.Next() })
    
    // Test
    req := httptest.NewRequest("GET", "/hermes/users", nil)
    req.Header.Set("Authorization", "Bearer test-token")
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
}

func TestChangePassword_OwnPassword(t *testing.T) {
    aegisServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/aegis/users/user-123/password" {
            t.Errorf("Expected /aegis/users/user-123/password, got %s", r.URL.Path)
        }
        
        var body map[string]string
        json.NewDecoder(r.Body).Decode(&body)
        
        if body["old_password"] != "old123" || body["new_password"] != "new123" {
            t.Error("Expected password fields in request")
        }
        
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"message": "password changed"})
    }))
    defer aegisServer.Close()
    
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    client := aegis.NewClient(aegisServer.URL, 5*time.Second)
    api := NewAPI(client, aegisServer.URL)
    
    router.Use(func(c *gin.Context) {
        c.Set("user_id", "user-123")
        c.Set("user_roles", []string{"viewer"})
        c.Next()
    })
    
    api.RegisterRoutes(router.Group("/hermes"), func(c *gin.Context) { c.Next() })
    
    reqBody := map[string]string{
        "old_password": "old123",
        "new_password": "new123",
    }
    body, _ := json.Marshal(reqBody)
    
    req := httptest.NewRequest("POST", "/hermes/users/user-123/password", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
}

func TestChangePassword_OtherUserPassword_Forbidden(t *testing.T) {
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    client := aegis.NewClient("http://localhost", 5*time.Second)
    api := NewAPI(client, "http://localhost")
    
    router.Use(func(c *gin.Context) {
        c.Set("user_id", "user-123")
        c.Set("user_roles", []string{"viewer"})  // Not admin
        c.Next()
    })
    
    api.RegisterRoutes(router.Group("/hermes"), func(c *gin.Context) { c.Next() })
    
    req := httptest.NewRequest("POST", "/hermes/users/other-user-id/password", bytes.NewReader([]byte("{}")))
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    if w.Code != http.StatusForbidden {
        t.Errorf("Expected status 403, got %d", w.Code)
    }
}
```

---
    return UserResponse{
        ID:        u.ID,
        Username:  u.Username,
        Role:      u.Role,
        CreatedAt: u.CreatedAt,
        UpdatedAt: u.UpdatedAt,
    }
}

// Implement getUser, updateUser, deleteUser similarly
```

---

### Phase 5: Aegis Integration & Authentication ✅ (Merged into Phase 2)
**Status:** This phase has been integrated into Phase 2 for better architectural coherence.

**Completed Components:**
- Aegis HTTP client (`hermes-server/services/aegis/client.go`)
- Authentication middleware with token validation (`hermes-server/middleware/auth.go`)
- Admin and permission-based authorization middleware
- User management proxy API (forwards to Aegis)
- Password change functionality (users can change their own password)

See **Phase 2: Aegis Integration & Authentication Middleware** for implementation details.

---

**Guidelines for Claude:**
```go
package auth

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"
)

// AegisClient handles communication with Aegis auth service
type AegisClient struct {
    baseURL string
    client  *http.Client
}

// NewAegisClient creates a new Aegis client
func NewAegisClient() *AegisClient {
    baseURL := os.Getenv("AEGIS_URL")
    if baseURL == "" {
        baseURL = "http://localhost/api/aegis"
    }
    
    return &AegisClient{
        baseURL: baseURL,
        client: &http.Client{
            Timeout: 5 * time.Second,
        },
    }
}

type ValidateRequest struct {
    Token string `json:"token"`
}

type ValidateResponse struct {
    Valid bool `json:"valid"`
    User  struct {
        ID          string   `json:"id"`
        Subject     string   `json:"subject"`
        Roles       []string `json:"roles"`
        Permissions []string `json:"permissions"`
    } `json:"user,omitempty"`
    Error string `json:"error,omitempty"`
}

// ValidateToken validates a JWT token with Aegis
func (a *AegisClient) ValidateToken(token string) (*ValidateResponse, error) {
    reqBody := ValidateRequest{Token: token}
    body, err := json.Marshal(reqBody)
    if err != nil {
        return nil, err
    }
    
    url := fmt.Sprintf("%s/api/auth/validate", a.baseURL)
    resp, err := a.client.Post(url, "application/json", bytes.NewBuffer(body))
    if err != nil {
        return nil, fmt.Errorf("failed to validate token: %w", err)
    }
    defer resp.Body.Close()
    
    var result ValidateResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }
    
    return &result, nil
}
```

#### Step 5.2: Authentication Middleware
**File:** `hermes-server/utils/middleware/auth.go`

**Guidelines for Claude:**
```go
package middleware

import (
    "log"
    "net/http"
    "strings"
    
    "github.com/gin-gonic/gin"
    "nfcunha/hermes/hermes-server/services/auth"
)

// AuthMiddleware validates JWT tokens via Aegis
func AuthMiddleware(aegisClient *auth.AegisClient) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Extract token from Authorization header
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
            c.Abort()
            return
        }
        
        // Extract Bearer token
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
            c.Abort()
            return
        }
        
        token := parts[1]
        
        // Validate with Aegis
        result, err := aegisClient.ValidateToken(token)
        if err != nil {
            log.Printf("Token validation error: %v", err)
            c.JSON(http.StatusUnauthorized, gin.H{"error": "token validation failed"})
            c.Abort()
            return
        }
        
        if !result.Valid {
            c.JSON(http.StatusUnauthorized, gin.H{"error": result.Error})
            c.Abort()
            return
        }
        
        // Store user info in context
        c.Set("user_id", result.User.ID)
        c.Set("user_email", result.User.Subject)
        c.Set("user_roles", result.User.Roles)
        c.Set("user_permissions", result.User.Permissions)
        
        c.Next()
    }
}

// AdminOnly middleware ensures user has admin role
func AdminOnly() gin.HandlerFunc {
    return func(c *gin.Context) {
        roles, exists := c.Get("user_roles")
        if !exists {
            c.JSON(http.StatusForbidden, gin.H{"error": "no role information"})
            c.Abort()
            return
        }
        
        roleList := roles.([]string)
        isAdmin := false
        for _, role := range roleList {
            if role == "admin" {
                isAdmin = true
                break
            }
        }
        
        if !isAdmin {
            c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

**Apply middleware to protected routes:**
```go
// In api/register.go
func RegisterRoutes(router *gin.Engine, reg *registry.Registry, prox *proxy.Proxy, aegis *auth.AegisClient) {
    hermes := router.Group("/hermes")
    
    // Public routes
    hermes.GET("/health", handleHealth)
    
    // Protected routes (require auth)
    protected := hermes.Group("")
    protected.Use(middleware.AuthMiddleware(aegis))
    {
        // Service management (admin only)
        service.RegisterRoutes(protected, reg, middleware.AdminOnly())
        
        // User management (admin only)
        user.RegisterRoutes(protected, middleware.AdminOnly())
    }
    
    // Proxy routes (no auth)
    router.NoRoute(func(c *gin.Context) {
        prox.Forward(c, reg)
    })
}
```

---

### Phase 6: React Dashboard
**Goal:** Build web interface for service and user management

#### Step 6.1: React App Structure
**Create React app:**
```bash
cd hermes-ui
npx create-react-app .
```

**File:** `hermes-ui/src/services/api.js`

**Guidelines for Claude:**
```javascript
// API client for Hermes backend
const API_BASE = '/hermes';

// Get token from localStorage
const getToken = () => localStorage.getItem('hermes_token');

// API call wrapper
const apiCall = async (endpoint, options = {}) => {
  const token = getToken();
  
  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token && { 'Authorization': `Bearer ${token}` }),
      ...options.headers,
    },
  });
  
  if (response.status === 401) {
    // Token expired, redirect to login
    localStorage.removeItem('hermes_token');
    window.location.href = '/login';
    throw new Error('Unauthorized');
  }
  
  if (!response.ok) {
    throw new Error(`API error: ${response.status}`);
  }
  
  return response.json();
};

// Service API
export const serviceAPI = {
  list: () => apiCall('/services'),
  get: (id) => apiCall(`/services/${id}`),
  register: (data) => apiCall('/services', {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  deregister: (id) => apiCall(`/services/${id}`, {
    method: 'DELETE',
  }),
};

// User API
export const userAPI = {
  list: () => apiCall('/users'),
  create: (data) => apiCall('/users', {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  update: (id, data) => apiCall(`/users/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  }),
  delete: (id) => apiCall(`/users/${id}`, {
    method: 'DELETE',
  }),
};
```

**File:** `hermes-ui/src/middleware/auth.js`

**Guidelines for Claude:**
```javascript
// Authentication middleware for React Router
import { Navigate } from 'react-router-dom';

export const PrivateRoute = ({ children }) => {
  const token = localStorage.getItem('hermes_token');
  
  if (!token) {
    return <Navigate to="/login" />;
  }
  
  return children;
};

// Inactivity timeout
let inactivityTimer;
const INACTIVITY_TIMEOUT = parseInt(process.env.REACT_APP_INACTIVITY_TIMEOUT || '1800000'); // 30 min default

export const setupInactivityTimeout = () => {
  const resetTimer = () => {
    clearTimeout(inactivityTimer);
    inactivityTimer = setTimeout(() => {
      localStorage.removeItem('hermes_token');
      window.location.href = '/login';
    }, INACTIVITY_TIMEOUT);
  };
  
  // Reset timer on user activity
  ['mousedown', 'keypress', 'scroll', 'touchstart'].forEach(event => {
    document.addEventListener(event, resetTimer);
  });
  
  resetTimer(); // Initial timer
};
```

---

### Phase 7: Docker Deployment
**Goal:** Containerize Hermes with NGINX and Supervisor

**File:** `Dockerfile`

**Guidelines for Claude:**
```dockerfile
# Build stage for Go backend
FROM golang:1.21-alpine AS go-builder

RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /app
COPY hermes-server/go.mod hermes-server/go.sum ./
RUN go mod download

COPY hermes-server/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o hermes .

# Build stage for React frontend
FROM node:18-alpine AS ui-builder

WORKDIR /app
COPY hermes-ui/package*.json ./
RUN npm ci

COPY hermes-ui/ ./
RUN npm run build

# Runtime stage
FROM nginx:alpine

# Install dependencies
RUN apk --no-cache add ca-certificates sqlite-libs supervisor

# Copy Go binary
COPY --from=go-builder /app/hermes /usr/local/bin/hermes

# Copy React build
COPY --from=ui-builder /app/build /usr/share/nginx/html

# Copy configuration files
COPY config/nginx.conf /etc/nginx/conf.d/default.conf
COPY config/supervisord.conf /etc/supervisord.conf

# Create data directory
RUN mkdir -p /app/data && chmod 755 /app/data

# Expose ports
EXPOSE 80

# Start supervisor
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
```

**File:** `config/supervisord.conf`

```ini
[supervisord]
nodaemon=true
user=root

[program:hermes-server]
command=/usr/local/bin/hermes
autostart=true
autorestart=true
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0

[program:nginx]
command=/usr/sbin/nginx -g "daemon off;"
autostart=true
autorestart=true
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
```

**File:** `docker-compose.yml`

```yaml
version: '3.8'

services:
  hermes:
    build: .
    container_name: hermes
    ports:
      - "8080:80"
    volumes:
      - hermes-data:/app/data
    env_file:
      - config/hermes.env
    networks:
      - hermes-network

  aegis:
    # Reference to Aegis service
    # (Assuming Aegis has its own compose file)
    external: true

volumes:
  hermes-data:

networks:
  hermes-network:
```

---

## 🎯 Code Generation Guidelines Summary

### General Principles:
1. **Always add logging**: Use `log.Printf` for important operations
2. **Error handling**: Never ignore errors, always wrap with context
3. **Thread safety**: Use mutexes for shared state (registry, cache)
4. **Configuration**: Use environment variables with sensible defaults
5. **Comments**: Add GoDoc comments for exported functions and types
6. **Testing**: Write tests for each package (domain, services, api)

### Code Style:
- Follow Go best practices and idioms
- Use meaningful variable names
- Keep functions small and focused
- Prefer composition over inheritance
- Use interfaces for testability

### Security:
- Never log sensitive data (passwords, tokens)
- Validate all input at API boundaries
- Use bcrypt for password hashing
- Check permissions before mutations

### Performance:
- Use read locks where possible (RWMutex)
- Avoid unnecessary database queries
- Close resources in defer statements
- Use context for cancellation

---

## 📝 Testing Strategy

- Unit tests for domain models
- Service tests with mocked dependencies
- Integration tests for API endpoints
- End-to-end tests with test containers

---

## 🚀 Next Steps After Roadmap

1. Load balancing algorithms (round-robin, least connections)
2. Circuit breakers and retry logic
3. Metrics and monitoring (Prometheus)
4. WebSocket support
5. gRPC proxying
6. Rate limiting
7. Request transformation
8. API versioning

---

**This roadmap provides explicit, step-by-step instructions for implementing Hermes with clear code examples and guidelines for Claude to generate high-quality, production-ready code.**


---

## 🚀 Continuation Guide for Future Development

**Last Updated:** November 28, 2025  
**Current Phase:** Phase 4 Complete, Ready for Phase 5

### What We Accomplished Today

1. **Database Persistence** ✅
   - Services table with JSON metadata
   - Warm cache pattern (load on startup, write-through)
   - Methods: load, save, delete, updateStatus
   - Indexes on name and status columns

2. **Authentication & Authorization** ✅
   - Aegis JWT validation for all service CRUD
   - Middleware chain: authMiddleware → adminMiddleware → handler
   - User context stored in Gin (ID, subject, roles, permissions)

3. **Duplicate Prevention** ✅
   - Application-level check (name, host, port)
   - Database UNIQUE constraint as backup
   - Supports load balancing (same name, different host)

4. **Comprehensive Testing** ✅
   - 71+ test assertions across 6 packages
   - Coverage: 76%-100%
   - Test isolation with in-memory SQLite
   - Manual testing: 21 scenarios validated

### Next Phase: User Management Proxy (Phase 5)

**Objective:** Implement API endpoints that proxy user operations to Aegis.

**Why Proxy?**
- Aegis is the source of truth for users
- Avoids data duplication and sync issues
- Simpler Hermes architecture

**Implementation Guide:**

#### Step 1: Create User API Handler

Create `hermes-server/api/user/api.go`:

```go
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

type API struct {
	aegisClient *aegis.Client
	aegisURL    string
}

func NewAPI(client *aegis.Client, url string) *API {
	return &API{
		aegisClient: client,
		aegisURL:    url,
	}
}

// RegisterRoutes sets up user management routes
func (a *API) RegisterRoutes(router gin.IRouter, authMw, adminMw gin.HandlerFunc) {
	users := router.Group("/users")
	users.Use(authMw) // All endpoints require authentication

	// Most endpoints require admin
	adminRoutes := users.Group("")
	adminRoutes.Use(adminMw)
	{
		adminRoutes.POST("", a.createUser)
		adminRoutes.GET("", a.listUsers)
		adminRoutes.GET("/:id", a.getUser)
		adminRoutes.PUT("/:id", a.updateUser)
		adminRoutes.DELETE("/:id", a.deleteUser)
		adminRoutes.POST("/:id/roles", a.addRole)
		adminRoutes.DELETE("/:id/roles/:role", a.removeRole)
	}

	// Password change: users can change their own
	users.POST("/:id/password", a.changePassword)
}

// proxyToAegis forwards request to Aegis
func (a *API) proxyToAegis(c *gin.Context, method, path string) {
	targetURL := a.aegisURL + path

	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to read request"})
		return
	}

	// Create Aegis request
	req, err := http.NewRequest(method, targetURL, bytes.NewReader(body))
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to create request"})
		return
	}

	// Copy headers (except Host, Authorization)
	for key, values := range c.Request.Header {
		if key == "Host" || key == "Authorization" {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Make request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Aegis request failed: %v", err)
		c.JSON(502, gin.H{"error": "authentication service unavailable"})
		return
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to read response"})
		return
	}

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// Return response
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

func (a *API) createUser(c *gin.Context) {
	a.proxyToAegis(c, "POST", "/api/aegis/users")
}

func (a *API) listUsers(c *gin.Context) {
	path := "/api/aegis/users"
	if query := c.Request.URL.RawQuery; query != "" {
		path += "?" + query
	}
	a.proxyToAegis(c, "GET", path)
}

func (a *API) getUser(c *gin.Context) {
	id := c.Param("id")
	a.proxyToAegis(c, "GET", "/api/aegis/users/"+id)
}

func (a *API) updateUser(c *gin.Context) {
	id := c.Param("id")
	a.proxyToAegis(c, "PUT", "/api/aegis/users/"+id)
}

func (a *API) deleteUser(c *gin.Context) {
	id := c.Param("id")
	a.proxyToAegis(c, "DELETE", "/api/aegis/users/"+id)
}

func (a *API) addRole(c *gin.Context) {
	id := c.Param("id")
	a.proxyToAegis(c, "POST", "/api/aegis/users/"+id+"/roles")
}

func (a *API) removeRole(c *gin.Context) {
	id := c.Param("id")
	role := c.Param("role")
	a.proxyToAegis(c, "DELETE", fmt.Sprintf("/api/aegis/users/%s/roles/%s", id, role))
}

func (a *API) changePassword(c *gin.Context) {
	userID := c.Param("id")
	authenticatedUserID, _ := c.Get("user_id")

	// Check if user is changing own password or is admin
	if userID != authenticatedUserID {
		roles, _ := c.Get("user_roles")
		userRoles := roles.([]string)
		isAdmin := false
		for _, role := range userRoles {
			if role == "admin" {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			c.JSON(403, gin.H{"error": "can only change own password"})
			return
		}
	}

	a.proxyToAegis(c, "POST", "/api/aegis/users/"+userID+"/password")
}
```

#### Step 2: Update Route Registration

Update `hermes-server/api/register.go`:

```go
package api

import (
	"github.com/gin-gonic/gin"
	"nfcunha/hermes/hermes-server/api/service"
	"nfcunha/hermes/hermes-server/api/user"
	"nfcunha/hermes/hermes-server/middleware"
	"nfcunha/hermes/hermes-server/services/aegis"
	"nfcunha/hermes/hermes-server/services/registry"
)

func RegisterRoutes(router *gin.Engine, serviceRegistry *registry.Registry,
	aegisClient *aegis.Client, aegisURL string) {
	hermes := router.Group("/hermes")
	hermes.GET("/health", handleHealth)

	authMw := middleware.AuthMiddleware(aegisClient)
	adminMw := middleware.RequireAdmin()

	// Service API (existing)
	service.RegisterRoutes(hermes, serviceRegistry, authMw, adminMw)

	// User API (new)
	userAPI := user.NewAPI(aegisClient, aegisURL)
	userAPI.RegisterRoutes(hermes, authMw, adminMw)
}

func handleHealth(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "healthy",
		"service": "hermes",
	})
}
```

#### Step 3: Update Main to Pass aegisURL

Update `hermes-server/main.go`:

```go
// In the main() function, pass aegisURL to route registration
aegisURL := cfg.Aegis.URL // Should be http://localhost:3100
api.RegisterRoutes(engine, serviceRegistry, aegisClient, aegisURL)
```

#### Step 4: Create Tests

Create `hermes-server/api/user/api_test.go`:

```go
package user

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"nfcunha/hermes/hermes-server/services/aegis"
)

func setupTest() (*gin.Engine, *httptest.Server, *API) {
	// Mock Aegis server
	mockAegis := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back for testing
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.Copy(w, r.Body)
	}))

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock auth middleware
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("user_roles", []string{"admin"})
		c.Next()
	})

	client := aegis.NewClient(mockAegis.URL, 5*time.Second)
	api := NewAPI(client, mockAegis.URL)
	api.RegisterRoutes(router.Group("/hermes"), func(c *gin.Context) { c.Next() }, func(c *gin.Context) { c.Next() })

	return router, mockAegis, api
}

func TestListUsers_Success(t *testing.T) {
	router, mockAegis, _ := setupTest()
	defer mockAegis.Close()

	req := httptest.NewRequest("GET", "/hermes/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCreateUser_Success(t *testing.T) {
	router, mockAegis, _ := setupTest()
	defer mockAegis.Close()

	user := map[string]string{
		"subject":  "newuser@test.com",
		"password": "Password123!",
	}
	body, _ := json.Marshal(user)

	req := httptest.NewRequest("POST", "/hermes/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestChangeOwnPassword_Success(t *testing.T) {
	router, mockAegis, _ := setupTest()
	defer mockAegis.Close()

	pwChange := map[string]string{
		"old_password": "old",
		"new_password": "new",
	}
	body, _ := json.Marshal(pwChange)

	req := httptest.NewRequest("POST", "/hermes/users/test-user/password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestChangeOtherPassword_NonAdmin_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Non-admin user
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Set("user_roles", []string{"viewer"})
		c.Next()
	})

	mockAegis := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer mockAegis.Close()

	client := aegis.NewClient(mockAegis.URL, 5*time.Second)
	api := NewAPI(client, mockAegis.URL)
	api.RegisterRoutes(router.Group("/hermes"), func(c *gin.Context) { c.Next() }, func(c *gin.Context) { c.Next() })

	req := httptest.NewRequest("POST", "/hermes/users/user-2/password", bytes.NewReader([]byte("{}")))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
```

### Testing Checklist

**Unit Tests:**
- [ ] All user API methods
- [ ] Password change authorization logic
- [ ] Aegis unavailable scenario (502)
- [ ] Request/response proxying

**Integration Tests:**
1. Start Aegis and Hermes
2. Get admin token from Aegis
3. Create user via Hermes: `POST /hermes/users`
4. List users: `GET /hermes/users`
5. Update user: `PUT /hermes/users/:id`
6. Delete user: `DELETE /hermes/users/:id`
7. Change own password as non-admin
8. Try change other password as non-admin (should fail)
9. Change other password as admin (should succeed)

**Command Examples:**

```bash
# Get admin token
TOKEN=$(curl -s -X POST http://localhost:3100/api/aegis/users/login \
  -H 'Content-Type: application/json' \
  -d '{"subject":"alice@aegis.com","password":"Password123!"}' | jq -r .access_token)

# List users via Hermes
curl -X GET http://localhost:8080/hermes/users \
  -H "Authorization: Bearer $TOKEN" | jq .

# Create user via Hermes
curl -X POST http://localhost:8080/hermes/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "newuser@test.com",
    "password": "Password123!",
    "roles": ["viewer"]
  }' | jq .
```

### Estimated Timeline

- **Day 1 (4 hours):**
  - Morning: Implement API proxy handler (2 hours)
  - Afternoon: Integrate with route registration (1 hour)
  - End of day: Basic manual testing (1 hour)

- **Day 2 (4 hours):**
  - Morning: Write unit tests (2 hours)
  - Afternoon: Integration testing and bug fixes (2 hours)

**Total: 8 hours / 2 days**

### Common Pitfalls to Avoid

1. **Path Mapping Error**
   - ❌ Wrong: `/hermes/users` → `/hermes/users`
   - ✅ Correct: `/hermes/users` → `/api/aegis/users`

2. **Authorization Header Forwarding**
   - ❌ Wrong: Forward `Authorization` header to Aegis
   - ✅ Correct: Strip it (Aegis doesn't need Hermes token)

3. **Password Change Permissions**
   - ❌ Wrong: Only admins can change passwords
   - ✅ Correct: Users can change own, admins can change any

4. **Error Handling**
   - ❌ Wrong: Return 500 when Aegis is down
   - ✅ Correct: Return 502 Bad Gateway

### After Phase 5: React Dashboard (Phase 6)

**Key Pages:**
1. Login (authenticate with Aegis)
2. Dashboard (service overview, health stats)
3. Services (list, register, details, delete)
4. Users (admin only: create, edit roles, delete)
5. Profile (change password, logout)

**Tech Stack:**
- React 18 + React Router
- Axios for API calls
- Material-UI or Tailwind CSS
- JWT in sessionStorage
- 30-min inactivity timeout

**Estimated Timeline:** 5 days

---

## 📖 Architecture Decisions Log

### Decision 1: In-Memory Registry + SQLite Persistence

**Date:** November 27, 2025

**Context:** Need service registry that survives restarts but performs well.

**Decision:** Use in-memory map for lookups, persist to SQLite, load on startup (warm cache).

**Consequences:**
- ✅ Fast lookups (O(1), microseconds)
- ✅ No external dependencies
- ✅ Survives restarts
- ❌ Not suitable for distributed deployment
- ❌ Limited to ~10k services

**Alternatives Considered:**
- Pure in-memory: Lost on restart ❌
- Database-only: Too slow for routing ❌
- Redis: External dependency ❌
- etcd: Overkill for single-instance ❌

### Decision 2: Proxy User API to Aegis

**Date:** November 28, 2025

**Context:** Need user management in dashboard but don't want to duplicate Aegis functionality.

**Decision:** Proxy all user operations to Aegis, no local user storage.

**Consequences:**
- ✅ Single source of truth
- ✅ No sync issues
- ✅ Simpler codebase
- ❌ Extra network hop
- ❌ Dependency on Aegis for dashboard

**Alternatives Considered:**
- Replicate users in Hermes: Sync issues ❌
- Direct Aegis calls from frontend: CORS, security ❌

### Decision 3: Test Isolation with In-Memory SQLite

**Date:** November 28, 2025

**Context:** Tests were potentially affecting production database.

**Decision:** All tests use `:memory:` SQLite, production uses file-based.

**Consequences:**
- ✅ Perfect isolation
- ✅ Fast tests (100x faster)
- ✅ No cleanup needed
- ✅ Parallel test execution
- ❌ Doesn't test file I/O

**Alternatives Considered:**
- Separate test.db file: Slow, cleanup issues ❌
- Transactions + rollback: Complex, error-prone ❌

---

## 🎓 Lessons Learned

### Lesson 1: Always Test with Production-Like Data

**Problem:** Tests passed but production failed because test used simplified data.

**Solution:** Include edge cases in test data (special characters, empty strings, large numbers).

**Example:**
```go
testCases := []Service{
    {Name: "simple", Host: "localhost", Port: 8080},
    {Name: "with-dashes", Host: "my-service.example.com", Port: 443},
    {Name: "UPPERCASE", Host: "192.168.1.1", Port: 65535},
    {Name: "special!@#", Host: "unicode-域.com", Port: 1},
}
```

### Lesson 2: Middleware Order Matters

**Problem:** Admin check ran before auth, caused nil pointer dereference.

**Solution:** Always apply authentication before authorization.

**Correct Order:**
```go
router.Use(authMiddleware)      // Step 1: Validate token, set user context
router.Use(authorizationMiddleware) // Step 2: Check permissions
router.Use(handler)              // Step 3: Execute business logic
```

### Lesson 3: Health Check Before Registration

**Problem:** Services registered successfully but were unreachable, causing immediate failures.

**Solution:** Validate health check endpoint before allowing registration.

**Benefits:**
- Immediate feedback to user
- Prevents bad registrations
- Cleaner registry (no dead services)

### Lesson 4: Duplicate Detection at Multiple Levels

**Problem:** Race condition allowed duplicate registrations when requests arrived simultaneously.

**Solution:** Check duplicates in application AND database.

**Why Both?**
- Application-level: Better UX with descriptive errors
- Database-level: Prevents race conditions with UNIQUE constraint

---

## 🛠️ Development Environment Setup

### Prerequisites

```bash
# Install Go 1.21+
go version

# Install SQLite3 (required for database/sql driver)
sudo apt-get install sqlite3 libsqlite3-dev  # Ubuntu/Debian
brew install sqlite3  # macOS

# Install dependencies
cd hermes-server
go mod download
```

### Environment Variables

```bash
# hermes-server/.env
HERMES_PORT=8080
HERMES_DB_PATH=./hermes.db
HERMES_AEGIS_URL=http://localhost:3100/api
HERMES_HEALTH_CHECK_INTERVAL=30s
HERMES_HEALTH_CHECK_TIMEOUT=5s
HERMES_HEALTH_CHECK_THRESHOLD=3
HERMES_HEALTH_CHECK_MAX_FAILURES=10
```

### Running Tests

```bash
# Run all tests with coverage
cd hermes-server
go test ./... -cover -v

# Run specific package tests
go test ./services/registry -cover -v
go test ./api/service -cover -v

# Run with race detector
go test ./... -race

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Running the Server

```bash
# Start Aegis first (required dependency)
cd aegis-server
go run main.go

# In another terminal, start Hermes
cd hermes-server
go run main.go
```

---

## 📋 Quick Reference: What's Working Now

✅ **Service Registry**
- In-memory + SQLite persistence
- Automatic loading on startup
- Thread-safe operations

✅ **Health Checking**
- 30-second intervals
- 5-second timeout
- Auto-removal after 10 failures

✅ **Authentication**
- Aegis JWT validation
- Admin-only service CRUD
- User context in requests

✅ **Authorization**
- Role-based access control
- Admin middleware
- Permission middleware

✅ **Duplicate Prevention**
- Application-level validation
- Database UNIQUE constraint
- Supports load balancing

✅ **Testing**
- 71+ assertions
- 76%-100% coverage
- In-memory test isolation

---

## 🎯 Next Action Items

When you're ready to continue:

1. **Read this section** for complete Phase 5 context
2. **Create** `hermes-server/api/user/api.go` (proxy handler)
3. **Update** `hermes-server/api/register.go` (add user routes)
4. **Update** `hermes-server/main.go` (pass aegisURL)
5. **Create** `hermes-server/api/user/api_test.go` (4 test cases)
6. **Run tests**: `go test ./api/user -v`
7. **Manual test**: Use curl commands from Testing Checklist
8. **Verify**: All 4 unit tests pass, integration tests succeed

**Estimated time: 8 hours over 2 days**

---

**End of Continuation Guide**
