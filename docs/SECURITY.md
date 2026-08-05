# Sicherheits- und Datenschutzkonzept (DSGVO)

Diese Dokumentation beschreibt die systemweiten Mechanismen zur Wahrung von Sicherheit und Datenschutz der Bibliotheks-Verwaltungssoftware.

> Zuletzt aktualisiert: 2026-06-24 (Session-Audit: alle 46 Dateien tief gescannt)

---

## 🛡️ Authentifizierung & Session-Management

### JWT (JSON Web Tokens)
- **Algorithmus-Pinning:** Der Server akzeptiert ausschließlich HMAC-signierte Tokens (HS256). Die `alg=none`-Schwachstelle (CVE-Klasse) ist damit verhindert — ein Token ohne Signatur wird abgelehnt.
- **Blacklist (fail-closed):** Abgemeldete Tokens werden in einer Datenbank-Blacklist registriert. Ist die Blacklist-Abfrage nicht erreichbar (DB-Fehler), wird der Request abgelehnt (HTTP 500), nicht durchgelassen. „Fail-Open"-Verhalten ist ausgeschlossen.
- **Lebensdauer:** 12 Stunden; danach ist eine erneute Anmeldung erforderlich.
- **Cookie-Attribute:** `HttpOnly` (kein JS-Zugriff), `SameSite=Lax`, in Produktion zusätzlich `Secure` (via `COOKIE_SECURE=true`).

### Brute-Force-Schutz (Login)
- **Schlüssel:** `lower(email)|ip` — sperrt ein Konto für eine IP-Adresse (5 Fehlversuche / 15 min).
- **Warum nicht nur IP?** An einer Schulnetzwerk-NAT sind alle Geräte hinter einer IP. Würde nur die IP gesperrt, würde ein einziger Fehlversuch die gesamte Schule aussperren. Der Composite-Key (`email|ip`) isoliert das betroffene Konto auf dieser IP und schützt trotzdem gegen gezielte Account-Angriffe.
- **Globaler Rate-Limiter:** Zusätzlich 50 Requests/s/IP über alle Endpunkte (Map+Mutex, kein externer Cache nötig).

---

## 🔒 Autorisierung (RBAC)

### RequirePermission-Middleware
- Alle schützenswerten Endpunkte sind über `RequirePermission` bzw. `RequireRoles` abgesichert.
- **Keine transiente 403-Cacheung:** Ist die Datenbank bei der Berechtigungsprüfung nicht erreichbar (Netzwerkfehler, Timeout), wird HTTP 500 zurückgegeben und **nicht** in den Permission-Cache geschrieben. Ein vorübergehender DB-Ausfall führt also nicht dazu, dass legitime Benutzer für 60 Sekunden ausgesperrt bleiben.
- **Stabile Verweigerung:** Nur `pgx.ErrNoRows` (Berechtigung definitiv nicht vorhanden) wird gecacht und als 403 gewertet.

### Rollenkonzept
- `admin`: Vollzugriff (`["*"]`). Berechtigungen werden beim Login direkt aus `role_permissions` geladen.
- `lehrer`: Granulare Rechte — jede Berechtigung muss explizit durch einen Admin freigeschaltet werden.
- `mitarbeiter`: Grundrechte für den Tresen-Betrieb.
- `helfer`: Kiosk-/Tresenbetrieb ohne die breiten Schülerrechte (Migration 042) — Scannen, Ausleihe, Rückgabe, Katalogzugriff, aber keine Schülerlisten und kein Mahnwesen.
- Alle Enum-Werte in der Datenbank sind **lowercase** (`admin`, `lehrer`, `mitarbeiter`, `helfer`). SQL-Vergleiche nutzen `LOWER(rolle::text)` um Casing-Fehler zu vermeiden (Bugfix: `LEHRER`-Enum führte zu HTTP 500 in der Omnibox).

### Endpunkte ohne Anmeldung
Ein Test erzwingt die Vollständigkeit dieser Liste: `TestAlleRoutenSindGeschuetzt`
(`api/routes_authz_coverage_test.go`) lässt jede registrierte Route ohne
`RequirePermission`/`RequireRoles` fehlschlagen, solange sie nicht mit Begründung auf der
Allowlist steht. Eine ungeschützte Route kann also nicht unbemerkt live gehen.

