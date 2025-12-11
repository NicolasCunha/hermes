> This is a personal learning project. You’re welcome to use it as a reference or as a starting point for your own ideas. However, I strongly recommend using more mature and stable solutions available in the market rather than relying on this project for production needs.

# Hermes API Gateway

A lightweight, dynamic API Gateway with service discovery, health monitoring, and user management. Built with Go and React.

## Features

- 🚀 **Dynamic Service Registry** - Services self-register on startup
- 💚 **Health Monitoring** - Automatic health checks with configurable intervals
- 🔒 **Authentication & Authorization** - JWT-based auth via Aegis
- 📊 **React Dashboard** - Modern UI for service and user management
- 🐳 **Docker Ready** - Single container deployment with nginx + supervisord
- 💾 **SQLite Storage** - Lightweight database with volume persistence
- 🔄 **Auto-Discovery** - Supports container IP auto-detection with overrides

## Quick Start with Docker Compose

**Option 1: Using Pre-built Images from DockerHub** (Recommended)

```bash
cd hermes
docker compose pull  # Pull latest images
docker compose up -d

# Access the services
# Hermes Dashboard: http://localhost:4000
# Aegis Dashboard: http://localhost:3200
```

**Option 2: Building from Source**

```bash
cd hermes
# Edit docker-compose.yml and change:
#   image: cunhanicolas/hermes:latest
# to:
#   build:
#     context: .
#     dockerfile: Dockerfile

docker compose build
docker compose up -d
```

Default credentials:
- Username: `hermes`
- Password: `hermes123`

### Changing the Default Port

To change the default port (4000), edit the `docker-compose.yml` file:

```yaml
services:
  hermes:
    ports:
      - "YOUR_PORT:8080"  # Change YOUR_PORT to your desired port
```

For example, to use port 5000:
```yaml
    ports:
      - "5000:8080"
```

Then restart the service:
```bash
docker compose up -d
```

## API Endpoints

### Public Endpoints

- `POST /hermes/register` - Service self-registration (no auth required)

### Management API (Authentication Required)

#### Services
- `GET /hermes/services` - List all registered services
- `POST /hermes/services` - Register a service (admin only)
- `GET /hermes/services/:id` - Get service details
- `DELETE /hermes/services/:id` - Deregister service (admin only)
- `GET /hermes/services/:id/health-logs` - Get health check history

#### Dynamic Routing

Hermes provides a powerful routing mechanism that forwards requests to registered services based on their name:

- `ANY /hermes/route/:serviceName/*path` - Route any HTTP method to a registered service

**How it works:**

1. **Service Lookup**: Hermes extracts the `:serviceName` from the URL and queries the service registry
2. **Health Check**: Only routes to services with `healthy` status (passed recent health checks)
3. **Load Balancing**: If multiple healthy instances exist with the same name, routes to the first available (future: round-robin)
4. **Request Forwarding**: Preserves the HTTP method, headers, query parameters, and request body
5. **Path Preservation**: The `*path` segment is appended to the service's base URL

**Example Flow:**

```bash
# 1. Register a service named "user-api"
curl -X POST http://localhost:4000/hermes/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user-api",
    "host": "192.168.1.100",
    "port": 3000,
    "health_check_path": "/health"
  }'

# 2. Route a request through Hermes
curl -X GET http://localhost:4000/hermes/route/user-api/v1/users/123 \
  -H "Authorization: Bearer <token>" \
  -H "X-Request-ID: abc-123"

# 3. Hermes forwards to: http://192.168.1.100:3000/v1/users/123
#    - Preserves: HTTP method (GET), headers (Authorization, X-Request-ID)
#    - Adds: X-Forwarded-For, X-Forwarded-Proto, X-Forwarded-Host
```

**Multiple Instances:**

If you register multiple instances of the same service (e.g., for high availability):

```bash
# Register instance 1
curl -X POST http://localhost:4000/hermes/register \
  -d '{"name":"api","host":"10.0.0.1","port":8080,"health_check_path":"/health"}'

# Register instance 2
curl -X POST http://localhost:4000/hermes/register \
  -d '{"name":"api","host":"10.0.0.2","port":8080,"health_check_path":"/health"}'

# Hermes routes to the first healthy instance found
curl http://localhost:4000/hermes/route/api/data
```

**Error Handling:**

- **404 Not Found**: Service name not registered
- **503 Service Unavailable**: All instances are unhealthy or service not found

#### Users (Proxied to Aegis)
- `GET /hermes/users` - List users (admin only)
- `POST /hermes/users` - Create user (admin only)
- `GET /hermes/users/:id` - Get user details
- `PUT /hermes/users/:id` - Update user (admin or self)
- `DELETE /hermes/users/:id` - Delete user (admin only)

### Authentication

