# Multi-stage Dockerfile for Hermes API Gateway
# Builds React UI + Go backend, serves with NGINX

# Stage 1: Build React UI
FROM node:20-alpine AS ui-builder

WORKDIR /app

# Copy package files
COPY hermes-ui/package*.json ./

# Install dependencies
RUN npm ci

# Copy UI source
COPY hermes-ui/ ./

# Remove .env files to ensure production uses relative paths
RUN rm -f .env .env.local .env.*.local

# Build for production
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.25.3-bookworm AS go-builder

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy go mod files
COPY hermes-server/go.mod hermes-server/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY hermes-server/ ./

# Build the application with SQLite support for musl
RUN CGO_ENABLED=1 GOOS=linux go build -tags "sqlite_omit_load_extension" -o hermes .

# Stage 3: Runtime with NGINX and Go binary
FROM nginx:bookworm

# Install runtime dependencies and supervisor
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    sqlite3 \
    libsqlite3-0 \
    supervisor \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Copy Go binary from builder
COPY --from=go-builder /app/hermes /usr/local/bin/hermes

# Create hermes user and data directory
RUN groupadd -g 1001 hermes && \
    useradd -r -u 1001 -g hermes -s /bin/false hermes && \
    mkdir -p /app/data && \
    chown -R hermes:hermes /app

# Copy built UI from ui-builder
COPY --from=ui-builder /app/dist /usr/share/nginx/html

# Copy nginx configuration
COPY config/nginx.conf /etc/nginx/conf.d/default.conf

# Copy supervisor configuration
COPY config/supervisord.conf /etc/supervisord.conf

# Expose port 8080
EXPOSE 8080

# Start supervisor to run both services
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
