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
FROM golang:1.24-alpine AS backend-builder

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

# Install ca-certificates for HTTPS, postgresql-client for migrations, ffmpeg for audio conversion
RUN apk add --no-cache ca-certificates postgresql-client ffmpeg

# Copy frontend build output
COPY --from=frontend-builder /app/.output ./.output

# Copy backend binary
COPY --from=backend-builder /app/api-meow ./api-meow

# Copy SQL schema for migrations
COPY back-end/sql/schema/unified_schema.sql ./migrations/schema.sql

# Create a startup script that runs migrations before starting services
RUN cat > /app/start.sh << 'EOF'
#!/bin/sh
set -e

echo "🚀 Starting API Meow..."

# Run database migrations if DATABASE_URL is set
if [ -n "$DATABASE_URL" ]; then
    echo "📦 Running database migrations..."
    psql "$DATABASE_URL" -f /app/migrations/schema.sql \
        && echo "✅ Migrations completed successfully!" \
        || echo "⚠️  Migration warning (tables may already exist, continuing...)"
else
    echo "⚠️  DATABASE_URL not set, skipping migrations"
fi

# Start Go backend in background
echo "🔧 Starting Go backend on port 8080..."
./api-meow &

# Start Nuxt frontend
echo "🌐 Starting Nuxt frontend on port 3000..."
node .output/server/index.mjs
EOF
RUN chmod +x /app/start.sh

# Expose ports (Nuxt: 3000, Go API: 8080)
EXPOSE 3000 8080

# Environment variables (can be overridden in EasyPanel)
ENV NODE_ENV=production
ENV HOST=0.0.0.0
ENV PORT=3000

# Start both services with migrations
CMD ["/app/start.sh"]
