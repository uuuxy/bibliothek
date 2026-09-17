# 2. Randbedingungen

Stand: 17.09.2026

Randbedingungen sind das, was **nicht zur Diskussion stand**. Sie erklären mehr von dieser
Architektur als jede Entwurfsvorliebe: Der Grund für die Omnibox, für IMAP als
Anmeldequelle und für die Nachsicht gegenüber alten Barcodes steht hier, nicht in
[Kapitel 4](04-loesungsstrategie.md).

---

## 2.1 Technische Randbedingungen

| #  | Randbedingung                                                                 | Hintergrund und Konsequenz                                                                                                                                                                               |
| -- | ----------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| T1 | **Go 1.27.1**, Version identisch in `go.mod` und `Dockerfile`                 | Zwei Versionen wären zwei Verhalten. Genutzt werden ausdrücklich neuere Fähigkeiten: Methoden-Routing im `net/http`-Mux (`GET /api/...`), `os.OpenRoot` gegen Path-Traversal, `slog` als Standard-Logger. |
| T2 | **Kein Web-Framework** — `net/http` mit `http.ServeMux`                       | Routing, Middleware-Kette und RBAC sind Eigenbau (`api/router.go`, `api/middleware.go`, `api/permission_middleware.go`). Preis: Die Kettenreihenfolge ist Handarbeit und braucht ein Gate (`routes_authz_coverage_test.go`). |
| T3 | **PostgreSQL 18** über `pgx/v5` (Pool), keine ORM-Schicht                     | SQL steht im Repository-Paket sichtbar da. Constraints sind ein Entwurfsmittel, nicht eine Absicherung „unten" — siehe [invarianten.md](../invarianten.md).                                                |
| T4 | **CGO_ENABLED=1** für das Hauptbinary                                         | `chai2010/webp` (Cover-Dekodierung) braucht CGO. Folge: Der Build braucht `build-base` im Builder-Image; die CLI-Werkzeuge werden dagegen mit `CGO_ENABLED=0` gebaut.                                      |
| T5 | **Svelte 5 (Runes), Tailwind 4, Vite — kein TypeScript**                       | Typsicherheit kommt über JSDoc + `svelte-check --fail-on-warnings`, nicht über `.ts`. Eine Umstellung wäre ein Umbau von 286 Komponenten und ist bewusst nicht erfolgt.                                    |
| T6 | **Ein Host, ein Prozess**                                                      | Der Prozess hält Zustand im Speicher: SSE-Abonnenten, Rechte-Cache, Rate-Limit-Zähler, Idempotenz-Warteschleife. Eine zweite Instanz hinter einem Load Balancer wäre **nicht** nur Konfiguration.         |
| T7 | **Anmeldung gegen den Schul-Mailserver (IMAP)**                                | Die Anwendung speichert **kein** Benutzerpasswort; eine Passwortspalte gibt es seit Migration 012 nicht. Folge: Die E-Mail **ist** die Identität, und `benutzer.email` schreiben zu dürfen heißt, ein Konto übernehmen zu können. |
| T8 | **Barcodes des Altbestands dürfen nicht neu geklebt werden**                   | Ausweise und Etiketten aus Littera tragen nackte Nummern ohne Präfix. Deshalb löst die Omnibox **ohne** Präfix der Reihe nach auf, und deshalb gibt es die Prüfzeichen-Nachsicht (`pkg/code39`).           |
| T9 | **Docker Compose hinter Caddy**, TLS per ACME                                  | Kein Kubernetes, kein Ingress-Controller. Die maßgebliche Caddy-Konfiguration liegt **auf dem Schulserver** (`/root/caddy/Caddyfile`, geschrieben von `update_caddy.sh`); die `Caddyfile` im Repo ist Vorlage zum Nachschlagen. |
| T10| **Der Betrieb läuft ggf. über HTTP im Schul-LAN**                              | Deshalb ist `COOKIE_SECURE` eine bewusst treffbare Betriebsentscheidung — Vorgabe `true` außerhalb lokaler Entwicklung, ein explizites `false` warnt laut, bricht aber nicht ab.                            |
| T11| **Zeitzone Europe/Berlin fachlich, UTC technisch**                              | Fristen pinnen `Europe/Berlin` im Code (`tagesEndeInSchulzeitzone`), der Cron-Zeitplan ist auf UTC genagelt (`cron.WithLocation(time.UTC)`) — sonst verschluckt die Zeitumstellung den 02:30-Backup-Job.  |
| T12| **Lizenz EUPL-1.2**                                                             | Jede Go-Quelldatei trägt den Lizenzkopf; `frontend/package.json` nennt dieselbe Lizenz. Abhängigkeiten müssen dazu passen.                                                                                 |

