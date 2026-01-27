# ============================================
# Stage 1: Build Frontend (Nuxt)
# ============================================
FROM node:22-alpine AS frontend-builder

WORKDIR /app

# Copy package files
COPY package.json package-lock.json ./

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

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go module files
COPY back-end/go.mod back-end/go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source
COPY back-end/ ./

# Build the Go binary
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

# Create a startup script that runs migrations before starting services
RUN cat > /app/start.sh << 'EOF'
#!/bin/sh
set -e

echo "🚀 Starting API Meow..."

# Run database migrations if DATABASE_URL is set
if [ -n "$DATABASE_URL" ]; then
    echo "📦 Running database migrations..."
    # Use IF NOT EXISTS to avoid errors on re-runs
    psql "$DATABASE_URL" -c "
        DO \$\$
        BEGIN
            -- Create instances table if not exists
            CREATE TABLE IF NOT EXISTS instances (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                name TEXT NOT NULL,
                status TEXT NOT NULL DEFAULT 'disconnected',
                phone_number TEXT,
                tag_id TEXT,
                ignore_groups BOOLEAN DEFAULT TRUE,
                webhook_url TEXT,
                receive_messages BOOLEAN DEFAULT TRUE,
                proxy_enabled BOOLEAN DEFAULT FALSE,
                proxy_url TEXT,
                created_at TIMESTAMP NOT NULL DEFAULT NOW(),
                updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
                CONSTRAINT instances_name_key UNIQUE (name)
            );

            -- Create tags table if not exists
            CREATE TABLE IF NOT EXISTS tags (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                name TEXT NOT NULL,
                color TEXT NOT NULL,
                created_at TIMESTAMP NOT NULL DEFAULT NOW()
            );
        END
        \$\$;
    " && echo "✅ Migrations completed successfully!" || echo "⚠️ Migration warning (tables may already exist)"
else
    echo "⚠️ DATABASE_URL not set, skipping migrations"
fi

# Start Go backend in background on port 80
echo "🔧 Starting Go backend on port ${API_PORT:-80}..."
./api-meow &

# Wait a moment for backend to start
sleep 2

# Start Nuxt frontend on port 3000
echo "🌐 Starting Nuxt frontend on port 3000..."
node .output/server/index.mjs
EOF
RUN chmod +x /app/start.sh

# Expose ports (Nuxt: 3000, Go API: 80)
EXPOSE 3000 80

# Environment variables (can be overridden in EasyPanel)
ENV NODE_ENV=production
ENV HOST=0.0.0.0
ENV PORT=3000
# Backend API port
ENV API_PORT=80
# Frontend should connect to backend on port 80 internally
ENV NUXT_PUBLIC_BACKEND_URL=http://localhost:80

# Start both services with migrations
CMD ["/app/start.sh"]
