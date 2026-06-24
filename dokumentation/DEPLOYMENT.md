# Deployment Guide

> Zuletzt aktualisiert: 2026-06-24

---

## Übersicht

Das System besteht aus:
- **Go-Backend** (Port 8083 Produktion / 8084 lokal)
- **PostgreSQL 15/16**
- **Caddy** als Reverse-Proxy (TLS-Terminierung)
- **Docker Compose** als Orchestrierung

---

## 1. Umgebungsvariablen (Pflicht)

Alle Secrets werden über Umgebungsvariablen übergeben. **Niemals Secrets in die `.env` des Repos committen.**

| Variable | Beschreibung | Anforderung |
|---|---|---|
| `DATABASE_URL` | PostgreSQL-DSN | Pflicht |
| `JWT_SECRET` | HMAC-Signatur-Schlüssel | Pflicht, ≥ 32 Zeichen |
| `APP_ENCRYPTION_KEY` | AES-256-Schlüssel für Schülerfotos | Pflicht, genau 32 Bytes |
| `APP_ENV` | Umgebung (`production` / `local`) | Standard: `production` |
| `COOKIE_SECURE` | `true` hinter TLS-Proxy (Caddy) | Standard: `false` |
| `PORT` | HTTP-Port des Backends | Pflicht |
| `SMTP_HOST` | SMTP-Server | Optional (Mahnwesen) |
| `SMTP_PORT` | SMTP-Port | Standard: 587 |
| `SMTP_USER` | SMTP-Benutzername | Optional |
| `SMTP_PASSWORD` | SMTP-Passwort | Optional |
| `SMTP_FROM` | Absender-Adresse | Optional |
| `SMTP_ALLOW_INSECURE_TLS` | TLS-Zertifikatsprüfung deaktivieren | Nur für Legacy-SMTP-Server |
| `INITIAL_ADMIN_EMAIL` | E-Mail des initialen Admins | Standard: pflasch@philipp-reis-schule.de |
| `SENTRY_DSN` | Sentry Error Tracking | Optional |

---

## 2. Produktions-Deployment (Hetzner/Docker)

### 2.1 `.env`-Datei anlegen

Auf dem Server eine `.env`-Datei (nicht im Repo) anlegen:

```bash
# /opt/bibliothek/.env
POSTGRES_PASSWORD=<sicheres-passwort>
JWT_SECRET=<mindestens-32-zeichen-geheimes-jwt-secret>
APP_ENCRYPTION_KEY=<genau-32-bytes-aes-schluessel>
APP_ENV=production
COOKIE_SECURE=true
SMTP_HOST=smtp.example.com
SMTP_USER=user@example.com
SMTP_PASSWORD=<smtp-passwort>
SMTP_FROM=bibliothek@schule.de
```

### 2.2 Secret Guard

Der Server **verweigert den Start** in Produktion (`APP_ENV != local`), wenn bekannte Default-Secrets aus dem Repository verwendet werden. Fehlermeldung:
```
FATAL: JWT_SECRET nutzt einen bekannten Default-Wert. Setze im Produktionsbetrieb
ein eigenes, geheimes JWT_SECRET (≥32 Zeichen) — oder APP_ENV=local für die Entwicklung.
```

### 2.3 Docker Compose starten

```bash
cd /pfad/zur/bibliothek
docker compose --env-file .env up -d --build
```

`docker-compose.yml` erzwingt per `${VAR:?Fehlermeldung}`, dass alle Pflicht-Secrets gesetzt sind — Docker-Compose bricht mit einer sprechenden Fehlermeldung ab, wenn eine Variable fehlt.

### 2.4 Deployment-Skript

Das Skript `scripts/deploy.sh` automatisiert den Prozess:
```bash
./scripts/deploy.sh
```

Führt aus: `git pull` → `docker compose up -d --build` → ggf. Caddy-Neuladung.

---

## 3. Caddy Reverse Proxy

Bibliothek läuft hinter Caddy als TLS-Proxy im Docker-Netzwerk `caddy_global_net`.

### Caddyfile-Eintrag
```caddyfile
flasch3.herzog-dupont.de {
    reverse_proxy bibliothek-backend:8083
}
```

### Zero-Downtime Reload
```bash
# Wenn Caddy als Docker-Container läuft:
docker exec caddy caddy reload -c /etc/caddy/Caddyfile

# Wenn Caddy als systemd-Dienst läuft:
systemctl reload caddy
```

**Wichtig:** `restart` statt `reload` würde aktive Verbindungen anderer Dienste kappen.

---

## 4. Lokale Entwicklung (docker-compose.local.yml)

```bash
docker compose -f docker-compose.local.yml up -d
```

- Backend: `http://localhost:8084`
- PostgreSQL: `localhost:5434`
- `APP_ENV=local` → Default-Secrets aus `docker-compose.local.yml` sind erlaubt
- `COOKIE_SECURE=false` → kein TLS nötig

Die lokale Compose-Datei enthält bereits gültige Entwicklungs-Secrets (≥32 Zeichen), die bewusst im Repo liegen dürfen — sie gelten **nur** für `APP_ENV=local`.

---

## 5. Datenbank-Migrationen

Migrationen laufen **automatisch beim Serverstart** (`database.RunMigrations`). Manuelles Eingreifen ist nur bei Problemen nötig.

### Migrations-Verzeichnis: `migrations/`

| Datei | Inhalt |
|---|---|
| `030_ziel_jahrgang.sql` | LMF-Mehrstufenfristen; idempotent (beide Fälle: Spalte existiert / existiert nicht) |
| `032_reconcile_titel_columns.sql` | Idempotente Angleichung aller `buecher_titel`-Spalten (behebt Schema-Drift aus alten Deployments) |
| `033_unique_active_loan.sql` | Dedup bestehender Duplikate + Unique-Partial-Indizes für aktive Ausleihen |

### Neue Migration hinzufügen
1. Datei `migrations/NNN_beschreibung.sql` anlegen (NNN = nächste Nummer, kein Namenskonflikt)
2. Hash in `schema.sql` unter `schema_migrations` eintragen (wird beim nächsten Start automatisch geprüft)

---

## 6. Backup & Recovery

Automatischer Backup-Cronjob täglich um 02:30 Uhr (konfigurierbar in `jobs/cron.go`):

```
pg_dump → gzip → AES-GCM-Verschlüsselung (Zufalls-Nonce) → 0600 auf Disk
```

Backup-Rotation: älteste Dateien werden nach Ablauf des Aufbewahrungsfensters gelöscht.

Manuelles Backup:
```bash
./scripts/backup.sh
```

---

## 7. Health Check & Monitoring

Der Docker-Container enthält einen eingebauten Health Check:
```yaml
healthcheck:
  test: ["CMD-SHELL", "pg_isready -U postgres -d bibliothek"]
  interval: 5s
  timeout: 5s
  retries: 5
```

Optional: Sentry-Integration für Error-Tracking via `SENTRY_DSN`.
