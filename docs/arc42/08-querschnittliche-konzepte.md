# 8. Querschnittliche Konzepte

Stand: 23.09.2026

Diese Konzepte gelten quer über alle Bausteine. Wer einen davon anfasst, ändert das System
an vielen Stellen zugleich — darum stehen sie hier zusammen und nicht in
[Kapitel 5](05-bausteinsicht.md).

| Abschnitt                                                        | Kernfrage                                                     |
| ---------------------------------------------------------------- | ------------------------------------------------------------- |
| [8.1 Domänenmodell und Sprache](#81-domänenmodell-und-sprache)   | Wie heißen die Dinge, und warum genau so?                      |
| [8.2 Sicherheit](#82-sicherheit)                                 | Welche Hürde steht wo, und was hält sie wirklich?              |
| [8.3 Datenschutz](#83-datenschutz-dsgvo)                         | Welche Daten, welche Frist, welcher Nachweis?                  |
| [8.4 Persistenz und Invarianten](#84-persistenz-und-invarianten) | Was ist strukturell unmöglich — und was nur beabsichtigt?      |
| [8.5 Nebenläufigkeit](#85-nebenläufigkeit)                       | Was passiert, wenn acht Stationen gleichzeitig zugreifen?      |
| [8.6 Fehlerbehandlung](#86-fehlerbehandlung)                     | Wie wird ein Fehler zum Fehler — und nicht zum stillen Erfolg? |
| [8.7 Zeit und Kalender](#87-zeit-und-kalender)                   | Welche Uhr gilt wo?                                            |
| [8.8 Echtzeit und Offline](#88-echtzeit-und-offline)             | Wie bleiben acht Stationen einig, auch ohne Netz?              |
| [8.9 Dokumente und Druck](#89-dokumente-und-druck)               | Warum der Druck ein Fachvorgang ist                            |
| [8.10 Mail](#810-mail)                                           | Wie Post das Haus verlässt                                     |
| [8.11 Protokollierung und Beobachtbarkeit](#811-protokollierung-und-beobachtbarkeit) | Woran merkt jemand etwas?                  |
| [8.12 Konfiguration](#812-konfiguration)                         | Was steht in der Umgebung, was in der Datenbank?               |
| [8.13 Barrierefreiheit und Bedienung](#813-barrierefreiheit-und-bedienung) | Bedienbar ohne Maus, messbar                          |
| [8.14 Teststrategie](#814-teststrategie)                         | Welche Ebene beweist was?                                      |
| [8.15 Abhängigkeits-Hygiene](#815-abhängigkeits-hygiene)         | Wie bleibt der Baum sauber?                                    |

---

## 8.1 Domänenmodell und Sprache

**Katalog und Bestand sind strikt getrennt.** `buecher_titel` ist das Werk (ISBN, Titel,
Autor, Jahrgang, `ist_lernmittel`), `buecher_exemplare` ist das physische Stück (Barcode,
Zustand, Ausleihbarkeit). Eine Ausleihe hängt immer am **Exemplar**, nie am Titel. Genau
diese Trennung ist der Grund, warum die Trefferliste an der Theke „3 Stück, 1 frei" sagen
kann — und warum ein Titel ohne Exemplar es ausdrücklich sagen muss.

**Rolle und Art sind zwei verschiedene Fragen.** Die **Rolle** sagt, was jemand im
Programm darf (Admin, Leitung, Mitarbeiter, Helfer — dazu `kollegium` als Grundzustand).
Die **Art** sagt, wer an der Theke Bücher bekommt (`schueler`, `lehrkraft`, `liv`). Ein
Mensch kann beides haben, eines oder keines: Eine Lehrkraft ohne Konto steht in der
Leserdatei und darf nichts im Programm; ein Admin muss nicht in der Leserdatei stehen.

**`schueler` ist eine Sicht, keine Tabelle** (seit Migration 124): `WHERE art = 'schueler'`
mit `WITH CHECK OPTION`. Das trägt die alte Bedeutung weiter — Klassenlisten,
LUSD-Abgleich, Mahnlauf und die Löschfristen lesen die Sicht und bekommen das Kollegium
nicht zu sehen.

> **Die Kehrseite ist eine eigene Bugklasse** („Schreibpfad gegen gefilterte Sicht",
> [sweeps.md](../sweeps.md)): Ein Pfad, der *alle* Leser meint und weiter gegen `schueler`
> schreibt, trifft beim Kollegen null Zeilen und meldet „nicht gefunden" — eine **stille
> 404** statt eines Fehlers. Genau so sind der Änderungspfad der Stammdaten und das
> Zusammenführen aufgefallen. Ein `JOIN schueler` sieht nach Zugehörigkeit aus und ist ein
> `WHERE`, das niemand geschrieben hat. Detektor:
> `docs/schreibpfade_gegen_sicht_test.go`.

**Enum-Schreibweise:** `benutzer_rolle` ist ein Postgres-ENUM mit
Kleinbuchstaben (`admin`, `kollegium`, `mitarbeiter`, `helfer`, `leitung`). SQL-Vergleiche
müssen `LOWER(rolle::text)` verwenden — kein `= 'KOLLEGIUM'`.

**Sprache:** Domänenbegriffe sind deutsch (`ausleihen`, `leser`, `vormerkungen`,
`Ersatzwert`, `Abgaenger`, `Nachbuchen`); aus der Frühzeit steht Englisch daneben
(`book`, `loan`, `student`). Gemischt, aber nie doppelt: Es gibt nicht zwei Namen für
dieselbe Sache. Das [Glossar](12-glossar.md) listet beide Seiten.

---

## 8.2 Sicherheit

### Die Hürden in der Reihenfolge, in der eine Anfrage sie trifft

| Hürde                     | Ausführung                                                                                                                                         |
| ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Security-Header**       | CSP `default-src 'self'` mit `script-src 'self'`, `img-src 'self' data: blob:`, `frame-ancestors 'none'`, `object-src 'none'`; HSTS 1 Jahr inkl. Subdomains; `X-Frame-Options: DENY`; `Referrer-Policy: strict-origin-when-cross-origin`; `Permissions-Policy: geolocation=(), microphone=(), camera=(self)` (Kamera: Barcode-Scan und Passbild) |
| **CORS**                  | Nur die konfigurierte Schuldomain (`ALLOWED_ORIGIN`)                                                                                               |
| **Body-Limit**            | 100 MB global über `http.MaxBytesReader` — begrenzt nur, **wie viel ein Handler lesen darf**, puffert nichts. Bewusst großzügig, weil Littera-, Bestands- und Excel-Import durch denselben Wert laufen. Für `/login` gilt zusätzlich eine eigene, enge Grenze (16 KB): Es ist der einzige unangemeldete Endpunkt, der JSON liest |
| **Fristen (Slowloris)**   | `ReadHeaderTimeout` 5 s, `ReadTimeout` als Lesefrist, `IdleTimeout` 120 s, kein `WriteTimeout` (SSE) — siehe [7.1](07-verteilungssicht.md#fristen-müssen-zueinander-passen) |
| **Rate-Limit**            | 50 Anfragen/s/IP (Map + Mutex, kein externer Cache). Ausgenommen sind Auslieferungspfade (`/api/images/cover`, `/uploads/`, `/api/barcode`, `/events`) — ein Seitenaufruf lädt dutzende Bilder, SSE ist eine Dauerverbindung. Der **teure** Zweig des Cover-Proxys hat eine eigene Bremse |
| **CSRF**                  | Double-Submit-Cookie; `GET /api/csrf-token` holt Token und Cookie vor der ersten Änderung; die Refresh-Route ist ausgenommen                        |
| **Anmeldung**             | IMAP über implizites TLS (:993, min. TLS 1.2, feste Cipher-Auswahl). **Kein** Passwort im System                                                     |
| **Brute-Force**           | Schlüssel `lower(email)|ip`, 5 Fehlversuche / 15 min. Nur-IP wäre an einer Schul-NAT eine Aussperrung der ganzen Schule                              |
| **Sitzung**               | JWT **HMAC-only** (HS256; `alg=none` ausgeschlossen), 12 h, Cookie `HttpOnly` + `SameSite=Strict` (+ `Secure` nach Betriebslage); Sperrliste für abgemeldete Token, Aufräum-Ticker alle 15 min |
| **Fail-closed**           | Ist die Sperrlisten- oder Kontostatus-Abfrage nicht erreichbar, wird die Anfrage mit **503** abgelehnt — nicht durchgelassen und **nicht** mit 401 beantwortet (eine 401 meldet den Arbeitsplatz ab, obwohl die Sitzung gültig ist) |
| **Autorisierung**         | `RequirePermission` je Route: Recht aus `role_permissions` (Cache 60 s mit Epochenzähler) **plus** Live-Kontostatus **plus** UUID-Form der Pfadparameter |
| **Eskalationsschutz**     | Ein Admin-Konto bleibt der Leitung auch mit `manage_users` verschlossen (`api/user_admin_eskalation.go`) — weil wer `benutzer.email` schreiben darf, ein Konto übernimmt |
| **Secret-Guard**          | Außerhalb von `local/development/test` verweigert der Server den Start bei bekannten Beispiel-Geheimnissen. Die Liste steht **einmal** (`api.IstBekanntesDefaultGeheimnis`) und wird von der Selbstprüfung mitbenutzt — zwei Listen würden „alles gut" melden, während der Server aus demselben Grund nicht startet |

### Injection und Datenausgänge

| Klasse                        | Maßnahme                                                                                                                                                                        |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **SQL-Injection**             | Ausschließlich parametrisierte Abfragen über `pgx`; kein String-Zusammenbau von Werten                                                                                          |
| **XSS**                       | Svelte escapet von selbst. Die **eine Naht**, an der das nicht mehr gilt, ist der Druckpfad (erzeugtes HTML für den Etiketten-/Ausweisdruck) — dort wird ausdrücklich kodiert   |
| **CSV-/Formel-Injection**     | `pkg/csvutil.SanitizeCell` (CWE-1236): Eine Zelle, die mit `=`, `+`, `-` oder `@` beginnt, ist für Tabellenkalkulationen eine Formel                                             |
| **SSRF**                      | `pkg/safehttp` lehnt Verbindungen zu **nicht-öffentlichen** IP-Adressen ab; `pkg/coverquelle` hält die Host-Allowlist                                                           |
| **Parsing-Differential**      | Die Cover-/Metadaten-URL wird aus geprüften Teilen **neu gebaut** (Schema fest HTTPS, Host aus der Konstante, nur Pfad und Query vom Aufrufer). Geprüft wurde bisher mit `url.Parse`, angefragt der rohe String — verschiedene Parser sind sich über `\`, `@`, `#`, `?` nicht einig |
| **Path-Traversal**            | `os.OpenRoot` bindet Dateizugriffe **OS-seitig** an ein Verzeichnis (SPA-Auslieferung, Cover-Einbettung ins PDF). Manuelle `filepath.Clean`+`HasPrefix`-Logik ist fehleranfällig und war als Existenz-Orakel außerhalb des Verzeichnisses nutzbar |
| **Log-Injection**             | `slog.SetDefault` mit JSON-Handler: Ein Zeilenumbruch aus einer Anfrage wird zu `\n` **innerhalb** des JSON-Strings und kann keinen zweiten Eintrag erzeugen. Am laufenden Server nachgemessen. Wer diese Zeile entfernt oder `log.SetOutput` dahinter setzt, hebt beides auf |
| **SMTP-Header-Injection**     | `mailservice.HeaderWert` weist CR/LF in Kopfzeilenwerten ab — direkt an der Schreibstelle, nicht nur bei der Empfängerprüfung                                                    |
| **Dekompressionsbombe**       | Bild-Uploads werden gegen aufgeblähte Pixelmaße geprüft, bevor dekodiert wird                                                                                                   |
| **Token im Pfad**             | Der Bestätigungslink des Händlers trägt sein Geheimnis im Pfad; `maskiereToken` hält es aus der Logzeile (sonst wäre die Hash-Speicherung in der Datenbank entwertet)             |
| **Panics**                    | `PanicRecoveryMiddleware` für HTTP-Handler, `pkg/safego` für unbeaufsichtigte Goroutinen — ein ungefangenes Panic in einer Goroutine reißt in Go den **ganzen** Prozess mit      |

### Endpunkte ohne Anmeldung (bewusst)

`/katalog` und `/monitor` (Titeldaten), `GET /api/images/cover` und `/uploads/` (Cover),
`POST /login`, `GET /api/csrf-token`, `GET /health`, der Bestätigungspfad
`/bestellung/<token>`. Jeder davon steht in der Allowlist von
`api/routes_authz_coverage_test.go`: Eine neue ungeschützte Route ohne Eintrag macht das
Gate rot.

---

## 8.3 Datenschutz (DSGVO)

### Einstufung als Gate, nicht als Zusage

[PII_MATRIX.de.md](../PII_MATRIX.de.md) stuft **jede** Route nach Schülerdaten ein
(Stufe 0–3). Drei Gates halten die Matrix ehrlich:

- `api/pii_matrix_test.go` — eine Route ohne Zeile wird rot, eine Zeile ohne Route ebenso,
  und das dokumentierte Recht wird gegen die Registrierung geprüft.
- `api/pii_antwort_gate_pg_test.go` — ruft jede GET-Route (und die lesenden POST-Routen:
  Theken-Scan und LUSD-Vorschau) über den **echten** Router mit genau dem Recht ihrer Zeile
  auf und prüft die Antwort — inklusive entpackter PDF-Ströme — gegen Kanarienwerte je
  Stufe. Schlüsselrouten tragen Positiv-Kontrollen gegen leere Antworten.
- `api/rechte_schreibwege_pg_test.go` — fährt alle 87 Schreibrouten mit Fachrecht mit einer
  Rolle, die das Recht **nicht** hat, und verlangt 403 mit der Begründung des
  Rechte-Wächters. Damit ist auch der Fall abgedeckt, den ein Textvergleich nie sieht: ein
  Recht, das im Seed ohnehin jede Rolle hält.

### Verschlüsselung

| Gegenstand              | Verfahren                                                                 | Ort                          |
| ----------------------- | ------------------------------------------------------------------------- | ---------------------------- |
| Schülerfotos            | AES-256-GCM (`internal/crypto`, `APP_ENCRYPTION_KEY`)                     | **in** der Datenbank         |
| SMTP-Passwort           | AES-256-GCM                                                               | `mail_settings_config`       |
| Datenbank-Backups       | gzip + AES-256-GCM mit **scrypt**-abgeleitetem Schlüssel                  | Volume + optional S3         |
| Transport               | HTTPS (Caddy/ACME), IMAP implizites TLS, SMTP erzwungenes STARTTLS        | —                            |

Der Schlüsselwechsel hat ein eigenes Werkzeug **mit Probelauf**
(`cmd/rotate-encryption-key`), und es liegt im Image — der Schulserver hat kein Go. Der
frühere Zweitname der Schlüsselvariable wird **nicht mehr gelesen**; ist er abweichend
gesetzt, bricht der Start laut ab, statt mit dem falschen Schlüssel zu verschlüsseln.

### Fristen und Löschung

```
Ausleihbearbeiter (bearbeiter_id)  →  nach 14 Tagen entfernt
Schüler-PII fällig                 →  anonymisiert (inkl. Audit-Spuren)
Abgänger ohne offene Vorgänge      →  Karenz (Vorgabe 90 Tage) → Anonymisierung
Anonymisierte Abgänger             →  Hard-Delete ab 30. Januar des Folgejahres
Audit-Einträge                     →  gelöscht jenseits der Frist (Vorgabe 24 Monate)
Lesehistorie, Anliegen             →  eigene Läufe in derselben Nachtkette
```

Die **Reihenfolge ist die Zusage**: Löschung läuft nach der Anonymisierung, damit die
Karenz für beides gilt. Import, Nachtlauf und Selbstprüfung lesen **denselben**
Einstellungsschlüssel und rechnen mit **demselben** Prädikat
(`repository.PredikatAnonymisierung`). Keine Frist darf eine andere verkürzen: Die Uhr der
Karenz liest den letzten abgeschlossenen Vorgang am Leser (`letzter_vorgang_am`, Migration
137), nicht aus den Ausleihen, deren Zuordnung die Lesehistorie-Befristung löst.

**Offene Vorgänge schlagen die Frist:** Wer eine offene Ausleihe oder einen unbezahlten
Schaden hat, wird gesperrt, behält aber Name und Anschrift — sonst ließe sich die Forderung
nicht mehr zustellen.

### Audit-Trail

- `audit_logs` ist **append-only als Konvention** (kein `UPDATE`/`DELETE` außer der
  DSGVO-Tilgung; kein Trigger-Zwang). Die Tilgung ist die bewusste Ausnahme — anders wäre
  das Löschrecht nicht erfüllbar.
- **Ausleihe und Rückgabe schreiben ihre Audit-Zeile in derselben Transaktion**, vor dem
  Commit: Kann die Spur nicht geschrieben werden, gibt es die Buchung nicht. Vorher lief
  die Zeile nach dem Commit in eigener Transaktion — brach die Verbindung dazwischen ab,
  galt die Ausleihe **ohne** Revisionsspur.
- Admin-Aktionen (Sperr-Override, Wareneingang-Sammelbuchung) stehen weiter neben dem
  Vorgang.
- **Keine IP-Adressen in der Anfrage-Logzeile.**

### Datenminimierung als Entwurfsprinzip

- Öffentliche Seiten liefern **nur** Titeldaten. Der Abholfach-Hinweis der Theke trägt
  Titel und Frist — bewusst **ohne** IDs, weil die Theke nur den Griff ins Fach braucht.
- Statistiken kommen ohne Klarnamen (Zirkulation, Wiederbeschaffungswert, Renner und
  Ladenhüter).
- Mahnlisten gehen an die Klassenleitung, **nie** an Schüler.
- Es gibt **kein** öffentliches Fotoverzeichnis mehr. `uploads/fotos` wurde bis zum
  08.08.2026 bei jedem Start neu angelegt, obwohl nichts mehr hineinschrieb — unter einem
  Pfad, der ohne Anmeldung lesbar ist, und mit Dateinamen, die die **Barcode-IDs von den
  Schülerausweisen** waren, also vollständig aufzählbar.

---

## 8.4 Persistenz und Invarianten

### Die Ebenenfrage

[invarianten.md](../invarianten.md) stellt zu jeder Invariante **nicht** die Frage „testen
wir sie?", sondern „auf welcher Ebene ist sie durchgesetzt?":

| Ebene       | Bedeutung                             | Umgehbar?                                               |
| ----------- | ------------------------------------- | ------------------------------------------------------- |
| 🟢 **DB**   | CHECK / UNIQUE / FK / Enum / NOT NULL | Nein — strukturell unmöglich                            |
| 🟡 **Code** | Go-Handler/Service-Logik              | Ja, sobald ein zweiter Schreibpfad die Prüfung auslässt |
| 🔴 **Doku** | nur im Kommentar/Konzept              | Ja — reine Hoffnung                                     |

Ziel ist, kritische Invarianten von 🔴/🟡 nach 🟢 zu schieben. Beispiele für 🟢:
`uniq_ausleihen_aktiv_exemplar`, `check_loan_item`, `check_return_date`,
`ausleihen_schueler_id_fkey`, `chk_leser_nur_schueler_werden_abgaenger`,
`UNIQUE lower(email)`.

Ein Beispiel für ein bewusst gebliebenes 🟡: die **Sperrreihenfolge** Schüler → Ausleihe →
Exemplar. Sie ist Konvention; jeder Schreibpfad hält sie, ein Gate gibt es nicht
([Kapitel 11](11-risiken-und-technische-schulden.md)).

### Migrations-Hygiene

- Nummeriert (`NNN_beschreibung.sql`), dedupliziert über `schema_migrations`, laufen beim
  Start.
- **Idempotent**: `IF NOT EXISTS` / `IF EXISTS` / `DO $$ BEGIN … EXCEPTION WHEN … END $$`.
- Die Seed-Liste in `schema.sql` muss **exakt** den Dateien entsprechen — kein
  Phantom-Eintrag, kein fehlender.
- Doppelte Zahlenpräfixe (003, 008, 021, 022) sortieren deterministisch und haben keine
  Reihenfolgeabhängigkeit: Style-Smell, funktional korrekt.
- **Eine Migrationsnummer ist kein haltbarer Beleg.** Die Datei existiert weiter, auch wenn
  eine spätere Migration ihr Werk zurücknimmt — eine Existenzprüfung bliebe grün. Deshalb
  wird in der Dokumentation das **lebende Objekt** beim Namen genannt.

### Erweiterbarkeit ohne Migration

`buecher_titel.erweiterte_eigenschaften`, `buecher_exemplare.erweiterte_eigenschaften` und
`audit_logs.details` sind `JSONB DEFAULT '{}'` — für Ad-hoc-Attribute (Regalposition,
Signatur, externe IDs) ohne Schemaschritt. GIN-Indizes können bei Bedarf darauf gelegt
werden.

---

## 8.5 Nebenläufigkeit

Vier Schichten, in dieser Reihenfolge wirksam:

1. **Transaktion und Zeilensperre.** `READ COMMITTED` (Postgres-Standard, hoher Durchsatz)
   mit `SELECT … FOR UPDATE`. Die Sperrreihenfolge ist **Schüler → Ausleihe → Exemplar**;
   Online-Scan und Nachbuchen halten sie gleich, sonst verklemmen sie sich gegeneinander.
   Die Rückgabe mit Vormerkung nimmt `FOR UPDATE OF v SKIP LOCKED`, damit eine fremde
   Sperre den Rückgabevorgang nicht anhält.
2. **Struktur.** Die partiellen Unique-Indizes sind an der entscheidenden Stelle **der
   einzige** Schutz: Zwei Stationen, die dasselbe Exemplar für verschiedene Leser scannen,
   greifen auf keine gemeinsame Zeile (siehe
   [6.4](06-laufzeitsicht.md#64-doppelscan-von-zwei-stationen)).
3. **Idempotenz.** Schlüssel je Vorgang mit gespeicherter Antwort; eine parallele Anfrage
   desselben Schlüssels wartet begrenzt (3 s) auf das Ergebnis der laufenden. 5xx wird
   **nie** gecacht, damit ein Wiederholen möglich bleibt. TTL-Lauf stündlich, Aufbewahrung
   24 h (zwei Zahlen, die man nicht verwechseln darf).
4. **Rechte-Cache mit Epoche.** Eine DB-Entscheidung darf nur in den Cache, wenn seit ihrem
   **Start** keine Invalidierung dazwischenkam — sonst schreibt ein überholter Leser den
   alten Stand zurück, wo er bis zu 60 s weiterwirkt.

**Bewegungsstempel:** Jede Bewegung eines Exemplars (Ausleihe, Rückgabe, Rückholen,
Aussonderung) setzt `letzte_bewegung_am`, und der Stempel **läuft nie rückwärts** — auch
nicht mit einem früheren Scan-Zeitpunkt aus dem Nachbuchen und nicht aus einer früher
begonnenen Transaktion. Jede Ausleihe trägt zusätzlich `erfasst_am` (Scan-Zeitpunkt,
höchstens Serverzeit).

---

## 8.6 Fehlerbehandlung

| Regel                                                          | Warum                                                                                                                                           |
| -------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| **Eine Unique-Verletzung wird 409, nicht 500**                  | Ein Doppelscan ist ein Fachfall, keine Störung. `mapLoanCreateErr` bildet SQLSTATE 23505 ab                                                     |
| **DB-Aussetzer bei der Sitzungsprüfung wird 503, nicht 401**    | Eine 401 meldet den Arbeitsplatz ab, obwohl die Sitzung gültig ist                                                                               |
| **Ein unbekannter `/api/`-Pfad wird 404, nicht die App-Shell**  | Vorher antwortete er „200 text/html", und der Aufrufer scheiterte erst beim JSON-Parsen — mit einer Meldung, die nichts mit der Ursache zu tun hat |
| **Jede `rows.Next()`-Schleife endet mit `rows.Err()`**          | Ohne das gilt ein Verbindungsabbruch mitten in der Iteration als **Erfolg** — die Liste wäre still unvollständig. In `audit_books.go` hätte das einen Titel trotz aktiver Ausleihen als „ausleihbar" behandeln können |
| **Schweigen ist kein Erfolg**                                   | Ein Nachbuch-Schlüssel ohne Antwort bleibt in der Warteschlange liegen                                                                          |
| **Eine gekappte Liste sagt es**                                 | Wo eine Ausgabe begrenzt wird, steht die Begrenzung dabei — auch wenn zusätzlich gefiltert wurde                                                 |
| **Fehlerantworten einheitlich**                                 | `apierrors.SendHTTPError`; Fehlermeldungen sind deutsch und nennen die nächste Handlung                                                          |
| **Ein Panic beendet nicht den Prozess**                         | `PanicRecoveryMiddleware` für Handler, `pkg/safego` für Goroutinen                                                                              |

Die Testklasse dazu heißt `phantom_erfolg_test.go` und `fehler_kollaps_test.go`: „Hat es
geklappt?" muss beantwortbar sein, und mehrere verschiedene Fehler dürfen nicht zu einer
einzigen unbrauchbaren Meldung kollabieren.

---

## 8.7 Zeit und Kalender

| Frage                        | Antwort                                                                                                          |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| Welche Zeitzone rechnet Fristen? | `Europe/Berlin`, im Code gepinnt (`TagesEndeInSchulzeitzone` → `schulzeit.TagesEnde`). Eine zweite, rohe Berechnung gibt es bewusst nicht |
| Welche Zeitzone fährt den Cron?  | **UTC**, genagelt über `cron.WithLocation(time.UTC)` — sonst verschluckt die Umstellnacht den 02:30-Job          |
| Was ist „jetzt" für die Fachlogik? | `pkg/schulzeit`; in Tests über `Server.Uhr` gestellt, sonst bewiese derselbe Test im Oktober das Gegenteil von dem im Juni |
| Wie wird der Lernmittel-Stichtag berechnet? | 31. Juli des laufenden bzw. kommenden Schuljahres — es sei denn, der LMF-Plan nennt für die Klasse einen Rückgabetermin; dann gilt der nächste Termin **nach** dem Ausleihtag |
| Feiertage?                    | `pkg/lmfplan` rechnet Ostersonntag nach der Gauß'schen Osterformel (Fassung Lichtenberg) und leitet die beweglichen Feiertage ab |
| Bestandsstichtage?            | 15.3. und 15.9. (`pkg/schulzeit`) — geprüft wird an den **Rändern**, weil dort entschieden wird, ob ein Abgang noch in den Nachweis gehört |
| Und die Uhr des Kiosk-Rechners? | Wird beim Nachbuchen gemessen; der Versatz wird herausgerechnet und ab einer Schwelle protokolliert (siehe [6.5](06-laufzeitsicht.md#65-theke-ohne-netz--und-das-nachbuchen)) |

Eine **Ferienautomatik gibt es nicht.** Das Werkzeug für „Bücher über die Sommerferien
mitnehmen" ist der Ferien-Leseclub (aktiv + Zieldatum ⇒ feste Rückgabefrist für alle
Ausleihen). Die Tabelle `ferien_schliesszeiten` hatte nie einen Schreiber und ist
ausgebaut; das Mahnwesen wird ohnehin nur von Hand bedient.

---

## 8.8 Echtzeit und Offline

### SSE

Nach jedem Commit geht ein Ereignis an alle verbundenen Stationen. Der Broker hat
**keinen** Event-Loop (Details und die Begründung:
[5.4.4](05-bausteinsicht.md#544-ssessego--broker-ohne-event-loop)). Im Client:
Reconnect mit Guards (`isLoggedIn`, Timeout), damit ein abgemeldeter Tab nicht endlos
gegen die Tür klopft.

### Offline

- **PWA** (`vite-plugin-pwa`, `registerType: 'autoUpdate'`): Der Service Worker beantwortet
  Navigationen aus dem Cache mit der App-Shell — aber nur für Pfade, die auch wirklich
  Anwendungspfade sind.
- **IndexedDB-Warteschlange** (`idb`): Scans werden lokal gehalten und gehen später durch
  die **Nachbuch-Tür**. Ein Eintrag verlässt die Warteschlange nur bei einem **endgültigen**
  Serverurteil (acht von neun Wörtern).
- **Die Vorsilbe ist offline unentbehrlich:** Ohne Netz ist sie die einzige Information, an
  der die Theke einen Buchscan von einem Ausweisscan unterscheiden kann. Es gibt niemanden
  zu fragen. (Littera kommt ohne aus, weil dort Nummer und Scanwert zwei verschiedene
  Felder sind — dieses zweite Feld gibt es hier nicht.)
- Portionen von 1–50 Einträgen je Aufruf: Mehr wäre eine Transaktion, die zu lange offen
  steht.

---

## 8.9 Dokumente und Druck

| Dokument                                  | Erzeuger                               | Besonderheit                                                                                         |
| ----------------------------------------- | -------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| Mahnliste, Kontoauszug, Rechnung, Schadensfall, LMF-Plan | `pdf/` (gofpdf/maroto)  | Der **Druck** der Mahnung ist der Verwaltungsakt: nur hier steigt die Mahnstufe                       |
| Bescheid (Landes-Lernmittel)              | `api/bescheid_pdf.go`                  | Nennt das Konto; Barzahlung ist laut Erlass nicht der Weg. Eigene Nummernfolge                        |
| Etiketten und Ausweise                    | `api/label_pdf.go`, `api/barcode_*.go` | Aufschrift nach **Art** des Lesers („Schülerausweis"/„Lehrerausweis"); Gültigkeit nur beim Schülerausweis |
| Zugangs-/Abgangsbuch                      | `api/bestandsbuch.go`, `api/abgangsbuch_*.go` | Getrennt nach Land und Träger; sagt ausdrücklich, was es **nicht** weiß (Aussonderungen ohne Abgangsdatum, Bücher „ohne Zuordnung") |
| DSGVO-Auskunft                            | `api/dsgvo_pdf.go`                     | Auskunftsrecht als Dokument                                                                           |
| Barcodebogen für den Händler              | `api/bestellbestaetigung_etiketten.go` | Für Händler, die selbst etikettieren                                                                  |

**Barcode-Symbologie:** Gedruckt wird **Code 128 ohne Prüfzeichen**. Vorher war es Code 39
**mit** Prüfzeichen — das Zeichen steht in den Strichcode-Daten, und ein Lesegerät gibt es
als Teil der Nummer zurück: Unter der Karte stand `A-10003`, gescannt wurde `A-100037`. Der
Server suchte eine Nummer, die es nicht gibt, und weil ein unbekannter Scan nur eine leere
Trefferliste erzeugt, sah es an der Theke aus, als täte der Scanner **gar nichts**.
`pkg/code39` rechnet das Zeichen für alte Aufdrucke wieder heraus.

Cover werden für die PDF-Einbettung aus WebP konvertiert (`pkg/coverdatei`) — weder
`gofpdf` noch `maroto` kennen WebP. Ein Gate (`npm run test:druck`) prüft die
Drucksektionen am gebauten Frontend.

---

## 8.10 Mail

- **Konfiguration aus der Datenbank, nicht aus der Umgebung.** `BindeSMTPKonfigAnDatenbank`
  wird beim Serverbau gesetzt; vorher benutzte der Test-Knopf die eine und jeder echte
  Versand die andere Quelle. `InitMailKonfig` übernimmt beim Start die Werte aus der
  Umgebung **nur**, solange in der Datenbank noch die Schema-Vorgabe steht — ohne diese
  Übernahme gingen die Mahnungen nach dem Umstieg an `localhost:1025`. Gemeldet wird nur,
  wenn wirklich übernommen wurde.
- **STARTTLS erzwungen**, mit Zertifikatsprüfung. Bietet der Server es nicht an, bricht der
  Versand ab (`ErrSMTPKlartext`) — vorher galt „dann eben ohne", und ein Angreifer, der die
  EHLO-Antwort streicht („STARTTLS stripping"), bekam Mahntexte mit Schülernamen.
  `SMTP_ALLOW_PLAINTEXT=true` erlaubt es ausdrücklich und protokolliert die Folgen.
- **Kopfzeilen-Härtung** direkt an der Schreibstelle (Betreff, Absender, Empfänger).
- **Wer Post bekommt:** Klassenleitung (Mahnlisten), Händler (Bestellung), Admins
  (Bereitschafts-Wächter), Lehrkräfte (Portal-Vorgänge). **Nicht** Schüler.
- Vorlagen mit Platzhaltern liegen in `mail_vorlagen` und sind in der Oberfläche pflegbar.

---

## 8.11 Protokollierung und Beobachtbarkeit

| Mittel                       | Inhalt                                                                                                              |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `slog` als JSON auf stdout   | Ein Handler für alles — auch für die noch vorhandenen `log.Printf`-Aufrufe (seit Go 1.21 hängt `log` mit daran). Das ist zugleich der Log-Injection-Schutz |
| Anfrage-Logzeile             | Methode und Pfad (Token maskiert), geschrieben VOR der Verarbeitung — **keine** IP, kein Status, keine Dauer, keine Anfragekennung. Endet die Anfrage mit 5xx, folgt eine zweite Zeile mit dem Status |
| 500er                        | Stack schreibt `PanicRecoveryMiddleware`; die Logging-Middleware hängt **keinen** eigenen Stack an (der zeigte nur auf sie selbst) |
| Docker-Logs                  | json-file, 3 × 10 MB je Container                                                                                    |
| Sentry (optional)            | `SENTRY_DSN`; `Repanic: true`, damit die eigene Recovery weiterhin greift                                            |
| `GET /health`                | Prozess **und** DB-Ping                                                                                              |
| Selbstprüfung                | „Was ist eingerichtet, aber nicht in Betrieb?" — Mail, Selbstanmeldung, Geheimnisse, Backup, Restore-Probe            |
| Bereitschafts-Wächter        | Meldet kritische Befunde per Mail an die Admins: 3 Minuten nach dem Start, danach täglich. Auf Spielwiesen schweigt er von selbst |
| `GIT_COMMIT` im Image        | „Gesund" heißt nicht „aktuell" — `update.sh` vergleicht nach dem Deploy                                              |

---

## 8.12 Konfiguration

Zwei Orte mit klarer Trennung:

| Ort                              | Inhalt                                                                                                         | Änderbar durch            |
| -------------------------------- | -------------------------------------------------------------------------------------------------------------- | ------------------------- |
| **Umgebung** (`.env`, Compose)   | Was zum Starten gebraucht wird und Sicherheitsentscheidungen: DSN, Geheimnisse, Ports, IMAP, Proxy-Vertrauen, Cookie-Secure, Backup- und S3-Zugang | Betrieb (Neustart nötig)  |
| **Datenbank** (`system_einstellungen`, `mail_settings_config`, `role_permissions`) | Fachliche Schalter der Einstellungskategorien (FACHKONZEPT §17): Fristen, Limits, Karenzzeit, Sitzungsfristen, SMTP-Zugang, Rechtematrix | Oberfläche (sofort wirksam) |

**Die Vorgabe ist immer die sichere Richtung.** `COOKIE_SECURE` ohne Wert ⇒ `true` mit
Warnung; unlesbarer Wert ⇒ Abbruch. Der Secret-Guard ist von selbst scharf. Fehlt
`BACKUP_ENCRYPTION_KEY`, überspringt sich der Backup-Job — und die Selbstprüfung macht
genau das sichtbar, statt dass es unbemerkt bleibt.

Was das Kollegium darf, steht **fest** in `db/seed.go` und ist bewusst **kein** Schalter je
Schule. Die Rechte der Leitung werden aus den Admin-Rechten **abgeleitet** (Admin minus
`manage_users` und `manage_settings`) — eine abgeschriebene Liste wäre eine zweite Wahrheit
und liefe beim nächsten neuen Recht auseinander, ohne dass es jemand merkt.

---

## 8.13 Barrierefreiheit und Bedienung

- Gebaut auf **WCAG 2.1 AA**, gemessen per Browser-Gate: axe über den Anfangszustand aller
  Hauptansichten, dazu Gates für Fokusfalle, Tabellensemantik und Bewegung
  (`frontend/e2e/barrierefreiheit-*.spec.js`).
- **Tastaturbedienung ist der Normalfall**, nicht die Ausnahme: Ein Handscanner ist eine
  Tastatur. Die Kurzbefehle stehen im [Handbuch](../HANDBUCH.md); Umfang, Grenzen und
  bekannte Lücken in [FACHKONZEPT.md §19](../FACHKONZEPT.md).
- **Sichtschutz statt Abmeldung:** 5 Minuten ⇒ Theke leeren, 15 Minuten ⇒ Sperrbildschirm.
  Als Bedienung zählen Zeiger, Tastatur, Berührung und Rad — **nicht** SSE-Pings oder
  Poller. Beide Fristen sind Einstellungen (0 = aus) und kommen aus einer Route, die nur
  zwei Zahlen liefert.
- **Fehlermeldungen sind deutsch und handlungsleitend.** Ein Ergebnis darf nicht so
  aussehen, als täte das Gerät nichts.

---

## 8.14 Teststrategie

| Ebene                     | Beweist                                                                     | Preis                                                                 |
| ------------------------- | --------------------------------------------------------------------------- | --------------------------------------------------------------------- |
| Unit (Go)                 | Regeln, Grenzfälle, Fehlerabbildung                                          | Sieht keine Constraints                                               |
| `*_pg_test.go`            | Constraints, Sichten, Sperren, Trigger — gegen **echtes** Postgres           | Überspringt sich ohne `TEST_DATABASE_URL` **still**, mit grünem „ok"  |
| Vitest (Frontend)         | Stores, Regeln, Komponentenverhalten                                         | Kein echter Browser                                                   |
| `svelte-check`            | Typfehler ohne TypeScript (JSDoc + `checkJs`)                                | Nur so gut wie die JSDoc-Abdeckung                                    |
| Playwright e2e            | Den **gebauten Container**: was ausgeliefert wird                            | Langsam; braucht den ganzen Stack                                     |
| Ratschen/Gates            | Dass ein behobener Fehler behoben **bleibt**                                 | Müssen einmal rot gesehen worden sein, sonst prüfen sie nichts        |
| Drift-Gates               | Dass zwei Wahrheiten nicht auseinanderlaufen (Swagger, Migrationen, Compose-Variablen, Dokumentzahlen, Fundstellen, Stand-Angaben) | Erkennen nur die Drift, die sie kennen |

**Die Skip-Bilanz ist Teil der Strategie.** Am 10.08.2026 stand acht Stunden ein Test auf
`main`, der das Gegenteil der geltenden Entscheidung behauptete: Lokal übersprang er sich,
und in CI fiel er in einen Lauf, in den niemand mehr schaute. Der pre-push-Hook zieht
deshalb keine Datenbank hoch (das gehört nicht in einen Push), **sagt aber, was er nicht
geprüft hat** — gezählt werden die Dateien, nicht die Skips, weil `go test` ohne `-v` keine
Skip-Gründe druckt und ein Zähler auf der Ausgabe immer auf 0 stünde.

Die Landkarte der Ratschen in [sweeps.md](../sweeps.md) nennt zu **jeder** Ratsche auch,
was sie systembedingt **nicht** sieht. Das ist der Teil, den man nur einmal aufschreibt,
wenn man ihn einmal gebraucht hat.

---

## 8.15 Abhängigkeits-Hygiene

- **Go:** `go mod verify` in CI, patch-genaue Toolchain, Paarung `go.mod` ↔ `Dockerfile`
  über ein Gate. Keine Build-Tags mehr (der frühere `//go:build odbc` ist mit dem Werkzeug
  entfallen — der Littera-Altbestand kommt über `mdb-export`-CSVs und braucht kein
  `unixODBC`).
- **npm:** `npm ci` (Lockfile verbindlich, **nicht** `npm install`), `overrides` für
  transitive Fixes, `npm audit` in CI und im Hook.
- **Schwachstellen:** `govulncheck` mit einem Gate, das **benannte, begründete** Ausnahmen
  kennt (`security/vuln-ausnahmen.json`) — nicht ein globales Abschalten. Ein Befund, der
  begründet keiner ist, wird am Code nachgemessen und dokumentiert (Beispiel: die
  Schwachstelle im Excel-Leser, die diesen Code nicht erreicht;
  `pkg/xlsxgrenze` hält die Grenze fest).
- **Dependabot** (`.github/dependabot.yml`) gruppiert Updates; die Gates entscheiden, ob
  sie durchgehen. Achtung: Dependabot hebt bei einem Go-Bump nur `go.mod` — die
  `Dockerfile`-Zeile muss mitziehen, und genau das erzwingt
  `docs/umgebung_paritaet_test.go`.
- **Erkenntnisse aus Sicherheitsbefunden** werden als Lehrsatz festgehalten
  (`.jules/sentinel.md`): `os.OpenRoot` statt Pfad-Strings, `.pgpass` statt `PGPASSWORD`,
  Kopfzeilen-Sanitisierung an der Schreibstelle. Das ist die Form, in der eine Lehre den
  nächsten Fall erreicht.
