# 7. Verteilungssicht

Stand: 26.09.2026 · Betriebsanleitung: [DEPLOYMENT.md](../DEPLOYMENT.md)

---

## 7.1 Produktion: ein Host, drei Prozesse

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│ Schulserver (Linux, Docker)                                                     │
│                                                                                 │
│  ┌──────────────────────────────┐                                               │
│  │ Caddy (Host bzw. eigener     │  :80/:443 öffentlich                          │
│  │ Container)                   │  ACME/Let's Encrypt, Zertifikat selbst geholt │
│  │ /root/caddy/Caddyfile        │  response_header_timeout 300s                 │
│  │ ← update_caddy.sh            │  read_timeout 600s, write_timeout 600s        │
│  └──────────────┬───────────────┘                                               │
│                 │ Docker-Netz »caddy_global_net« (extern)                       │
│                 ▼                                                               │
│  ┌──────────────────────────────────────────────┐                               │
│  │ bibliothek-backend                           │  127.0.0.1:8083 → :8083      │
│  │ alpine:3.24, USER appuser (non-root)          │  restart: unless-stopped     │
│  │ HEALTHCHECK wget /health (10s/3s/30s, 3×)     │  logging json-file 3×10 MB   │
│  │ Volumes: /app/uploads, /app/backups           │  ENV GIT_COMMIT (Build-Arg)  │
│  └──────────────┬───────────────────────────────┘                               │
│                 │ postgres-db:5432 (nur Docker-Netz)                            │
│                 ▼                                                               │
│  ┌──────────────────────────────────────────────┐                               │
│  │ bibliothek-db — postgres:18-alpine            │  127.0.0.1:5434 → :5432      │
│  │ Volume postgres_data:/var/lib/postgresql      │  healthcheck pg_isready      │
│  └──────────────────────────────────────────────┘                               │
│                                                                                 │
│  Volumes: postgres_data · bibliothek_uploads · bibliothek_backups (benannt)      │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### Die Betriebsentscheidungen, die in dieser Zeichnung stecken

| Entscheidung                                                   | Grund                                                                                                                                                                              |
| -------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Alle Host-Ports auf `127.0.0.1` gebunden**                    | `"5434:5432"` veröffentlichte die Datenbank auf **allen** Interfaces: Wer den Host im Netz erreichte, sprach mit dem DSN aus der `.env` direkt die Datenbank an — vorbei an Rolle, Audit und Anonymisierung. Für einen Zugriff von außen: SSH-Tunnel. |
| **`restart: unless-stopped` für beide Container**               | `unattended-upgrades` startete den Docker-Daemon neu, und der Dienst lag **zwölf Stunden** still. Die Policy stand danach als uncommitteter Hotfix auf dem Server und blockierte jedes `git pull` — sie gehört ins Repo. |
| **Log-Rotation 3 × 10 MB je Container**                          | Der Docker-Default `json-file` kennt **keine** Größengrenze. Das fällt nicht in Wochen auf, sondern in dem Moment, in dem die Platte voll ist — und mit ihr die Datenbank desselben Hosts. |
| **`GIT_COMMIT` als Build-Argument ins Image**                    | Beantwortet die Frage, die kein Healthcheck beantwortet: Läuft der Container, der aus **diesem** Commit gebaut wurde? „Gesund" heißt nicht „aktuell".                              |
| **Volume-Mount auf `/var/lib/postgresql`, nicht `…/data`**        | Seit dem 18er-Image liegt `PGDATA` unter `/var/lib/postgresql/18/docker`. Der alte Mount ginge ins Leere — die Daten landeten in einem **anonymen** Volume und wären beim nächsten Neuerstellen weg. |
| **`bibliothek_backups` als benanntes Volume**                     | Damit die verschlüsselten Dumps einen Container-Neubau überleben.                                                                                                                 |
| **Netz `caddy_global_net` extern**                                | Auf dem Host laufen mehrere Dienste (schul-orga, inventur, bibliothek) hinter **einem** Caddy.                                                                                     |
| **Die `Caddyfile` im Repo ist nicht maßgeblich**                   | Maßgeblich ist `/root/caddy/Caddyfile` auf dem Server, geschrieben von `update_caddy.sh`. Die Repo-Datei sagt das in ihrer ersten Zeile — eine Vorlage, die niemand ausprobiert, fällt erst im Ernstfall auf. |

### Fristen müssen zueinander passen

