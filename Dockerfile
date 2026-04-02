# ============================================
# Stage 1: Build Frontend (Nuxt)
# ============================================
FROM node:22-alpine AS frontend-builder

WORKDIR /app

# Copy package files
COPY package.json package-lock.json* ./

# Install dependencies
RUN npm ci

# Copy frontend source
COPY nuxt.config.ts tsconfig.json ./
COPY app/ ./app/
COPY public/ ./public/
COPY server/ ./server/

# Build Nuxt
RUN npm run build

# ============================================
# Stage 2: Build Backend (Go)
# ============================================
FROM golang:1.25-alpine AS backend-builder

WORKDIR /app

# Copy go module files
COPY back-end/go.mod back-end/go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source
COPY back-end/ ./

# Build the Go binary (CGO disabled - no gcc needed)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o api-meow .

# ============================================
# Stage 3: Production Image
# ============================================
FROM node:22-alpine AS production

WORKDIR /app

# Install ca-certificates for HTTPS and postgresql-client for migrations
RUN apk add --no-cache ca-certificates postgresql-client

# Copy frontend build output
COPY --from=frontend-builder /app/.output ./.output

# Copy backend binary
COPY --from=backend-builder /app/api-meow ./api-meow

# Copy SQL schema for migrations
COPY back-end/sql/schema/unified_schema.sql ./migrations/schema.sql

# Create a startup script that runs migrations before starting services.
# Both processes run in background so the shell (PID 1) can trap SIGTERM and
# forward it to both — guaranteeing the Go binary gets a clean shutdown signal.
# NOTE: no "set -e" — we don't want the shell to exit if Node.js crashes.
# The container lifecycle is driven by the Go backend (critical service).
RUN cat > /app/start.sh << 'EOF'
#!/bin/sh

echo "Starting API Meow..."

# Run database migrations if DATABASE_URL is set
if [ -n "$DATABASE_URL" ]; then
    echo "Running database migrations..."
    psql "$DATABASE_URL" -f /app/migrations/schema.sql \
        && echo "Migrations completed successfully!" \
        || echo "Migration warning (tables may already exist, continuing...)"
else
    echo "DATABASE_URL not set, skipping migrations"
fi

# Start Go backend and Nuxt frontend in background, capturing PIDs
echo "Starting Go backend on port 8080..."
./api-meow &
GO_PID=$!

echo "Starting Nuxt frontend..."
node .output/server/index.mjs &
NODE_PID=$!

# Forward SIGTERM/INT to both processes so the Go binary performs graceful
# shutdown (closes WhatsApp connections cleanly and preserves session state).
trap 'echo "Shutdown signal received, stopping services..."; kill -TERM $GO_PID $NODE_PID 2>/dev/null' TERM INT

# Container lifecycle is tied to the Go backend only.
# If Nuxt crashes, the Go API keeps serving and EasyPanel won't restart the
# container unnecessarily. If Go exits (crash or SIGTERM), the container stops.
wait $GO_PID
echo "Go backend stopped, shutting down..."
kill -TERM $NODE_PID 2>/dev/null
wait $NODE_PID 2>/dev/null
EOF
RUN chmod +x /app/start.sh

# Expose ports (Nuxt: 3000, Go API: 8080)
EXPOSE 3000 8080

# Environment variables (can be overridden in EasyPanel)
ENV NODE_ENV=production
ENV HOST=0.0.0.0
ENV PORT=3000

# Health check uses the /health endpoint exposed by the Go backend.
# --start-period gives time for migrations + startup before checks begin.
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

# Start both services with migrations
CMD ["/app/start.sh"]
