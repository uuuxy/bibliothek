# ==============================================================================
# Stage 1: Build the Svelte 5 frontend
# ==============================================================================
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend

# Copy dependencies first for Docker caching
COPY frontend/package*.json ./
RUN npm ci

# Copy the rest of the frontend files and build
COPY frontend/ ./
RUN npm run build

# ==============================================================================
# Stage 2: Build the Go backend
# ==============================================================================
FROM golang:alpine AS backend-builder
WORKDIR /app

# Disable Go workspace mode to build using root go.mod directly
ENV GOWORK=off

# Copy module definitions first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy Go source code
COPY main.go ./
COPY api/ ./api/
COPY apierrors/ ./apierrors/
COPY auth/ ./auth/
COPY db/ ./db/
COPY inventur/ ./inventur/
COPY jobs/ ./jobs/
COPY migrations/ ./migrations/
COPY repository/ ./repository/
COPY sse/ ./sse/
COPY docs/ ./docs/
COPY plugins/ ./plugins/
COPY mailservice/ ./mailservice/
COPY pdf/ ./pdf/

# Compile static Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main main.go

# ==============================================================================
# Stage 3: Runner container
# ==============================================================================
FROM alpine:3.19
WORKDIR /app

# Install ca-certificates for secure outgoing connections (e.g. cover APIs)
RUN apk --no-cache add ca-certificates tzdata

# Copy database schema file (for reference / first-run init)
COPY schema.sql ./

# Copy SQL migration files
COPY migrations/ ./migrations/

# Copy compiled Go binary
COPY --from=backend-builder /app/main .

# Copy built Svelte static files
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Create non-privileged user, create uploads dir to inherit permissions, and give ownership
RUN adduser -D appuser && \
    mkdir -p /app/uploads/fotos && \
    chown -R appuser:appuser /app

# Switch context
USER appuser

# Expose port (matched with default PORT in environment)
EXPOSE 8081

# Environment variables defaults
ENV PORT=8081
ENV DATABASE_URL=""
ENV COOKIE_SECURE="false"

HEALTHCHECK --interval=30s --timeout=3s CMD wget --no-verbose --tries=1 --spider http://localhost:$PORT/health || exit 1

# Run the single-binary application
CMD ["./main"]
