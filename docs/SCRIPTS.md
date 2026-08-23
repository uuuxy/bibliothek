# Kommandozeilen-Skripte und Werkzeuge

---

## 1. LITTERA-Altbestand (`cmd/littera-altbestand`)

Überträgt den Littera-Altbestand nach PostgreSQL. Quelle sind `mdb-export`-CSVs, nicht
die Access-Datei selbst: Deren ODBC-Treiber gibt es nur unter Windows.

```bash
# 1. Export erzeugen (mdbtools, plattformunabhängig)
for t in Titel Exemplar Verlag Medienart Personen Personen_Zuordnung Leser Leser_UG Verleih; do
  mdb-export littera_sav.mdb "$t" > "littera-export/$(echo "$t" | tr 'A-Z' 'a-z').csv"
done

# 2. Trockenlauf – liest und berichtet, schreibt nichts
go run ./cmd/littera-altbestand -csv ./littera-export -trocken

# 3. Übernahme
go run ./cmd/littera-altbestand -csv ./littera-export -db "$DATABASE_URL"
```

**Was übernommen wird**, steuern `-bestand` (Vorgabe an), `-personen` und `-ausleihen`
(Vorgabe aus). Die Vorgabe ist Absicht: Der geprüfte Export ist ein Stand von 2010
(Schüler-Geburtsjahre 1989–2001, letzte Ausleihe 2010). Personen und Ausleihen daraus
legen Schüler an, die heute Mitte dreißig sind, und melden ein Viertel des Bestands als
verliehen. Für einen aktuellen Export sind beide Schalter richtig.

**Barcodes:** `-barcodes littera` (Vorgabe) schreibt den Wert, den ein Scanner vom
Buchetikett liest — eine **EAN-13**, nicht die daneben gedruckte Exemplarnummer:

```
1 0 5 7 8 5   0   0 3 9 5   6   7      Exemplar-Nr. 105785 → 1057850039567
└─ Exemplarnr └─ Bibl.-Nr ─┘   └ Prüfziffer
   6-stellig       0395
```

Gemessen an zwei echten Büchern der Schule, Prüfziffer jeweils verifiziert (siehe
`littera.EtikettBarcode`). Der Bestand bleibt damit ohne Neubeklebung scannbar. Eine
zweite Etikettengeneration gibt es nicht mehr: Die alten Littera-Aufkleber (Zeichenkette
für eine Barcode-Schrift, `8 *pkpööp#-c.bc-*`) sitzen auf Büchern, die nicht mehr im
Regal stehen — bestätigt am 04.08.2026.
`-barcodes neu` vergibt stattdessen frische `B-XXXXX` aus `barcode_seq` — derselben
Sequenz, aus der die Anwendung ihre Barcodes zieht — und setzt voraus, dass jedes Buch
ein neues Etikett bekommt.

Trägt Litteras Tabelle `FremdBarcode` für ein Exemplar ein Ersatzetikett, gewinnt das
gegen die gerechnete EAN-13. Dasselbe gilt für Schülerausweise: `FremdLeserNummer` hält
die Nummer des Kartenherstellers (`B97601826457`), und die steht in keinem Stammdatenfeld.
Beide Dateien sind optional; fehlen sie, rechnet der Import mit den Littera-Nummern.

**Härtung:** Ein Savepoint je Datensatz (`internal/uebernahme`), Postgres-Fehler nach
SQLSTATE eingeordnet, Abwertungen (verworfene ISBN, gekürztes Feld) getrennt von
Ausfällen protokolliert nach `littera_import.log`. Nach jedem Abschnitt wird die gemeldete
Zahl gegen den tatsächlichen Zeilenzuwachs gehalten.

**Wiederholung:** Ein zweiter Lauf wird abgelehnt — es gibt keinen natürlichen Schlüssel,
an dem Postgres die Dublette erkennen könnte. Zum Aufräumen:

```sql
DELETE FROM ausleihen a USING buecher_exemplare e
  WHERE a.exemplar_id = e.id AND e.erweiterte_eigenschaften ? 'littera_id';
DELETE FROM buecher_exemplare WHERE erweiterte_eigenschaften ? 'littera_id';
DELETE FROM buecher_titel     WHERE erweiterte_eigenschaften ? 'littera_id';
DELETE FROM ausleihen WHERE schueler_id IN (SELECT id FROM schueler WHERE lusd_id LIKE 'littera:%');
DELETE FROM schueler  WHERE lusd_id LIKE 'littera:%';
DELETE FROM benutzer  WHERE rolle = 'kollegium' AND email LIKE '%@littera.invalid';
```

> Hier stand bis zum 11.08.2026 `rolle = 'lehrer'`. Seit Migration 069 gibt es diesen
> Enum-Wert nicht mehr, und Postgres bricht den **Vergleich** ab, nicht nur den Treffer:
> `ERROR: invalid input value for enum benutzer_rolle: "lehrer"`. Die fünf Anweisungen
> davor laufen durch — wer den Block einfügt, räumt also alles auf **außer** den
> Littera-Lehrkraftkonten und sieht am Ende einen Fehler, der nach „nichts passiert"
> aussieht.

**Lehrkräfte** werden mit `aktiv = true` angelegt — die Ausweis-Abfrage der Omnibox
filtert darauf, eine inaktive Lehrkraft wäre am Scanner unauffindbar. Den Login sperrt
statt dessen die Adresse: Anmeldung geht ausschließlich über IMAP gegen den Schul-Mail-
server, und `littera-4908@littera.invalid` gibt es dort nicht. `-lehrer-inaktiv` kehrt das
um, macht die Karten aber wertlos.

**Rückgabewerte:** 0 vollständig · 1 abgebrochen · 2 unvollständig (Details im Protokoll).

> Das frühere `cmd/littera_migration` ist entfallen. Es fragte `SELECT TitelID, Titel,
> Autor, ISBN, Verlag, Jahr, Signatur FROM TITEL` ab — von diesen Spalten existiert in
> einer echten Littera-Datei einzig `ISBN`. Es ist nie gegen echte Daten gelaufen.

### 1a. Katalogisat aus dem MAB2-Export (`cmd/littera-import`)

Der zweite Weg in denselben Bestand — und für den Titelkatalog inzwischen der bessere.
Quelle ist kein Access-Backup, sondern ein **MAB2-Katalogisat (XML)**, das Littera selbst
ausgibt. Der Unterschied ist die Aktualität: Die vorliegende `littera_sav.mdb` ist ein
Stand von 2010, `katalogisat.xml` von Juni 2026 mit 13.708 Titeln.

```bash
# Trockenlauf gibt es hier nicht — der Import läuft in EINER Transaktion,
# ein Fehler rollt alles zurück. Vorher ein Backup ziehen.
go run ./cmd/littera-import -file katalogisat.xml -db "$DATABASE_URL"
```

- `-file` (Pflicht): Pfad zur Katalogisat-XML. `-db` fällt auf `$DATABASE_URL` zurück.
- Läuft über **denselben** Service-Pfad wie `POST /api/import/littera` — es gibt also
  keine zweite Importlogik, die eigene Fehler machen könnte.
- **Keine Dubletten bei Re-Imports:** Titel werden über ISBN oder Titel gegen den Bestand
  gematcht. Der Lauf ist damit wiederholbar.
- Signaturen landen in `buecher_titel.signatur` (der echten Spalte), LMF-Bestand wird über
  das Präfix `LMF-` geflaggt.
- Zeitlimit 10 Minuten: ~15.000 Titel brauchen gegen eine entfernte Datenbank Minuten,
  nicht Sekunden.
- **Rückgabewerte:** 0 erfolgreich · 1 abgebrochen (fehlende Datei, DB nicht erreichbar,
  Importfehler). Die Zahl verarbeiteter Titel steht als `verarbeitete_titel` im JSON-Log.

---

## 2. Foto-Migration (`cmd/migrate-fotos`)

