# Bibliothek – Schulbibliotheks-Software

Eine webbasierte Verwaltungssoftware für Schulbibliotheken. Das System unterstützt die Abwicklung von Buch- und Hardware-Ausleihen mittels eines integrierten Barcode-Scanner-Konzepts.

---

## Tech-Stack

| Komponente | Technologie |
|---|---|
| Backend | Go 1.27.0 (go.mod = Dockerfile), `net/http`, `pgx/v5` |
| Frontend | Svelte 5 (Runes), Tailwind CSS, Vite |
| Datenbank | PostgreSQL 18 |
| Echtzeit | Server-Sent Events (SSE) |
| Deployment | Docker Compose, Caddy (Reverse Proxy) |

---

## Hauptfunktionen

- **Zentrale Omnibox (Scanner-Dispatcher):** Ein Eingabefeld verarbeitet alle Barcode-Scans. Ohne Präfix wird in der Reihenfolge Buch → Schülerausweis → Lehrerausweis → Volltextsuche aufgelöst — die Ausweise aus dem Littera-Altbestand tragen nackte Nummern und dürfen nicht neu gedruckt werden. Die Präfixe `S-`, `L-`, `B-`, `G-` sind eine Abkürzung, keine Voraussetzung.
- **Fristenberechnung:** Berücksichtigung von LMF-Büchern (Stichtag 31. Juli), Sonderbeständen (CDs, DVDs, Hörbücher) und Ferien-Leseclub.
- **Audit-Trail:** Ereignisprotokollierung für administrative Aktionen (append-only als Konvention; die DSGVO-Tilgung ist die bewusste Ausnahme).
- **Datenschutz-Funktionen:** Löschroutinen für Schulabgänger, AES-256-Verschlüsselung für Schülerfotos.
- **LUSD-Schnittstelle:** Import von Schülerdaten aus dem LUSD-System.
- **Littera-Altbestandsübernahme:** Titel, Exemplare, Personen und offene Ausleihen aus der Vorgängersoftware — mit Savepoint je Datensatz und Abgleich gegen den tatsächlichen Zeilenzuwachs (`cmd/littera-altbestand`).
- **Hardware-Verwaltung:** Ausleihe von Laptops/Tablets inklusive Zubehör-Checklisten.
- **Druck-Center:** Erstellung von Barcode-Etiketten und Schülerausweisen.
- **Bestellwesen:** Bedarfsvorschläge aus dem Bestand, Bestellmail an den Händler samt Barcodebogen, Wareneingang — und für Händler, die selbst etikettieren, ein Bestätigungs-Link, über den der Lieferant seine Etiketten druckt und die Bestellung selbst bestätigt.
- **Inventur:** Session-gebundene Bestandsaufnahme mit Scanner, Fehlbestandsliste und Aufarbeitung.
- **Rollenbasierte Zugriffskontrolle (RBAC):** Rollen für Admin, Kollegium (nur Klassensatz-Reservierung im eigenen Portal), Mitarbeiter (Tresen-Betrieb) und Helfer (Kiosk-Betrieb ohne Schülerrechte). Angemeldet wird gegen den Schul-Mailserver per IMAP — die Anwendung speichert kein Benutzerpasswort.
- **Öffentliche Seiten:** Katalog `/katalog` (Suche mit Cover und Verfügbarkeit) und Bibliotheks-Monitor `/monitor` (Slideshow für den Flur) — beide ohne Anmeldung, beide ohne Personendaten ([FACHKONZEPT.md §16](FACHKONZEPT.md)).
- **Kollegiums-Portal & Selbstanmeldung:** Lehrkräfte melden sich mit dem Schul-Postfach an, reservieren Klassensätze und geben Wünsche/Meldungen ab; die Bibliothek schaltet Zugangsanfragen frei.
- **Selbstprüfung der Betriebsbereitschaft:** Eine Seite unter *System*, die eine einzige Frage beantwortet — was ist eingerichtet, aber nicht in Betrieb? Sie fängt die wiederkehrende Fehlerart ab, bei der eine fertige Funktion still nichts tut, weil eine Einstellung fehlt (Details: [FACHKONZEPT.md §15](FACHKONZEPT.md)).

---

## Dokumentation

