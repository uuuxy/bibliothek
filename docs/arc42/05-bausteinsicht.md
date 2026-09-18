# 5. Bausteinsicht

Stand: 18.09.2026 · alle Umfangszahlen gemessen am 17.09.2026
(Befehle im [Anhang](#anhang-die-zahlen-selbst-nachmessen))

---

## 5.1 Level 1 — Whitebox Gesamtsystem

```
┌──────────────────────────────────────────────────────────────────────────────┐
│  Bibliothek (ein Deployable)                                                 │
│                                                                              │
│  ┌────────────────────────────┐        ┌───────────────────────────────────┐ │
│  │  Frontend (SPA + PWA)      │        │  Backend (Go)                     │ │
│  │  Svelte 5 Runes, Tailwind  │◄──────►│  net/http, pgx/v5                 │ │
│  │  286 .svelte, 61.812 Zeilen│  JSON  │  65.883 Zeilen Produktivcode      │ │
│  │  IndexedDB-Warteschlange   │  SSE   │  206 Routen, 132 Migrationen      │ │
│  └────────────────────────────┘        └──────────────┬────────────────────┘ │
│           ausgeliefert AUS dem Backend                │                       │
│           (frontend/dist, os.OpenRoot)                │ pgx-Pool              │
│                                                        ▼                      │
│                                          ┌───────────────────────────────┐   │
│                                          │  PostgreSQL 18                │   │
│                                          │  42 Tabellen, 2 Sichten       │   │
│                                          └───────────────────────────────┘   │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────────┐│
│  │  Einmal-/Betriebswerkzeuge (cmd/, 9 Kommandos) — NICHT über HTTP        ││
│  └──────────────────────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────────────────────┘
```

| Baustein                   | Verantwortung                                                                                         | Nicht verantwortlich für                                             |
| -------------------------- | ----------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| **Frontend (SPA/PWA)**     | Bedienung, Ansichten, Offline-Warteschlange, Menü-Sichtbarkeit, Druckvorbereitung                     | Autorisierung (nur Blende — Autorität ist immer das Backend)          |
| **Backend (Go)**           | Routing, Auth/RBAC, Fachlogik, Persistenz, PDF, Mail, Hintergrundjobs, Echtzeit, Auslieferung der SPA  | Rendering der Oberfläche; Zustandshaltung über Prozessgrenzen hinweg  |
| **PostgreSQL**             | Datenhaltung **und Durchsetzung der Invarianten** (Constraints, partielle Unique-Indizes, Sichten)    | Geschäftsregeln, die Ermessen enthalten (Ersatzwert-Vorschlag, Sperr-Override) |
| **cmd-Werkzeuge**          | Altbestandsübernahme, Migrationen von Hand, Foto-Migration, Backup/Restore, Schlüsselwechsel, Seeding, Lasttest | Alles, was im laufenden Betrieb über die Oberfläche erreichbar sein muss |

---

## 5.2 Level 2 — Backend, Whitebox

### 5.2.1 Die Anfragekette

```
HTTP-Anfrage
   │
   ├─ 1 PanicRecovery ────────── fängt Panics, 500 statt Prozess-Ende
   ├─ 2 Sentry (Repanic: true) ─ Fehlerweitergabe, optional
   ├─ 3 SecurityHeaders ──────── CSP, HSTS, X-Content-Type-Options … (internal/middleware)
   ├─ 4 CORS ─────────────────── nur die konfigurierte Schuldomain (ALLOWED_ORIGIN)
   ├─ 5 Logging ──────────────── Status + Dauer, ohne IP, Token im Pfad maskiert
   ├─ 6 HTTPSRedirect ────────── nur wenn der Proxy es anzeigt
   ├─ 7 Lesefrist-Erweiterung ── hebt ReadTimeout für Import-Routen an (VOR dem Body-Lesen)
   ├─ 8 BodyLimit (100 MB) ───── MaxBytesReader, puffert nichts
   ├─ 9 Timeout ──────────────── Kontext-Deadline je Route (StandardBearbeitungsfrist)
   ├─10 RateLimit ────────────── je Client-IP (pkg/clientip: genau ein Proxy-Hop)
   ├─11 CSRF ─────────────────── Double-Submit-Cookie, Refresh-Route ausgenommen
   │
   ▼  http.ServeMux (Methoden-Routing, Go 1.22+)
   │
   ├─ RequirePermission("…") / RequireRoles(…) / RequireAuthenticated()
   │     └─ Cookie → JWT prüfen → Kontostatus in der DB → Recht (Cache 60 s, Epoche)
   │        → UUID-Pfadparameter prüfen  ← hier, weil r.PathValue erst nach dem Routing gefüllt ist
   ▼
  Handler (api/, inventur/, auth/)
   ▼
  Service (internal/service/) — Regeln, Orchestrierung, Transaktionsklammer
   ▼
  Repository (repository/) — SQL, pgx, Mapping
   ▼
  PostgreSQL
   │
   └─ nach dem Commit: SSE-Broadcast an alle Stationen
```

> **Warum Auth nicht in der globalen Kette sitzt:** Eine Middleware außen um den Mux
> gelegt kennt die aufgelöste Route nicht. Die UUID-Prüfung hing dort einmal und lief
> deshalb **nie** (`r.PathValue("id")` ist vor dem Routing leer, Audit-Befund vom
> 01.08.2026). Seither sitzt sie in `RequirePermission`. Zugleich ist die frühere
> `RBACBlockMiddleware` mit ihrer hartkodierten Pfad-Allowlist entfallen: Sie überstimmte
> die konfigurierbare Rechtetabelle und entzog Rechte, die der Admin gerade vergeben hatte.

### 5.2.2 Pakete des Backends

| Paket                   | Umfang (Produktivcode) | Verantwortung                                                                                                                                                                     |
| ----------------------- | ---------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `main.go`               | 1 Datei                | Konfiguration lesen **und hart prüfen** (DSN, JWT ≥ 32 Zeichen, AES-Schlüssel exakt 32 Byte, IMAP, Secret-Guard), Pool, Migrationen, Rechte-Seed, Admin-Bootstrap, SMTP-Übernahme, Broker, Scheduler, Server, Graceful Shutdown |
| `api/`                  | 27.810 Zeilen, 153 Dateien | HTTP-Schicht: Router, Middleware, CSRF, Rate-Limit, Handler je Fachbereich, PDF-Endpunkte, Selbstprüfung, Mail-Routen, LUSD-Parser und -Anwendung, öffentliche Seiten     |
| `repository/`           | 12.948 Zeilen, 78 Dateien  | SQL gegen `pgx`: Abfragen, Schreibpfade, Mapping auf Go-Strukturen, Sperren, Bewegungsstempel, Audit-Schreiber, Systemeinstellungen                                        |
| `internal/service/`     | Teil von 8.481 Zeilen  | Fachlogik mit Transaktionsklammer: Ausleihe/Rückgabe (`loan_*.go`), Omnibox, Nachbuchen, Geräte, Cover, Fotos, Bestellungen, Importe, Littera-Etiketten                          |
| `inventur/`             | 5.780 Zeilen, 43 Dateien   | **Eigenständiges Untermodul** mit eigenem Handler-Baum und eigener Datenbankschicht: Medienkatalog-CRUD, Excel-Import, ISBN-Suche, Metadaten- und Cover-Beschaffung, Dublettenkontrolle, Lernmittel-Sichten, Uploads |
| `auth/`                 | 1.342 Zeilen, 8 Dateien    | Anmeldung gegen IMAP, JWT-Erzeugung/-Prüfung, Sperrliste widerrufener Token (Ticker alle 15 min), Selbstanmeldung des Kollegiums, `/api/auth/me`, Refresh                  |
| `jobs/`                 | 1.536 Zeilen, 11 Dateien   | Cron-Scheduler (UTC) und die Läufe: DSGVO-Kette, Audit-Aufbewahrung, Backup (+ optional S3), Idempotenz-TTL, Vormerkungs-Verfall, Cover-Sync, Restore-Probe               |
| `db/`                   | 715 Zeilen, 4 Dateien      | Verbindungspool, Migrations-Runner, Rechte-Seed (`seed.go` = Vorgabe je Rolle), Admin-Bootstrap, SMTP-Konfig-Übernahme                                                    |
| `pkg/` (18 Pakete)      | 2.039 Zeilen, 22 Dateien   | Wiederverwendbares ohne Fachbezug bzw. mit **isoliertem** Fachbezug — siehe Tabelle unten                                                                                 |
| `pdf/`                  | 1.392 Zeilen, 11 Dateien   | Erzeugte Dokumente: Mahnliste, Kontoauszug, Rechnung, Schadensfall, LMF-Plan, Zahlungsweg, Schulkopf                                                                      |
| `mailservice/`          | 476 Zeilen, 4 Dateien      | SMTP-Versand mit erzwungenem STARTTLS, Kopfzeilen-Härtung (CR/LF), SMTP-Konfiguration aus der Datenbank                                                                   |
| `sse/`                  | 193 Zeilen, 1 Datei        | Broker und Handler für Server-Sent Events                                                                                                                                 |
| `apierrors/`            | 242 Zeilen, 1 Datei        | Einheitliche Fehlerantworten (`SendHTTPError`) und ihre Abbildung auf HTTP-Status                                                                                          |
| `internal/*` (übrige)   | Teil von 8.481 Zeilen  | `crypto` (AES-256-GCM), `backupkrypto` (scrypt + Dateiformat), `littera` (Altbestand lesen/abbilden/schreiben), `uebernahme` (Savepoint, Fehlerklassen, ISBN, Protokoll), `ausweis` (Gültigkeit), `middleware` (Security-Header), `pgtest`/`smtptest`/`pdftest` (Prüfhilfen) |
| `migrations/`           | 132 Dateien            | Nummerierte, idempotente Schema-Schritte; laufen beim Start                                                                                                                |
| `docs/` (Go-Anteil)     | `docs.go` generiert    | Swagger-Spezifikation, ausgeliefert **nur** bei `APP_ENV=local`/`development`                                                                                              |
| `cmd/` (9 Kommandos)    | 2.359 Zeilen, 13 Dateien   | `littera-altbestand`, `littera-import`, `migrate`, `migrate-fotos`, `encrypt-backup`, `restore-backup`, `rotate-encryption-key`, `seed`, `stresstest`                      |

#### Die `pkg/`-Pakete im Einzelnen

| Paket              | Zweck                                                                                                                             |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------- |
| `clientip`         | Echte Client-IP hinter dem Proxy — `X-Forwarded-For` wird **nur** von konfigurierten Proxies geglaubt (sonst wäre Rate-Limiting ein globaler DoS) |
| `safehttp`         | HTTP-Clients für **fremde** Ziele; Verbindungen zu nicht-öffentlichen IP-Adressen werden abgelehnt (SSRF)                          |
| `coverquelle`      | Host-Allowlists für Cover und Metadaten; baut die URL aus geprüften Teilen **neu** auf (Parsing-Differential)                     |
| `coverdatei`       | Lokal gespeicherte WebP-Cover in einer Form, die `gofpdf`/`maroto` einbetten können                                               |
| `imageutil`        | Bildkonvertierung (JPEG/PNG/GIF/WebP → JPEG), Qualitätsvorgabe                                                                    |
| `csvutil`          | Schutz vor CSV-/Formel-Injection (CWE-1236) beim Export                                                                           |
| `xlsxgrenze`       | Grenzprüfung für den Excel-Leser (Anlass: `GO-2026-6452`)                                                                          |
| `isbnutil`         | ISBN normalisieren                                                                                                                |
| `code39`           | Rechnet das Prüfzeichen wieder heraus, das bis zum 17.09.2026 auf jedem Aufdruck stand                                            |
| `kennung`          | UUID-Form genau so prüfen, wie Postgres sie annimmt (nicht `urn:uuid:…`)                                                          |
| `schulzeit`        | „Jetzt" aus Sicht der Schule, Stichtage der Bestandskartei (15.3./15.9.), Kalendertag-Rechnung                                    |
| `lmf`              | Das Wissen über Lernmittel: Schuljahresfrist, Ausleihlimit, Katalogsichtbarkeit, Löschfrist                                       |
| `lmfplan`          | Feiertage (Osterformel), freie Tage, Terminlagen des LMF-Plans                                                                    |
| `ersatzwert`       | Schadensersatz-**Vorschlag** nach Staffel, **mit Herleitung** (der Betrag liegt im Ermessen der Schule)                           |
| `httpresp`         | Antwortkörper schreiben, wenn Status und Header schon draußen sind (dann bleibt nur Logging)                                       |
| `closeutil`        | Schließen mit protokolliertem, aber nicht behebbarem Fehler                                                                       |
| `logger`           | Log-Injection-Schutz für Werte, die nicht durch `slog` gehen (CWE-117)                                                            |
| `safego`           | Unbeaufsichtigte Goroutinen überleben ein Panic — ohne den Prozess mitzunehmen                                                    |

### 5.2.3 Warum `inventur/` ein eigenes Untermodul ist

`inventur/` ist der Medienkatalog samt Beschaffung von Titeldaten und Covern. Es hat seine
**eigene** Datenbankschicht (`inventur/datenbank*.go`) und seinen eigenen Handler
(`inventur.NewAPIHandler`), der vom Hauptrouter unter wenigen Pfadpräfixen eingehängt wird:
`/api/books`, `/api/class-books`, `/api/portal/`, `/api/lookup/`, `/api/admin`, `/uploads/`.

Die Rechteprüfung kommt dabei **von außen hinein**: Der Hauptrouter übergibt die fertigen
Middleware-Wrapper (`RequireViewBooks`, `RequireEditBooks`, `RequireDeleteBooks`,
`RequireAuthenticated`) in der Konfiguration. Das Untermodul entscheidet also nicht selbst
über Rechte — es bekommt sie.

Das ist historisch gewachsen (der Katalog war ein eigenes Projekt) und bleibt so, weil die
Grenze sauber ist: Der Katalog kennt keine Ausleihen, keine Leser und kein Mahnwesen.

### 5.2.4 Neben der Anfragekette: die Einmal-Werkzeuge

Die Altbestandsübernahme läuft **nicht** über Handler/Service/Repository, sondern als
eigenes Kommando gegen dieselbe Datenbank:

| Paket                    | Aufgabe                                                                                                                                         |
| ------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/littera`       | Liest den Littera-Export (`mdb-export`-CSVs), bildet ihn auf die Begriffe dieser Anwendung ab und schreibt ihn: Bestand, Personen, Barcodes, Verleih, Abgang |
| `internal/uebernahme`    | Das Gemeinsame **jeder** Übernahme: Savepoint je Datensatz, Einordnung von Postgres-Fehlern nach SQLSTATE, ISBN-Prüfung, Spaltenbreiten, Protokoll mit getrennten Zählern für Abwertung und Ausfall |
| `cmd/littera-altbestand` | Das Kommando davor ([SCRIPTS.md](../SCRIPTS.md))                                                                                                |

Warum `internal/uebernahme` ein eigenes Paket ist und nicht in `cmd/` liegt: Die Härtung
entstand in `cmd/migrate` gegen echtes PostgreSQL. Eine zweite Kopie für Littera hätte
bedeutet, dass die zweite Fassung dieselben Fehler noch einmal macht — der fehlende
Savepoint war jahrelang unbemerkt und kostete im Fehlerfall ganze Batches.

### 5.2.5 Fremdbibliotheken des Backends

Maßgeblich ist `go.mod` (23 direkte Abhängigkeiten am 18.09.2026). Die Tabelle nennt, **wo**
jede eingesetzt wird — gemessen über die Importe des Produktivcodes, nicht abgeschrieben:

| Modul                                                          | eingesetzt in                                                    | Rolle                                                                                   |
| -------------------------------------------------------------- | ---------------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| `jackc/pgx/v5`                                                 | überall, wo SQL läuft (`repository`, `db`, `api`, `auth`, `jobs`, `inventur`, `internal/*`, `cmd/*`) | Postgres-Treiber und Pool, keine ORM-Schicht (T3)                            |
| `golang-jwt/jwt/v5`                                            | `auth`, `cmd/stresstest`                                         | Sitzungs-Token HS256; die Prüfung akzeptiert nur HMAC (`auth/jwt.go`)                   |
| `emersion/go-imap`                                             | `auth`                                                           | Anmeldung gegen den Schul-Mailserver (A2)                                               |
| `go-playground/validator/v10`                                  | `api`                                                            | Strukturprüfung von Request-Körpern                                                     |
| `getsentry/sentry-go`                                          | `main.go`, `api`                                                 | Fehlerweitergabe, nur mit `SENTRY_DSN`                                                  |
| `chai2010/webp`                                                | `api`, `pkg/imageutil`                                           | WebP-Dekodierung der Cover — der Grund für CGO (T4)                                     |
| `jung-kurt/gofpdf`, `phpdave11/gofpdf`, `johnfercher/maroto/v2` | `pdf`, `api`, `inventur`                                        | Erzeugte Dokumente (8.9)                                                                |
| `boombuler/barcode`                                            | `api`                                                            | Code 128, Code 39 und QR auf Aufdrucken (A14)                                           |
| `xuri/excelize/v2`                                             | `api`, `inventur`, `pkg/xlsxgrenze`                              | Excel lesen (`OpenReader`) und schreiben (`NewFile`); Grenzprüfung in `pkg/xlsxgrenze`  |
| `robfig/cron/v3`                                               | `jobs`                                                           | Zeitplan der Hintergrundläufe, auf UTC (A16)                                            |
| `minio/minio-go/v7`                                            | `jobs`                                                           | optionaler S3-Upload des Backups (A17)                                                  |
| `google/uuid`                                                  | `api`, `cmd/seed`                                                | Kennungen erzeugen                                                                      |
| `swaggo/swag`, `swaggo/http-swagger`                           | `docs`, `api`                                                    | Swagger, nur lokal (A23)                                                                |
| `golang.org/x/crypto`                                          | `internal/backupkrypto`                                          | scrypt-Schlüsselableitung des Backups                                                   |
| `golang.org/x/image`, `golang.org/x/net`, `golang.org/x/text`  | `pkg/imageutil`, `inventur`, `internal/service`, `repository`    | Bildformate; Zeichensatz-Erkennung (`html/charset`); Unicode-Normalisierung (`unicode/norm`) |
| `go-sql-driver/mysql`                                          | nur `cmd/migrate`                                                | Einmal-Werkzeug, nicht im Server                                                        |
| `pashagolub/pgxmock/v5`, `stretchr/testify`                    | nur `*_test.go`                                                  | Prüfhilfen                                                                              |

Nachmessen: `awk '/^require \(/{f=1;next} /^\)/{f=0} f&&!/indirect/{print $1}' go.mod` und je
Modul `grep -rl '"<modul>' --include='*.go' . | grep -v _test.go`.

---

## 5.3 Level 2 — Frontend, Whitebox

```
frontend/src
├─ main.js               Einstiegspunkt, Service-Worker-Registrierung
├─ App.svelte            App-Shell: Layout, Menü, SSE-Abonnement, Sperrbildschirm
├─ lib/                  179 Einträge — Ansichten, Komponenten, Stores, Metadaten
│   ├─ Router.svelte     Client-Routing; fragt canSeeItem() aus menu.js
│   ├─ menu.js           EINZIGE Wahrheitsquelle: welche Rolle erreicht welche Seite
│   ├─ Omnibox.svelte    Das Eingabefeld des Tresens
│   ├─ stores/*.svelte.js Geteilter Zustand (Runes-Singletons): authStore, uiStore,
│   │                     omnibox, offlineSync, printQueue, toastStore, mahnwesen*, …
│   └─ …                 Fachansichten (StudentProfile, BookAkte, Mahnwesen, LmfPlan,
│                         DruckCenter, KollegiumPortal, Betriebsbereitschaft, Monitor, …)
├─ inventur/             Die Katalog-Oberfläche des Untermoduls (eigene lib/ + routes/)
└─ styles/, assets/
```

| Baustein             | Verantwortung                                                                                                     |
| -------------------- | ----------------------------------------------------------------------------------------------------------------- |
| `App.svelte`         | Rahmen, Navigation, SSE-Verbindung samt Reconnect-Guards, Leerlauf-Sperre                                         |
| `Router.svelte`      | Pfad → Ansicht, **mit derselben Sichtbarkeitsfunktion wie das Menü**                                              |
| `menu.js`            | `canSeeItem()` — die einzige Stelle, an der Rolle/Recht über Sichtbarkeit entscheidet                              |
| `stores/omnibox`     | Eingabe, Auflösung, Warteschlange, Sperrdialog, Leserauswahl, Zusammenführen                                        |
| `stores/offlineSync` | IndexedDB-Warteschlange; entfernt einen Eintrag **nur** bei einem endgültigen Serverurteil                         |
| `stores/authStore`   | Sitzungszustand, Rechte-Map für die Blende, Abmeldung                                                              |
| `stores/printQueue`  | Sammelt Etiketten/Ausweise für den Stapeldruck                                                                     |
| `inventur/`          | Katalogpflege, Import, Cover — spiegelt die Grenze des Backend-Untermoduls                                          |

**Regeln, die im Frontend architektonisch tragen:**

- **Die Autorität ist immer das Backend.** Menü-Blenden verhindern Verwirrung, nicht
  Zugriff: Jede Datenabfrage ist rechtegeschützt, eine erzwungene Ansicht endet in 403.
- **≤ 200 Zeilen je neuer Komponente**, Altbestand darüber darf nicht wachsen (Ratsche).
- **Geteilter Zustand nur in `stores/*.svelte.js`**, Komponentenzustand lokal mit Runes.
- **Datenarrays in `.js`-Metadatendateien** (z. B. `permissionMetadata.js`) statt in
  Komponenten.
- **Logikfreie Teilkomponenten mit `{#snippet}` / `{@render}`** statt kopierter
  Markup-Blöcke (gemessen 18.09.2026: 67 Dateien).
- **Flat & Edge-to-Edge:** Trennung über `border-b`, nicht über Karten; Karten bleiben
  Modals, Toasts, Dropdowns und Cover-Kacheln.

---

## 5.4 Level 3 — Ausgewählte Bausteine im Detail

### 5.4.1 `internal/service` — Ausleihe (`loan_*.go`)

| Datei                          | Aufgabe                                                                                                  |
| ------------------------------ | -------------------------------------------------------------------------------------------------------- |
| `loan.go`                      | Dienstdefinition, Abhängigkeiten (Pools und Repositories)                                                |
| `loan_checkout.go`             | Die Transaktionsklammer: Sperren in der Reihenfolge Schüler → Ausleihe → Exemplar, Limits, Vormerkungskonflikt |
| `loan_checkout_validation.go`  | Sperren des Lesers, Überfällig-Automatik, Override mit Audit                                             |
| `loan_checkout_cases.go`       | Abbildung der DB-Fehler auf HTTP-Fälle (`mapLoanCreateErr` → 409)                                        |
| `loan_return.go`               | Rückgabe, inklusive Vormerkungs-Nachrücken (`FOR UPDATE OF v SKIP LOCKED`)                                |
| `loan_rules.go`                | Fristenberechnung je Medienart und Entleiherart                                                          |

`HandleUnifiedCheckout` ist bewusst **eine** Tür für Ausleihe und Rückgabe: Wird ein
bereits ausgeliehenes Exemplar gescannt, entscheidet die Methode selbst, ob das eine
reguläre Rückgabe, eine Fremdrückgabe oder ein Konflikt ist. Zwei Türen wären zwei
Sperrreihenfolgen.

### 5.4.2 `internal/service/omnibox_service.go` — der Scan-Dispatcher

```
ProcessQuery(eingabe)
   │
   ├─ verarbeite(eingabe)              ← der Schalter: Präfix B- / A- / S- / L- / G-,
   │                                     sonst Buch → Ausweis → Volltextsuche
   │
   ├─ scanBliebOhneTreffer(resp, err)? ← erkennt BEIDE Formen: lauter ErrNotFound
   │                                     und stille leere Trefferliste
   │
   └─ ja → Prüfzeichen abschneiden, EINMAL erneut verarbeiten
           (nur als zweiter Versuch — bei 43 möglichen Zeichen sieht jeder 43. gültige
            Code zufällig so aus, als hinge ein Prüfzeichen dran)
```

Ein Fehler, der **kein** `ErrNotFound` ist (Datenbank weg, Sperre, Gerät ohne Checkliste),
gilt ausdrücklich nicht als „ohne Treffer": Ihn ein zweites Mal auszulösen hieße, eine
Sperrmeldung doppelt zu schreiben oder eine echte Störung zu verschleiern.

### 5.4.3 `api/permission_middleware.go` — Rechteprüfung mit Epoche

Der Rechte-Cache (60 s) hat einen Zähler, nicht nur einen Inhalt:

```
leseCacheEpoche()  ──►  DB lesen  ──►  nur cachen, wenn die Epoche unverändert ist
                                        └─ sonst: Entscheidung ist überholt, nicht speichern
InvalidatePermissionCache()  ──►  Cache leeren UND Epoche erhöhen
```

Ohne die Epoche konnte ein Leser, der **vor** einer Rechteänderung startete, den alten
Stand **nach** der Invalidierung zurückschreiben — und er wirkte bis zu 60 s weiter
(Nebenläufigkeits-Audit vom 19.08.2026).

### 5.4.4 `sse/sse.go` — Broker ohne Event-Loop

| Eigenschaft            | Ausführung                                                                                           |
| ---------------------- | ---------------------------------------------------------------------------------------------------- |
| Zustand                | `map[chan string]struct{}` hinter `sync.RWMutex` — **keine** eigene Goroutine, keine Registerkanäle   |
| Verteilen              | In der Goroutine des Aufrufers, unter **Lese**sperre                                                  |
| Abmelden/Herunterfahren| Schließen der Kanäle unter **Schreib**sperre → kann sich mit dem Senden nie überschneiden             |
| Rückstau               | Puffer 10 je Client, `select`/`default`: ein langsamer Client wird übersprungen, nicht abgewartet     |
| Lebenszeichen          | Heartbeat alle 15 s als Dead-Man-Switch                                                               |
| Abbruch                | Kontext-Abbruch beim Shutdown; Zeitfenster 10 s in `main.go`                                          |

> **Wer hier einen „zentralen Event-Loop" wiederherstellt, baut einen Betriebsfehler
> zurück ein.** Die frühere Bauweise mit register-/unregister-Kanälen verhinderte das
> saubere Herunterfahren: Kehrte `Start` durch den abgebrochenen Kontext zurück, las
> niemand mehr aus den Kanälen, jeder SSE-Handler blieb in seinem `defer` stehen,
> `Shutdown` wartete auf eben diese Handler bis zum Timeout — und `main` endete mit
> `os.Exit(1)`. In einer Schule mit dauerhaft verbundenen Arbeitsplätzen war das **jeder**
> Deploy.

### 5.4.5 `jobs/` — der Zeitplan

| Job                       | Zeitplan (UTC)                  | Inhalt                                                                                                     |
| ------------------------- | ------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `RunNaechtlicheDSGVO`     | `0 0 * * *`                     | Anonymisierung Ausleihen → Anonymisierung fälliger Schüler-PII → Hard-Delete Abgänger → Lesehistorie → Anliegen, **in dieser Reihenfolge** |
| `RunAuditAufbewahrung`    | `0 3 * * *`                     | Audit-Einträge jenseits der Frist löschen (Vorgabe 24 Monate)                                              |
| `RunDatabaseBackup`       | `30 2 * * *`                    | `pg_dump` → gzip → AES-256-GCM (scrypt), Ablage im Volume, optional S3-Upload                              |
| Idempotenz-TTL            | `17 * * * *`                    | Schlüssel älter als 24 h entfernen (**stündlicher Takt**, 24 h ist die Aufbewahrung)                        |
| Vormerkungs-Verfall       | `23 * * * *`                    | Abgelaufene „abholbereit"-Reservierungen abräumen                                                           |
| Cover-Sync                | `0 */6 * * *` + on demand       | Worker-Pool (8), Re-Entrancy-Guard, Retry für FAILED                                                        |
| `RunRestoreProbe`         | `30 3 * * 0`                    | Jüngstes Backup in eine Wegwerf-Datenbank einspielen; Ergebnis wird Befund der Selbstprüfung                |

Daneben laufen **zwei** Starter im Prozess selbst: `startGDPRWorker` (beim Start und alle
24 h, nur `RunGDPRAnonymizeLoans` + `RunGDPRDeleteAbgaenger` — **nicht**
`RunGDPRAnonymizeOldData`) und `startBereitschaftsWaechter` (3 Minuten nach dem Start,
danach alle 24 h). Der erste Lauf nach 3 Minuten ist Absicht: Ein Deploy mit kaputter
Umgebung meldet sich noch am selben Vormittag, nicht erst am nächsten Tag.

---

## 5.5 Wichtige Datenstrukturen (Auszug)

Vollständig: `schema.sql` (42 Tabellen) und [invarianten.md](../invarianten.md).

| Tabelle / Sicht                | Bedeutung                                                                                                        |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------- |
| `buecher_titel`                | Katalog: Metadaten (ISBN, Titel, Autor, Verlag, Jahrgang, `ist_lernmittel`), `erweiterte_eigenschaften JSONB`      |
| `buecher_exemplare`            | Bestand: das physische Stück (Barcode, Zustand, `ist_ausleihbar`, `letzte_bewegung_am`)                          |
| `ausleihen`                    | Aktive und historische Ausleihen; `exemplar_id` XOR `geraet_id`; aktiv = `rueckgabe_am IS NULL`                    |
| `leser`                        | **Alle** Entleiher mit `art` ∈ {`schueler`, `lehrkraft`, `liv`}, ein gemeinsamer Ausweis-Nummernkreis             |
| `schueler` (**Sicht**)         | `WHERE art = 'schueler'` + `WITH CHECK OPTION` — trägt die alte Bedeutung „wirklich Schüler" weiter               |
| `benutzer`                     | Anmeldekonten; `UNIQUE lower(email)`; **keine** Passwortspalte                                                    |
| `role_permissions`             | Die konfigurierbare Rechtematrix; Vorgabe kommt aus `db/seed.go`                                                  |
| `vormerkungen`                 | Einzel-Vormerkungen inkl. Abholfach und Abholfrist                                                                |
| `klassensatz_reservierungen`   | Reservierungen des Kollegiums                                                                                     |
| `lmf_plaene`, `lmf_termine`, `lmf_termin_klassen`, `lmf_plan_freie_tage`, `lmf_plan_ausgelassen` | Büchertausch und -ausgabe je Klasse                            |
| `inventur_sessions`, `inventur_erfassungen`, `inventur_verluste` | Sitzungsgebundene Zählung, damit parallele Zählungen sich nicht überschreiben   |
| `schadensfaelle`, `schadensersatz_bescheide`, `schadensersatz_nummern` | Schaden, Forderung, Bescheid mit eigener Nummernfolge                     |
| `bestellungen_verlauf`, `bestellungen_positionen`, `lieferanten` | Bestellwesen inkl. Mittelherkunft (Land/Träger)                                 |
| `idempotency_keys`             | Schlüssel → gespeicherte Antwort (TTL 24 h)                                                                       |
| `audit_logs`                   | Ereignisprotokoll, `details JSONB`; append-only als **Konvention**, DSGVO-Tilgung als bewusste Ausnahme            |
| `revoked_tokens`               | Sperrliste abgemeldeter JWTs                                                                                      |
| `schueler_fotos`               | AES-256-GCM verschlüsselte Passbilder (kein öffentliches Dateiverzeichnis)                                        |
| `system_einstellungen`         | Die 14 Einstellungskategorien der Oberfläche                                                                      |
| `mail_settings_config`, `mail_vorlagen` | SMTP-Zugang (Passwort verschlüsselt) und Textvorlagen                                                    |
| `view_buecher_bestand` (Sicht) | Bestandszahlen je Titel für Katalog und Theke                                                                     |

---

## Anhang: Die Zahlen selbst nachmessen

```bash
# Migrationen
ls migrations/*.sql | wc -l

# Go-Produktivcode (ohne das generierte Swagger-Dokument)
find . -name '*.go' -not -name '*_test.go' -not -path './node_modules/*' \
     -not -path './docs/docs.go' | xargs cat | wc -l

# Go-Tests
find . -name '*_test.go' -not -path './node_modules/*' | wc -l
find . -name '*_test.go' -not -path './node_modules/*' | xargs cat | wc -l

# Frontend — node_modules ausschließen: Vitest legt einen Cache unter
# frontend/src/lib/node_modules an (gitignored); mit ihm zählt der Befehl das Doppelte.
find frontend/src -name '*.svelte' -not -path '*/node_modules/*' | wc -l
find frontend/src \( -name '*.svelte' -o -name '*.js' \) -not -path '*/node_modules/*' | xargs cat | wc -l
ls frontend/e2e | wc -l

# Routen und Schema
grep -rhoE 'mux\.(Handle|HandleFunc)\(' api/*.go | wc -l
grep -c 'CREATE TABLE' schema.sql
```

Für das vollständige Routenverzeichnis samt Abgleich gegen die Frontend-Aufrufer in
**beide** Richtungen: `./scripts/api_inventar.sh` → [api_inventar.md](../api_inventar.md).
