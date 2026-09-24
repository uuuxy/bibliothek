# 9. Architekturentscheidungen

Stand: 24.09.2026

Vierundzwanzig Entscheidungen, die diese Architektur tragen. Format je Eintrag:
**Entscheidung — Anlass — Folge — Fundstelle.** Wo eine Entscheidung eine längere
Geschichte hat, steht sie knapp; ausführlicher ist die Commit-Historie, und die kann nicht
veralten.

> **Warum das hier steht:** Mehrere dieser Entscheidungen sehen von außen wie ein Fehler
> aus und sind das Ergebnis eines echten Vorfalls. Wer sie „aufräumt", baut den Vorfall
> zurück ein. Die betroffenen Einträge sind mit ⚠️ markiert.

---

## Überblick

| #   | Entscheidung                                                                   | Wann          | Status   |
| --- | ------------------------------------------------------------------------------ | ------------- | -------- |
| A1  | Geschichteter Monolith, `net/http` statt Framework                             | von Anfang an | gültig   |
| A2  | Anmeldung gegen IMAP, keine Passwortspalte                                     | Migration 012 | gültig   |
| A3  | Autorisierung pro Route; `RBACBlockMiddleware` entfernt                        | —             | gültig   |
| A4  | UUID-Prüfung hinter das Routing verlegt                                        | 01.08.2026    | gültig   |
| A5  | Die Datenbank ist die letzte Instanz (partielle Unique-Indizes)                | Migration 033 | gültig   |
| A6  | ⚠️ SSE-Broker ohne Event-Loop                                                  | 11.08.2026    | gültig   |
| A7  | Feste Sperrreihenfolge Schüler → Ausleihe → Exemplar                           | 15.09.2026    | gültig (🟡) |
| A8  | Idempotenz-Keys als Vertrag der Schreibtüren                                   | —             | gültig   |
| A9  | Rechte-Cache mit Epochenzähler                                                 | 19.08.2026    | gültig   |
| A10 | Eine Tabelle `leser`, `schueler` als Sicht                                     | Migr. 123–125 | gültig   |
| A11 | `kollegium` ist Grundzustand, keine Rolle; Rechte fest im Seed                 | 16.09.2026    | gültig   |
| A12 | Rolle `leitung` wird **abgeleitet**, nicht abgeschrieben                       | Migration 122 | gültig   |
| A13 | Eine Vorsilbe `A-` für alle Ausweise — die Vorsilbe selbst bleibt              | 16.09.2026    | gültig   |
| A14 | Code 128 ohne Prüfzeichen, mit Nachsicht für alte Aufdrucke                    | 17.09.2026    | gültig   |
| A15 | Secret-Guard von selbst scharf                                                 | 05.09.2026    | gültig   |
| A16 | Cron auf UTC genagelt                                                          | 19.08.2026    | gültig   |
| A17 | Backup: nur die Datenbank, verschlüsselt, mit wöchentlicher Restore-Probe      | 11.07.2026    | gültig   |
| A18 | Mahnstufe steigt nur beim PDF-Druck                                            | —             | gültig   |
| A19 | Frist nach Tagen endet an einem Schultag; Ferien als Tabelle im Programm      | 24.09.2026    | gültig   |
| A20 | Schülerfotos verschlüsselt in der Datenbank, kein öffentliches Verzeichnis     | 08.08.2026    | gültig   |
| A21 | Offline-Sync über die Nachbuch-Tür statt über die Stapel-Tür                    | 16.09.2026    | gültig   |
| A22 | Kein TypeScript; JSDoc + `svelte-check`                                        | von Anfang an | gültig   |
| A23 | Swagger nur lokal; `api_inventar.md` ist das vollständige Verzeichnis           | —             | gültig   |
| A24 | Keine Changelog-Datei; `OFFEN.md` ist die einzige Liste                        | 15.09.2026    | gültig   |

---

## A1 — Geschichteter Monolith, `net/http` statt Framework

**Entscheidung.** Ein Deployable mit den Schichten Handler → Service → Repository,
geroutet über `http.ServeMux` mit Methoden-Mustern; Middleware-Kette von Hand.

