# Befund-Register

**Zweck:** Ein Ort für Dinge, die beim Arbeiten auffallen, aber nicht sofort angefasst
werden. Ohne diese Liste gibt es nur zwei Umgangsformen mit einem Fund — sofort
reparieren oder vergessen — und beide sind falsch, wenn das System kurz vor dem
Pilotbetrieb steht.

Ergänzt [`invarianten.md`](invarianten.md): Dort steht, was immer wahr sein *muss*.
Hier steht, was aufgefallen ist und noch nicht entschieden oder erledigt wurde.

---

## Die Einordnung

Vor jedem Fund steht dieselbe Frage — **nicht** „ist das hässlich?", sondern
**„kann das still jemandem schaden?"**:

| | Kategorie | Umgang |
|---|---|---|
| **A** | Kann **stillschweigend** ein falsches Ergebnis für einen echten Menschen erzeugen: doppelte Mahnung an Eltern, Daten beim falschen Empfänger, verlorene Eingabe, ein Gate, das nicht rot werden kann | **Sofort**, eigener Commit, eigener Test, eigene Deploy-Entscheidung |
| **B** | Fehler, der sich **laut** meldet, oder Unordnung ohne Wirkung nach außen: totes Codestück, wackeliger Test, Doppelung | **Hier notieren**, gebündelt nach dem Pilotstart |
| **C** | „Wenn ich schon mal hier bin" — Umbenennungen, Stilfragen, Refactorings ohne Anlass | **Nicht** vor dem Pilotstart |

Zwei Regeln dazu:

1. **Ein Fund = ein Commit.** Kein Anhängen verwandter Aufräumarbeiten. Was beim
   Reparieren zusätzlich auffällt, kommt in diese Liste, nicht in denselben Commit.
2. **Kategorie A wird belegt, nicht behauptet.** Ein Fund dieser Klasse braucht einen
   Test, der mit dem alten Code rot wird. Ohne diese Gegenprobe ist unklar, ob
   überhaupt etwas repariert wurde.

---

## Offen