Bewusst öffentlich sind:

| Endpunkt | Warum unbedenklich |
|---|---|
| `GET /api/public/opac/suche` | Katalogsuche: Titel/Autor/Verfügbarkeit, keine personenbezogenen Daten. LMF-Schulbücher sind ausgefiltert. |
| `GET /api/monitor/slides` | Bibliotheks-Monitor, nur Buchdaten. |
| `GET /api/images/cover`, `/uploads/` | Cover-Bilder (SSRF-Host-Allowlist). Schülerfotos liegen **nicht** hier, sondern AES-verschlüsselt in der Datenbank. |
| `GET /api/csrf-token`, `/api/auth/*`, `POST /login` | Bootstrap bzw. selbst-authentifizierend. |
| `GET /api/public/bestellung/{token}` + `/etiketten/{groesse}` + `POST …/bestaetigen` | Bestätigungs-Link an den Lieferanten — siehe unten. |

**Der Bestätigungs-Link ist der einzige schreibende Zugang ohne Anmeldung.** Sein
Zuschnitt begrenzt den Schaden:

- **Der Token ist der Ausweis:** 32 Byte aus `crypto/rand` (256 Bit), Base64-URL. Raten ist
  ausgeschlossen; der globale Rate-Limiter steht zusätzlich davor.
- **In der Datenbank nur der SHA-256** (`bestellungen_verlauf.bestaetigungs_token_hash`,
  Migration 063). Ein DB-Auszug oder ein entwendetes Backup enthält keine benutzbaren
  Links. Kein langsamer KDF nötig — die Eingabe ist bereits 256 Bit Zufall, es gibt nichts
  zu erraten.
- **Kein Passwort davor, bewusst:** Der Lieferant müsste sonst ein Geheimnis verwalten;
  das würde den Ablauf entwerten, den der Link erst möglich macht.
- **Reichweite:** genau eine Bestellung. Sichtbar sind Lieferant, Datum, Kundennummer und
  Titelzeilen — dieselben Angaben, die der Lieferant ohnehin als Mailanhang hat. Keine
  Schülerdaten, keine Preise, kein Zugriff auf den Bestand.
- **Einmalig und atomar:** Bestätigen läuft über `WHERE bestaetigt_am IS NULL`; der zweite
  Klick bekommt 409 statt eines stillen Überschreibens.
- **Ablauf und Rückruf:** 180 Tage gültig; ein neu erzeugter Link entwertet den alten sofort.
- **Ungültig ist immer 404** — abgelaufen, zurückgezogen und nie existiert sehen von außen
  gleich aus, sonst verriete die Antwort, dass ein geratener Token einmal echt war.
- **Nicht in Logfiles:** Der Token steht im Pfad, also maskiert `maskiereToken`
  (`api/middleware.go`) ihn in beiden Logzeilen — Request-Log und 500er-Stacktrace. Ohne
  das hätte ein weitergereichtes Logfile funktionierende Links enthalten und die
  Hash-Speicherung entwertet (gefunden im Smoke-Test am laufenden Server, 05.08.2026).
- **CSRF gilt weiter:** Die Seite nutzt den normalen Double-Submit-Weg; ein POST ohne
  `X-CSRF-Token` wird mit 403 abgewiesen (nachgewiesen am laufenden Server).

---

## 🛡️ Schutz vor Injection-Angriffen

### SQL-Injection
- Alle Datenbankinteraktionen erfolgen ausschließlich über parametrisierte Queries (`$1`, `$2`, …) mit `jackc/pgx/v5`. String-Konkatenation in SQL-Statements existiert nicht.

### CSV-Formel-Injection (CWE-1236)
- **Angriffsvektor:** Buchtitel oder Autornamen, die mit `=`, `+`, `-`, `@`, `\t`, `\r`, `\n` beginnen, können in CSV-Dateien als Formeln interpretiert werden (Excel/LibreOffice führt diese bei Öffnen aus).
- **Schutz:** `pkg/csvutil.SanitizeRow()` setzt einen Apostroph-Präfix vor alle Zellen, die mit einem dieser Zeichen beginnen (OWASP-Empfehlung). Wird bei allen CSV-Exporten verwendet (`inventur/export_csv.go`, Bestellexporte).

