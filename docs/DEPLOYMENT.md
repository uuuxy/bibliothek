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
| `BACKUP_ENCRYPTION_KEY` | AES-256-Schlüssel der Backups | **Ohne ihn läuft kein Backup** — der Job überspringt sich mit einer Logzeile; sichtbar wird das im Admin-Dashboard über `/api/admin/system/backup-status` |
| `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`, `S3_USE_SSL` | Optionaler Offsite-Upload der verschlüsselten Backups (`jobs/backup.go`) | Nur gemeinsam sinnvoll; fehlt eine, unterbleibt der Upload |
| `SMTP_HOST` | SMTP-Server | Optional (Mahnwesen) |
| `SMTP_PORT` | SMTP-Port | Standard: 587 |
| `SMTP_USER` | SMTP-Benutzername | Optional |
| `SMTP_PASSWORD` | SMTP-Passwort | Optional |
| `SMTP_FROM` | Absender-Adresse | Optional |
| `SMTP_ALLOW_INSECURE_TLS` | TLS-Zertifikatsprüfung deaktivieren (TLS bleibt) | Nur für Legacy-SMTP-Server |
| `SMTP_ALLOW_PLAINTEXT` | Versand **ganz ohne** TLS erlauben, wenn der Server kein STARTTLS anbietet | Nur für ein Legacy-Relay. Ohne diese Variable bricht der Versand in dem Fall ab, statt Mahntexte im Klartext zu schicken |
| `INITIAL_ADMIN_EMAIL` | E-Mail des initialen Admins — **nur** wenn die Tabelle `benutzer` leer ist | Standard aus `docker-compose.yml`: pflasch@philipp-reis-schule.de. Ein Passwort gibt es dazu nicht (Anmeldung per IMAP); die Adresse muss auf dem Schulserver existieren, sonst kommt der so angelegte Admin nicht hinein |
| `RATE_LIMIT` | Globales Limit in Requests/s/IP | Vorgabe **50** (der Produktionswert). Ein unlesbarer Wert wird protokolliert und ignoriert, es gilt dann die Vorgabe |
| `SMTP_PASS` | Historischer Zweitname für `SMTP_PASSWORD` | Wird nur gelesen, wenn `SMTP_PASSWORD` leer ist. `docker-compose.yml` setzt beide Namen — beim Suchen nach „warum nimmt er das alte Passwort" ist das die Stelle |
| `SENTRY_DSN` | Sentry Error Tracking | Optional |

**Nicht als Variable, sondern in der Oberfläche:** Mail-Zugangsdaten (Einstellungen → Mail)
und die **Öffentliche Adresse** (Einstellungen → Erreichbarkeit & Alarme) leben in der Datenbank. Die `SMTP_*`-Variablen werden
beim ersten Start einmalig übernommen und sind danach nur noch Rückfall — beim Debuggen
also die DB-Zeile ansehen, nicht die `.env`. Die Öffentliche Adresse (z. B.
`https://bibliothek.schule.de`) ist die Grundlage des Bestätigungs-Links an Lieferanten;
ohne sie verschickt das System Bestellungen ohne Link. Der Server kann sie nicht erraten:
Hinter dem Reverse-Proxy sieht er nur seinen internen Namen.

---

## 2. Produktions-Deployment (Hetzner/Docker)

### 2.1 `.env`-Datei anlegen

Auf dem Server eine `.env`-Datei (nicht im Repo) anlegen. **Die Geheimnisse werden
erzeugt, nicht abgetippt** — ein wörtlich übernommenes `<mindestens-32-zeichen-…>` ist
43 Zeichen lang, besteht damit jede Längenprüfung und ist zugleich ein Wert, der in diesem
Repository nachzulesen ist:

```bash
cd /opt/bibliothek

# Zufällige Geheimnisse direkt in die .env schreiben.
# ÜBERALL -hex, NIRGENDS -base64 — die Begründung steht direkt darunter.
{
  echo "POSTGRES_PASSWORD=$(openssl rand -hex 24)"
  echo "JWT_SECRET=$(openssl rand -hex 32)"
  echo "APP_ENCRYPTION_KEY=$(openssl rand -hex 32)"      # 64 Hex = 32 Byte
  echo "BACKUP_ENCRYPTION_KEY=$(openssl rand -hex 32)"
} >> .env
```