| Fund | Warum offen |
|---|---|
| `golang.org/x/crypto/openpgp` gilt als unwartbar (`GO-2026-5932`) | Kein Fix verfügbar (`Fixed in: N/A`), transitive Abhängigkeit, **kein** Aufrufer im eigenen Code (`govulncheck`: „your code doesn't appear to call these"). Beobachten, nicht jagen. |
| `pgtest_support_test.go` liegt echt dupliziert in fünf Paketen | Sauber wäre ein `internal/pgtest`-Paket; fasst die Test-Infrastruktur von fünf Paketen auf einmal an und ist nur gegen echtes PostgreSQL beweisbar. Steht als Befund in `sonar-project.properties`. |
| Etikettenraster stehen an vier Stellen | Maßgeblich ist `api/label_formats.go`; das Druck-Center führt eigene Kopien (`LabelLayoutOptionen.svelte`, `stores/labels.svelte.js`). Der Umbau auf die Server-Liste — wie bei der Lieferantenseite gemacht — wäre eine Verhaltensänderung an einem täglich benutzten Bildschirm. Bis dahin hält `etikettformate-konsistenz.test.js` die Kopien deckungsgleich. |
| Zwei verschiedene Vorgaben für dasselbe Raster | Druck-Center `avery_3475`, Lieferanten-Weg `zweckform_l4760`. Fällt praktisch nie auf, weil beide Oberflächen immer ein Format mitschicken — die Vorgabe greift nur bei einem Aufruf ohne `?format=`. Zu entscheiden ist, welche gelten soll. |

---

## SonarQube-Lauf 23.08.2026 — was der Scan sagt und was er nicht sagt

Erster Lauf auf dem neuen Server (26.7). Endstand nach der Abarbeitung: **0 Bugs,
0 Vulnerabilities, Ratings A/A/A, Coverage 58,3 %, Duplikate 0,5 %, 32 offene Smells,
Quality Gate OK** (`http://localhost:9000/dashboard?id=bibliothek`).

**Drei Messfehler, die wichtiger waren als die Funde:**

1. **Das Gate war eine Attrappe.** „Sonar way" prüft ausschließlich *neuen* Code; die
   New-Code-Periode ist `PREVIOUS_VERSION`, und es gab keine vorherige Analyse. Ergebnis:
   null Bedingungen, Status OK — eine Aussage über nichts. Ab der zweiten Analyse greift es
   (und meldete sofort zwei echte Punkte).
2. **Security-Rating E hing an einer Kommentarzeile.** Zwei BLOCKER `secrets:S6698` in
   `scripts/pruefe_secrets.sh`: Das „Passwort" ist das Platzhalterwort `PASSWORT` in einem
   Erklärtext — ausgerechnet in dem Skript, das falsch konfigurierte Geheimnisse *findet*.
   Begründet ausgeschlossen (e4). **Falle dabei:** Die erste Fassung der Begründung zitierte
   die beanstandete Zeile wörtlich — der Detektor meldete daraufhin die Begründung. Wer
   einen Secrets-Fehlalarm dokumentiert, darf das Muster nicht abschreiben.
3. **Svelte ist im Scan gar nicht enthalten.** Sprachverteilung: `go=70.117`, `js=5.148`,
   `web=619`, `css=874`. Im Repo liegen **195 `.svelte`-Dateien mit rund 26.000 Zeilen** —
   SonarQube hat keinen Svelte-Parser. Was dieser Server über das Frontend aussagt, gilt für
   die 49 produktiven `.js`-Dateien, **nicht für die Komponenten**. Steht so auch in
   `sonar-project.properties`, damit die Zahl niemanden in Sicherheit wiegt.

**Behoben:** `klassifiziereZeileName` 8 → 7 Parameter (der achte stammte vom A2-Fix desselben
Tages; jetzt Modus statt Schlüssel+Flag, symmetrisch zu `bestandsSchluessel`) ·
`settingsWerte.js` `String(unknown)` → Typ-Zweige · `menue-fuehrt-irgendwohin.spec.js`:
Kommentar versprach „auf den Navigationszustand warten", darunter stand `waitForTimeout(600)`
— eine feste Frist, nach der ein langsam rendernder Bildschirm einen gesunden Menüpunkt als
tot gemeldet hätte (am Rückbau rot gesehen) · Frontend-Coverage (`@vitest/coverage-v8`, lcov,
an `sonar_scan.sh` gebunden mit Abbruch bei roten Tests) — schließt den offenen Punkt oben.

**Bewusst offen:**

| Fund | Warum offen |
|---|---|
| 20× `go:S3776` (Cognitive Complexity > 15) | Alle im Go-Backend, **null im Frontend** — die Ratsche vom 13.07. hält. Mehrere Handler stehen seit Mai/Juni über der Schwelle; Grund ist die Zählweise (ein `http.HandlerFunc`-Closure ist eine zusätzliche Verschachtelungsebene, siehe Projektgedächtnis). Ein Aufsplitten nur für die Zahl macht die Handler nicht lesbarer. Lohnend sind allenfalls die drei größten: `OverrideDueDateHandler` (30), `behandleAbgaenger` (23), `ErledigeAnliegenHandler`/`PatchStudentHandler` (21). |
| 1× `javascript:S2925` (`kontrast.spec.js:44`) | 700 ms Setzzeit vor der Kontrastmessung. Eine beobachtbare Bedingung gäbe es nur je Bildschirm (Selektor-Tabelle) — ehrlicher wäre eine Stabilitäts-Schleife wie in den Schwester-Specs. Eigene kleine Runde. |
| 1× `javascript:S6551` (`escapeHtml.js`) | `String(wert)` ist dort genau die Absicht: Der Helfer soll jeden Wert sicher machen, auch einen versehentlich übergebenen. |
| 1× `javascript:S8783` (`schuelerprofil-sperre.spec.js:75`) | `hover({force:true})` auf einem **absichtlich deaktivierten** Knopf — ohne `force` wartet Playwright ewig auf Aktionierbarkeit. Genau der Testzweck (Tooltip am gesperrten Knopf). |
| 3× `go:S1192`, 2× `go:S107`, Rest JS-Stil | Kleinkram, gebündelt bei Gelegenheit. |
| `sonar.projectVersion` nicht gesetzt | Die New-Code-Periode `PREVIOUS_VERSION` hat damit keinen Bezugspunkt; „neuer Code" heißt derzeit „seit dem letzten Scan". Mit `--define sonar.projectVersion=$(git describe --tags)` hieße es „seit dem letzten Release" — passt zum Release-Workflow, aber ändert die Gate-Semantik. Entscheidung offen. |

---

## Prüfung 22.08.2026 (Daniel-Raster über alle Commits seit 21.08.) — Stand nach der Abarbeitung

Sechs unabhängige, nur lesende Durchgänge nach dem abstrahierten Raster des externen
DB-Prüfberichts (Konvention statt Regel · Spezialwert/Doppeljob · zwei Wahrheitsquellen ·
wer sieht was · stille Fehler · Zeit/Reihenfolge · Gate-Ehrlichkeit · Lebenszyklus) über
die ~70 Commits vom 21./22.08. Jeder Fund unten ist **am Code verifiziert** (Datei:Zeile),
die HOCH-Funde zusätzlich am Live-Pfad bzw. per Probe gegen echtes Postgres.

**Abarbeitung 22.08. abends (je Fund ein Commit, Gate am alten Code rot gesehen):**
A1/A2 `bdb48ca7` · A3 `6d01f27a` · A4 `b5b50a2e` · A5 `4cf559cc` · A6 `729a7271` · A7 `f7e39361` ·
A8 `9fa3eae4` · B-Betrieb `5a55147f` (restore-backup im Image, S3-Durchreichung, Probe-Startlauf,
Stderr-Scrub, scrypt-Texte, Doku) · B-DB `8c9c8042` (Migration 085, conname-Parität,
Selbstprüfung „DSGVO-Löschroutinen", Restore-409, Append-only-Ratsche) · B-Rest `63d09011`
(Art.-15-Angaben aus den Fristen, LUSD-Import-Audit, PATCH-Geburtsdatum, Release-Wächter +
Tag-Ruleset, actionlint, trivy-Pin, Jules-Nacharbeit, Doku) · LUSD-Modus-Wechsel `968ee01a`.

**Bewusst offen (gelistet, entschieden „nicht jetzt"):**
- Paritätstest vergleicht weiterhin keine COMMENTs/Seeds (nur Kosmetik; 085 deckt die
  strukturellen Lücken) — Prod-Check der vier Indexe nach dem Deploy: `SELECT indexname FROM
  pg_indexes WHERE indexname IN ('idx_schueler_deleted_at','idx_ausleihen_rueckgabe_am')`.
- 082-Dedupe der Vormerkungen (neuerer `abholbereit` verloren) — auf Prod bereits gelaufen,
  nicht rückholbar; nur relevant, falls eine weitere gewachsene DB hinzukommt.
- Lernmittel-/Schülerbücherei-Frist hängt an der `LMF-`-Namenskonvention; kein
  `ist_lernmittel`-Flag (Produktentscheidung, siehe LMF-Memory).
- LUSD: Namensschlüssel nur `lower+trim` (Umlaut-/Bindestrich-Varianten gelten als
  verschiedene Menschen → „mehrdeutig"/neu) — sicher, aber nicht klug; ID-Modus mit gemischter
  Datei behandelt ID-lose Zeilen weiterhin nur als Zähler.
- Jules: sieben Testdateien > 200 Zeilen; Export-CSV-„breaks stream"-Test bleibt schwach.
- Release: Go/Node-Versionen sind jetzt dokumentiert, aber Dependabot kann `golang:`/`node:`
  im Dockerfile weiter unabhängig heben (kein `ignore`).
- Altbackups vor 21.08. sind unlesbar; Re-Encrypt-Tool bewusst nicht gebaut (Pilot, keine
  schützenswerten Altbestände) — dafür Betreiber-Hinweis in resilience_and_recovery.md.

### Kategorie A — kann still jemandem schaden (ALLE ERLEDIGT 22.08.)

| Fund | Nachweis | Fix-Idee |
|---|---|---|
| **LUSD Nur-Name: bestätigter Bestandsschüler, dessen Name ZWEIMAL in der Datei steht, wird Abgänger und anonymisiert.** Der „Mehrdeutig“-Zweig setzt `gesehen` nicht; `sammleAbgaenger` hält ihn für „nicht im Export“. Vorschau zeigt ihn unter „Mehrdeutig (wird nicht angefasst)“ UND unter „Abgänger“. | `api/lusd_klassifizierung.go:59-63` + `:155-163`; Probe gegen PG: 1/13 → Schwelle greift nicht, Apply ohne Bestätigung, danach `ist_abgaenger=true, vorname='Abgänger'` | im Default-Zweig `gesehen[rec.namensschluessel()] = true` vor `continue`; Regressionstest mit Bestandstreffer + zwei Dateizeilen |
| **LUSD Nur-Name: gesperrter Abgänger wird von Namensvetter „reaktiviert“** — Fünftklässler landet auf dem Datensatz (Schulden, Sperre, Lesehistorie) eines anderen Kindes. | `api/lusd_klassifizierung.go:136-143` (`idx.abgaenger[key]` nur über Name) | im Nur-Name-Modus Abgänger-Treffer als „Rückkehrer-Kandidat“ melden statt zuordnen |
| **LUSD-Abgänger-Anonymisierung setzt kein `anonymized_at`; Purge läuft VOR der Cron-Tilgung und tilgt `audit_logs` nie** → LUSD-ID (LUSD_ID_NACHGETRAGEN) überlebt 24 Monate in `audit_logs.details`; Vormerkungen bleiben bis zum Purge. | `api/lusd_apply.go:319-349` (kein `anonymized_at`), `repository/audit_users.go:168-203` (nur `audit_log`), `jobs/cron.go:43-45` (Reihenfolge), Purge-Cutoff 30.01. vs. 360 d | `anonymisiereAbgaenger` setzt `anonymized_at` + `ANON-`-Barcode; Purge tilgt `audit_logs` vor dem DELETE; PG-Test für den Purge-Pfad |
| **Leeres Zahlenfeld in „Datenschutz & Sitzung“ schaltet Befristung/Sperre still auf 0 = aus.** `bind:value` liefert `null`, `Number(null) \|\| 0` → 0; Backend nimmt 0. | `frontend/src/lib/SystemSettingsAllgemein.svelte:86-89`, `SettingField.svelte:42`, `repository/system_settings_datenschutz.go:49-57` | „Aus“ als eigener Schalter + Zahl ≥ 1; leer ⇒ Vorgabe; Selbstprüfung meldet „Befristung aus“ |
| **Lesehistorie lebt im `audit_log` weiter** (CHECKOUT/RETURN mit `details.schueler_id`, 24 Monate) — Art.-13/VVT sagen „Zuordnung automatisch entfernt“; Art.-15-Auskunft liest diese Einträge nicht einmal. | `repository/audit_books.go` (`details["schueler_id"]`), `jobs/cron_dsgvo_lesehistorie.go` fasst nur `ausleihen` an, `api/dsgvo_auskunft.go:255` nur `tabelle='schueler'` | Job tilgt `details - 'schueler_id'` für `tabelle='ausleihen'` nach derselben Frist; Auskunft um Ausleih-Einträge ergänzen |
| **Sperrbildschirm: Druckvorschau (Strg+P) zeigt die Seite dahinter** (`no-print` am Overlay), **kein Fokus-Fang/`inert`** (Tab verlässt die Sperre, Screenreader liest dahinter), **Kamera-Scanner bucht weiter**. | `Sperrbildschirm.svelte:33`, `styles/druck-grundlagen.css:38-42`, `App.svelte` rendert App unter dem Overlay, `idleLock.svelte.js` fasst `showCamera` nicht an | App-Fläche bei Sperre nicht rendern (löst Druck + Fokus + SR zusammen); `thekeLeeren()` stoppt Kamera |
| **Login-Handler-Kontext 10 s < IMAP-Frist 15 s** → korrektes, langsames Login scheitert am DB-Lookup, zählt als Fehlversuch (401 + Sperre). Umgekehrt: ≥ 15 s-Tarpit des Mailservers macht jedes falsche Passwort zum 503 ohne Zählung. | `auth/handlers.go:127`, `auth/imap.go:192,215-216`, `:264`, `selbstanmeldung.go:118-123` | Handler-ctx an `AuthenticateIMAP` durchreichen, EINE Frist; Klassifikation aus der IMAP-Antwort (NO = Passwort), nicht aus der Zeit; Test mit Mini-IMAP-Listener (sofort NO / verzögert NO) |
| **Bulk-Mahnmail: SMTP-Hänger je Klasse bis 70 s, Ausfälle zählen als „übersprungen (keine E-Mail hinterlegt)“**, End-Audit mit totem `r.Context()` schweigt. | `api/mail_sender.go:84` (kein ctx), `api/mahnwesen_bulk_mail.go:288,324-327,128,374` | „fehlgeschlagen“ getrennt zählen; Audit mit `context.Background()`+Frist; nach erstem Versandfehler abbrechen |

### Kategorie B — laut oder ohne Außenwirkung (erledigt bis auf die oben gelisteten Punkte)

| Fund | Nachweis |
|---|---|
| `restore-backup` liegt **nicht** im Image, Doku sagt „liegt im Image“ — im Ernstfall auf dem Schulserver ohne Go nicht baubar | `Dockerfile:67,103-107`, `docs/resilience_and_recovery.md:76` |
| Selbstprüfung rät „S3_* in der .env setzen“ — Compose reicht `S3_*` nicht durch (gleiche Klasse wie BACKUP_ENCRYPTION_KEY) | `api/betriebsbereitschaft.go:380`, `docker-compose.yml` ohne `S3_`, `.env.example:122-126` |
| Altbackups (SHA-256) seit 5265698c **unlesbar**, bis 02:30 des Folgetags gibt es kein lesbares Backup; Doku widerspricht sich (sed-Reparatur für Dateien, die nicht mehr entschlüsselbar sind), `backup.go:297-299` behauptet „alte Backups bleiben lesbar“ | `jobs/backup_krypto.go:94-97`, `docs/resilience_and_recovery.md:20-23` vs. `:95-100` |
| IMAP `Logout()` ohne Frist (go-imap `Timeout=0`) — schweigender Server hängt den Login-Handler | `auth/imap.go:202-208` |
| Restore-Probe: „per Neustart erneut proben“ stimmt nicht (nur So 03:30) → bis 7 Tage „kritisch“ nach Behebung; psql-Stderr mit Datenkontext (`COPY schueler, line …`) landet in DB/Seite/Alarm-Mail | `api/betriebsbereitschaft.go:185-186`, `jobs/cron.go:116`, `restore_probe_hilfen.go:87` |
| Veraltete SHA-256-Texte nach scrypt-Umstellung | `betriebsbereitschaft.go:238`, `backup_status.go:30`, `cron.go:75-77`, `backup.go:297-299` |
| Art.-15-Auskunft nennt pauschal lit. e, Speicherdauer ohne 90/730 Tage, keine Einwilligung für die Schülerbücherei — zwei Wahrheitsquellen zu SECURITY.md/VVT | `api/dsgvo_auskunft.go:121-129` |
| VVT sagt „Protokoll ohne IP-Adresse“ — `audit_logs.ip_adresse` wird geschrieben | `repository/audit.go:132`, `api/littera_import.go`, `api/mahnwesen_bulk_mail.go:371` |
| Lernmittel-/Schülerbücherei-Frist hängt an der `LMF-`-Namenskonvention; keine Gegenprobe „Lernmittel-Fach ohne Kennung“ | `jobs/cron_dsgvo_lesehistorie.go:71-75`, `pkg/lmf/lmf.go:43-45` |
| DSGVO-Jobs (Anonymisierung, Lesehistorie) scheitern still (nur Log) — exakt die Klasse, die zweimal monatelang ins Leere lief; Selbstprüfung hat keinen DSGVO-Bereich | `jobs/cron_dsgvo.go:102,112,159,172,183`, `api/betriebsbereitschaft.go` |
| Paritäts-Ratsche sieht nur die älteste Baseline: DBs aus schema.sql vom 14.06.–21.08. fehlen `idx_ausleihen_ausgeliehen_am/rueckgabe_am`, `idx_buecher_titel_erstellt_am`, `idx_schueler_deleted_at` (082 heilt bewusst nur die Gegenrichtung); Test vergleicht keine Constraint-NAMEN (Kommentar behauptet es), keine COMMENTs (084-Kommentar fehlt in schema.sql), keine Seeds | `migrations/082:14-15`, `db/migrations_schema_paritaet_pg_test.go:33-41`; Prod prüfen: `SELECT indexname FROM pg_indexes WHERE indexname IN ('idx_schueler_deleted_at','idx_ausleihen_rueckgabe_am')` |
| Papierkorb-Restore stellt anonymisierte Zeilen wieder her (gesperrt, Name „Anonym“); 082-Dedupe der Vormerkungen löscht den NEUEREN Eintrag auch wenn er `abholbereit` ist | `api/student_deleted.go:86-90`, `migrations/082` |
| Append-only auf `audit_log(s)` ist seit 083 reine Konvention ohne Ratsche (App läuft als `postgres`) | `docker-compose.yml:101`, Schreibtüren `audit_users.go:176`, `cron_dsgvo.go:154/168`, `cron_audit_retention.go:37/43` |
| LUSD: Modus-Wechsel Nur-Name → Name+Geburtsdatum dupliziert den Bestand (Bestand ohne Datum nur in `ohneSchluessel`; Datum wird nie nachgetragen); ID-Modus mit gemischter Datei macht ID-lose Zeile zum Abgänger; kein Audit-Eintrag für den Import; PATCH blankt das Pflicht-Geburtsdatum; Parser loggt Namen bei 400 | `api/lusd_bestand.go:75-76`, `lusd_apply.go:155-166`, `lusd_klassifizierung.go:87-90`, `student_update.go:145-150`, `lusd_parser.go:204` |
| Mahnketten-TZ-Fix ist Test-Reparatur, aber am DST-Ende (24.10., 22–23 UTC) wieder rot: `time.Now().AddDate` in Runner-TZ statt Schulzeitzone | `api/mahnwesen_kette_pg_test.go:153` vs. `internal/service/loan_rules.go:162` |
| Release: v-Tag ist ungeschützter Kanal (jeder Push-Collaborator; kein main-/CI-Check), `latest` hat zwei Schreiber (Doku behauptet einen), Release-Notes versprechen ein Image, das ein zweiter Workflow erst baut; ghcr-Image ist öffentlich (Kommentar sagt privat); kein actionlint; `trivy-action@master`; Go-Version 1.26.5 in docs vs. 1.26.6; Node 22 (Image) vs. 24 (CI) | `.github/workflows/release.yml:16-18,49,54`, `docker-publish.yml:13,17-25`, `security-scan.yml:169`, `docs/README.md:11,67`, `ci.yml:157,195` |
| update.sh 4b: leerer GIT_COMMIT = nur Warnung | `update.sh:275-279` |
| Jules-Batches: `TestMigriereFoto_VerschluesselungFehlgeschlagen` belegt seine Behauptung nicht; Dubletten (`TestKuerze`, `TestSammleSignaturUpdatesDynamic`, DeleteBooks-Mock); 7 Testdateien > 200 Zeilen; Nil-Deref im Fehlerzweig (`schreiber_bestand_test.go:113-124`); DeleteBooks-Test schreibt `inventur/uploads/` ins Repo | `cmd/migrate-fotos/main_test.go:174-193` u. a.; Cruft: keiner, `go test -race -count=3` + `TZ=Pacific/Midway` grün |

### Geprüft, kein Befund (Auszug)
SMTP-Frist deckt alle Phasen (Hänger-Test echt, am Rückbau rot); keine Geheimnisse in argv/Log (PGPASSFILE); scrypt 32 MB ohne mem_limit; Backup-Alter eine Quelle; Cron-Zeiten UTC ohne Überlappung; Restore-Probe gegen echtes pg_dump/psql; Dockerfile-Pin 16 ≥ Prod 15; 080–084 idempotent, Migrationslauf aus sechs Baselines bricht nirgends; `ANON-<uuid>` kollisionsfrei; LUSD-Apply atomar mit Advisory-Lock, Upsert-Blanking gepaart, Handanlagen nie Abgänger; Dependabot-PRs laufen durch ci+security-scan; 3b80f7f1 `npm ci` konsistent; Jules-Merges ohne Cruft, nur zwei bewusste Produktions-Nähte.

---

## Erledigt (2026-08-06) — Etiketten nach Rückmeldung Naacher

Zwei Wünsche aus einem Telefonat, beide am fertigen PDF abgesichert:

| Fund | Was es behauptete | Was stimmte |
|---|---|---|
| Große Lernmittel-Etiketten | „ein Etikett je Seite (A6)" | Auf einem A4-Drucker kam damit **ein** Etikett je Blatt heraus. Vier passen ohne jede Skalierung darauf — A6 ist exakt ein Viertel von A4. Jetzt 2×2 mit Schnittlinien. |
| Kleine Etiketten im Lieferanten-Link | „Bogen wie im Druck-Center" | Der Weg hatte `"zweckform_l4760"` **fest verdrahtet**, während das Druck-Center längst drei Raster anbietet. Wer andere Bögen im Drucker hatte, bekam einen Ausdruck, der danebenliegt. |

**Der Fund kam aus der Rückbau-Probe, nicht aus dem Code.** Der erste Test zur
Formatwahl rief den Generator direkt auf und blieb grün, als der Lieferanten-Weg
probeweise wieder auf das feste Raster gesetzt wurde — er sah die Durchreichung gar
nicht. Erst ein Test über den echten Endpunkt (`bestellbestaetigung_format_pg_test.go`)
wurde rot. Genau die Bugklasse dieses Projekts: ein Gate, das eine Stelle prüft, die
niemand kaputtmacht.

Dazu ein drittes Fundstück am Rand: Die Rasterdaten stehen an **vier** Stellen
(`api/label_formats.go`, `LabelLayoutOptionen.svelte`, `stores/labels.svelte.js`, dazu
zwei verschiedene Vorgaben — `avery_3475` im Druck-Center, `zweckform_l4760` im
Lieferanten-Weg). Der Umbau des Druck-Centers auf die Server-Liste wäre eine
Verhaltensänderung an einem täglich benutzten Bildschirm und ist **nicht** gemacht;
stattdessen hält `etikettformate-konsistenz.test.js` die Kopien deckungsgleich.

Stand: 2026-08-06

---

## Erledigt (2026-08-06) — Audit-Nachlese

Ein externes Audit meldete 16 Punkte. Nachgeprüft haben sich davon acht bestätigt, drei
in anderer Ursache als gemeldet, zwei waren falsch. Vier weitere Funde kamen erst beim
Nachprüfen dazu — sie standen in keiner Meldung.

**Der teuerste Fund stand nicht auf der Liste:** `internal/crypto` las den
Master-Schlüssel vorrangig aus `ENCRYPTION_KEY` und erst danach aus
`APP_ENCRYPTION_KEY`. Geprüft wird beim Start aber nur der zweite Name — auf Länge, auf
Hex-Form und (mit `ENFORCE_PROD_SECRETS`) gegen die bekannten Default-Werte. Ein
gesetztes `ENCRYPTION_KEY` hätte alle drei Prüfungen umgangen und still gewonnen; ver-
und entschlüsselt worden wäre mit einem anderen Schlüssel als dem geprüften.
Schülerfotos und das gespeicherte SMTP-Passwort wären damit nicht falsch, sondern
**weg**. Dieselbe Machart wie „zwei Türen zum selben Zustand": ein zweiter Weg zu einem
Wert, der nur an einem Weg abgesichert ist.

| Fund | Was es behauptete | Was stimmte |
|---|---|---|
| `SECURITY.md`: „docker-compose.yml erzwingt per `${VAR:?}`, dass **alle** Secrets gesetzt sind" | Absicherung durch Compose | Nur `POSTGRES_PASSWORD` und `IMAP_HOST`. `JWT_SECRET` und `APP_ENCRYPTION_KEY` fallen auf die **committeten** Defaults zurück; einzige Absicherung ist der Code-Guard, und der ist standardmäßig aus. |
| `starttlsWennMoeglich` | „verifiziertes STARTTLS" | Bot der Server keines an, ging die Mahnung im Klartext raus — mit Schülernamen und Elternadressen. Ein MITM konnte das durch Streichen der Erweiterung selbst herbeiführen. |
| Cover-Proxy `/api/images/cover` | öffentlicher Bild-Cache | Einziger unauthentifizierter Pfad, der eine ausgehende Verbindung **und** ein volles `image.Decode` auslöst — ohne Größengrenze, ohne Dimensionsprüfung, ohne Bremse, mit dem Query-Parameter als Dateinamen. |
| `maskiereToken` „in beiden Logzeilen" | Token nirgends im Log | Die Panic-Recovery schrieb den rohen Pfad. Beim ersten Fix schlicht übersehen. |
| `ReadHeaderTimeout` am Server | Anfragen sind zeitlich begrenzt | Nur die Kopfzeilen. Ein Rumpf durfte beliebig langsam kommen; der Kontext-Deadline der `TimeoutMiddleware` bricht kein blockierendes `Read` ab. |
| `docker-compose.local.yml` | Entwicklungsstack, „nicht laxer als Prod" | Der DB-Port war auf `127.0.0.1` gebunden und begründet — das Backend eine Zeile darunter auf `0.0.0.0`, mit `IMAP_HOST=mock`, das jedes Passwort durchwinkt. |
| `SCRIPTS.md`: Backup-Pipeline „→ AES-GCM → 0600" | gilt für alle Backups | Galt nur für den nächtlichen Job. `scripts/backup.sh` **und** `./update.sh` legten unverschlüsselte Volldumps mit Standardrechten ab — letzterer war überhaupt nicht dokumentiert. |
| `scripts/backup.sh` meldete „Backup erfolgreich" | pg_dump lief durch | Ohne `pipefail` lieferte die Pipe den Status von `gzip`, und `gzip` gelingt auch dann, wenn `pg_dump` abgebrochen ist. |
| `Caddyfile` im Repo-Root | Vorlage für den Reverse-Proxy | Zeigte auf `bibliothek-backend-local:8083` — ein Container dieses Namens existiert nur lokal und lauscht auf 8084. |
| `useStudentEditForm({ student, … })` | Formular des ausgewählten Schülers | Destrukturieren nimmt einen Schnappschuss. `save()` hätte das PATCH an die **zuvor** geöffnete ID geschickt — mit Erfolgsmeldung. |
| `bind:this` auf `let` in `UnifiedInventory` | `$effect` setzt den Fokus ins Scanfeld | Ohne `$state` ist die Zuweisung nicht reaktiv: Der Effekt lief einmal, bevor das Feld existierte, und nie wieder. Jeder Scan wäre ins Leere gelaufen — ohne Fehlermeldung. |

**Zwei Meldungen waren falsch**, und beide auf dieselbe Weise: Sie beschrieben eine
Grenze, die es gab, als fehlend. Die XLSX-Importe haben `http.MaxBytesReader`
(100/20 MB) und hängen hinter `RequirePermission("manage_inventory")`. Und der
Klartext-Versand hätte die **SMTP-Zugangsdaten** nie preisgegeben: `smtp.PlainAuth`
verweigert die Übertragung über eine unverschlüsselte Verbindung von sich aus. Der
Inhalt war das Problem, nicht die Anmeldung — bei einem Relay ohne Zugangsdaten fiel
diese Bremse allerdings ganz weg.

**Merksatz dieses Tages:** Ein gemeldeter Befund ist eine Behauptung wie jede andere.
Drei der 16 stimmten in der Wirkung, aber nicht in der Ursache — wer sie so repariert
hätte, wie sie dastanden, hätte die falsche Stelle angefasst und den Fund für erledigt
gehalten.

Stand: 2026-08-06

---

## Erledigt (2026-08-04) — Littera-Schreibpfad

Fünf Funde, alle derselben Machart wie am 30.07.: Etwas behauptete etwas, und die
Behauptung war nie gegen die Wirklichkeit gehalten worden. Nur diesmal war die
Wirklichkeit nicht der Code, sondern **das Buch im Regal**.

| Fund | Was es behauptete | Was stimmte |
|---|---|---|
| `barcode_id` aus der Exemplarnummer | „Auf dem Etikett steht die Exemplarnummer" | Der Scanner liest eine EAN-13: aufgedruckt `105785`, gescannt `1057850039567`. Keines der 61.520 Exemplare wäre auffindbar gewesen — gemerkt hätte man es an der Theke. |
| Schülerausweis `[0395] 37` | Der Aufdruck ist der Scanwert | Gescannt kommt `B97601826457`, die Nummer des Kartenherstellers. Sie steht in keinem Stammdatenfeld, nur in Litteras `FremdLeserNummer`. |
| „10.002 Titel mit Autor (93 %)" | Autorenabdeckung | Mitgezählt waren Standortvermerke, die als Personen erfasst sind: `Buchbestand Bibliothek` allein auf 6.711 Titeln. Bei 7.131 Titeln stünde ein Regalvermerk in der Autorenangabe. Echt sind 9.029 (84 %). |
| Geburtsdaten aus Littera | Jahrgänge der Personen | Gos Jahrhundertgrenze liegt bei 69: 69 Lehrkräfte der Jahrgänge 1946–1968 kamen als **2046–2068** an. |
| Präfixlose Omnibox-Auflösung | Buch → Schüler → Suche | Lehrkräfte stehen in `benutzer`, nicht in `schueler`. Ein gescannter Lehrerausweis lief bis in die Volltextsuche. Die passende Abfrage gab es längst — sie hing allein hinter `L-` und ließ sich als rohes SQL am Pool nicht testen. |

**Merksatz dieses Tages:** Aufdruck, Datenbankspalte und Scanwert sind drei verschiedene
Dinge. Für jede Barcode-Quelle einmal real in einen Texteditor scannen, bevor irgendetwas
nach `barcode_id` geschrieben wird.

Dazu, aus demselben Lauf und nicht kleiner: **`npm ci` lief gegen das committete Lockfile
gar nicht durch** (`typescript@^7` gegen `typescript-eslint@8.65`, das `<6.1.0` verlangt).
Lokal war ESLint grün, weil `node_modules` älter war als das Lockfile — auf einem frischen
Klon oder in CI wäre es rot gewesen. Ein grünes Gate beweist nichts über das, was im Repo
steht, wenn die Umgebung abgedriftet ist.

Stand: 2026-08-04

## Erledigt (2026-07-30)

Alle fünf Funde eines Tages waren dieselbe Art Fehler — etwas behauptete zu
funktionieren, und nichts prüfte die Behauptung nach:

| Fund | Was es behauptete | Was stimmte |
|---|---|---|
| `expect([200, 400, 500])` im Mail-Test | prüft den Testversand | konnte nicht rot werden; schickte Felder, die die API nie gelesen hat |
| CSRF-Ausnahme für `/api/admin` & Co. | „Inventur-Modul hat eigenes CSRF" | dieses System gab es im Code nicht; sechs Admin-Mutationen ungeschützt |
| SMTP-Einstellungen im Admin-Bereich | steuern den Mailversand | wurden nur vom Test-Knopf gelesen; echte Mails nahmen die Umgebung |
| Diagnose bei SMTP-Fehlern | steht im Formular | jede 500 wurde zu „interner Datenbankfehler" eingedampft |
| Ergebnis des Mailversands | Nachricht zugestellt | war das Ergebnis der Verabschiedung; Abbruch danach galt als Fehlschlag |

**Merksatz:** Der Fundort ist immer die Stelle, an der eine Zusicherung steht, die
niemand prüft — ein Kommentar, ein Testname, eine Eingabemaske. Wer dort sucht,
findet; wer im Code sucht, findet Schönheitsfehler.

### Nachtrag desselben Tages: die Liste selbst abgearbeitet

| Fund | Was es behauptete | Was stimmte |
|---|---|---|
| `schadensfall.spec.js` fiel unter Last um | Test prüft den Elternbrief | `popup.url()` wurde gelesen, bevor die Navigation übernommen hatte — allein grün, im vollen Lauf manchmal rot. Jetzt wird die Adresse abgewartet und die Route zusätzlich am Inhaltstyp geprüft. |
| `SendTemplateMail` | Vorlagen-Versand des Systems | kein einziger Aufrufer; die Vorlagen werden anderswo direkt geladen. Entfernt. |
| `inventur`-Backup-Modul samt Benachrichtigungs-Mail | zweites Backup-System mit Mail bei Änderungen | `NewAPIHandler` hat das Feld nie gesetzt — unerreichbar. Gesichert wird über `jobs.BackupJob`. Die `.env.example` lud dazu ein, `BACKUP_EMAIL_TO` zu setzen und auf Mails zu warten, die nie kommen konnten. Ersatzlos entfernt. |

Der dritte Fund kam aus der Prüfung des zweiten: Der Eintrag stand hier als
„bewusst so entschieden" — und die Begründung hielt der Nachfrage nicht stand. Auch
eine Notiz in dieser Liste ist eine Zusicherung, die jemand prüfen muss.

Stand: 2026-07-30
