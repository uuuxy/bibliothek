# ==============================================================================
# Stage 1: Build the Svelte 5 frontend
# ==============================================================================
FROM node:24-alpine AS frontend-builder
WORKDIR /app/frontend

# Copy dependencies first for Docker caching. .npmrc gehört dazu: Es schaltet die
# Installationsskripte der Abhängigkeiten ab und wirkt nur, wenn es vor npm ci im Image liegt.
COPY frontend/package*.json frontend/.npmrc ./
RUN npm ci

# Copy the rest of the frontend files and build
COPY frontend/ ./
RUN npm run build

# ==============================================================================
# Stage 2: Build the Go backend
# ==============================================================================
# Patch-genau gepinnt, nicht "1.27": go.mod verlangt genau diese Toolchain, und CVE-Fixes
# stecken in Stdlib-Patches — ein Build mit einem älteren Tag kompiliert die verwundbaren
# Pakete ins Binary (GOTOOLCHAIN=local lädt nichts nach). Dependabot hebt nur diese Zeile;
# docs/umgebung_paritaet_test.go hält sie gleich mit go.mod.
FROM golang:1.27.3-alpine AS backend-builder
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
COPY mailservice/ ./mailservice/
COPY pdf/ ./pdf/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY cmd/ ./cmd/

# Install build-base for WebP CGO compilation
RUN apk add --no-cache build-base

# Compile static Go binary with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o main main.go

# Der Schlüsselwechsel gehört dorthin, wo die Daten sind: Der Server hat kein Go, ein
# Werkzeug außerhalb des Images ließe sich dort nicht starten. Kein zusätzliches Risiko: Das
# Kommando liest APP_ENCRYPTION_KEY aus der Umgebung, die im Container ohnehin gesetzt ist.
# Wer eine Shell im Container hat, hat den Schlüssel bereits.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o rotate-encryption-key ./cmd/rotate-encryption-key
# Die beiden Backup-Werkzeuge gehören aus demselben Grund ins Image. Sie bauen ohne cgo: Sie
# binden nur internal/backupkrypto ein, das an der Standardbibliothek und scrypt hängt, nicht
# jobs mit der WebP-Bibliothek.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o restore-backup ./cmd/restore-backup
# Gegenstück: verschlüsselt einen komprimierten Dump von stdin nach stdout. Die beiden
# Shell-Wege (update.sh, scripts/backup.sh) dumpen am nächtlichen Job vorbei; mit diesem
# Werkzeug im Container verschlüsseln sie über dieselbe Ableitung wie der Job, ohne den
# Schlüssel je an den Host zu reichen.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o encrypt-backup ./cmd/encrypt-backup

# ==============================================================================
# Stage 3: Runner container
# ==============================================================================
FROM alpine:3.24
WORKDIR /app

# ca-certificates: sichere ausgehende Verbindungen (Cover-/Metadaten-APIs).
#
# postgresql-client: pg_dump für jobs/backup.go und psql für die Restore-Probe. Die
# Hauptversion ist die des Servers (docker-compose.yml): Ein älterer Client verweigert einen
# neueren Server, ein neuerer schreibt Einstellungen in den Dump (`SET transaction_timeout`
# ab 17), die ein älterer Server unter ON_ERROR_STOP nicht einspielt. Beim Upgrade des
# Servers diese Zeile mitziehen, sonst schlägt die Restore-Probe am ersten Sonntag Alarm.
#
# apk upgrade zuerst: Das Basisimage trägt die Paketstände seines Release-Tags;
# Sicherheits-Fixes kommen nur über das Upgrade herein.
RUN apk --no-cache upgrade && apk --no-cache add ca-certificates tzdata postgresql18-client

# Copy database schema file (for reference / first-run init)
COPY schema.sql ./

# Copy SQL migration files
COPY migrations/ ./migrations/

# Copy compiled Go binary
COPY --from=backend-builder /app/main .

# Wartungswerkzeug: Schlüsselwechsel ohne Datenverlust (docs/SECURITY.md).
#   docker compose exec backend ./rotate-encryption-key -neu <neu> -pruefen
COPY --from=backend-builder /app/rotate-encryption-key .
# Wiederherstellung (docs/resilience_and_recovery.md 2a); die Argumente stehen an fester
# Stelle, ohne Schalter:
#   docker compose exec backend ./restore-backup backups/<datei>.sql.gz.enc /tmp/dump.sql
COPY --from=backend-builder /app/restore-backup .
# Verschlüsselung für die Shell-Wege (docs/resilience_and_recovery.md 1b):
#   … | docker exec -i bibliothek-backend ./encrypt-backup > <datei>.sql.gz.enc
COPY --from=backend-builder /app/encrypt-backup .

# Copy built Svelte static files
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Create non-privileged user, create the volume mount points, and give ownership.
#
# Beide Verzeichnisse müssen im Image existieren: Docker übernimmt beim ersten Mounten eines
# leeren Named Volume Inhalt und Eigentümer aus dem Image. Fehlt das Verzeichnis, legt Docker
# es als root an, und der Container, der als appuser (uid 1000) läuft, kann nie
# hineinschreiben; der nächtliche Backup-Job scheitert dann an "permission denied".
RUN adduser -D appuser && \
    mkdir -p /app/uploads /app/backups && \
    chown -R appuser:appuser /app

# Switch context
USER appuser

# Expose port (matched with default PORT in environment)
EXPOSE 8081

# Environment variables defaults
ENV PORT=8081
ENV DATABASE_URL=""

# Welcher Commit steckt in diesem Image? Gesund heißt nicht aktuell: Nach `git pull` ohne
# `./update.sh` melden Healthcheck und /health "gesund", während ein altes Image läuft.
# update.sh vergleicht nach dem Deploy `git rev-parse HEAD` mit diesem Wert. Ein
# Cache-Treffer beim Build stört nicht — das Build-Argument ändert sich mit jedem Commit und
# bricht den Cache genau dieser Schicht.
#
# Leer gelassen, wenn niemand es setzt: Ein lokaler `docker build` ohne Argument soll
# weiterhin funktionieren, und update.sh sagt dann "unbekannt" statt zu behaupten, der
# Stand sei falsch.
ARG GIT_COMMIT=""
ENV GIT_COMMIT=${GIT_COMMIT}

# 127.0.0.1 statt localhost: Busybox-wget nimmt die erste Adresse der Auflösung; auf Hosts
# mit IPv6 im Container ist das ::1, und lauscht dort niemand, gälte der Container als krank,
# obwohl die Anwendung über IPv4 antwortet. start-period und kurzes Intervall: Ohne
# start-period beginnt die erste Prüfung erst nach einem vollen Intervall, und ein
# Deploy-Skript, das 60 s wartet, sähe den Container nie gesund werden.
HEALTHCHECK --interval=10s --timeout=3s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:$PORT/health || exit 1

# Run the single-binary application
CMD ["./main"]