Migriert unverschlüsselte Bilddateien vom Dateisystem in die Datenbank.

- **Funktionsweise:** Iteriert über ein Verzeichnis mit Schülerfotos, validiert und verschlüsselt diese (AES-256-GCM), speichert sie als `BYTEA` in `schueler_fotos`, **liest sie zur Gegenprobe zurück** und **löscht die Quelldatei erst danach**.
- **Zweck:** Konsolidierung der Infrastruktur (kein separates Foto-Verzeichnis) + Datensicherheit.

**Warum es selbst aufräumt (seit 23.08.2026).** Bis dahin sagte das Werkzeug nur „Du
kannst das Verzeichnis `uploads/fotos` jetzt sicher löschen" — ob das jemand tat, wusste
niemand. Was liegen blieb, sind unverschlüsselte Schülerfotos unter `/uploads/`, einem
Pfad, der bewusst ohne Anmeldung lesbar ist (Cover für Katalog und Monitor), und ihre
Dateinamen sind die Barcode-IDs vom Schülerausweis — also vollständig aufzählbar. Ein
Hinweis auf der Konsole ist für diesen Zustand die falsche Sicherung.

Gelöscht wird **erst nach bestandener Gegenprobe**: Das eben geschriebene Foto wird
zurückgelesen, entschlüsselt und mit dem Original verglichen. Bis dahin ist die Datei die
einzige Kopie; ein „INSERT ohne Fehler" heißt noch nicht, dass sich das Bild je wieder
anzeigen lässt. Scheitert die Probe, bleibt die Datei liegen und der Lauf meldet es.

Am Ende sagt das Werkzeug, wie viele Dateien es entfernt hat — und wenn welche übrig
sind, warum das zählt und wie man sie los wird (`shred -u uploads/fotos/*.jpg`).

```bash
# Aufräumen ausdrücklich abschalten (die Quelldateien bleiben dann unverschlüsselt liegen):
FOTOS_BEHALTEN=1 docker compose exec backend ./migrate-fotos
```

> **Auf dem Schulserver prüfen:** Der Lauf von vor dem 23.08.2026 hat nichts gelöscht.
> `docker compose exec backend ls -la uploads/fotos` sagt, ob dort noch Altbestand liegt.

---

## 3. Datenbank-Backup (`jobs/backup.go` / `scripts/backup.sh`)

Es gibt **drei Wege**, und seit dem 23.08.2026 verschlüsseln alle drei über dieselbe
Ableitung (`internal/backupkrypto`, scrypt + AES-256-GCM). Hier stand bis zum 06.08.2026
eine gemeinsame Zeile für alle — sie beschrieb den automatischen Weg und ließ die
Shell-Wege sicherer aussehen, als sie waren.

| | Automatisch (`jobs/backup.go`) | Manuell (`scripts/backup.sh`) | Vor jedem Deploy (`./update.sh`) |
|---|---|---|---|
| Auslöser | Täglich 02:30 via `jobs/cron.go` | `./scripts/backup.sh` | Schritt 1 von `./update.sh` |
| Pipeline | `pg_dump → gzip → AES-GCM` | `pg_dump → gzip → AES-GCM` (in EINER Pipe) | `pg_dump → gzip`, AES-GCM in Schritt 5 |
| Verschlüsselt | **ja** (`BACKUP_ENCRYPTION_KEY`) | **ja** (seit 23.08.2026) | **ja, nach gesundem Deploy** (seit 23.08.2026) |
| Dateirechte | 0600 | 0600 (seit 06.08.2026) | 0600 (seit 06.08.2026) |
| Rotation | letzte 14 | 7 Tage (`.enc`) / 2 Tage (Klartext) | 30 Tage (`.enc`) / 2 Tage (Klartext) |
| Dateiname | `backup_<Zeitstempel>.sql.gz.enc` | `bibliothek_backup_<Datum>.sql.gz.enc` | `backup_<Zeitstempel>.sql.gz.enc` |