---

## 2.2 Organisatorische und fachliche Randbedingungen

| #  | Randbedingung                                                        | Konsequenz für die Architektur                                                                                                                                                       |
| -- | -------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| O1 | **Ein Entwickler, ein Betreiber — dieselbe Person**                   | Alles, was nur durch Disziplin funktioniert, funktioniert nicht. Daher: Gates statt Checklisten, Selbstprüfung statt Runbook-Gedächtnis, `OFFEN.md` als **einzige** Liste.            |
| O2 | **Der Betrieb läuft, während gebaut wird**                            | Migrationen müssen idempotent sein und beim Start von selbst laufen; ein Deploy darf offene SSE-Verbindungen nicht mit `os.Exit(1)` beenden (der Grund für den Broker-Umbau).         |
| O3 | **Lernmittelfreiheit ist Landesrecht**                                | Der Stichtag 31. Juli, die Ersatzwert-Staffel und der Bescheid-Weg (Konto statt Bargeld) sind Vorgaben, keine Produktentscheidungen. Die Staffel ist bewusst ein **Vorschlag mit Herleitung**, weil der Betrag im Ermessen der Schule liegt. |
| O4 | **~160 Lehrkräfte legt niemand von Hand an**                          | Selbstanmeldung über die Schuldomain ist Pflicht, die Freischaltung bleibt aber bei der Schule: IMAP beantwortet „wer bist du", nicht „darfst du rein".                               |
| O5 | **Anforderungsprotokoll des Medienzentrums (16.09.2026, zwölf Punkte)** | Das aktuelle Abnahme-Gate. Zwei Punkte wiegen architektonisch: alte Aufdrucke müssen lesbar bleiben, und mehrjährige Ausleihen an dasselbe Kind müssen möglich sein. Stand und Fragen: [OFFEN.md](../OFFEN.md) Abschnitt 9. |
| O6 | **Kein Passwort-Selbstservice, kein Nutzerverzeichnis**                | Es gibt keinen „Passwort vergessen"-Pfad und keine Registrierung außer der Selbstanmeldung — beides liegt beim Schul-IT-Betrieb.                                                       |
| O7 | **Altbestand aus Littera muss verlustfrei übernommen werden**          | Übernahme als eigenes Kommando gegen dieselbe Datenbank, mit Savepoint **je Datensatz** und Abgleich gegen den tatsächlichen Zeilenzuwachs — ein abgebrochener Batch darf nicht alles mitnehmen. |
| O8 | **Datenschutz ist nachweispflichtig, nicht nur einzuhalten**            | Die PII-Einstufung je Route ist ein Dokument **mit Gate** (`api/pii_matrix_test.go`), nicht eine Zusage.                                                                              |

---

## 2.3 Konventionen