### XSS (Cross-Site Scripting)
- Svelte 5 escaped alle Template-Variablen automatisch.
- `{@html}` wird im gesamten Frontend nicht eingesetzt.
- SVG-Icons sind hartcodierte Konstanten, keine benutzerkontrollierten Werte.

---

## 📁 Datei-Uploads

### Foto-Uploads (Schülerfotos)
- **Decompression-Bomb-Schutz:** `pkg/imageutil.GuardImageDimensions()` liest per `image.DecodeConfig` nur den Bild-Header (ohne volle Dekodierung). Bilder über 50 Megapixel werden abgelehnt, bevor `image.Decode` die vollständige Pixelmatrix allokiert (Schutz gegen RAM-Erschöpfung durch präparierte Bilder).
- **MIME-Prüfung:** Über echte Dekodierung, nicht nur Dateiendung.
- **Verschlüsselung:** Fotos werden AES-256-GCM-verschlüsselt als `BYTEA` in der Datenbank gespeichert — kein Klarpfad auf dem Dateisystem.
- **Path-Traversal:** Alle Pfadoperationen nutzen `filepath.Base` + `filepath.Clean` + Prefix-Guard.

### Cover-Uploads
- 10 MB Body-Limit, 0600 Dateiberechtigungen.
- Ebenfalls `GuardImageDimensions` vor dem vollständigen Decode.

---

## 📧 E-Mail-Sicherheit (SMTP/IMAP)

### SMTP STARTTLS
- **Zertifikatsprüfung aktiv:** `ServerName` wird gesetzt, `MinVersion: TLS 1.2` erzwungen. `InsecureSkipVerify` war zuvor auf `true` gesetzt — ein MITM-Angreifer konnte dadurch SMTP-Credentials und den gesamten E-Mail-Inhalt (inkl. Personendaten für Mahnwesen) mitlesen. **Behoben.**
- **Opt-out für interne/Legacy-Server:** Umgebungsvariable `SMTP_ALLOW_INSECURE_TLS=true` erlaubt das Abschalten der Zertifikatsprüfung bei Bedarf (mit expliziter Warnung im Log).
- **Header-Injection:** Attachment-Dateinamen werden gegen CRLF-Injection bereinigt.

### IMAP
- Implizites TLS (Port 993), `MinVersion: TLS 1.2`, ServerName-Verifikation, Timeouts.

---

## 🔏 CSRF-Schutz

- **Methode:** Double-Submit Cookie mit Constant-Time-Vergleich.
- **Achtung (behoben):** `sync-covers` und `import-bestand` sind global registrierte Endpunkte unter `/api/admin/…`. Durch eine zu breite Ausnahme-Regel für `/api/admin/*` waren diese temporär ohne CSRF-Schutz. Die Ausnahme wurde entfernt — beide Endpunkte durchlaufen jetzt die globale CSRF-Prüfung. Das Frontend sendet den Token bereits korrekt (keine Frontend-Änderung nötig).

---

## 🐳 Produktions-Absicherung (Secret Guard)

### Problem
Wenn `JWT_SECRET` oder `APP_ENCRYPTION_KEY` die committeten Entwicklungs-Defaults verwenden, kann jeder mit Repo-Zugriff Admin-JWTs fälschen (vollständige Übernahme) oder AES-verschlüsselte Schülerfotos entschlüsseln.

### Lösung (`main.go/loadConfig`)
Der Server **verweigert den Start**, wenn der Schalter `ENFORCE_PROD_SECRETS=true` gesetzt ist und bekannte Default-Secrets erkannt werden:
```go
enforceProdSecrets := strings.ToLower(os.Getenv("ENFORCE_PROD_SECRETS")) == "true"
if enforceProdSecrets {
    knownDefaultSecrets := map[string]bool{
        "super-secret-default-key-at-least-32-bytes": true,
        "super-secure-aes-key-32-chars-ok":           true,
        "supergeheim_lokal":                          true,
    }
    // … log.Fatalf bei Treffer
}
```

