# ==============================================================================
# Stage 1: Build the Svelte 5 frontend
# ==============================================================================
FROM node:22-alpine AS frontend-builder
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
FROM golang:1.26-alpine AS backend-builder
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
COPY internal/ ./internal/
COPY pkg/ ./pkg/

# Install build-base for WebP CGO compilation
RUN apk add --no-cache build-base

# Compile static Go binary with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o main main.go

# ==============================================================================
# Stage 3: Runner container
# ==============================================================================
FROM alpine:3.21
WORKDIR /app

# ca-certificates: sichere ausgehende Verbindungen (Cover-/Metadaten-APIs).
#
# postgresql-client: liefert pg_dump, das jobs/backup.go per exec aufruft. Ohne das
# Paket schlug JEDER nächtliche Backup-Lauf bereits beim Start des Prozesses fehl
# ("executable file not found in $PATH") — auch mit korrekt gesetztem
# BACKUP_ENCRYPTION_KEY. Das Backup war damit im Container-Betrieb nie möglich,
# und der Wächter in api/backup_status.go stand dauerhaft auf "critical", ohne dass
# das Setzen des Schlüssels daran etwas geändert hätte.
#
# Alpine 3.21 liefert hier pg_dump 17. Das ist Absicht: Der Prod-Stack fährt
# postgres:15, der lokale postgres:16 — ein pg_dump darf jeden ÄLTEREN Server
# dumpen, aber keinen neueren. Der neueste Client bedient also beide.
RUN apk --no-cache add ca-certificates tzdata postgresql-client

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
#
# COOKIE_SECURE steht hier bewusst NICHT mehr: Ein im Image gebackenes "false" ist
# ein gesetzter Wert, und ermittleCookieSecure() (main.go) greift seine sichere
# Vorgabe nur bei NICHT gesetzter Variable. Wer das Image ohne Compose startete,
# verlor damit still das Secure-Flag am Sitzungscookie. Ungesetzt gelassen liefert
# APP_ENV=local/development/test weiterhin false, alles andere true.
ENV PORT=8081
ENV DATABASE_URL=""

# Zwei Änderungen gegenüber "--interval=30s ... http://localhost":
#
#  * 127.0.0.1 statt localhost. Busybox-wget nimmt die erste Adresse, die die Auflösung
#    liefert — auf Hosts mit IPv6 im Container ist das ::1, und wenn dort niemand
#    lauscht, gilt der Container als krank, obwohl die Anwendung über IPv4 tadellos
#    antwortet. Die Adresse ist hier bekannt; sie muss nicht aufgelöst werden.
#  * start-period + kürzeres Intervall. Ohne start-period beginnt die erste Prüfung
#    erst nach einem vollen Intervall; bis dahin steht "starting", und ein Deploy-Skript,
#    das 60 s wartet, sieht ein gesundes System nie grün werden.
HEALTHCHECK --interval=10s --timeout=3s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:$PORT/health || exit 1

# Run the single-binary application
CMD ["./main"]