**Anlass.** Eine Schule, ein Server, ein Betreiber. Die Korrektheit der Ausleihe hängt an
einer Transaktion über `leser`, `ausleihen` und `buecher_exemplare`.

**Folge.** Kein Framework-Update kann das Routing verändern; die Kettenreihenfolge ist
aber Handarbeit und braucht Gates (`routes_authz_coverage_test.go`). Ein Schnitt in
Dienste würde die Transaktionsklammer zerreißen, die heute Q1 trägt.

**Fundstelle.** `api/router.go`, `api/middleware.go`.

---

## A2 — Anmeldung gegen IMAP, keine Passwortspalte

**Entscheidung.** Zugangsdaten werden gegen den Schul-Mailserver geprüft. Die Anwendung
speichert **kein** Benutzerpasswort und hasht keines.

**Anlass.** Ein zweiter Passwortspeicher in einer Schule ist ein Risiko ohne Nutzen; die
Schule pflegt die Postfächer ohnehin.

**Folge.** Die **E-Mail ist die Identität**. Wer `benutzer.email` schreiben darf, übernimmt
damit ein Konto — ein Rechte-Audit, das nur auf `rolle` schaut, sieht diesen Weg nicht.
Daraus folgen A12 und der Eskalationsschutz. Es gibt keinen „Passwort vergessen"-Pfad.
Nebenwirkung: Der Brute-Force-Schutz schützt auch den **Mailserver** vor
Credential-Stuffing über diesen Weg. Ein Helfer braucht ein Schulpostfach.

**Fundstelle.** `auth/handlers.go` (`verifyIMAPCredentials`), Migration 012.

**Frühere Doku-Lage.** Bis zum 11.08.2026 stand in [FACHKONZEPT.md](../FACHKONZEPT.md)
„E-Mail und Passwort (Bcrypt-gehasht)" — das war nie so, und zwei Absätze weiter stand
bereits das Gegenteil.

---

## A3 — Autorisierung pro Route; `RBACBlockMiddleware` entfernt

**Entscheidung.** Jede nicht-öffentliche Route trägt `RequirePermission(...)` oder
`RequireRoles(...)`. Es gibt **keine** globale Autorisierungs-Middleware.

**Anlass.** Die frühere `RBACBlockMiddleware` führte eine hartkodierte Pfad-Allowlist für
einzelne Rollen. Sie **überstimmte** die konfigurierbare Rechtetabelle: Eine Lehrkraft
konnte im PermissionManager gewährte Rechte nicht nutzen.

**Folge.** Zwei Wahrheitsquellen wurden zu einer (`role_permissions`). Der Preis ist die
Vollständigkeitspflicht — deshalb das Coverage-Gate mit Allowlist für die bewusst
öffentlichen Routen.

**Fundstelle.** `api/permission_middleware.go`, `api/routes_authz_coverage_test.go`.

---

## A4 — UUID-Prüfung hinter das Routing verlegt

**Entscheidung.** `ValidateUUIDParamsMiddleware` sitzt **in** `RequirePermission`, nicht als globale
Middleware um den Mux.

**Anlass.** Audit-Befund vom 01.08.2026: Von außen um den Mux gelegt las die Middleware
`r.PathValue("id")`, bevor die Route aufgelöst war. Der Wert ist dort **immer leer** — die
Prüfung lief also nie.

**Folge.** Ungültige UUIDs werden abgewiesen, bevor sie Postgres erreichen (`pkg/kennung`
prüft genau die Schreibweise, die Postgres annimmt — **nicht** `uuid.Parse`, das auch
`urn:uuid:…` akzeptiert, was Postgres abweist).

**Fundstelle.** `api/middleware.go` (Kommentar an `ValidateUUIDParamsMiddleware`),
`pkg/kennung`.

---

## A5 — Die Datenbank ist die letzte Instanz

**Entscheidung.** Zwei aktive Ausleihen auf demselben Exemplar bzw. Gerät sind durch
**partielle Unique-Indizes** strukturell unmöglich; die Verletzung wird zu 409 gemappt.