**Warum `update.sh` erst in Schritt 5 verschlüsselt.** Seine Vorab-Sicherung ist der
Rückweg für genau das Zeitfenster, in dem der neue Container nicht hochkommt — und in dem
damit auch das Verschlüsselungswerkzeug nicht erreichbar wäre. Sie bleibt deshalb im
Klartext, bis Gesundheits- und Commit-Prüfung bestanden sind (Schritt 4/4b); erst dann
verschlüsselt Schritt 5 sie und löscht den Klartext. Der dokumentierte Rollback-Weg
`gunzip … | psql` bleibt für dieses Fenster unangetastet.

**Der Rückweg wird bewiesen, nicht angenommen.** Bevor ein Klartext-Dump gelöscht wird
(und bevor `backup.sh` „erfolgreich" meldet), geht die fertige `.enc`-Datei durch das
`restore-backup` im Container. Die bloße Formprüfung genügt nicht: Läuft die Platte
während des Schreibens voll, trägt die abgeschnittene Datei ihre `BKDF`-Kennung und hat
plausible Größe — auffallen würde der Verlust erst beim Wiederherstellen.

**Klartext als Ausnahme.** Beide Skripte fallen darauf zurück, wenn die Verschlüsselung
nicht möglich ist (Backend-Container aus, `BACKUP_ENCRYPTION_KEY` nicht gesetzt, Image
älter als 23.08.2026) — kein Abbruch, denn ein lesbares Backup ist besser als keines. Sie
sagen es dann laut, und die Datei verfällt nach 2 Tagen. Wiederherstellung und
Restore-Probe: [resilience_and_recovery.md](resilience_and_recovery.md).

---

## 4. Deployment (`./update.sh`)

**Der Weg, der benutzt wird.** Auf dem Server:

```bash
git pull && ./update.sh
```

`git pull` zuerst und getrennt, weil `update.sh` sich sonst in der gerade laufenden
Fassung selbst aktualisiert.

Führt aus: Backup (siehe oben) → `git pull` → `docker compose up -d --build` →
Gesundheitsprüfung (Docker-Status **und** `/health` im Container) → alte Backups
aufräumen. Bei Fehlschlag: Abbruch mit Rollback-Anleitung, die auf das eben erzeugte
Backup zeigt.

`scripts/deploy.sh` ist der ältere, schlankere Weg (`git pull` →
`docker compose up -d --build` → prüft, ob der Domain-Block im Caddyfile steht, und
hängt ihn ggf. an). Er macht **kein** Backup und **keine** Gesundheitsprüfung.

`./update_caddy.sh` schreibt die Caddy-Konfiguration des Servers neu
(`/root/caddy/Caddyfile`, alle Dienste des Hosts) und startet Caddy neu. Die Datei
`Caddyfile` im Repo-Root ist nur eine Vorlage zum Nachschlagen — sie wird nirgends
ausgeliefert.

### `scripts/stack-neu.sh` — dasselbe lokal, mit Beweis

```bash
./scripts/stack-neu.sh
```

Baut den Entwicklungs-Stack (`docker-compose.local.yml`) neu und **belegt**, dass der
Container danach den aktuellen Stand ausliefert. `docker compose up -d --build` bricht
bei einem fehlgeschlagenen Build zwar ab, lässt aber den **alten** Container
weiterlaufen — wer nur die letzte Ausgabezeile liest, sieht „Container Started" und
misst anschließend eine Fassung, die es im Repo nicht mehr gibt (am 10.08.2026 war die
e2e-Suite deshalb grün für einen Knopf, den es nicht mehr gab).

Der Beweis läuft über den Bundle-Hash: Vite hängt an jeden Bundle-Namen einen Hash über
den Inhalt. Stimmt der Name des ausgelieferten Bundles mit dem des lokalen Builds
überein, liefert der Container exakt diesen Stand. Zeitstempel und Image-IDs
beantworten die Frage **nicht** — sie ändern sich auch ohne Inhaltsänderung und bleiben
gleich, wenn ein Cache-Layer den alten Stand konserviert.

Das Skript legt denselben `GIT_COMMIT` ins Image wie `update.sh` auf dem Server, sodass
auch lokal jederzeit gilt:

```bash
docker exec bibliothek-backend-local printenv GIT_COMMIT
```

---

## 5. Concurrency-Lasttest (`cmd/stresstest`)

Simuliert Race Conditions für parallele Barcode-Scans.

```bash
go run cmd/stresstest/main.go -port 8084
```

- Feuert via `sync.Cond` + Goroutinen zeitgleich Dutzende Requests gegen `/api/action`
- Zweck: Verifikation der Transaktionssicherheit (FOR UPDATE + Unique Partial Index)

---

## 6. Paket-Utilities (`pkg/`)

### `pkg/csvutil`
CSV-Formel-Injection-Schutz (OWASP CWE-1236):
```go
import "bibliothek/pkg/csvutil"

safeRow := csvutil.SanitizeRow([]string{titel, autor, ...})
```
Setzt Apostroph-Präfix bei Zellen die mit `= + - @ \t \r \n` beginnen.

### `pkg/imageutil`
Decompression-Bomb-Guard:
```go
import "bibliothek/pkg/imageutil"

if err := imageutil.GuardImageDimensions(r.Body, 50_000_000); err != nil {
    // Bild zu groß oder ungültig
}
```
Liest nur den Bild-Header (`image.DecodeConfig`) — ohne volle RAM-Allokation. Limit: 50 Megapixel.

---

## 7. Übrige Skripte in `scripts/` (Kurzübersicht)

Bis zum 05.08.2026 beschrieb dieses Dokument nur die großen Werkzeuge; die folgenden
Skripte lagen undokumentiert im Verzeichnis. Sie sind bewusst kurz gehalten — der
ausführliche Kommentar steht jeweils im Dateikopf.

### Qualitäts-Gates (lokal, es gibt dafür keinen CI-Job)

| Skript | Zweck |
|---|---|
| `api_inventar.sh` | Erzeugt `docs/api_inventar.md`: alle registrierten Go-Routen, alle `/api/`-Aufrufer im Frontend und den Abgleich in beide Richtungen (tote Handler / Geister-Aufrufe). |
| `deadcode_gate.sh` | Gate gegen unerreichbaren Go-Code (`x/tools/cmd/deadcode`), Erreichbarkeit ab allen `main`-Paketen; nur von Tests erreichter Code zählt mit, begründete Ausnahmen stehen in `deadcode_baseline.txt`. Tote **Interface**-Methoden sieht das Werkzeug nicht — dafür läuft `tote_tueren_test.go` in jedem `go test`. |
| `sonar_scan.sh` | SonarQube-Analyse **inklusive beider** Coverage-Berichte (Go + Frontend-lcov; seit 23.08.2026 erzeugt Schritt 2 `npm run test:coverage` und bricht bei roten Frontend-Tests ab). Ein bloßer `sonar-scanner`-Aufruf lädt keine Coverage hoch — fehlende Coverage zählt dort als 0 %. Braucht `SONAR_TOKEN` in der Umgebung (nie als `-Dsonar.token=`, das stünde in `ps`). **Vorher `TEST_DATABASE_URL` setzen** — siehe unten, sonst misst der Lauf rund 13 Punkte zu niedrig. |
| `install-hooks.sh` | Installiert `scripts/git-hooks/` (pre-commit, pre-push) in `.git/hooks`. |
| `backup_krypto.sh` | Kein eigenständiges Skript, sondern der gemeinsame Verschlüsselungs-Helfer von `backup.sh` und `update.sh` (`source`). Prüft, ob verschlüsselt werden kann, reicht Daten durch `cmd/encrypt-backup` im Backend-Container und beweist am fertigen `.enc` den Rückweg über `restore-backup`, **bevor** ein Klartext-Dump gelöscht wird. |
| `../security-scan.sh` | Sammel-Scan im **Repo-Root**: `gosec` (SAST), `trivy fs` (Abhängigkeiten/Konfiguration), OWASP-ZAP-API-Scan gegen `/swagger/doc.json`. Der ZAP-Teil braucht einen laufenden Server, Docker und `ADMIN_TOKEN` — er ist kein stiller Durchläufer, sondern eine bewusste Sitzung. |

#### Warum die Coverage niedriger aussieht, als sie ist

Gemessen am 06.08.2026, dreimal dieselbe Codebasis:

| Lauf | Gesamtabdeckung |
|---|---|
| `go test ./...` ohne Datenbank | **32,5 %** |
| … mit `TEST_DATABASE_URL` | **45,2 %** |
| … und ohne die Fremddatei aus `node_modules` | **45,9 %** |

Zwei Messfehler, kein Codefehler:

1. **58 Dateien `*_pg_test.go` überspringen sich ohne Datenbank** — still, mit „ok" in
   der Ausgabe. Ihr Produktivcode zählt dann als ungedeckt. Das sind rund 13 Punkte.
2. **`frontend/node_modules/flatted/golang/pkg/flatted/flatted.go`** ist eine fremde
   Go-Datei in einem JS-Paket. `go list ./...` führt sie als Projektpaket — Go kennt
   `node_modules` nicht als Sonderfall. 115 ungedeckte Zeilen im Profil.
   `sonar_scan.sh` filtert sie seit dem 06.08.2026 über `go list | grep -v node_modules`.

**100 % sind kein Ziel und wären kein gutes.** Der Rest verteilt sich so: `cmd/*`
(sechs CLI-Werkzeuge, 0 %) und `internal/smtptest` (Testserver, wird von Tests benutzt
statt getestet) sind strukturell ungedeckt und sollen es bleiben. Das Quality Gate misst
deshalb **neuen** Code gegen 80 % — die richtige Frage ist nicht „wie hoch ist die Zahl",
sondern „ist das, was ich gerade geändert habe, abgesichert".

Echte Zahlen erzeugen:
```bash
docker run -d --name biblio-test-pg -e POSTGRES_PASSWORD=test \
  -e POSTGRES_DB=bibliothek_test -p 55432:5432 postgres:16-alpine
export TEST_DATABASE_URL="postgres://postgres:test@localhost:55432/bibliothek_test?sslmode=disable"
SONAR_TOKEN=sqp_… ./scripts/sonar_scan.sh
docker rm -f biblio-test-pg
```
Der DB-Name **muss** „test" enthalten (Sicherheits-Notbremse in `pgtest_support_test.go`
vor dem `DROP SCHEMA`).