| Ort                                | Frist                                        | Warum genau dort                                                                          |
| ---------------------------------- | -------------------------------------------- | ----------------------------------------------------------------------------------------- |
| `http.Server.ReadHeaderTimeout`    | 5 s                                          | Slowloris auf der Header-Phase                                                            |
| `http.Server.ReadTimeout`          | `api.StandardLesefrist`                      | Begrenzt das **Senden des Rumpfes** — eine Kontext-Deadline bricht kein blockierendes Read auf |
| `ErweitereLesefristFuerLangeUploads` | angehoben für Import-Routen                | Steht **vor** dem Body-Limit, also vor jedem Lesen — danach gesetzt käme sie zu spät       |
| `http.Server.IdleTimeout`          | 120 s                                        | Getrennt gesetzt, damit nicht `ReadTimeout` beide Fristen aneinanderkettet                 |
| `http.Server.WriteTimeout`         | **bewusst nicht gesetzt**                    | `/events` ist definitionsgemäß eine Antwort, die nie endet; ein globaler Wert würde jede Live-Verbindung kappen |
| `TimeoutMiddleware`                | `StandardBearbeitungsfrist` (je Route)       | Begrenzt die Bearbeitungsdauer im Handler                                                  |
| Caddy                              | 300 s Header, 600 s Read/Write               | Muss ≥ der längsten Anwendungsfrist sein — läuft die kürzeste Frist vorne weg, bricht der Vorgang dort ab, wo niemand die Meldung sieht |

---

## 7.2 Das Image (mehrstufiger Build)

| Stufe                    | Basis                  | Erzeugnis                                                                                                          |
| ------------------------ | ---------------------- | ------------------------------------------------------------------------------------------------------------------ |
| 1 `frontend-builder`     | `node:24-alpine`       | `npm ci` (Lockfile verbindlich) → `npm run build` → `frontend/dist`                                                |
| 2 `backend-builder`      | `golang:1.27.1-alpine` | `main` mit **CGO_ENABLED=1** (WebP), dazu `rotate-encryption-key`, `restore-backup`, `encrypt-backup`, `migrate-fotos` mit **CGO_ENABLED=0** |
| 3 Laufzeit               | `alpine:3.24`          | `apk upgrade` + `ca-certificates`, `tzdata`, **`postgresql18-client`**; Binaries, `schema.sql`, `migrations/`, `frontend/dist`; `USER appuser`; `EXPOSE 8081`; `HEALTHCHECK` auf `/health`; `CMD ["./main"]` |

**Warum die Werkzeuge mit ins Image gehören:** Der Schulserver hat kein Go. Ein
Wiederherstellungswerkzeug, das man genau dort nicht starten kann, wo man es braucht, ist
keines — aufgefallen beim ersten echten Einsatz am 06.08.2026. Deshalb liegen
`restore-backup`, `encrypt-backup`, `rotate-encryption-key` und `migrate-fotos` im Image,
und ein Gate (`docs/werkzeuge_im_image_test.go`) hält die Anleitungen damit deckungsgleich.

**Warum die Go-Version patch-genau gepinnt ist:** CVE-Fixes stecken in
Stdlib-Patches; ein Build mit `golang:1.27` könnte die verwundbaren Pakete einkompilieren.
`docs/umgebung_paritaet_test.go` erzwingt die Paarung von `go.mod` und `Dockerfile` —
Dependabot hebt nur eine der beiden Zeilen.

**Warum `postgresql18-client` im Image ist:** `pg_dump` für das nächtliche Backup und
`psql` für die Restore-Probe. Der Client **muss** bei jedem Server-Upgrade mitziehen, sonst
schlägt die Restore-Probe sonntags Alarm (ein älterer Client verweigert den neueren Server).

---

## 7.3 Lokale Entwicklung

| Variante                  | Kommando                                              | Erreichbar                                                    |
| ------------------------- | ----------------------------------------------------- | ------------------------------------------------------------- |
| Vollständiger Stack       | `docker compose -f docker-compose.local.yml up -d`    | Anwendung `http://localhost:8084`, DB `127.0.0.1:5434`         |
| Frontend mit Hot Reload   | `cd frontend && npm ci && npm run dev`                | `http://localhost:5173`, reicht `/api`, `/login`, `/uploads`, `/events` an `127.0.0.1:8084` durch |
| Backend von Hand          | `go run main.go`                                      | Port aus der `.env` (Beispiel: 8081) — dann `VITE_API_TARGET=http://127.0.0.1:8081 npm run dev` |

Der lokale Stack fährt **dieselbe** Postgres-Major-Version wie Produktion und CI. Die
Migrationen laufen beim Start automatisch. `IMAP_HOST=mock` akzeptiert in
Entwicklung/Test jedes Passwort für jede in `benutzer` vorhandene E-Mail — und der Server
protokolliert das beim Start ausdrücklich.

> **Auf den Port achten.** Der Dev-Server zeigt auf den **lokalen Docker-Stack** (8084).
> Bis zum 12.08.2026 war er fest auf 8083 verdrahtet — den Port des **Produktions**-Stacks.
> Wer der Anleitung folgte, bekam eine Oberfläche, deren API-Aufrufe alle ins Leere liefen,
> ohne dass irgendetwas auf den Port hinwies.

