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
# Patch-genau gepinnt, nicht "1.26": go.mod verlangt exakt diese Toolchain, und die
# CVE-Fixes vom 16.08.2026 stecken in der Stdlib — ein Build mit einem aelteren
# 1.26-Tag kompiliert die verwundbaren Pakete ins Binary (GOTOOLCHAIN=local laedt
# nichts nach). Beim naechsten go.mod-Go-Bump diese Zeile mitziehen.
FROM golang:1.27.0-alpine AS backend-builder
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
COPY cmd/ ./cmd/

# Install build-base for WebP CGO compilation
RUN apk add --no-cache build-base

# Compile static Go binary with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o main main.go

# Der Schlüsselwechsel gehört dorthin, wo die Daten sind.
#
# Ohne dieses Binary war das Werkzeug auf dem Server nicht ausführbar: Der Runner
# enthält kein Go, und cmd/ lag nicht einmal im Build-Kontext. Ein Werkzeug, das man
# genau dort nicht starten kann, wo man es braucht, ist keines — aufgefallen beim
# ersten echten Einsatz auf dem Schulserver am 06.08.2026.
#
# Kein zusätzliches Risiko: Das Kommando liest APP_ENCRYPTION_KEY aus der Umgebung,
# die im Container ohnehin gesetzt ist. Wer eine Shell im Container hat, hat den
# Schlüssel bereits.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o rotate-encryption-key ./cmd/rotate-encryption-key
# Die beiden Backup-Werkzeuge. Beide gehören INS Image — im Ernstfall hat der Schulserver
# kein Go, und docs/resilience_and_recovery.md behauptete bis 22.08.2026, restore-backup
# läge dort bereits (Prüfung 22.08., B).
#
# CGO_ENABLED=0, seit die Krypto in internal/backupkrypto liegt (23.08.2026). Vorher stand
# hier CGO_ENABLED=1: cmd/restore-backup importierte jobs, und jobs zieht über seine
# übrigen Dateien die WebP-Bibliothek (cgo) mit — ohne cgo brach der Build. Die Krypto
# selbst hängt nur an der Standardbibliothek und scrypt; ein zweiter cgo-Durchlauf je
# Deploy war der Preis für eine Abhängigkeit, die das Werkzeug nie brauchte.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o restore-backup ./cmd/restore-backup
# Gegenstück: verschlüsselt einen komprimierten Dump von stdin nach stdout. Die beiden
# Shell-Wege (update.sh, scripts/backup.sh) dumpen am nächtlichen Job vorbei und legten
# den Klartext offen ab (Befund A5); mit diesem Werkzeug im Container verschlüsseln sie
# über dieselbe Ableitung wie der Job — ohne den Schlüssel je an den Host zu reichen.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o encrypt-backup ./cmd/encrypt-backup

# ==============================================================================
# Stage 3: Runner container
# ==============================================================================
FROM alpine:3.24
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
# VERSIONIERT auf 16, nicht "neuester Client bedient beide" (so stand es hier bis zum
# 21.08.2026 — und das war nur die halbe Wahrheit): pg_dump 17 DUMPT einen älteren
# Server zwar, schreibt aber `SET transaction_timeout` (GUC ab PG 17) in den Dump —
# und der ließ sich damit unter ON_ERROR_STOP nicht mehr in den eigenen postgres:15
# einspielen. Die Backups der Produktion waren monatelang mit dem dokumentierten
# Restore-Weg NICHT wiederherstellbar; gefunden hat es die Restore-Probe
# (jobs/restore_probe.go) an ihrem ersten Testlauf. Client 16 dumpt Server 15 UND 16,
# und seine Dumps spielen nachweislich sauber in 15 ein (CI-Drill, Client 16→Server 15).
# Beim nächsten Server-Upgrade über 16 hinaus diese Zeile mitziehen — die Probe
# schlägt sonst am ersten Sonntag Alarm.
# apk upgrade zuerst: Das Basisimage trägt die Paketstände seines Release-Tags; Sicherheits-
# Fixes (z. B. openssl 3.5.8, CVE-2026-14456) kommen nur über das Upgrade herein. Ohne diese
# Zeile war der Trivy-Scan auf main rot, obwohl kein eigener Code betroffen war (26.08.2026).
RUN apk --no-cache upgrade && apk --no-cache add ca-certificates tzdata postgresql16-client

