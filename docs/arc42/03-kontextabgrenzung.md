# 3. Kontextabgrenzung

Stand: 17.09.2026

---

## 3.1 Fachlicher Kontext

```mermaid
graph TB
    subgraph Schule
        MA[Bibliotheks-<br/>Mitarbeiter]
        LTG[Bibliotheks-<br/>leitung]
        HELF[Helfer]
        KOLL[Kollegium<br/>~160 Lehrkräfte]
        SEK[Sekretariat]
    end
    subgraph Öffentlich
        SCH[Schüler & Eltern]
        MON[Flur-Monitor]
    end
    subgraph Extern
        HAENDL[Buchhändler]
        MZ[Schulträger]
        DSB[Datenschutz-<br/>beauftragte]
    end

    SYS[("Bibliothek<br/>(dieses System)")]

    MA -->|Scan, Ausleihe, Rückgabe, Mahnwesen, Inventur| SYS
    LTG -->|Bestellung, Statistik, Bescheide| SYS
    HELF -->|Ausleihe/Rückgabe, Katalog lesen| SYS
    KOLL -->|Klassensatz-Reservierung, LMF-Plan, Anliegen| SYS
    SEK -->|LUSD-Bericht (Schülerdaten)| SYS
    SCH -->|Katalog /katalog, ohne Anmeldung| SYS
    SYS -->|Slideshow /monitor| MON
    SYS -->|Bestellmail + Barcodebogen| HAENDL
    HAENDL -->|Bestätigungslink /bestellung/token| SYS
    SYS -->|Zugangs-/Abgangsbuch, Bestandsnachweis| MZ
    SYS -->|Auskunft, VVT, Löschnachweis| DSB
    SYS -->|Mahnliste an Klassenleitung, Abgängerbrief, Bescheid| KOLL
```

### Fachliche Nachbarn im Einzelnen

| Nachbar                     | Eingang in das System                                                                                       | Ausgang aus dem System                                                                                                | Besonderheit                                                                                                    |
| --------------------------- | ----------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| **Bibliothekspersonal**     | Barcode-Scans, Stammdatenpflege, Mahnlauf, Inventurzählung, Wareneingang                                     | Bildschirm-Rückmeldung je Scan, PDFs (Mahnungen, Etiketten, Ausweise, Bescheide, Listen)                              | Bedienung ist auf **Tastatur und Scanner** ausgelegt; die Maus ist optional                                     |
| **Kollegium**               | Selbstanmeldung mit dem Schulpostfach, Reservierungen, Buchwünsche, Meldungen                                 | Portal-Ansichten (Klassensätze, LMF-Plan, Schulbücher als PDF), Terminbestätigungen                                    | Zugang entsteht **inaktiv** und wird von der Bibliothek freigeschaltet                                          |
| **Sekretariat**             | LUSD-Bericht als `.xlsx` **oder** Semikolon-CSV (LANIS-Klassenliste, UTF-8 mit BOM)                           | Abgleichbericht: neu, geändert, Umbenennung, Abgänger                                                                  | Die Kopfzeile wird **gesucht, nicht vorausgesetzt** — LUSD-Berichte tragen Titelzeilen darüber                  |
| **Schüler und Eltern**      | Suchanfragen im öffentlichen Katalog                                                                         | Titel, Cover, Verfügbarkeit                                                                                            | **Nie** Ausleiherdaten — PII-Stufe 0, im Gate `api/pii_matrix_test.go` festgehalten                              |
| **Buchhändler**             | Bestätigung über einen Token-Link, ohne Konto                                                                 | Bestellmail mit Positionsliste und Barcodebogen                                                                        | Der Token steht **im Pfad** und wird im Log maskiert (`maskiereToken`); gespeichert wird nur sein Hash           |
| **Schulträger**             | Anforderungsprotokoll, Nachweispflichten                                                                      | Zugangs-/Abgangsbuch je Halbjahr, getrennt nach Land und Träger; Bestandsnachweis zum Stichtag (15.3./15.9.)           | Bücher ohne hinterlegte Bestellung erscheinen ausdrücklich „ohne Zuordnung" — die ehrliche Lücke statt einer Erfindung |
| **Datenschutzbeauftragte**  | Prüffragen                                                                                                    | DSGVO-Auskunft als PDF, PII-Matrix, VVT-Entwurf, Nachweis der Löschläufe im Audit-Trail                                | Die Audit-Tilgung ist die **bewusste Ausnahme** von der Append-only-Konvention                                  |

