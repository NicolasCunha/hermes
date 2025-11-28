# Hermes API Gateway

A lightweight, dynamic API Gateway with service discovery and user management.

## Quick Start

```bash
# Build
go build -o hermes hermes-server/main.go

# Run with defaults
./hermes

# Run with custom configuration
HERMES_SERVER_PORT=8081 HERMES_LOG_LEVEL=debug ./hermes
```

## Environment Variables

All configuration is done via environment variables with the `HERMES_` prefix.

### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `HERMES_SERVER_HOST` | `0.0.0.0` | Server bind address |
| `HERMES_SERVER_PORT` | `8080` | Server listen port |
| `HERMES_SERVER_READ_TIMEOUT` | `30s` | HTTP read timeout |
| `HERMES_SERVER_WRITE_TIMEOUT` | `30s` | HTTP write timeout |
| `HERMES_SERVER_IDLE_TIMEOUT` | `60s` | HTTP idle timeout |
| `HERMES_SERVER_MAX_HEADER_BYTES` | `1048576` | Max header size (1MB) |

### Database Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `HERMES_DB_PATH` | `./hermes.db` | SQLite database file path |
| `HERMES_DEFAULT_USERNAME` | `hermes` | Default admin username |
| `HERMES_DEFAULT_PASSWORD` | `hermes` | Default admin password |

### Authentication Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `HERMES_AEGIS_URL` | `http://localhost/api/aegis` | Aegis authentication service URL |
| `HERMES_AEGIS_TIMEOUT` | `5s` | Aegis request timeout |

### Health Check Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `HERMES_HEALTH_CHECK_INTERVAL` | `30s` | Service health check interval |
| `HERMES_HEALTH_CHECK_TIMEOUT` | `5s` | Health check request timeout |
| `HERMES_HEALTH_CHECK_THRESHOLD` | `3` | Failures before marking unhealthy |
| `HERMES_HEALTH_CHECK_MAX_FAILURES` | `10` | Failures before deregistration |

### Logging Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `HERMES_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |

## API Endpoints

### Management API (`/hermes`)

All Hermes management endpoints are under the `/hermes` context path:

#### Public Endpoints
- `GET /hermes/health` - Health check endpoint

#### Service Management (Admin Auth Required)
- `POST /hermes/services` - Register a service
- `GET /hermes/services` - List all registered services
- `GET /hermes/services/:id` - Get service details
- `DELETE /hermes/services/:id` - Deregister a service

#### User Management (Coming in Phase 5)
- `POST /hermes/users` - Create user (proxy to Aegis)
- `GET /hermes/users` - List users (proxy to Aegis)
- `GET /hermes/users/:id` - Get user (proxy to Aegis)
- `PUT /hermes/users/:id` - Update user (proxy to Aegis)
- `DELETE /hermes/users/:id` - Delete user (proxy to Aegis)

#### Authentication
All service and user management endpoints require:
1. Valid JWT token from Aegis in `Authorization: Bearer <token>` header
2. Admin role for CRUD operations

Example:
```bash
# Get token from Aegis
curl -X POST http://localhost:3100/api/aegis/users/login \
  -H 'Content-Type: application/json' \
  -d '{"subject":"alice@aegis.com","password":"Password123!"}'

# Use token with Hermes
curl -X GET http://localhost:8080/hermes/services \
  -H "Authorization: Bearer <your-token>"
```

### Proxy Behavior

All requests not matching `/hermes/*` are considered for proxying to registered backend services. Services must be registered via the management API or through service check-in.

## Database

Hermes uses SQLite for local data storage:

- **Location**: `./hermes.db` (configurable via `HERMES_DB_PATH`)
- **Schema**: Managed via migrations in `database/migrations.go`
- **Persistence**: In containers, map this file to a volume

### Current Schema

**services** table:
- `id` (TEXT PRIMARY KEY) - Service UUID
- `name` (TEXT) - Service name
- `host` (TEXT) - Service hostname
- `port` (INTEGER) - Service port
- `protocol` (TEXT) - http/https
- `health_check_path` (TEXT) - Health check endpoint
- `status` (TEXT) - healthy/unhealthy/draining
- `metadata` (TEXT JSON) - Custom metadata
- `registered_at` (TIMESTAMP)
- `last_checked_at` (TIMESTAMP)
- `failure_count` (INTEGER)
- **Unique constraint**: (name, host, port)
- **Indexes**: name, status

### Testing

Unit tests use in-memory SQLite (`:memory:`) to avoid affecting production data:
- Each test gets a fresh database via `setupTestDB(t)`
- Production `hermes.db` remains untouched during test runs
- Verified with 71+ test assertions across 6 packages

### Docker Volume Mapping

```yaml
volumes:
  - ./data:/app
environment:
  - HERMES_DB_PATH=/app/hermes.db
```

## Development

### Project Structure

```
hermes-server/
├── api/                 # HTTP route handlers
├── database/            # Database layer
├── domain/              # Domain models (Phase 2+)
├── services/            # Business logic
│   ├── proxy/          # HTTP reverse proxy
│   └── router/         # Request routing
└── utils/
    └── config/         # Environment config loader
```

### Build

```bash
go build -o hermes hermes-server/main.go
```

### Run Tests

```bash
go test ./...
```

### Debug Mode

```bash
HERMES_LOG_LEVEL=debug ./hermes
```

## Deployment

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o hermes hermes-server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite-libs
WORKDIR /app
COPY --from=builder /app/hermes .
EXPOSE 8080
CMD ["./hermes"]
```

### Docker Compose

```yaml
version: '3.8'
services:
  hermes:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app
    environment:
      - HERMES_DB_PATH=/app/hermes.db
      - HERMES_LOG_LEVEL=info
      - HERMES_AEGIS_URL=http://aegis/api/aegis
```

## Roadmap

See [ROADMAP.md](ROADMAP.md) for detailed implementation plan.

- ✅ **Phase 1**: Project restructure & database layer
- ✅ **Phase 2**: Aegis integration & authentication middleware
- ✅ **Phase 3**: Service registry & health checking with database persistence
- ✅ **Phase 4**: Service management API with authentication
- 🔜 **Phase 5**: User management dashboard (proxy to Aegis)
- 🔜 **Phase 6**: React dashboard
- 🔜 **Phase 7**: Docker deployment

## Current Features

### ✅ Completed
- **Service Registry**: In-memory service registry with SQLite persistence
- **Service Discovery**: Dynamic service registration with health monitoring
- **Database Persistence**: Services persist across restarts (warm cache pattern)
- **Health Checking**: Configurable periodic health checks (30s interval, 5s timeout)
- **Authentication**: Aegis JWT token validation for all service management
- **Authorization**: Role-based access (admin-only for CRUD operations)
- **Duplicate Prevention**: Enforces (name, host, port) uniqueness
- **Load Balancing Ready**: Allows same service name on different hosts
- **Comprehensive Testing**: 71+ test assertions, 76%-100% coverage
- **Test Isolation**: Uses in-memory SQLite for tests, production data protected

## License

MIT