**Bewusst per Schalter einschaltbar (entkoppelt von `APP_ENV`):**
- Test-/Pilotphase: `ENFORCE_PROD_SECRETS=false` (Standard) → Stack startet auch mit Defaults.
- Echter Prod-Deploy: `ENFORCE_PROD_SECRETS=true` → harte Start-Verweigerung bei Default-Secrets.

Die Entkopplung von `APP_ENV` ist Absicht: `APP_ENV=local` würde sonst gleichzeitig das Cookie-`Secure`-Flag deaktivieren und Swagger öffentlich freischalten. So bleibt `APP_ENV=production` (sichere Cookies, kein Swagger), während die Secret-Härtung separat geschaltet wird.

### Mindestanforderungen
- `JWT_SECRET`: ≥ 32 Zeichen
- `APP_ENCRYPTION_KEY`: genau 32 Bytes (oder 64 Hex-Zeichen)
- In Produktion: `docker-compose.yml` erzwingt per `${VAR:?Fehlermeldung}`, dass alle Secrets gesetzt sind

---

## 🔒 Datenschutz und DSGVO-Konformität

### Automatisierte Löschroutinen
Die Applikation führt automatisierte Cronjobs (`jobs/cron.go`) durch:

- **Ausleihen-Anonymisierung (`RunGDPRAnonymizeLoans`):** Entfernt `bearbeiter_id` von Ausleihen, die vor mehr als 14 Tagen zurückgegeben wurden.
- **Abgänger-Löschung (`RunGDPRDeleteAbgaenger`):** Hard-Delete von Schülerdatensätzen (`ist_abgaenger = true`) nach Karenzzeit (30 Tage im neuen Schuljahr), sofern keine offenen Ausleihen oder unbezahlten Schadensfälle bestehen. Historische Ausleihdaten werden anonymisiert (`schueler_id = NULL`).

### Datenverschlüsselung
- Schülerfotos: AES-256-GCM-verschlüsselt als `BYTEA` in der Datenbank. Kein Klartext auf dem Dateisystem.
- DB-Backups: `pg_dump → gzip → AES-GCM` (Zufalls-Nonce), 0600 Dateiberechtigungen, Rotation.

### Adressdaten (DSGVO vs. Mahnwesen)
Adressspalten (`strasse`, `plz`, `ort`) und `eltern_email` werden für das Mahnwesen (Briefversand für Schadens-Rechnungen und E-Mail für Mahnungen) benötigt und sind **bewusst vorhanden**. Migration 003 enthielt ursprünglich einen `RAISE EXCEPTION`-Wächter, der Adressspalten blockiert hätte — dieser wurde entfernt, da die Daten fachlich essenziell sind.

**Dokumentation für das Verzeichnis von Verarbeitungstätigkeiten (VVT):**
- **Rechtsgrundlage:** Art. 6 Abs. 1 lit. c DSGVO (Erfüllung einer rechtlichen Verpflichtung, z.B. Schulgesetz/Lernmittelfreiheit) in Verbindung mit Art. 6 Abs. 1 lit. b DSGVO (Vertragserfüllung bzgl. Ausleihe) und Art. 5 Abs. 1 lit. c DSGVO (Zweckbindung & Datensparsamkeit).
- **Zweck:** Ausschließlich für den Versand von Schadens-Rechnungen (Anschrift) und Eltern-Mahnungen (E-Mail).
- **Aufbewahrungsfrist/Löschung:** Beim Abgang eines Schülers (ohne offene Vorgänge wie Ausleihen oder unbezahlte Rechnungen) werden diese Felder durch die Anonymisierungsroutine (`anonymisiereAbgaenger`) umgehend geleert.

---

## 🛡️ Netzwerksicherheit & Security-Header

Restriktive HTTP-Header in `api/middleware.go`:
- `frame-ancestors 'none'` — verhindert Clickjacking via iFrame
- `form-action 'self'` — Formulare nur an eigene API
- `script-src 'self'` — kein externes Script-Loading
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`

---

## 📋 Audit-Trail

- Alle administrativen Aktionen und Buchbewegungen werden in `audit_logs` protokolliert (Append-Only).
- Auditierung erfolgt **nach** dem Transaktions-Commit (kein Rollback-Risiko).