**Anlass.** Acht Stationen. Der Scan eines Exemplars sperrt die **aktive Ausleihe**, nicht
das Exemplar — zwei Stationen mit verschiedenen Lesern greifen also auf keine gemeinsame
Zeile.

**Folge.** Der Index ist an dieser Stelle **der einzige** Schutz, nicht der Hosenträger zum
Gürtel. Er deckt zugleich den TOCTOU-Fall bei Idempotenz-Keys ab.

⚠️ **Wer ihn entfernt, verliert die Zusage von Q1** — und zwar ohne dass ein Test
außerhalb der PG-Integrationstests es merkt.

**Fundstelle.** `uniq_ausleihen_aktiv_exemplar`, `uniq_ausleihen_aktiv_geraet`,
`check_loan_item`, `mapLoanCreateErr`.

**Doku-Korrektur.** Bis zum 11.08.2026 stand in der Architekturübersicht, der Scan sperre
`buecher_exemplare`. Das tut nur der Schadens-Pfad.

---

## A6 ⚠️ — SSE-Broker ohne Event-Loop

**Entscheidung.** Der Broker hat **keine** eigene Goroutine: Der Zustand
(`map[chan string]struct{}`) liegt hinter einem `sync.RWMutex`, das Verteilen läuft in der
Goroutine des Aufrufers.

**Anlass.** Die frühere Bauweise hatte register-/unregister-Kanäle. Kehrte `Start` durch
den abgebrochenen Kontext zurück, las niemand mehr aus den Kanälen, jeder SSE-Handler blieb
in seinem `defer` stehen, `httpServer.Shutdown` wartete auf eben diese Handler bis zum
Timeout — und `main` endete mit `os.Exit(1)`. Bei dauerhaft verbundenen Arbeitsplätzen war
das **jeder** Deploy.

**Folge.** `Broadcast` sendet unter Lesesperre, `unsubscribe`/`shutdown` schließen Kanäle
unter Schreibsperre — beides kann sich nie überschneiden (ein Senden auf einen geschlossenen
Kanal würde den Prozess abbrechen). Rückstau wird übersprungen (Puffer 10, `select`/
`default`), Heartbeat alle 15 s.

⚠️ **Wer den „zentralen Event-Loop" wiederherstellt, baut den Deploy-Abbruch zurück ein.**

**Fundstelle.** `sse/sse.go`.

---

## A7 — Feste Sperrreihenfolge Schüler → Ausleihe → Exemplar

**Entscheidung.** Jeder Schreibpfad, der Zeilen sperrt, sperrt in dieser Reihenfolge.

**Anlass.** Online-Scan und Nachbuchen fassen dieselben Zeilen an. Zwei verschiedene
Reihenfolgen verklemmen sich gegeneinander.

**Folge.** Die Regel ist eingehalten, aber **nur Konvention (🟡)**: Ein Gate gibt es nicht.
Sie steht als Invariante im Katalog und in den Kommentaren der beteiligten Funktionen.

**Fundstelle.** `internal/service/loan_checkout.go`, `repository/loan.go`
(`StempleBewegungZum`), [invarianten.md](../invarianten.md) Abschnitt 1.

---

## A8 — Idempotenz-Keys als Vertrag der Schreibtüren

**Entscheidung.** Schreibende Scan-Vorgänge tragen einen Idempotenz-Schlüssel; die Antwort
wird gespeichert und bei Wiederholung zurückgegeben.

**Anlass.** Ein Scan kommt doppelt an (Netz, nervöse Hand, Nachbuchen). Eine zweite Buchung
oder eine zweite Sperrmeldung wäre falsch.

**Folge.** 5xx wird **nicht** gecacht (Retry bleibt möglich). Eine parallele Anfrage
desselben Schlüssels wartet begrenzt. Aufbewahrung 24 h, TTL-Lauf **stündlich** — bis zum
11.08.2026 stand in der Doku „täglich (24h-Cron)", die beiden Zahlen waren verwechselt.

