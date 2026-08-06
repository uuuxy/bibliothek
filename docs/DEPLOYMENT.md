# Deployment Guide

> Zuletzt aktualisiert: 2026-08-06

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
| `APP_ENCRYPTION_KEY` | AES-256-Schlüssel für Schülerfotos **und** das gespeicherte SMTP-Passwort | Pflicht, genau 32 Bytes (oder 64 Hex-Zeichen). **Der einzige gültige Name** — `ENCRYPTION_KEY` wurde bis zum 06.08.2026 vorrangig gelesen und umging dabei jede Startprüfung; der Server bricht jetzt ab, wenn er abweichend gesetzt ist |
| `APP_ENV` | Umgebung (`production` / `local`) — steuert Cookie-Secure & Swagger | Standard: `production` |
| `ENFORCE_PROD_SECRETS` | Harte Start-Verweigerung bei Default-Secrets | Standard: `false` (Testphase) |
| `COOKIE_SECURE` | `true` hinter TLS-Proxy (Caddy) | Standard: **`true`**, außerhalb von `APP_ENV=local/development/test`. Nicht gesetzt → `true` mit Warnung im Log; unlesbarer Wert → harter Abbruch (`ermittleCookieSecure`). `docker-compose.yml` setzt zusätzlich `${COOKIE_SECURE:-true}` |
| `PORT` | HTTP-Port des Backends | Pflicht |
| `IMAP_HOST` | IMAP-Server der Schule — die Anmeldung prüft Zugangsdaten dagegen | **Pflicht.** Ohne diese Variable bricht der Start ab (`FATAL: IMAP_HOST ist nicht gesetzt`); lokal `IMAP_HOST=mock` zusammen mit `APP_ENV=local` |
| `IMAP_PORT` | IMAP-Port | Standard: 993 |
| `ALLOWED_ORIGIN` | Erlaubte Herkunft für CORS (die Frontend-Adresse der Schule) | Empfohlen in Produktion |
| `TRUSTED_PROXIES` | CIDRs/IPs, deren `X-Forwarded-For` geglaubt wird (Rate-Limit, Login-Brute-Force, Audit-Log) | Ohne sie gilt **nur Loopback** als vertrauenswürdig — hinter Caddy auf einem anderen Host also nötig |
| `BACKUP_DIR` | Zielverzeichnis der automatischen Backups | Standard siehe [resilience_and_recovery.md](resilience_and_recovery.md) |
| `BACKUP_ENCRYPTION_KEY` | AES-256-Schlüssel der Backups | **Ohne ihn läuft kein Backup** — der Job überspringt still |
| `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`, `S3_USE_SSL` | Optionaler Offsite-Upload der verschlüsselten Backups (`jobs/backup.go`) | Nur gemeinsam sinnvoll; fehlt eine, unterbleibt der Upload |
| `SMTP_HOST` | SMTP-Server | Optional (Mahnwesen) |
| `SMTP_PORT` | SMTP-Port | Standard: 587 |
| `SMTP_USER` | SMTP-Benutzername | Optional |
| `SMTP_PASSWORD` | SMTP-Passwort | Optional |
| `SMTP_FROM` | Absender-Adresse | Optional |
| `SMTP_ALLOW_INSECURE_TLS` | TLS-Zertifikatsprüfung deaktivieren (TLS bleibt) | Nur für Legacy-SMTP-Server |
| `SMTP_ALLOW_PLAINTEXT` | Versand **ganz ohne** TLS erlauben, wenn der Server kein STARTTLS anbietet | Nur für ein Legacy-Relay. Ohne diese Variable bricht der Versand in dem Fall ab, statt Mahntexte im Klartext zu schicken |
| `INITIAL_ADMIN_EMAIL` | E-Mail des initialen Admins | Standard: pflasch@philipp-reis-schule.de |
| `SENTRY_DSN` | Sentry Error Tracking | Optional |