### Datenbank-Helfer

| Skript | Zweck | Vorsicht |
|---|---|---|
| `seed_demo.sql` | Realistischer Demo-Datensatz für Pilot und Schulung. | Nur auf Test-/Demo-Datenbanken. |
| `seed_loadtest.sql` | Datenbestand für den k6-Lasttest. | Nur auf Wegwerf-Datenbanken. |
| `tabula_rasa.sql` | Bereinigt die Datenbank für den Echtbetrieb (Bewegungsdaten raus). | **Löscht Daten.** Vorher Backup. |
| `repair_titel_dubletten.sql` | Räumt Titel-Dubletten aus dem Import auf. | Vorher Backup, Ergebnis prüfen. |
| `repair_titel_ortssuffix.sql` | Entfernt Ortssuffixe aus Titelfeldern (Import-Artefakt). | Vorher Backup. |
| `repair_altbestand_etiketten.sql` | Einmalige Prod-Reparatur (16.08.2026): setzt `etikett_gedruckt` für importierten Altbestand, der physisch längst beklebt ist. Ohne sie zählte das Druck-Center-Badge den ganzen Altbestand als „Etikett offen“. | **Ändert Daten.** Vorher Backup; nur für Bestände, die aus Littera kamen und bereits Etiketten tragen. |
| `signatur_report.sql` | Report zur Signatur-Harmonisierung nach Littera-Import (Migration 038). | Nur lesend. |
| `e2e_altlasten.sql` | Entfernt den Bestands-Bodensatz der E2E-Suite (Titel mit Präfix `E2E `, deren Exemplare und Ausleihen). | **Löscht Daten.** Vorher Backup; Probelauf mit `ROLLBACK` statt `COMMIT` möglich. |