**Fundstelle.** `repository/idempotenz.go`, `jobs/cron.go` (`17 * * * *`). Offline ist der
Schlüssel die `item.id` des Warteschlangen-Eintrags (`frontend/src/lib/stores/offlineSync.svelte.js`);
sie wandert beim Nachbuchen als `schluessel` mit (`api/nachbuchen_schluessel.go`) — eine
zweimal eingespielte Sicherung führt deshalb nichts doppelt aus.

---

## A9 — Rechte-Cache mit Epochenzähler

**Entscheidung.** Eine DB-Rechteentscheidung darf nur in den Cache, wenn seit ihrem
**Start** keine Invalidierung dazwischenkam.

**Anlass.** Nebenläufigkeits-Audit vom 19.08.2026: Ein Leser, der vor einer Rechteänderung
startete, schrieb den alten Stand **nach** der Invalidierung zurück — und er wirkte bis zu
60 s weiter.

**Folge.** `InvalidatePermissionCache()` leert den Cache **und** erhöht die Epoche; der
Schreiber vergleicht. 60 s Cache bleiben als Obergrenze für die Wirksamkeit einer
Rechteänderung.

**Fundstelle.** `api/permission_middleware.go`.

---

## A10 — Eine Tabelle `leser`, `schueler` als Sicht

**Entscheidung.** Schüler und Kollegium stehen in **einer** Tabelle `leser` mit der Spalte
`art` (`schueler` | `lehrkraft` | `liv`). `schueler` ist eine Sicht mit
`WHERE art = 'schueler'` und `WITH CHECK OPTION`. Ein gemeinsamer Ausweis-Nummernkreis.

**Anlass.** Lehrkräfte waren Entleiher über einen Umweg (`schueler.klasse = 'lehrer'`) und
standen teils doppelt in der Datei — Ausweis und Ausleihen am einen Eintrag, die Anmeldung
am anderen.

**Folge.** Jede Abfrage, die **wirklich** Schüler meint (Klassenlisten, LUSD-Abgleich,
Mahnlauf, Löschfristen), liest die Sicht. Preis ist die Bugklasse „Schreibpfad gegen
gefilterte Sicht": eine **stille 404** statt eines Fehlers. `chk_leser_nur_schueler_werden_abgaenger`
verhindert, dass ein Schüler seine Art wechselt; Lehrkraft ⇄ LiV ist erlaubt.

**Fundstelle.** Migrationen 123–125, `docs/schreibpfade_gegen_sicht_test.go`,
`db/sicht_schueler_vollstaendig_pg_test.go`. Der frühere Umweg ist seit Migration 072
geschlossen; `api/student_klasse_regel.go` weist `lehrer` als Klassennamen an beiden Türen
ab (`student_create.go`, `student_update.go`).

---

## A11 — `kollegium` ist Grundzustand, keine Rolle

**Entscheidung.** Der Admin vergibt **vier** Rollen (Admin, Leitung, Mitarbeiter, Helfer).
`kollegium` ist technisch derselbe Enum-Wert, fachlich aber der Grundzustand jeder
Lehrkraft. Was das Kollegium darf, steht **fest** in `db/seed.go`.

**Anlass.** Bis zum 16.09.2026 stand Kollegium als fünfte Spalte in der Rechte-Matrix und
sah dort wie eine Stufe in einer Rangfolge aus. Es ist keine: Jede Lehrkraft meldet sich
selbst an, wird freigeschaltet und ist damit erst einmal niemand Besonderes.

**Folge.** Der Rechte-Editor führt nur noch vier Spalten. Erteilt ist genau ein Recht
(`create_reservations`); „Mein Portal" hängt seit dem 26.08.2026 an **diesem Recht**, nicht
an der Rolle — eine Lehrkraft, die als Mitarbeiter in der Bibliothek mitarbeitet, sieht das
Portal ebenfalls.

**Historie, die man kennen muss:** Am 10.08.2026 sah ein Kollegiums-Konto auf dem
Schulserver **zehn von fünfzehn** Menüpunkten, darunter Schülerdatei, Mahnwesen,
System-Logs und Einstellungen — und es war keine reine Anzeigefrage: Dieselbe Tabelle
entscheidet in `RequirePermission`. Migration 070 hat alles außer `create_reservations`
entzogen.