---

## 3.2 Technischer Kontext

```
                   ┌──────────────────────────────────────────┐
  Browser          │  Caddy (Host, ACME/Let's Encrypt)        │
  (SPA, PWA)  ────► │  flasch3.herzog-dupont.de :443          │
                   │  response_header_timeout 300s            │
                   │  read/write_timeout 600s                 │
                   └───────────────┬──────────────────────────┘
                                   │ HTTP, Docker-Netz
                                   ▼
        ┌───────────────────────────────────────────────────────────┐
        │  backend (Go, alpine, non-root »appuser«)                 │
        │  :8083 (Produktion) / :8084 (lokaler Stack)               │
        │  /api  JSON · /events SSE · /uploads Cover · /login       │
        │  /health · /swagger (nur local|development)               │
        └───┬───────────┬────────────┬───────────┬──────────────────┘
            │ pgx/v5    │ IMAP 993   │ SMTP      │ HTTPS (Allowlist)
            ▼           ▼            ▼           ▼
     ┌────────────┐  ┌────────┐  ┌────────┐  ┌──────────────────────┐
     │ PostgreSQL │  │ Schul- │  │ Schul- │  │ DNB · OpenLibrary ·  │
     │ 18-alpine  │  │ Mail-  │  │ SMTP   │  │ Google Books         │
     │ :5432      │  │ server │  │ STARTTLS│ │ (Cover + Metadaten)  │
     └────────────┘  └────────┘  └────────┘  └──────────────────────┘
            │                                        ▲
            │ pg_dump → gzip → AES-256-GCM           │ nur öffentliche IPs
            ▼                                        │ (pkg/safehttp)
     ┌────────────────────┐   optional   ┌────────────────────────┐
     │ Volume /app/backups│ ───────────► │ S3-kompatibler Speicher│
     └────────────────────┘              └────────────────────────┘
                                          optional: Sentry (SENTRY_DSN)
```

### Schnittstellen technisch