---

## 7.4 Konfiguration (Umgebungsvariablen)

| Variable                                        | Pflicht     | Wirkung bei Fehlen / Besonderheit                                                                            |
| ----------------------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------ |
| `DATABASE_URL`                                  | ja          | Start bricht ab                                                                                              |
| `JWT_SECRET` (≥ 32 Zeichen)                     | ja          | Start bricht ab; bekannter Default-Wert → Start verweigert (Secret-Guard)                                     |
| `APP_ENCRYPTION_KEY` (32 Byte / 64 Hex)         | ja          | Start bricht ab; ein abweichend gesetzter **früherer Zweitname** bricht ebenfalls ab, statt still zu gelten    |
| `PORT`                                          | ja          | Start bricht ab                                                                                              |
| `POSTGRES_PASSWORD`                             | ja (Compose)| Compose startet nicht — es gibt bewusst kein committetes Default-Passwort                                      |
| `APP_ENV`                                       | Vorgabe `production` | Steuert Cookie-Secure-Vorgabe, Swagger-Sichtbarkeit und den Secret-Guard                              |
| `ENFORCE_PROD_SECRETS`                          | nein        | Seit 05.09.2026 **von selbst scharf**; nur ein ausdrückliches `false` schaltet es für eine Testphase ab — mit Warnung und Vermerk in der Selbstprüfung |
| `COOKIE_SECURE`                                 | nein        | Fehlt → sichere Vorgabe `true` (mit Warnung); unlesbarer Wert → **Abbruch**; `false` außerhalb lokal → laute Warnung |
| `TRUSTED_PROXIES`                               | praktisch ja| Ohne den Wert sieht das Backend hinter Caddy nur eine Proxy-IP: fünf Fehl-Logins eines Nutzers sperren **alle** (globaler DoS). Compose setzt das private Docker-Netz als Vorgabe |
| `IMAP_HOST`, `IMAP_PORT`                        | ja          | `PruefeIMAPKonfiguration()` bricht früh ab — fehlt der Host, kann sich niemand anmelden; steht er auf `mock`, jeder |
| `SELBSTANMELDUNG_DOMAIN`                        | nein        | Leer ⇒ der Weg ist zu; richtige Zugangsdaten enden in „Anmeldung fehlgeschlagen". Die Selbstprüfung meldet das als Warnung |
| `INITIAL_ADMIN_EMAIL`                           | nein        | Leer ⇒ System startet ohne Admin-Zugang und sagt das in einer Logzeile. **Kein** Passwort — es gibt keins      |
| `BACKUP_ENCRYPTION_KEY`, `BACKUP_DIR`           | nein        | Ohne Schlüssel **überspringt** sich der Backup-Job — die Selbstprüfung macht genau das sichtbar                 |
| `S3_ENDPOINT/ACCESS_KEY/SECRET_KEY/BUCKET/USE_SSL` | nein     | Nur bei vollständiger Angabe läuft der Offsite-Upload; sonst überspringt der Job ihn und protokolliert es       |
| `ALLOWED_ORIGIN`                                | nein        | CORS-Herkunft der Schuldomain                                                                                 |
| `SENTRY_DSN`                                    | nein        | Ohne DSN kein Sentry                                                                                          |
| `SMTP_ALLOW_PLAINTEXT`                          | nein        | Nur `true` erlaubt Versand ohne STARTTLS — mit deutlicher Warnung über die Folgen                              |
| `TZ`                                            | nein        | Hält Logs konsistent; gefahrlos setzbar, weil der Cron-Plan auf UTC genagelt ist                               |

`docs/compose_variablen_test.go` prüft die Richtung, die ohne Liste auskommt: Was der Code
liest, muss in Compose ankommen.

---

## 7.5 Deployment-Wege

| Weg                        | Was er tut                                                                                                                                                              | Wann                             |
| -------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------- |
| **`./update.sh`**          | 1. `pg_dump`-Vorabsicherung → 2. `git pull` → 3. `docker compose up -d --build` → 4. Gesundheitsprüfung **und** Commit-Abgleich (`GIT_COMMIT` im Image vs. `git rev-parse HEAD`) → 5. Vorabsicherung verschlüsseln, Klartext löschen → 6. Backups > 30 Tage aufräumen → 7. Build-Cache > 7 Tage aufräumen. Bei Fehler: Abbruch **mit ausgedrucktem Rückweg** auf den Stand, der vorher lief (Commit des laufenden Images, seit 25.09.2026; ohne lesbares Image der Commit des Arbeitsverzeichnisses, mit Warnung) | **Der gepflegte Weg** für Updates |
| `scripts/deploy.sh`        | Stammt aus der Einrichtung (trägt den Caddy-Block nach). Baut und geht — **kein** Backup, **keine** Gesundheitsprüfung, **kein** Commit-Beweis                            | Ersteinrichtung                   |
| `scripts/stack-neu.sh`     | Stack neu aufsetzen                                                                                                                                                      | Notfall/Neuaufbau                 |
| `update_caddy.sh`          | Schreibt die maßgebliche Caddy-Konfiguration auf dem Server                                                                                                              | Proxy-Änderungen                  |