```bash
# 1. Get JWT token from Aegis
curl -X POST http://localhost:4000/hermes/users/login \
  -H 'Content-Type: application/json' \
  -d '{"subject":"hermes","password":"hermes123"}'

# 2. Use token for authenticated requests
curl http://localhost:4000/hermes/services \
  -H "Authorization: Bearer <your-token>"
```

## Service Self-Registration

Services can dynamically register themselves on startup:

```bash
# Example: Register from inside a container
curl -X POST http://172.17.0.1:8080/hermes/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-api",
    "port": 3000,
    "health_check_path": "/health",
    "protocol": "http",
    "metadata": {
      "version": "1.0.0",
      "environment": "production"
    }
  }'
```

**Auto-detection**: If `host` is not provided, Hermes auto-detects the client IP (supports `X-Forwarded-For` and `X-Real-IP` headers for proxied requests).

### Example: Docker Container Self-Registration

```dockerfile
FROM node:20-alpine
WORKDIR /app
COPY . .
RUN npm install

# Self-register on startup
CMD sh -c "node server.js & \
  sleep 2 && \
  wget -O- --post-data='{\"name\":\"my-service\",\"port\":3000,\"health_check_path\":\"/health\"}' \
  --header='Content-Type: application/json' \
  http://172.17.0.1:8080/hermes/register && \
  wait"
```

## Configuration

### Environment Variables

Edit `hermes/config/hermes.env`:

```bash
# Server Configuration
HERMES_SERVER_HOST=0.0.0.0
HERMES_SERVER_PORT=8081

# Aegis Integration
HERMES_AEGIS_URL=http://aegis:3100/api

# Bootstrap Admin
HERMES_ADMIN_USER=hermes
HERMES_ADMIN_PASSWORD=hermes123

# Health Check Configuration (optional - defaults shown)
# HERMES_HEALTH_CHECK_INTERVAL=30s
# HERMES_HEALTH_CHECK_TIMEOUT=5s
# HERMES_HEALTH_CHECK_THRESHOLD=3
```

## Development

### Prerequisites

### Prerequisites

- Go 1.25.3+
- Node.js 20+
- Docker & Docker Compose

### Local Development (without Docker)

```bash
# Terminal 1: Start Aegis
docker run -p 3100:3100 cunhanicolas/aegis:latest

# Terminal 2: Start Go backend
cd hermes/hermes-server
go run .

# Terminal 3: Start React UI
cd hermes/hermes-ui
npm install
npm run dev
```

Access:
- UI: http://localhost:3000
- API: http://localhost:4000
- Aegis: http://localhost:3100

### Project Structure

```
hermes/
├── Dockerfile              # Multi-stage build (React + Go + nginx)
├── docker-compose.yml      # Aegis + Hermes orchestration
├── config/
│   ├── hermes.env         # Environment variables
│   ├── nginx.conf         # NGINX configuration
│   └── supervisord.conf   # Process manager config
├── hermes-server/         # Go backend
│   ├── main.go
│   ├── core/              # Business logic
│   │   ├── registry.go    # Service registry
│   │   ├── health_checker.go
│   │   └── bootstrap/
│   ├── handler/           # HTTP handlers
│   │   ├── service/
│   │   ├── user/
│   │   └── middleware/
│   ├── database/          # Data access
│   └── utils/
└── hermes-ui/             # React frontend
    ├── src/
    │   ├── components/    # Reusable components
    │   ├── pages/         # Dashboard, Services, Users
    │   ├── services/      # API client
    │   └── context/       # Auth context
    └── public/
```

### Building for Production

```bash
# Build Docker image
docker build -t hermes:latest -f hermes/Dockerfile .

# Or use docker-compose
docker-compose build hermes
```

### Database

- **Type**: SQLite
- **Location**: `/app/data/hermes.db` (in container)
- **Persistence**: Mapped to Docker volume `hermes-data`

#### Schema

**services**:
- `id`, `name`, `host`, `port`, `protocol`
- `health_check_path`, `status`, `metadata`
- `registered_at`, `last_checked_at`, `failure_count`

**health_check_logs**:
- `id`, `service_id`, `checked_at`, `status`
- `error_message`, `response_time_ms`, `response_body`

## Testing

```bash
# Run all tests
cd hermes/hermes-server
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test ./handler/service -v
```

## Troubleshooting

### Service Registration Fails

Check if Hermes is accessible from the service's network:

```bash
# From inside container
wget -O- http://172.17.0.1:8080/hermes/services

# Check Docker network
docker inspect <container-id> | grep IPAddress
```

### Health Checks Failing

1. Ensure health endpoint returns 2xx status
2. Verify network connectivity
3. Check logs: `docker logs hermes`

### Authentication Issues

1. Verify Aegis is running: `curl http://localhost:3100/api/aegis/health`
2. Check JWT token validity
3. Ensure admin role for protected endpoints

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `go test ./...`
5. Submit a pull request

## License

MIT