**Fundstelle.** `db/seed.go`, `auth/selbstanmeldung.go`; Migrationen 042 (`helfer` kommt in
das Enum), 069 (`lehrer` → `kollegium`, weil das Wort doppelt belegt war), 070 (Rechte des
Kollegiums auf das Portal zurückgenommen), 121 (`leitung`).

---

## A12 — Rolle `leitung` wird abgeleitet, nicht abgeschrieben

**Entscheidung.** `leitung` = Rechte des Admins **minus** `manage_users` und
`manage_settings`. Migration 122 **erzeugt** die Zeilen aus den Admin-Zeilen und setzt genau
die zwei Ausnahmen auf `false`.

**Anlass.** Eine abgeschriebene Rechteliste wäre eine zweite Wahrheit neben `db/seed.go`
und liefe beim nächsten neuen Recht auseinander — die Leitung bekäme es nicht, und niemand
merkte es.

**Folge.** Das Gate `db/rolle_leitung_test.go` leitet dasselbe aus der Vorgabe ab.
`manage_users` fehlt **deshalb**, weil man mit dem Recht die E-Mail eines Kontos ändert und
die Anmeldung eine Person allein an ihrer E-Mail erkennt (A2): Die Rechtevergabe wäre der
Weg in jedes Konto. Ein Admin-**Konto** bleibt der Leitung auch mit dem Recht verschlossen.

**Fundstelle.** Migrationen 121/122, `db/rolle_leitung_test.go`,
`api/user_admin_eskalation.go`.

---

## A13 — Eine Vorsilbe `A-` für alle Ausweise; die Vorsilbe bleibt

**Entscheidung.** Neue Ausweisnummern tragen `A-`. `S-` und `L-` werden weiter **gelesen**.
Bücher `B-`, Geräte `G-`.

**Anlass.** Die beiden alten Vorsilben behaupteten etwas über die Person (`S-` aus
Handanlage/LUSD, `L-` aus dem Littera-Personenlauf). Seit A10 ist das die falsche Aussage:
Wer jemand ist, steht in den Stammdaten, nicht auf der Karte.

**Folge.** Nummern werden nicht recycelt, alte Karten funktionieren weiter — seit Migration 146
(24.09.2026) auch dann nicht, wenn die Nummer aus `leser` verschwindet: Die Tabelle
`ausweisnummern_ausgeschieden` hält sie als Zahl fest, und der Generator zählt über sie hinweg.
Eine Sequenz reichte dafür nicht, weil der LUSD-Lauf selbst weiterzählt, die Littera-Übernahme
Ersatznummern bildet und Nummern von Hand nur in der Tabelle stehen. **Die Vorsilbe
selbst bleibt** — und der Grund ist nicht offensichtlich: Ohne Netz ist sie die einzige
Information, an der die Theke einen Buchscan von einem Ausweisscan unterscheiden kann
(A21). Littera kommt ohne aus, weil dort Nummer und Scanwert zwei verschiedene Felder sind.

**Fundstelle.** `internal/service/omnibox_service.go`, `internal/service/vorsilben_zwilling_test.go`.

---

## A14 — Code 128 ohne Prüfzeichen, mit Nachsicht für alte Aufdrucke

**Entscheidung.** Gedruckt wird Code 128 **ohne** Prüfzeichen. Bleibt ein Scan ohne
Treffer, wird ein mögliches Code-39-Prüfzeichen abgeschnitten und **einmal** erneut
aufgelöst.

**Anlass.** Die Anwendung druckte Code 39 **mit** Prüfzeichen. Das Zeichen steht in den
Strichcode-Daten, und ein Lesegerät gibt es als Teil der Nummer zurück: Unter der Karte
stand `A-10003`, gescannt wurde `A-100037`. Der Server suchte eine Nummer, die es nicht
gibt — und weil ein unbekannter Scan nur eine leere Trefferliste erzeugt, sah es an der
Theke aus, als täte der Scanner **gar nichts**.