| Schnittstelle                    | Protokoll / Format                              | Richtung        | Absicherung                                                                                                                       |
| -------------------------------- | ----------------------------------------------- | --------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| **Browser → Backend**            | HTTPS, JSON über `/api`, `POST /login`          | eingehend       | Security-Header/CSP, CORS auf die Schuldomain, Body-Limit 100 MB, Timeouts, Rate-Limit, CSRF (Double-Submit), JWT im HttpOnly-Cookie |
| **Backend → Browser (Echtzeit)** | Server-Sent Events, `GET /events`               | ausgehend       | Nur authentifiziert (`RequireAuthenticated`); Heartbeat alle 15 s; **kein** `WriteTimeout` am Server, weil der Strom nie endet     |
| **Statische Auslieferung**       | SPA aus `frontend/dist`, PWA-Service-Worker      | ausgehend       | Dateizugriffe OS-seitig an `dist/` gebunden (`os.OpenRoot`); ein unbekannter `/api/`-Pfad antwortet 404 statt App-Shell            |
| **Cover-Dateien**                | `GET /uploads/...`                              | ausgehend       | **Bewusst ohne Anmeldung** (Katalog/Monitor brauchen Cover); in der Allowlist von `routes_authz_coverage_test.go` vermerkt          |
| **Backend → PostgreSQL**         | Postgres-Wire über `pgx/v5`-Pool                | ausgehend       | Netz nur intern; Passwort aus `.env`; `pg_dump` erhält es über eine temporäre `.pgpass`, nicht über `PGPASSWORD`                    |
| **Backend → IMAP**               | IMAP über implizites TLS, Port 993, min. TLS 1.2 | ausgehend       | Feste Cipher-Auswahl; `IMAP_HOST=mock` akzeptiert in Entwicklung/Test jedes Passwort und wird beim Start laut protokolliert         |
| **Backend → SMTP**               | SMTP mit **erzwungenem** STARTTLS und Zertifikatsprüfung | ausgehend | Bietet der Server kein STARTTLS, bricht der Versand ab (`ErrSMTPKlartext`); nur ein ausdrückliches `SMTP_ALLOW_PLAINTEXT=true` erlaubt es. Kopfzeilen werden gegen CR/LF-Einschmuggelung geprüft |
| **Backend → Metadaten/Cover**    | HTTPS GET an eine **Host-Allowlist**             | ausgehend       | `pkg/coverquelle` baut die URL aus geprüften Teilen **neu** auf (Schema fest HTTPS, Host aus der Konstante); `pkg/safehttp` lehnt Verbindungen zu nicht-öffentlichen IPs ab (SSRF) |
| **Backend → S3 (optional)**      | S3-API über `minio-go`                           | ausgehend       | Nur aktiv, wenn `S3_ENDPOINT/ACCESS_KEY/SECRET_KEY/BUCKET` vollständig gesetzt sind; sonst überspringt der Job den Offsite-Upload und sagt das |
| **Backend → Sentry (optional)**  | HTTPS                                            | ausgehend       | Nur bei gesetztem `SENTRY_DSN`                                                                                                     |
| **Littera-Altbestand**           | CSV aus `mdb-export` (einmalig, CLI)             | eingehend       | Kein HTTP-Weg: eigenes Kommando `cmd/littera-altbestand` gegen dieselbe Datenbank, Savepoint je Datensatz                            |
| **Bestellbestätigung Händler**   | `GET/POST /bestellung/<token>` ohne Anmeldung     | eingehend       | Token einmalig, Hash gespeichert, im Log maskiert; die Bestätigung ist über `WHERE bestaetigt_am IS NULL` atomar                    |
| **Scanner / Kamera**             | USB-HID (Tastatureingabe) bzw. `html5-qrcode`     | eingehend       | Kein Gerätetreiber im System: Ein Handscanner ist eine Tastatur, die Kamera läuft im Browser                                        |
| **Drucker**                      | PDF-Download, Druck aus dem Browser              | ausgehend       | Etiketten/Ausweise als PDF mit festen Bogenmaßen; Gate `npm run test:druck` prüft die Drucksektionen                               |

### Was ausdrücklich **nicht** angebunden ist

- **Kein Single-Sign-On** (kein LDAP, kein OAuth, kein Kerberos) — IMAP ist die einzige
  Identitätsquelle.
- **Keine Schnittstelle zur Schulverwaltung außer dem LUSD-Bericht als Datei.** Es gibt
  keinen Webservice, keine Datenbankkopplung und kein automatisches Nachladen.
- **Keine Zahlungsschnittstelle.** Forderungen werden über Bescheid und Rechnung
  abgewickelt; das System bucht keine Zahlungseingänge ein und nennt für Landes-Lernmittel
  ausdrücklich das Konto statt Bargeld.
- **Kein Push an Schüler oder Eltern.** Mahnlisten gehen an die Klassenleitung; Abgänger-
  und Schadensbriefe gehen über die Schule.
- **Kein externer Monitoring-Agent.** Das System stellt `/health` und die Selbstprüfung
  bereit und schickt den Bereitschafts-Wächter per Mail; ein externes Uptime-Signal ist als
  Handgriff offen ([OFFEN.md](../OFFEN.md) 7.5).