# Copy database schema file (for reference / first-run init)
COPY schema.sql ./

# Copy SQL migration files
COPY migrations/ ./migrations/

# Copy compiled Go binary
COPY --from=backend-builder /app/main .

# Wartungswerkzeug: Schlüsselwechsel ohne Datenverlust (docs/SECURITY.md).
#   docker compose exec backend ./rotate-encryption-key -neu <neu> -pruefen
COPY --from=backend-builder /app/rotate-encryption-key .
# Wiederherstellung (docs/resilience_and_recovery.md 2a) — die Argumente sind POSITIONAL,
# hier stand bis zum 23.08.2026 eine -in/-out-Form, die das Werkzeug nie kannte:
#   docker compose exec backend ./restore-backup backups/<datei>.sql.gz.enc /tmp/dump.sql
COPY --from=backend-builder /app/restore-backup .
# Verschlüsselung für die Shell-Wege (docs/resilience_and_recovery.md 1b):
#   … | docker exec -i bibliothek-backend ./encrypt-backup > <datei>.sql.gz.enc
COPY --from=backend-builder /app/encrypt-backup .

# Copy built Svelte static files
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Create non-privileged user, create the volume mount points, and give ownership.
#
# BEIDE Verzeichnisse müssen hier existieren — /app/uploads UND /app/backups. Docker
# übernimmt beim ersten Mounten eines leeren Named Volume Inhalt UND Eigentümer aus dem
# Image. Fehlt das Verzeichnis im Image, legt Docker es selbst an: als root. Der Container
# läuft aber als appuser (uid 1000) und kann dann nie hineinschreiben.
#
# Genau das war auf der Produktion der Fall (gefunden am 06.08.2026): /app/backups fehlte
# hier, das Volume gehörte root:root mit drwxr-xr-x, und der nächtliche Backup-Job scheiterte
# seit Anlegen des Volumes am 04.08. an "permission denied". BACKUP_ENCRYPTION_KEY war
# gesetzt, pg_dump lag im Image — es fehlte nur das Schreibrecht. Der Job protokolliert den
# Fehler (jobs/backup.go: "writing backup file failed"), aber jeder Container-Neustart
# verwischt die Logzeile, und niemand liest sie.
#
# /app/uploads funktionierte deshalb, weil es hier steht. Der Unterschied zwischen beiden
# war eine vergessene Zeile, kein Konzept.
RUN adduser -D appuser && \
    mkdir -p /app/uploads/fotos /app/backups && \
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

# Welcher Commit steckt in diesem Image?
#
# Am 11.08.2026 stand auf dem Produktivserver `git pull` durch, `./update.sh` aber nicht:
# Das Arbeitsverzeichnis zeigte den neuen Commit, das laufende Image war zehn Stunden alt
# und lieferte den Stand von vorgestern aus. Beide Gesundheitsprüfungen — Docker-Healthcheck
# und /health von aussen — meldeten dabei völlig zu Recht "gesund". Gesund heisst nicht
# aktuell, und keine der beiden Fragen war die richtige.
#
# Mit diesem Wert lässt sich die richtige Frage stellen: Läuft der Container, der aus DEM
# Commit gebaut wurde, der gerade im Arbeitsverzeichnis liegt? update.sh vergleicht nach
# dem Deploy `git rev-parse HEAD` damit. Ein Cache-Treffer beim Build ist dabei kein
# Problem — das Build-Argument ändert sich mit jedem Commit und bricht den Cache genau
# dieser Schicht.
#
# Leer gelassen, wenn niemand es setzt: Ein lokaler `docker build` ohne Argument soll
# weiterhin funktionieren, und update.sh sagt dann "unbekannt" statt zu behaupten, der
# Stand sei falsch.
ARG GIT_COMMIT=""
ENV GIT_COMMIT=${GIT_COMMIT}

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