> **Warum `-hex` und nicht `-base64`?** `POSTGRES_PASSWORD` landet unverändert in der
> `DATABASE_URL`:
>
> ```
> postgres://postgres:PASSWORT@postgres-db:5432/bibliothek?sslmode=disable
> ```
>
> Base64 erzeugt `/`, `+` und `=`. Ein `/` im Passwort beendet für den URL-Parser den
> Host-Teil, ein `+` verschiebt den Rest — der Server startet dann nicht mehr:
>
> ```
> failed to parse as URL (invalid port ":sGWO+wgLlTIj" after host)
> ```
>
> Genau das ist am 06.08.2026 auf der Produktion passiert: Der Passwortwechsel lief
> scheinbar sauber durch (`ALTER ROLE`, Container „Started", `pruefe_secrets.sh` meldete
> **alles grün**) — und die Anwendung war trotzdem unten. Die Konfigurationsprüfung liest
> die Datei, nicht die Verbindung; sie kann diesen Fehler nicht sehen.
>
> `-hex` liefert nur `0-9a-f` und ist in jeder URL unverändert gültig. Für die anderen
> Werte wäre Base64 unschädlich, aber eine Regel ohne Ausnahme ist leichter zu befolgen
> als eine mit.

**Nach jedem Passwortwechsel am Live-Pfad prüfen, nicht an der Datei** — sonst wiederholt
sich genau der Fall oben:

```bash
docker compose up -d
docker logs bibliothek-backend --since 2m 2>&1 | grep -iE "pool|error|fatal"
curl -s -o /dev/null -w "health: %{http_code}\n" https://DEINE-DOMAIN/health
```

Richtig ist: `Database connection pool successfully initialized.` und `health: 200`.

Den Rest von Hand ergänzen — das sind Einstellungen, keine Geheimnisse:

```bash
# /opt/bibliothek/.env
APP_ENV=production
ENFORCE_PROD_SECRETS=true   # erst beim echten Prod-Deploy scharf schalten
COOKIE_SECURE=true
SMTP_HOST=smtp.example.com
SMTP_USER=user@example.com
SMTP_FROM=bibliothek@schule.de
```

`SMTP_PASSWORD` gehört ebenfalls hinein, wird aber nicht erzeugt, sondern vom Mailserver
vorgegeben. Es steht nur für die Erstbefüllung in der `.env`: Ab dem ersten Start ist die
Einstellung in der Oberfläche maßgeblich (`mail_settings_config`).

> **`BACKUP_ENCRYPTION_KEY` nicht vergessen** — ohne ihn werden **keine** Backups erstellt;
> der nächtliche Job überspringt sich mit einer Logzeile (`jobs/backup.go`). Das fällt
> sonst erst auf, wenn man ein Backup braucht. Mindestens 32 Zeichen: Die Ableitung läuft
> per **scrypt** (speicherhart, gesalzen) — das erschwert den Offline-Rateangriff auf eine
> entwendete Backup-Datei erheblich, ersetzt aber keine Passphrase-Entropie; eine kurze
> bleibt kurz.
> Und: Diesen Schlüssel **aufbewahren**, außerhalb des Servers. Ohne ihn ist kein
> verschlüsseltes Backup wiederherstellbar.

Kontrolle, bevor der Stack startet:

```bash
./scripts/pruefe_secrets.sh          # alles grün?
grep -c '^JWT_SECRET=' .env          # genau 1
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

Automatischer Backup-Cronjob täglich um **02:30 UTC** (konfigurierbar in `jobs/cron.go`) —
in Deutschland also 03:30 Winter- bzw. 04:30 Sommerzeit:

```
pg_dump → gzip → AES-GCM-Verschlüsselung (Zufalls-Nonce) → 0600 auf Disk
```

Backup-Rotation: älteste Dateien werden nach Ablauf des Aufbewahrungsfensters gelöscht.

Manuelles Backup:
```bash
./scripts/backup.sh
```

Auch die beiden **Shell-Wege** verschlüsseln seit dem 23.08.2026 (Befund A5):
`scripts/backup.sh` schreibt `.sql.gz.enc` direkt aus der Pipe — der Klartext berührt die
Platte nie —, und `./update.sh` verschlüsselt seine Vorab-Sicherung in Schritt 5, sobald
der neue Container gesund und aus dem richtigen Commit gebaut ist. Beide nutzen
`cmd/encrypt-backup` im Backend-Container; der Schlüssel verlässt den Container nicht.

**Klartext gibt es nur noch als Ausnahme** — bei fehlgeschlagenem Deploy (dann ist die
Datei der Rollback-Weg) oder wenn die Verschlüsselung nicht möglich war (Container aus,
`BACKUP_ENCRYPTION_KEY` nicht gesetzt). Solche Dateien werden nach **2 Tagen** gelöscht,
und beide Skripte melden bei jedem Lauf, wie viele davon noch liegen. Solange eine da ist,
gehört `backups/` zum schutzbedürftigen Bestand — sie enthält jeden Schülernamen, jede
Adresse und jede Ausleihe im Klartext. Details:
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

### 7.0 Externes Uptime-Signal (offen — 5-Minuten-Handgriff)

Alle Wächter unten laufen AUF dem Server: Fällt er ganz aus (Strom, Netz, Hetzner),
meldet es niemand — die Nutzer merken es zuerst. Abhilfe ist kein Code, sondern ein
externer Monitor (z. B. UptimeRobot, kostenloser Plan), der alle 5 Minuten
`https://flasch3.herzog-dupont.de/health` abruft und bei Ausfall eine Mail schickt.
Der Endpunkt ist öffentlich und gibt keine Interna preis (`{"status":"healthy"}`).


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