**Warum der Bestand von Hand aufgeräumt wird, Lieferanten aber automatisch:** Der globale
Teardown der E2E-Suite (`frontend/e2e/global-teardown.js`) räumt nach jedem Lauf
Lieferanten, Testbestellungen und Klassensatz-Reservierungen ab — dort kann ein zu weites
Muster wenig anrichten. Bestand und Ausleihen sind eine andere Klasse:
`ausleihen.exemplar_id` und `schadensfaelle.exemplar_id` stehen auf **RESTRICT**, ein
Löschen bräuchte also eine Kette über vier Tabellen, darunter die Ausleihhistorie. Eine
solche Kette automatisch nach jedem Lauf feuern zu lassen, ist das Aufgeräumtsein nicht
wert — ein Fehler im Zuschnitt löscht dort echte Daten. Gemessen am 12.08.2026: Ein
vollständiger Lauf hinterlässt rund 42 Titel, 75 Exemplare und 22 Ausleihen.

Benutzerkonten der Suite räumt seit dem 12.08.2026 die erzeugende Spec selbst weg
(`admin-mail-config.spec.js`, `afterAll`) — dort ist das Muster eindeutig und die
Fremdschlüssel stehen auf `SET NULL`.

### Einmal-Werkzeuge (`//go:build ignore`, per `go run` gestartet)