**Nicht als Variable, sondern in der Oberfläche:** Mail-Zugangsdaten und die **Öffentliche
Adresse** (Einstellungen → Allgemein) leben in der Datenbank. Die `SMTP_*`-Variablen werden
beim ersten Start einmalig übernommen und sind danach nur noch Rückfall — beim Debuggen
also die DB-Zeile ansehen, nicht die `.env`. Die Öffentliche Adresse (z. B.
`https://bibliothek.schule.de`) ist die Grundlage des Bestätigungs-Links an Lieferanten;
ohne sie verschickt das System Bestellungen ohne Link. Der Server kann sie nicht erraten:
Hinter dem Reverse-Proxy sieht er nur seinen internen Namen.

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
ENFORCE_PROD_SECRETS=true   # erst beim echten Prod-Deploy scharf schalten
COOKIE_SECURE=true
# OHNE diesen Schlüssel werden KEINE Backups erstellt — der nächtliche Job
# überspringt sich (jobs/backup.go). Das fällt sonst erst auf, wenn man ein Backup
# BRAUCHT. Mindestens 32 Zeichen: Die Ableitung läuft per SHA-256, kurze Passphrasen
# sind an einer entwendeten Backup-Datei offline angreifbar.
BACKUP_ENCRYPTION_KEY=<mindestens-32-zeichen-passphrase>
SMTP_HOST=smtp.example.com
SMTP_USER=user@example.com
SMTP_PASSWORD=<smtp-passwort>
SMTP_FROM=bibliothek@schule.de
```

### 2.2 Secret Guard (per Schalter einschaltbar)

Die harte Start-Verweigerung ist von `APP_ENV` **entkoppelt** und wird über den dedizierten Schalter `ENFORCE_PROD_SECRETS` gesteuert:

| Phase | `ENFORCE_PROD_SECRETS` | Verhalten |
|---|---|---|
| Test-/Pilotbetrieb | `false` (Standard) | Stack startet auch mit Default-Secrets — bequemes Testen |
| Echter Prod-Deploy | `true` | Server **verweigert den Start**, wenn ein bekannter Default für `JWT_SECRET` oder `APP_ENCRYPTION_KEY` aktiv ist |

> **Warum entkoppelt von `APP_ENV`?** `APP_ENV=local` würde gleichzeitig das Cookie-`Secure`-Flag deaktivieren und die Swagger-Docs öffentlich freischalten — auf einem über das Internet erreichbaren Test-Server unerwünscht. Mit `ENFORCE_PROD_SECRETS` bleibt `APP_ENV=production` (sichere Cookies, kein Swagger), während die Secret-Härtung unabhängig davon ein-/ausgeschaltet wird.

Fehlermeldung bei `ENFORCE_PROD_SECRETS=true` + Default-Secret:
```
FATAL: JWT_SECRET nutzt einen bekannten Default-Wert. Setze ein eigenes, geheimes
JWT_SECRET (≥32 Zeichen) — oder ENFORCE_PROD_SECRETS=false während der Testphase.
```

**Prüfen statt hoffen:** `./scripts/pruefe_secrets.sh /pfad/zur/.env` liest die Datei,
ändert nichts und meldet genau die Fehlkonfigurationen, die im Betrieb still bleiben —
Default-Secrets, fehlender Backup-Schlüssel, `IMAP_HOST=mock`, offene Produktionsschalter.
Exit-Code 1 bei kritischem Befund, damit es sich in ein Deploy-Skript hängen lässt.

**Checkliste vor dem ersten echten Prod-Deploy:** `ENFORCE_PROD_SECRETS=true` setzen und dazu echte Werte für `JWT_SECRET`, `APP_ENCRYPTION_KEY`, `POSTGRES_PASSWORD`, `BACKUP_ENCRYPTION_KEY` sowie `COOKIE_SECURE=true` (hinter Caddy-HTTPS).

> **`APP_ENCRYPTION_KEY` auf einem System mit Bestand ändern?** Nicht einfach
> überschreiben — Schülerfotos und das gespeicherte SMTP-Passwort sind damit
> verschlüsselt und wären danach verloren. Der Weg mit Umschlüsselung steht in
> [SECURITY.md](SECURITY.md#app_encryption_key-wechseln) (`cmd/rotate-encryption-key`).

> **Fehlt `BACKUP_ENCRYPTION_KEY`,** protokolliert der Server das beim Start
> (`ACHTUNG: … es werden KEINE Datenbank-Backups erstellt`) und das Admin-Dashboard
> meldet den Backup-Status als `critical`.

### 2.3 Docker Compose starten

```bash
cd /pfad/zur/bibliothek
docker compose --env-file .env up -d --build
```

`docker-compose.yml` liefert für alle Secrets bequeme Defaults (`${VAR:-…}`), damit der Stack in der Testphase ohne weitere Konfiguration startet. Die Produktions-Absicherung übernimmt der Code-Guard (`ENFORCE_PROD_SECRETS=true`), nicht die Compose-Datei.

### 2.4 Deployment-Skript

Der übliche Weg auf dem Server ist `./update.sh`:
```bash
git pull && ./update.sh
```
`git pull` zuerst und getrennt, weil das Skript sich sonst in der gerade laufenden
Fassung selbst aktualisiert.

Führt aus: **Backup** → `git pull` → `docker compose up -d --build` →
**Gesundheitsprüfung** → alte Backups aufräumen. Bei Fehlschlag bricht es ab und gibt
eine Rollback-Anleitung samt Pfad zum eben erzeugten Backup aus.

`scripts/deploy.sh` ist der ältere, schlankere Weg (`git pull` →
`docker compose up -d --build` → Caddy-Block prüfen/ergänzen) — **ohne** Backup und
**ohne** Gesundheitsprüfung. Details zu beiden: [SCRIPTS.md](SCRIPTS.md).

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

**Achtung — zwei Wege legen UNVERSCHLÜSSELTE Dumps ab:** `scripts/backup.sh` (7 Tage
Rotation) und `./update.sh` vor jedem Deploy (30 Tage, `backups/backup_<Zeitstempel>.sql.gz`).
Beide enthalten jeden Schülernamen, jede Adresse und jede Ausleihe im Klartext und werden
seit dem 06.08.2026 mit `0600` angelegt. Das Verzeichnis `backups/` gehört damit zum
schutzbedürftigen Bestand — es ist **kein** Ersatz für das verschlüsselte Backup und
darf nicht mit ausgeliefert oder weitergereicht werden. Details:
[resilience_and_recovery.md](resilience_and_recovery.md).

### Backup-Umfang: nur die Datenbank (bewusste Entscheidung, 11.07.2026)

Das `uploads/`-Volume (Buchcover als lokale WebP-Dateien) wird **absichtlich nicht**
mitgesichert:

- **Schülerfotos** liegen verschlüsselt in der Datenbank (`schueler_fotos.foto_encrypted`)
  und sind damit vom pg_dump abgedeckt — es gehen keine personenbezogenen Daten verloren.
- **Cover sind reproduzierbar**: Der Cover-Sync-Job (alle 6 h + bei Serverstart,
  `internal/service/cover_service.go`) lädt PENDING/FAILED-Titel gedrosselt
  (2 Titel/s) von DNB/Google/OpenLibrary nach. **Achtung:** Titel mit Status
  `FOUND` und totem `/uploads/`-Pfad überspringt der Job — nach einem Restore
  ohne Volume einmalig zurücksetzen, dann heilt der nächste Lauf alles nach:
  ```sql
  UPDATE buecher_titel SET cover_status = 'PENDING'
  WHERE cover_url LIKE '/uploads/%';
  ```
- **Etiketten/PDFs** werden on-demand generiert und nie persistiert.

Wer das Nachladen nach einem Restore vermeiden will (z. B. Offline-Betrieb), kann das
Volume zusätzlich mit `docker run --rm -v bibliothek_uploads:/data alpine tar czf - /data`
wegsichern — Pflicht ist es nicht.

---

## 7. Health Check & Monitoring

Es gibt **zwei** Health Checks, und sie prüfen Verschiedenes. Hier stand bis zum
06.08.2026 nur der Datenbank-Check, ausgegeben als wäre es der des Anwendungscontainers.

**Anwendung** — im `Dockerfile` (nicht in der Compose-Datei):
```dockerfile
HEALTHCHECK --interval=10s --timeout=3s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:$PORT/health || exit 1
```
`--start-period=30s` ist nötig, weil beim ersten Start die Migrationen laufen; ohne sie
galt der Container in dieser Zeit als krank. Die Sonde spricht bewusst `127.0.0.1`
im Container an — über den öffentlichen Namen hätte sie die HTTPS-Umleitung getroffen
und wäre an der 301 gescheitert (Commit `ea5bf6d`).

**Datenbank** — in `docker-compose.yml` beim Dienst `postgres-db`:
```yaml
healthcheck:
  test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-postgres} -d ${POSTGRES_DB:-bibliothek}"]
```
Er ist die Bedingung für `depends_on: service_healthy` — das Backend startet erst, wenn
die Datenbank Anfragen annimmt.

`./update.sh` fragt beim Deploy **beide Quellen** ab (Docker-Status *und* `/health` im
Container), weil keine für sich genügt: Der Docker-Status braucht Anlauf, und ein
Container ohne Healthcheck liefert dort gar nichts.

Optional: Sentry-Integration für Error-Tracking via `SENTRY_DSN`.