### Gesund heißt nicht aktuell (Schritt 4b)

Beide Prüfungen oben beantworten die Frage „läuft etwas?" — nicht die Frage „läuft der
**neue** Stand?". Am 11.08.2026 stand auf dem Server `git pull` durch, `./update.sh`
aber nicht: Das Arbeitsverzeichnis zeigte den neuen Commit, das laufende Image war zehn
Stunden alt und lieferte den Stand von vorgestern aus. Die Gesundheitsprüfungen meldeten
dabei völlig zu Recht „gesund". Aufgefallen ist es nur, weil jemand von Hand den Namen
der ausgelieferten Bundle-Datei verglichen hat.

Seither legt `update.sh` den Commit als Build-Argument ins Image (`Dockerfile`:
`ARG`/`ENV GIT_COMMIT`) und gleicht ihn in **Schritt 4b** gegen das Arbeitsverzeichnis
ab:

```bash
docker exec bibliothek-backend printenv GIT_COMMIT   # muss git rev-parse HEAD entsprechen
```

Weicht er ab, bricht der Deploy mit Exit-Code 1 ab und nennt beide Stände; der Ausweg
steht in der Meldung (`docker compose build --no-cache && docker compose up -d`). Trägt
das Image gar keinen Commit, gibt es nur eine Warnung — das ist beim ersten Deploy nach
dieser Neuerung der Normalfall, ab dem zweiten ist der Abgleich scharf. Lokal macht
`./scripts/stack-neu.sh` dasselbe (siehe [SCRIPTS.md](SCRIPTS.md) §4).

### Neustart nach Host-Neustart

Beide Dienste in `docker-compose.yml` tragen `restart: unless-stopped`. Ohne das bleibt
die Anwendung nach einem Neustart des Hosts einfach aus — `unattended-upgrades` hat
genau das auf diesem Server schon einmal ausgelöst, und gemerkt hat es niemand, weil
kein Fehler auftrat: Es lief nur nichts mehr.

Optional: Sentry-Integration für Error-Tracking via `SENTRY_DSN`.

---

## 8. Release & Image (ghcr.io)

Zwei Dinge laufen automatisch, ein drittes bewusst nicht:

| Ereignis | Was passiert | Workflow |
|---|---|---|
| Push auf `main` | Image `ghcr.io/uuuxy/bibliothek:latest` + `main-<kurz-sha>` | `docker-publish.yml` |
| Git-Tag `v1.2.3` | Image `1.2.3`, `1.2`, `latest` **und** GitHub-Release mit generierten Notes | `docker-publish.yml` + `release.yml` |
| Deploy auf dem Server | **nicht** automatisch — `./update.sh` (git pull + lokaler Build), siehe §2 | — |

**Release machen** (auf grünem `main`, vom Repo-Root):

```bash
TAG=v1.2.3
git tag -a "$TAG" -m "$TAG" && git push origin "$TAG"
```

Nur das Muster `v<major>.<minor>.<patch>` (mit führendem „v") löst beides aus — derselbe
Filter in beiden Workflows, damit Release und Image nie auseinanderlaufen. Das alte
Release „1.0" (ohne „v", von Hand angelegt am 21.08.2026) zeigt fest auf `dd8a31c6`
und hat kein versioniertes Image; es bleibt als Historie stehen.

`latest` heißt „neuester main", nicht „neuestes Release" — seit 22.08.2026 auch technisch:
der Tag-Lauf setzt `latest` nicht mehr (`flavor: latest=false`). Wer einen festen Stand
will, nimmt den Versions-Tag. Beide Tag-Workflows prüfen vorher: striktes Muster
`v<major>.<minor>.<patch>`, Commit liegt auf `main`, CI des Commits ist grün — ein Tag auf
einem Feature-Branch oder rotem Stand erzeugt weder Release noch Image.

**Deploy-Weg bleibt `git pull` + lokaler Build** (`update.sh`); die ghcr-Images sind
Nachweis und Artefakt, kein zweiter Deploy-Pfad. Rollback = Backup zurückspielen
(Abschnitt 6); Migrationen laufen nur vorwärts.