> Beide Skripte kapseln ihren gesamten Rumpf in `{ … }` mit einem `exit` **innerhalb** der
> Klammer. Grund: Sie ziehen per `git pull` neuen Code und überschreiben sich dabei
> **selbst, während sie laufen**. Bash merkt sich eine Position in der Datei; wird sie
> unterwegs länger, führt bash Bruchstücke aus (am 11.08.2026 nachgestellt). `{ … }` ist
> **ein** zusammengesetzter Befehl, den bash vollständig parst, bevor er beginnt.
>
> Daraus folgt eine Eigenschaft, die man kennen muss: **Ein Deploy-Skript ändert sich
> immer erst für den nächsten Lauf.**

---

## 7.6 CI/CD

| Workflow              | Auslöser                          | Stufen                                                                                                                                                                                       |
| --------------------- | --------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ci.yml`              | Push, Pull Request                | `actionlint` · `build-and-test` (gofmt, **golangci-lint**, Deadcode-Gate, Postgres-Client für die Restore-Probe, `go test -race ./...`, **Skip-Bilanz**) · `frontend-test` (ESLint, Prettier, `svelte-check`, Vitest) · e2e (lokaler Stack **bauen**, Playwright-Browser aus Cache, Druck-Sektionen-Gate, Playwright; bei Fehlschlag Traces und Backend-Logs als Artefakt) |
| `security-scan.yml`   | Push, Pull Request, **Zeitplan**  | `govulncheck` über `scripts/govulncheck-gate.sh` (benannte Ausnahmen in `security/vuln-ausnahmen.json`) · `gosec` · `npm audit` · Trivy-Scan des **gebauten Images** samt Container-Smoke (läuft unprivilegiert? sind die Laufzeitwerkzeuge da? ist jedes Volume beschreibbar?) |
| `docker-publish.yml`  | Push auf `main`, `v*.*.*`, manuell | Build und Push nach `ghcr.io/uuuxy/bibliothek`, `linux/amd64`                                                                                                                                 |
| `release.yml`         | Tag `v*.*.*`                      | Prüft Muster, Zugehörigkeit zu `main` und **grüne CI**, dann GitHub-Release mit generierten Notes                                                                                             |

Dazu lokal: `scripts/install-hooks.sh` installiert pre-commit (Formatierung, ESLint,
`golangci-lint`) und pre-push (Go-Tests, `golangci-lint`, `svelte-check`, Vitest,
`npm audit`, `govulncheck`, Trivy, `deadcode` — plus die Skip-Bilanz).

**Die Lehre vom 12.09.2026 steckt in dieser Reihenfolge:** Weil der Lint-Schritt in CI
**vor** den Tests kommt, hat ein einziger ungenutzter Typ in einer Testdatei auch
Unit-Tests, Deadcode-Gate und Restore-Probe mit übersprungen; `main` stand knapp drei
Stunden rot, ohne dass lokal etwas zu sehen war. Seither kennt der pre-push-Hook denselben
Linter — `go vet` meldet ungenutzte **Typen** nicht.

---

## 7.7 Datensicherung im Betrieb

```
täglich 02:30  pg_dump → gzip → AES-256-GCM (scrypt)  →  Volume bibliothek_backups
                                                       └─ optional S3 (minio-go)
vor jedem Deploy  Vorabsicherung (update.sh), nach Erfolg verschlüsselt, Klartext gelöscht
wöchentlich So 03:30  Restore-Probe in eine Wegwerf-Datenbank  →  Befund der Selbstprüfung
Aufbewahrung  Backups > 30 Tage werden von update.sh aufgeräumt
```

**Gesichert wird nur die Datenbank** — eine bewusste Entscheidung vom 11.07.2026. Cover
unter `/app/uploads` sind aus ISBN und Quelle reproduzierbar; Schülerfotos liegen
verschlüsselt **in** der Datenbank und sind damit im Dump enthalten.

Das Passwort erreicht `pg_dump` über eine temporär angelegte `.pgpass` mit engen Rechten,
**nicht** über `PGPASSWORD` — Umgebungsvariablen sind für andere Prozesse desselben Systems
sichtbar (`.jules/sentinel.md`).

Der Rückweg steht Schritt für Schritt in
[resilience_and_recovery.md](../resilience_and_recovery.md), inklusive der Gegenprobe
**vor** dem Löschen und der Sicherung des aktuellen Stands **vor** dem Einspielen.