**Folge.** Die Nachsicht greift **nur** als zweiter Versuch: Bei 43 möglichen Zeichen sieht
im Schnitt jeder 43. gültige Code zufällig so aus, als hinge ein Prüfzeichen dran. Sie
sitzt **an einer** Stelle (in `ProcessQuery`, nicht in den einzelnen Zweigen), weil ein
alter Aufdruck jede Form haben kann — drei Stellen wären drei Gelegenheiten, eine zu
vergessen. Ein Fehler, der **kein** `ErrNotFound` ist, gilt ausdrücklich nicht als „ohne
Treffer".

**Offen bleibt:** Alle vor dem 17.09.2026 gedruckten Ausweise und Etiketten sind mit dem
Lesegerät weiterhin nur über diesen Weg nutzbar ([OFFEN.md](../OFFEN.md) 9.2).

**Fundstelle.** `pkg/code39`, `internal/service/omnibox_service.go`,
`frontend/e2e/barcode-lesbar.spec.js`.

---

## A15 — Secret-Guard von selbst scharf

**Entscheidung.** Außerhalb von `APP_ENV=local/development/test` verweigert der Server den
Start bei bekannten Beispiel-Geheimnissen. Nur ein ausdrückliches
`ENFORCE_PROD_SECRETS=false` schaltet das für eine Testphase ab — mit Warnung.

**Anlass.** Vorher musste die Prüfung **eingeschaltet** werden. Eine vergessene Zeile
reichte, damit der Schulserver mit dem Schlüssel aus dem Repository lief: Jeder mit
Repo-Zugriff hätte Admin-JWTs fälschen (JWT_SECRET) bzw. die verschlüsselten Schülerfotos
entschlüsseln können (APP_ENCRYPTION_KEY).

**Folge.** Die Liste der Beispiel-Geheimnisse steht **einmal**
(`api.IstBekanntesDefaultGeheimnis`) und wird von der Selbstprüfung mitbenutzt. Zwei Listen
wären genau die Fehlerart, gegen die die Selbstprüfung antritt: Sie meldete „alles gut",
während der Server aus demselben Grund den Start verweigert.

**Fundstelle.** `main.go/loadConfig`, `api/prod_geheimnisse.go`, `scripts/pruefe_secrets.sh`.

---

## A16 — Cron auf UTC genagelt

**Entscheidung.** `cron.New(cron.WithLocation(time.UTC))` — **nicht** `time.Local`.

**Anlass.** Mit `TZ=Europe/Berlin` hätte die Nacht der Sommerzeit-Umstellung den Backup-Job
`30 2 * * *` verschluckt: 02:30 existiert dann nicht.

**Folge.** `TZ` ist seither gefahrlos setzbar und hält Logs und Zeitstempel konsistent. Die
**fachliche** Zeitrechnung bleibt getrennt davon bei `Europe/Berlin` im Code.

**Fundstelle.** `jobs/cron.go`, `.env.example` (Abschnitt `TZ`).

---

## A17 — Backup: nur die Datenbank, verschlüsselt, mit Restore-Probe

**Entscheidung.** Gesichert wird ausschließlich die Datenbank: `pg_dump` → gzip →
AES-256-GCM (scrypt), Ablage im benannten Volume, optional Offsite zu S3. Wöchentlich wird
das jüngste Backup in eine **Wegwerf-Datenbank** eingespielt.

**Anlass.** Cover sind aus ISBN und Quelle reproduzierbar; Schülerfotos liegen
verschlüsselt **in** der Datenbank und sind damit im Dump enthalten. Ein Backup, das nie
eingespielt wurde, ist eine Hoffnung.

**Folge.** Das Ergebnis der Probe wird **Befund der Selbstprüfung** — es muss niemand
daran denken, nachzusehen. Die beiden Shell-Wege verschlüsseln seit dem 23.08.2026 über
dieselbe Ableitung wie der Job, ohne den Schlüssel je an den Host zu reichen. Das Passwort
erreicht `pg_dump` über eine temporäre `.pgpass`, **nicht** über `PGPASSWORD`. Ohne
`BACKUP_ENCRYPTION_KEY` überspringt sich der Job — und sagt das.