| Konvention                                        | Regel                                                                                                                                                                                                              | Durchsetzung                                                        |
| ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------- |
| **Fachsprache deutsch**                           | Domänenbegriffe im Code deutsch (`ausleihen`, `leser`, `vormerkungen`, `Abgaenger`, `Ersatzwert`). Englisch nur, wo es aus der Frühzeit stammt (`book`, `loan`, `student`). Gemischt, aber nie übersetzt-doppelt.   | Review; [Glossar](12-glossar.md)                                     |
| **Migrationen**                                   | `NNN_beschreibung.sql`, idempotent (`IF NOT EXISTS` / `DO $$ … EXCEPTION`), dedupliziert über `schema_migrations`; die Seed-Liste in `schema.sql` muss exakt den Dateien entsprechen.                               | `db/migrations_drift_test.go`, `db/migrations_nummern_test.go`, `db/migrations_schema_paritaet_pg_test.go` |
| **Rows-Iteration**                                | Jede `rows.Next()`-Schleife endet mit `rows.Err()`. Ohne das gilt ein Verbindungsabbruch mitten in der Iteration als Erfolg — die Liste wäre still unvollständig.                                                   | `golangci-lint`, Review                                              |
| **Komponenten-Größe Frontend**                    | ≤ 200 Zeilen je **neuer** `.svelte`-Datei; der Altbestand darüber darf nicht wachsen.                                                                                                                              | Ratsche `frontend/src/lib/frontend-hygiene-dateigroesse.test.js`     |
| **Eine Wahrheitsquelle für Menü und Router**      | Welche Seite eine Rolle erreicht, entscheidet `canSeeItem()` in `frontend/src/lib/menu.js` — und nur diese Funktion.                                                                                                | `frontend/e2e/menue-fuehrt-irgendwohin.spec.js`                      |
| **Autorisierung pro Route**                       | Kein globaler Auth-Filter: jede nicht-öffentliche Route trägt `RequirePermission(...)` oder `RequireRoles(...)`; öffentliche Routen stehen in einer Allowlist.                                                      | `api/routes_authz_coverage_test.go`                                  |
| **Fundstellen beim Namen**                        | In der Dokumentation werden Constraint-, Index-, Datei- und Paketnamen genannt — keine Zeilennummern und keine Migrationsnummern als Beleg (eine Datei existiert weiter, auch wenn eine spätere Migration sie aufhebt). | `docs/invarianten_fundstellen_test.go`                               |
| **Stand-Angaben**                                 | Kein Dokument behauptet im Kopf einen Stand, unter dem es jüngere Vorgänge beschreibt.                                                                                                                             | `docs/stand_angaben_test.go` (greift auch für `docs/arc42/*.md`)     |
| **Keine Changelog-Datei**                         | Die Commit-Historie ist Teil der Dokumentation; Erledigtes wird aus `OFFEN.md` gelöscht, nicht archiviert.                                                                                                          | Entscheidung vom 15.09.2026                                          |
| **Jedes Gate einmal rot gesehen**                 | Ein Detektor, dessen Aussage nicht verloren gehen kann, prüft nichts. Neue Gates werden gegen den echten Fehlerfall gehalten, bevor sie grün bleiben dürfen.                                                        | [sweeps.md](../sweeps.md) Regel 2; „Rot-Beweis-Battery"              |

---

## 2.4 Werkzeuge, die zur Architektur gehören

Diese Werkzeuge sind keine Beigabe: Ohne sie wären mehrere Entscheidungen in
[Kapitel 9](09-architekturentscheidungen.md) nicht tragbar.

| Werkzeug                                | Rolle in der Architektur                                                                                            |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `golangci-lint` (`.golangci.yml`)       | Steht in CI **vor** allem anderen. Ein ungenutzter Typ hat am 12.09.2026 Tests, Deadcode-Gate und Restore-Probe mit übersprungen — seither kennt auch der pre-push-Hook den Linter. |
| `scripts/deadcode_gate.sh` + Baseline   | Unerreichbarer Code ist eine Fehlerquelle: `Blacklist.Stop()` war geschrieben, getestet und von niemandem aufgerufen. |
| `govulncheck` + `scripts/govulncheck-gate.sh`, `security/vuln-ausnahmen.json` | Schwachstellen-Gate mit **benannten, begründeten** Ausnahmen statt globalem Abschalten.               |
| `gosec`, CodeQL, Trivy, `npm audit`      | Vier unabhängige Blickwinkel; Befunde, die begründet keine sind, werden dokumentiert (`.jules/sentinel.md`).          |
| Playwright gegen den **gebauten Container** | e2e misst, was ausgeliefert wird — nicht einen Dev-Server.                                                       |
| `internal/pgtest`, `*_pg_test.go`        | Constraints kann man nicht mocken. Ohne `TEST_DATABASE_URL` überspringen sie sich still — deshalb die Skip-Bilanz.    |
| `internal/smtptest`, `internal/pdftest`  | Mail- und PDF-Pfade werden gegen eine Attrappe bzw. am erzeugten Dokument geprüft, nicht am Aufruf.                  |
| `scripts/api_inventar.sh`                | Erzeugt [api_inventar.md](../api_inventar.md): alle Go-Routen gegen alle Frontend-Aufrufer, in **beide** Richtungen.  |