| Skript | Zweck |
|---|---|
| `import_isbns.go` | Nachträglicher ISBN-Import in bestehende Titel. |
| `monitor_stats.sh` | Protokolliert Systemkennzahlen über ~6 Stunden (Begleitung von Lasttests). |

> `scripts/migrate_photos.go` stand hier bis zum 11.08.2026. Es war ein Doppel von
> `cmd/migrate-fotos` (Abschnitt 2) und wurde entfernt — zwei Wege in dieselbe
> verschlüsselte Ablage, von denen nur einer gepflegt wurde.

### Testdaten-Generator (`cmd/seed`)

Füllt eine Datenbank mit Test-Admin, Schülern, Titeln und Exemplaren — die Vorstufe zum
k6-Lasttest (Abschnitt 5). Liest `DATABASE_URL` aus der Umgebung, kennt keine Flags und
fragt **nicht** nach, bevor es schreibt:

```bash
DATABASE_URL="postgres://…/bibliothek_test" go run ./cmd/seed
```

> **Nur auf Wegwerf-Datenbanken.** Das Werkzeug legt Massendaten an und prüft vorher
> nicht, ob die Zieldatenbank leer ist. Auf einem Echtbestand vermischen sich Testdaten
> unrettbar mit echten Schülern.

## `pruefe_secrets.sh` — Konfigurationsprüfung vor dem Deploy

```bash
./scripts/pruefe_secrets.sh                 # nutzt ./.env
./scripts/pruefe_secrets.sh /opt/bibliothek/.env
```

Liest eine `.env` und meldet die Fehlkonfigurationen, die **still** bleiben: bekannte
Default-Secrets aus dem Repository, fehlender `BACKUP_ENCRYPTION_KEY` (der nächtliche Job
überspringt sich dann kommentarlos), `IMAP_HOST=mock` (akzeptiert jedes Passwort),
ungesetztes `ENFORCE_PROD_SECRETS` oder `COOKIE_SECURE`.

Das Skript **ändert nichts**. Exit-Code 0 = sauber, 1 = kritischer Befund.

Bei einem Treffer auf `APP_ENCRYPTION_KEY` den Schlüssel **nicht einfach ersetzen** —
Schülerfotos und das SMTP-Passwort wären verloren. Der Weg mit Umschlüsselung steht in
[SECURITY.md](SECURITY.md#app_encryption_key-wechseln).