**Fundstelle.** `jobs/backup.go`, `internal/backupkrypto`, `jobs/restore_probe.go`,
[resilience_and_recovery.md](../resilience_and_recovery.md).

---

## A18 — Mahnstufe steigt nur beim PDF-Druck

**Entscheidung.** `mahnstufe` wird an **genau einer** Stelle erhöht: beim Erzeugen des
Mahn-PDFs. Der Mailversand erhöht sie nicht.

**Anlass.** Der Druck ist der physische Verwaltungsakt. Was nicht gedruckt wurde, ist nicht
gemahnt.

**Folge.** Ein Mailversand ist wiederholbar, ohne die Stufe zu verschieben. Am 11.08.2026
zeilenweise nachgeprüft: Es gibt wirklich nur diese eine Schreibstelle. Mahnlisten gehen an
die Klassenleitung, nie an Schüler.

**Fundstelle.** `api/mahnwesen_bulk.go`, [invarianten.md](../invarianten.md).

---

## A19 — Frist nach Tagen endet an einem Schultag; Ferien als Tabelle im Programm

**Entscheidung (24.09.2026).** Fällt eine Frist, die in Tagen zählt — Buch, Medium, Gerät,
Verlängerung —, auf ein Wochenende, einen Feiertag Hessens oder in die Ferien, gilt der
nächste Schultag. Die Ferien stehen als Tabelle im Programm, nicht als Datei, die die Schule
hochlädt. Stichtage (Lernmittel, LMF-Plan, Ferien-Leseclub), Fristen von Hand und die
Jahresfrist der Dauerleihe rücken nicht. Eine Ferien-Pause des Mahnwesens gibt es nicht; für
„Bücher über die Sommerferien mitnehmen" bleibt der **Ferien-Leseclub** (aktiv + Zieldatum ⇒
festes Rückgabedatum).

**Anlass.** Ausleihe am 24.09.2026 plus 21 Tage ist der 15.10., mitten in den Herbstferien.
Vom 06.09. bis zum 24.09.2026 galt „keine Ferienautomatik": Die Tabelle
`ferien_schliesszeiten` (Migration 017) hatte nie einen Schreiber, Banner und Sperre liefen
ins Leere, und sie ist mit Migration 102 ausgebaut. Littera führte Schließtage als Liste von
Hand; in der Sicherung von 2010 ist sie leer.

**Folge.** Das Kultusministerium legt die Termine vier Schuljahre im Voraus fest. Zwei Jahre
vor dem Ende der Tabelle melden es die Betriebsbereitschaft und zwei Horizont-Tests. Die
beweglichen Ferientage der Schule kennt die Tabelle nicht.

**Fundstelle.** `service.Tagesfrist` (`internal/service/loan_rules.go`),
`pkg/lmfplan/schulferien.go`, `pkg/lmfplan/ferien.go`, [FACHKONZEPT.md §2.1](../FACHKONZEPT.md).

---

## A20 — Schülerfotos verschlüsselt in der Datenbank, kein öffentliches Verzeichnis

**Entscheidung.** Passbilder liegen AES-256-GCM verschlüsselt in `schueler_fotos`. Das
Verzeichnis `uploads/fotos` wird **nicht** mehr angelegt.

**Anlass.** Es wurde bis zum 08.08.2026 bei jedem Start neu erzeugt, obwohl seit der
Foto-Migration nichts mehr hineinschrieb — unter `/uploads/`, das **bewusst ohne
Anmeldung** lesbar ist (Cover für Katalog und Monitor). Die Dateinamen waren die
Barcode-IDs von den Schülerausweisen, also vollständig aufzählbar.

**Folge.** `/uploads/` bleibt öffentlich lesbar (mit Vermerk in der Allowlist), enthält aber
keine Personendaten. Wer die Datei-Funktion je zurückholt, muss das Verzeichnis **bewusst**
anlegen und stolpert dabei über den Kommentar an dieser Stelle.

**Fundstelle.** `api/router.go` (`registerInventurSubmoduleRoutes`), `cmd/migrate-fotos`,
`internal/crypto`.