| Dokument | Inhalt |
|---|---|
| [HANDBUCH.md](HANDBUCH.md) | Benutzerhandbuch — jeder Bereich aus Sicht der Bibliothek, mit den öffentlichen Seiten und den 13 Einstellungs-Kategorien |
| [FACHKONZEPT.md](FACHKONZEPT.md) | Vollständige fachliche Feature-Spezifikation (Ausleihregeln, Mahnwesen, Vormerkungen, DSGVO, RBAC, Katalog …) |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Schichtenarchitektur, Concurrency-Modell, Datenbankdesign, Frontend |
| [SECURITY.md](SECURITY.md) | Sicherheitskonzept, DSGVO, Schutzmaßnahmen |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Produktions-Deployment, Umgebungsvariablen, Caddy, Backups |
| [SCRIPTS.md](SCRIPTS.md) | CLI-Werkzeuge: Littera-Altbestand, Foto-Migration, Backup, Deployment, Lasttest |
| [invarianten.md](invarianten.md) | Invarianten-Katalog: was immer gelten muss und auf welcher Ebene es durchgesetzt ist |
| [befunde.md](befunde.md) | Befund-Register: was aufgefallen ist, was davon erledigt wurde |
| [resilience_and_recovery.md](resilience_and_recovery.md) | Backup (verschlüsselt + manuell), Restore-Probe, Notfall-Wiederherstellung, Cronjob-Einrichtung |
| [api_inventar.md](api_inventar.md) | **Vollständiges** Routenverzeichnis (generiert): alle Go-Routen, alle Frontend-Aufrufer, Abgleich in beide Richtungen — `./scripts/api_inventar.sh` |
| `docs.go` (Swagger) | Interaktive API-Doku, **nur bei `APP_ENV=local`/`development`** unter `/swagger`. Deckt die **annotierten** Endpunkte ab (aktuell 49 Operationen auf 43 Pfaden von 168 registrierten Routen) — das vollständige Verzeichnis ist `api_inventar.md`. Neu erzeugen: `swag init -g main.go -o docs`; ein Test (`docs/swagger_drift_test.go`) schlägt fehl, sobald die Datei von den `@Router`-Annotationen abweicht |
| [datenschutz_offene_punkte.md](datenschutz_offene_punkte.md) | Datenschutz-Arbeitsliste gegen den hessischen Rahmen (SchDSV, HBDI-Muster, HKM-LMF): Code-Punkte, Schul-/DSB-Punkte, bewusste Nicht-Befunde |
| [datenschutz/vvt_entwurf.md](datenschutz/vvt_entwurf.md) | Entwurf Verzeichnis von Verarbeitungstätigkeiten (HBDI-Muster) — zwei Tätigkeiten: Lernmittelausleihe, Schülerbücherei; TOM-Anhang aus SECURITY.md |
| [datenschutz/datenschutzhinweis_art13.md](datenschutz/datenschutzhinweis_art13.md) | Entwurf Datenschutzhinweis nach Art. 13 DSGVO für Schüler/Eltern — zwei Fassungen (Lernmittel, Schülerbücherei) |
| [abnahme_checkliste.md](abnahme_checkliste.md) | Durchlauf für die manuellen Abnahmen (LUSD, Versetzung, Klassensatz) |
| [littera_schema_befund.md](littera_schema_befund.md) | Littera-Altbestand: Schema, Barcodes, Schreibpfad — alle Zahlen gemessen |

> Eine Änderungshistorie gibt es bewusst nicht als Datei — `git log` ist ausführlicher und
> kann nicht veralten.

---

## Schnellstart (lokal)

### Voraussetzungen
- Go 1.27.0 (wie go.mod/Dockerfile)
- Node.js (npm)
- PostgreSQL (lokal oder via Docker)

> **PG-Integrationstests auf macOS:** `pg_dump`/`psql` 18 liegen in
> `/opt/homebrew/opt/libpq/bin` und müssen VOR den PATH (`brew`-Standard ist 16) —
> ein älterer Client verweigert den neueren Server, und die zwei
> Backup-Proben-Tests scheitern dann scheinbar grundlos (so passiert am 31.08.2026).

### Mit Docker
```bash
docker compose -f docker-compose.local.yml up -d
```
Backend: `http://localhost:8084` · DB: `localhost:5434`

### Manuell

**1. Umgebungsvariablen**
```bash
cp .env.example .env
# DATABASE_URL, JWT_SECRET (≥32 Zeichen), APP_ENCRYPTION_KEY (32 Bytes) anpassen
```

**2. Backend starten**
```bash
go run main.go
# Führt Datenbank-Migrationen automatisch aus
```

**3. Frontend starten**
```bash
cd frontend
npm ci                    # nicht `npm install` — der Lockfile ist verbindlich
npm run dev
# → http://localhost:5173
```

> **Auf den Port achten.** Der Entwicklungs-Server reicht `/api`, `/login`, `/uploads` und
> `/events` an `127.0.0.1:8084` durch — das ist der **lokale Docker-Stack**. Startest du
> das Backend wie oben von Hand, gilt der `PORT` aus deiner `.env` (Vorgabe der
> Beispieldatei: 8081), und du musst das Ziel mitgeben:
>
> ```bash
> VITE_API_TARGET=http://127.0.0.1:8081 npm run dev
> ```
>
> Bis zum 12.08.2026 war das Ziel fest auf 8083 verdrahtet — den Port des
> **Produktions**-Stacks. Wer dieser Anleitung folgte, bekam eine Oberfläche, deren
> API-Aufrufe alle ins Leere liefen, ohne dass irgendetwas auf den Port hinwies.

---

## Systemarchitektur (Kurzübersicht)

```
Globale Kette (api/router.go): Security-Header → CORS → Body-Limit → Timeout
                               → Rate-Limit → CSRF
        │
        ▼
Pro Route: RequirePermission(...) / RequireRoles(...)   ← Auth + RBAC sitzen hier,
        │                                                 nicht in der globalen Kette
        ▼
Handler (api/) → Service (internal/service/) → Repository (repository/)
        │                                               │
        ▼                                               ▼
SSE Broker (Echtzeit)                         PostgreSQL (pgx/v5)
```

Details: [ARCHITECTURE.md](ARCHITECTURE.md)

---

## Sicherheit

- JWT HMAC-only (kein `alg=none`)
- Brute-Force-Schutz: `email|ip`-Composite-Key
- CSRF: Double-Submit Cookie
- AES-256-GCM für Schülerfotos
- SMTP: STARTTLS erzwungen, mit Zertifikatsprüfung
- CSV-Formel-Injection-Schutz (OWASP CWE-1236)
- Decompression-Bomb-Guard bei Bild-Uploads
- Produktions-Secret-Guard: **muss scharf geschaltet werden** — mit `ENFORCE_PROD_SECRETS=true`
  verweigert der Server den Start bei bekannten Default-Secrets. Der Standard ist `false`
  (Testphase), damit der Stack ohne Konfiguration hochkommt; vor dem echten Prod-Deploy
  gehört der Schalter gesetzt ([DEPLOYMENT.md](DEPLOYMENT.md#22-secret-guard-per-schalter-einschaltbar))

Details: [SECURITY.md](SECURITY.md)