---

## A21 — Offline-Sync über die Nachbuch-Tür

**Entscheidung.** Die Warteschlange der Theke geht an `POST /api/action/nachbuchen`, nicht
an die Stapel-Tür `/api/action/batch`.

**Anlass.** Die Nachbuch-Tür war gebaut, geroutet und getestet — und niemand rief sie auf.
Sie kann, was der Stapel nicht kann: den **Scan-Zeitpunkt** buchen, einen Schlüssel genau
einmal buchen, einen offline gescannten Ausweis auflösen und jede Abweichung als Meldung
festhalten.

**Folge.** Neun Urteile je Eintrag, acht davon endgültig; `wiederholen` bedeutet
ausdrücklich das Gegenteil. Ein Schlüssel, den der Server **gar nicht** beantwortet hat,
bleibt liegen — bis zum 15.09.2026 galt Schweigen als erledigt. Die Uhr des Kiosk-Rechners
wird über `performance.now()` gegen die Wanduhr gehalten, weil sie oft genau dann gestellt
wird, wenn das Netz zurückkommt. Die Stapel-Tür blieb „eine Version länger" für Theken-Tabs mit
altem Stand (v3.0.0) und ist seit dem 22.09.2026 entfernt.

**Fundstelle.** `api/nachbuchen_handler.go`, `internal/service/nachbuchen.go`,
`frontend/src/lib/stores/offlineSync.svelte.js`.

---

## A22 — Kein TypeScript

**Entscheidung.** Das Frontend ist JavaScript mit JSDoc-Typen; geprüft wird über
`jsconfig.json` (`checkJs`) und `svelte-check --fail-on-warnings`.

**Anlass.** Der Nutzen wäre Typprüfung — und die gibt es so auch. Der Preis wäre der Umbau
von 286 Komponenten.

**Folge.** Die Typabdeckung hängt an der JSDoc-Disziplin; das Gate ist `npm run check` in
CI und im pre-push-Hook.

**Fundstelle.** `frontend/jsconfig.json`, `frontend/package.json`.

---

## A23 — Swagger nur lokal; `api_inventar.md` ist das vollständige Verzeichnis

**Entscheidung.** Die interaktive API-Doku wird **nur** bei `APP_ENV=local`/`development`
gemountet. Das vollständige Routenverzeichnis ist ein generiertes Dokument.

**Anlass.** Swagger deckt die **annotierten** Endpunkte ab (am 17.09.2026 63 Operationen auf 54
Pfaden von 206 registrierten Routen). Eine unvollständige Liste, die vollständig aussieht,
ist schlechter als keine.

**Folge.** `./scripts/api_inventar.sh` erzeugt [api_inventar.md](../api_inventar.md) und
gleicht Go-Routen gegen Frontend-Aufrufer in **beide** Richtungen ab.
`docs/swagger_drift_test.go` wird rot, sobald `docs.go` von den `@Router`-Annotationen
abweicht.

**Fundstelle.** `api/router.go` (`registerSwaggerRoutes`), `docs/swagger_drift_test.go`.

---

## A24 — Keine Changelog-Datei; `OFFEN.md` ist die einzige Liste

**Entscheidung.** Es gibt keine gepflegte Änderungshistorie als Datei. Offenes steht an
**einem** Ort; Erledigtes wird gelöscht, nicht archiviert.

**Anlass.** Das Befund-Register und sechs GitHub-Issues führten parallele Listen; am
13.09.2026 sind sie zusammengeführt worden. Zwei Listen driften, und die Drift fällt genau
dann auf, wenn man sich auf eine verlässt.

**Folge.** `git log` (und `git log -p docs/OFFEN.md`) ist die Historie — ausführlicher als
jede gepflegte Liste und nicht veraltbar. **Auch diese arc42-Dokumentation führt keine
eigene Offen-Liste**; Kapitel 11 benennt Risiken und verweist für den Stand auf `OFFEN.md`.

**Fundstelle.** [OFFEN.md](../OFFEN.md), [docs/README.md](../README.md).
