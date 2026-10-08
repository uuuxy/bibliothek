# Pflegekonzept und Wartungshandbuch

Stand: 08.10.2026 (Entwurf)

Dieses Dokument beantwortet zwei Fragen. Für die Schule und den Schulträger: Wer betreibt und
pflegt das Programm, wie kommt eine Änderung auf den Server, und was geschieht, wenn die Pflege
endet? Für die Vertretung: Wie spielt man ohne die Entwicklung ein Update ein und holt eine
Sicherung zurück?

Befehlsfolgen stehen hier nicht. Sie stehen jeweils an einer Stelle, auf die dieses Dokument
verweist; einen Teil davon prüfen Tests gegen den Code (`docs/rueckweg_anleitungen_test.go`,
`docs/deployment_anleitung_test.go`).

**Nicht in diesem Dokument**, weil das Repository öffentlich ist: Zugänge, die Adresse des
Servers, der Ort der Schlüssel und der Quelldokumente, Namen und Telefonnummern. Das alles
steht auf einem Blatt, das bei der Schule liegt (Abschnitt 7.3).

---

## 1. Zuständigkeiten

| Rolle             | Wer                                       | Aufgabe                                                                                    |
| ----------------- | ----------------------------------------- | ------------------------------------------------------------------------------------------ |
| Betrieb           | die Schule, auf eigenem Server            | betreibt und nutzt das Programm                                                            |
| Hardware und Netz | die IT des Schulträgers                   | Server, Netzanbindung; vorgesehen auch Betriebssystem und Docker (Abschnitt 9)             |
| Pflege            | die Entwicklung des Programms             | Fehler beheben, Sicherheitsupdates, Termine und Vorgaben nachziehen, Updates bereitstellen |
| Vertretung        | eine benannte Person (Name auf dem Blatt) | ein Update einspielen (3.1), eine Sicherung zurückholen (3.2)                              |

Von der Vertretung wird nicht mehr verlangt als die zwei Handgriffe in Abschnitt 3. Änderungen am Code
bleiben bei der Entwicklung oder bei einer Stelle, die die Pflege übernimmt (Abschnitt 8).

**Für welche Schulen** (entschieden am 28.09.2026): Gepflegt wird das Programm für die eigene
Schule; nach einem Schuljahr Echtbetrieb wird neu entschieden. Jede weitere Schule bräuchte einen
eigenen Server und eine eigene Vertretung. Betreiben darf das Programm nach der Lizenz jede
Stelle, eine Pflege ist damit nicht zugesagt (Abschnitt 8).

**Fehler melden** (entschieden am 28.09.2026): per E-Mail an die Entwicklung, die Vertretung in
Kopie; bei Stillstand zusätzlich ein Anruf. Nie über die Issues auf GitHub, weil das Repository
öffentlich ist. Die Adressen stehen auf dem Blatt (Abschnitt 7.3).

- **Stillstand:** Antwort am selben Schultag. Hat ein Update ihn ausgelöst, geht die Vertretung
  auf den Stand davor zurück (3.1).
- **Fehler ohne Stillstand:** innerhalb einer Woche.
- **Wünsche:** mit dem nächsten Release.

---

## 2. Wie eine Änderung zur Schule kommt

1. **Code:** öffentlich auf GitHub (`uuuxy/bibliothek`), Lizenz EUPL 1.2 (`LICENSE`).
2. **Prüfung vor dem Hochladen:** Auf dem Arbeitsplatz der Entwicklung laufen vor jedem Commit
   Formatierung und Lint, vor jedem Push die Tests und die Sicherheitsprüfungen
   (`scripts/install-hooks.sh`, [SCRIPTS.md](SCRIPTS.md) §7).
3. **Prüfung auf GitHub:** vier CI-Jobs und vier Sicherheitsprüfungen je Stand. Ein Release
   (eine Versionsnummer wie `v2.14.0`) entsteht nur, wenn alle acht grün sind
   ([DEPLOYMENT.md](DEPLOYMENT.md) §8, `scripts/tag-gate.sh`). Was beim Einspielen von Hand zu
   tun ist, steht in der Release-Notiz.
4. **Auf den Server** kommt ein Stand nur von Hand, mit `update.sh`: Vorab-Sicherung der
   Datenbank → Bau des neuen Stands → Gesundheitsprüfung → Abgleich, ob der laufende Stand der
   eingespielte ist ([DEPLOYMENT.md](DEPLOYMENT.md) §2.4 und §7). Ein automatisches Update gibt
   es nicht. `update.sh` fragt nicht ab, ob die Prüfläufe des Stands auf GitHub grün sind.
   **Entschieden am 28.09.2026:** Der Schulserver bekommt nur Releases, der Testserver folgt
   `main` als Vorstufe. Heute holt `update.sh` mit `git pull` den neuesten Stand des Zweigs;
   der Weg über Releases ist nicht gebaut ([OFFEN.md](OFFEN.md) 5.31).
5. **Die Datenbank** passt sich beim Start selbst an (Migrationen). Migrationen laufen nur
   vorwärts: Zurück geht es über die Vorab-Sicherung, nicht über den alten Code allein.

**Eine Änderung von außen** — ein Pull Request, auch von Dependabot oder von einem Werkzeug,
das Vorschläge selbst erzeugt — wird vor dem Übernehmen so geprüft:

- Grün in der CI genügt nicht; rot ist ein Befund und kein Rauschen.
- Welche Dateien ändert er (`gh pr view <Nummer> --json files`)? Beifang fällt hier auf: eine
  fremde Sperrdatei, Notizdateien, eine zurückgestufte Go-Version.
- Was er ändert, zeigt `git diff --stat origin/main...origin/<Zweig>` mit drei Punkten. Titel
  und Zweigname sagen es nicht, und mit zwei Punkten erscheint bei einem alten Zweig alles als
  Löschung, was `main` seither dazubekam.
- Ändert er einen Test, eine Ratsche oder die Stelle, an der ein Browser-Test sein Element
  sucht, ist das eine Lockerung, bis das Gegenteil belegt ist (Gegenprobe am Rückbau,
  [ARCHITEKTUR.md](ARCHITEKTUR.md) 8.14).
- „Schneller" wird gemessen, bevor es geglaubt wird (`EXPLAIN ANALYZE` an echten Mengen). Ein
  Zwischenspeicher ohne Messung und ohne Regel, wann er verfällt, wird abgelehnt.
- Neue Tests an der Datenbank laufen am Arbeitsplatz gegen Postgres; die CI zeigt nur den
  ersten Fehler.
- Ein Sammel-Update der Pakete wird in einem eigenen Arbeitsverzeichnis mit `npm ci` geprüft,
  bevor es übernommen wird.
- Ein Zweig auf GitHub ist kein offener Vorschlag: `gh pr list --state all --head <Zweig>`
  zeigt, ob er schon übernommen oder abgelehnt ist.
- Ändert der Vorschlag eine Funktion ohne Test, gehört der Test zur Übernahme.

---

## 3. Die zwei Handgriffe der Vertretung

### 3.1 Ein Update einspielen

1. Auf GitHub nachsehen, ob die Prüfläufe des neuesten Stands grün sind, und die Hinweise
   zum Einspielen lesen (Release-Notiz). Am Schulserver wird nur ein Release eingespielt; der
   Weg dafür ist noch nicht gebaut (Abschnitt 2).
2. Am Server im Programmverzeichnis erst `git pull`, dann `./update.sh` — zwei getrennte
   Befehle, weil sonst die alte Fassung des Skripts das Update fährt
   ([DEPLOYMENT.md](DEPLOYMENT.md) §2.4).
3. **Erfolg** heißt: Das Skript endet mit „UPDATE ERFOLGREICH ABGESCHLOSSEN", und unter
   System → Einstellungen → Betriebsbereitschaft steht kein kritischer Befund.
4. **Misserfolg:** Das Skript bricht ab und gibt eine Anleitung zum Zurückgehen aus: den
   Commit, der vor dem Update lief (aus dem laufenden Image gelesen), und den Pfad der
   Vorab-Sicherung. Ihr folgen. Die Vorab-Sicherung ist in diesem Fall nicht verschlüsselt;
   die Einzelheiten stehen in [resilience_and_recovery.md](resilience_and_recovery.md) §2b
   („Der Klartext-Fall") und §2c.

### 3.2 Eine Sicherung zurückholen

**Was es gibt:** Jede Nacht um 02:30 UTC eine verschlüsselte Sicherung der Datenbank; die
jüngsten 14 bleiben, dazu seit dem 29.09.2026 von den älteren je Kalenderwoche eine für 12
Wochen, damit ein Fehler, der erst nach den Sommerferien auffällt, noch eine Sicherung von
davor vorfindet. Ein älterer Stand bringt Personen zurück, die seitdem von Hand endgültig
gelöscht wurden; wie sie nachgeholt werden, steht in
[resilience_and_recovery.md](resilience_and_recovery.md), Abschnitt 2a, Schritte 5b und 8. Dazu legt `update.sh` vor jedem Update eine
Sicherung an, die beim ersten Update nach 30 Tagen gelöscht wird; eine Sicherung von Hand mit
`scripts/backup.sh` wird beim ersten Lauf nach 7 Tagen gelöscht. Die Nachtsicherungen liegen im
Container, diese beiden in `backups/` im Programmverzeichnis
([SCRIPTS.md](SCRIPTS.md) §3). Alle Sicherungen liegen auf demselben Server —
eine Kopie außer Haus ist vorbereitet, aber nicht eingerichtet ([OFFEN.md](OFFEN.md) 7.3).
Wird sie als S3-Speicher eingerichtet, braucht der Speicher eine eigene Löschregel: Das
Programm lädt dorthin nur hoch und löscht nie. Sonntags um 03:30 UTC spielt das Programm
die jüngste Sicherung probeweise in eine Wegwerf-Datenbank ein; misslingt das, meldet es die
Betriebsbereitschaft als kritisch, und die tägliche Alarm-Mail geht hinaus.

**Was man braucht:** die zwei Schlüssel aus der `.env`, beide außerhalb des Servers verwahrt
(Ort auf dem Blatt). Die `.env` steht in keiner Sicherung.

- Ohne `BACKUP_ENCRYPTION_KEY` — den, der zur Zeit der Sicherung galt — lässt sich keine
  Sicherung öffnen.
- Ohne `APP_ENCRYPTION_KEY` kommt die Datenbank zurück, aber Schülerfotos und das gespeicherte
  Mail-Passwort bleiben unlesbar.

**Ablauf:** [resilience_and_recovery.md](resilience_and_recovery.md) §2a (einspielen), §2c
(der Weg zurück, falls es misslingt), §2d (Arbeitsdateien löschen — sie enthalten alle Namen
im Klartext). Ist der Server selbst verloren: §2f, auf einem neuen Rechner und mit einer
Sicherung, die nicht auf dem alten lag. Danach die Betriebsbereitschaft ansehen: „Schlüssel und Bestand" meldet, ob der
laufende Schlüssel zu den Daten passt. Buchcover sind nicht in der Sicherung. Fehlen sie nach
einer Wiederherstellung auf einem neuen Server, lädt das Programm sie nach einem Befehl aus
[DEPLOYMENT.md](DEPLOYMENT.md) §6 neu; von Hand hochgeladene Cover kommen so nicht zurück.

**Üben:** Die Probe von Hand an einem fremden Ziel ([resilience_and_recovery.md](resilience_and_recovery.md)
§2e, [OFFEN.md](OFFEN.md) 7.4) ist zugleich die Probe dieses Dokuments: Die Vertretung macht sie
einmal allein, nur mit diesen Seiten.

---

## 4. Wiederkehrende Aufgaben

| Was                          | Wann                                                                                                                        | Woran man es merkt                                                                                                                               | Was zu tun ist                                                                                                                                                                                                                       |
| ---------------------------- | --------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Abhängigkeiten               | wöchentlich, montags 06:00 Berliner Zeit                                                                                    | Dependabot öffnet Pull Requests; die CI prüft sie                                                                                                | einzeln ansehen, bei grüner CI übernehmen. Kein automatisches Übernehmen; TypeScript 7 ist bewusst zurückgehalten (`.github/dependabot.yml`)                                                                                         |
| Sicherheitsprüfung           | bei jedem Push und montags 07:00 UTC                                                                                        | `.github/workflows/security-scan.yml` wird rot                                                                                                   | Abhängigkeit heben. Gibt es keinen Fix und trifft die Lücke den Code nicht: Ausnahme nach den Regeln in `security/vuln-ausnahmen.json` — mit Nachweis als Test und Wiedervorlage                                                     |
| Wiedervorlage einer Ausnahme | je Eintrag in `security/vuln-ausnahmen.json`; seit dem 25.09.2026 ist die Liste leer                                        | ab dem Tag nach der Wiedervorlage ist die Sicherheitsprüfung rot, bei jedem Push und im Wochenlauf                                               | nachsehen, ob es einen Fix gibt; dann die Abhängigkeit heben und die Ausnahme löschen                                                                                                                                                |
| Go                           | halbjährlich (Februar, August); unterstützt sind die zwei neuesten Linien, zurzeit 1.26 und 1.27                            | Dependabot schlägt die neue Docker-Basis vor; ein Test verlangt dieselbe Version in `go.mod` und `Dockerfile` (`docs/umgebung_paritaet_test.go`) | `go.mod` und `Dockerfile` gemeinsam heben; `golangci-lint` und `govulncheck` am Arbeitsplatz mitziehen, sonst verweigern die Hooks; die festen Versionen von gosec und govulncheck in `.github/workflows/security-scan.yml` mitheben |
| Node                         | Node 24 ist bis zum 20. Oktober 2026 aktive LTS, danach in Wartung bis 30. April 2028; Node 26 wird am 28. Oktober 2026 LTS | Projektregel: immer die aktive LTS; ein Test verlangt dieselbe Hauptversion an allen Stellen                                                     | `Dockerfile` und `.github/workflows/ci.yml` gemeinsam heben                                                                                                                                                                          |
| Runner-Abbild der Prüfläufe  | wenn GitHub ein neues Ubuntu-Abbild bereitstellt oder das eingesetzte abkündigt; seit dem 08.10.2026 `ubuntu-26.04` | Ankündigung von GitHub; alle vier Workflows nennen das Abbild fest, `ubuntu-latest` nimmt keiner | Probelauf mit `gh workflow run ci.yml -f runner=<Abbild>`, dann `runs-on` in allen vier Workflows umstellen und den Namen in `.github/actionlint.yaml` eintragen, solange actionlint ihn nicht kennt; den ersten Lauf je Workflow ansehen |
| PostgreSQL, Hauptversion     | 18 wird bis 14. November 2030 gepflegt                                                                                      | ein Test verlangt eine Hauptversion an allen Stellen (`docs/umgebung_paritaet_test.go`)                                                          | nur über Sicherung und Wiederherstellung ([DEPLOYMENT.md](DEPLOYMENT.md) §5); den `pg_dump`-Client im `Dockerfile` mitziehen, sonst schlägt die sonntägliche Probe Alarm                                                             |
| PostgreSQL, Nebenversion     | vierteljährlich                                                                                                             | —                                                                                                                                                | entschieden am 28.09.2026: `update.sh` holt das Datenbank-Image bei jedem Update neu; nicht gebaut ([OFFEN.md](OFFEN.md) 5.31)                                                                                                       |
| Pakete im Backend-Image      | laufend (Sicherheitskorrekturen von Alpine)                                                                                 | —                                                                                                                                                | entschieden am 28.09.2026: `update.sh` baut mit `--pull --no-cache`; bis dahin hält der Build-Cache die Schicht mit `apk upgrade` fest, und ein Neubau holt die Korrekturen nicht ([OFFEN.md](OFFEN.md) 5.31)                        |
| Update einspielen            | mindestens einmal im Monat, auch ohne neue Funktionen (entschieden am 28.09.2026)                                           | —                                                                                                                                                | ein Update nach 3.1; sobald 5.31 gebaut ist, holt es dabei die Sicherheitskorrekturen von Datenbank und Paketen                                                                                                                      |
| Ferien                       | Sommerferien Hessens stehen bis 2030 im Programm, die übrigen Ferien bis zum Schuljahr 2029/30                              | ab Januar 2029: Test rot, Warnung in der Betriebsbereitschaft                                                                                    | neue Termine von KMK und Kultusministerium eintragen (`pkg/lmfplan/ferien.go`, `schulferien.go`). Sommerferien kann die Schule selbst eintragen: Einstellungen → LUSD & Versetzung → Sommerferien                                    |
| Betriebsbereitschaft         | täglich                                                                                                                     | Alarm-Mail bei kritischem Befund                                                                                                                 | Befund, Folge und Abhilfe stehen in der Mail ([FACHKONZEPT.md](FACHKONZEPT.md) §15)                                                                                                                                                  |

Feiertage rechnet das Programm selbst aus (`pkg/lmfplan/feiertage.go`); sie brauchen keine
Pflege. Die beweglichen Ferientage legt jede Schule selbst; sie stehen nicht im Programm.

Die Alarm-Mails gehen an die Adressen unter Einstellungen → Erreichbarkeit & Alarme, ist das
Feld leer, an alle aktiven Admin-Konten. **Vorschlag:** Entwicklung und Vertretung stehen dort.

---

## 5. Wenn ein Prüflauf rot ist

Für die Entwicklung und für jeden, der sie übernimmt.

- **Auf `main` rot:** nicht einspielen. Ein Release entsteht ohnehin nicht (Abschnitt 2).
- **Eine Ratsche** (ein Test, der eine bekannte Fehlerart im ganzen Code sucht) nennt im
  Kopfkommentar ihren Anlass und was sie nicht sieht, meist auch, was bei Rot zu tun ist; die
  Übersicht steht in [sweeps.md](sweeps.md) („Landkarte der Ratschen"). Repariert wird die Ursache. Eine
  Liste erlaubter Ausnahmen zu verlängern oder eine Zahl zu erhöhen, lockert die Prüfung und
  braucht eine Begründung in der Commit-Nachricht.
- **Rot ohne Änderung am Code** ist in drei Fällen gewollt: Eine Schwachstelle wird neu
  veröffentlicht, eine Wiedervorlage der Sicherheits-Ausnahmen läuft ab, oder ein
  Horizont-Test der Ferien erreicht sein Jahr. Dann ist die Aufgabe aus Abschnitt 4 fällig.
- **Wo die Begründungen stehen:** in den Commit-Nachrichten (`git log --grep`), in
  [invarianten.md](invarianten.md) (was immer gelten muss, und das Raster aus neunzehn Fragen,
  wenn ein Schreibpfad seine Form wechselt), in [ARCHITEKTUR.md](ARCHITEKTUR.md#9-architekturentscheidungen),
  Kapitel 9 (Entscheidungen), und in [OFFEN.md](OFFEN.md) (alles Offene).

**Fälle, die vorkamen,** je mit dem Tag:

- **Die Sicherheitsprüfung ist rot, am Stand hat sich nichts geändert (npm, 06.10.2026).** Zu
  einem Paket, das nur über ein anderes hereinkommt, wurde eine Schwachstelle veröffentlicht.
  Im Ordner `frontend` nennt `npm ls <Paket>`, wer es hereinzieht. Geprüft wird ohne die
  Entwicklungspakete (`npm audit --audit-level=high --omit=dev`); `@tailwindcss/vite` steht
  unter `dependencies`, sein Unterbaum zählt deshalb mit. Abhilfe: `npm update` mit genau den
  gemeldeten Paketnamen; das ändert nur `package-lock.json`. Die Sammel-PRs von Dependabot
  heben direkte Pakete und enthielten die gemeldeten nicht.
- **Nach einem Update der Pakete baut das Frontend nicht mehr (21.08.2026).** Ein `npm update`
  ohne Paketnamen hob auch den Bundler, und `npm run build` brach an gültigem Code;
  svelte-check, ESLint und Vitest blieben grün. Pakete deshalb einzeln heben und danach
  `npm run build` laufen lassen; der Hook vor dem Push baut das Frontend nicht.
- **Go, eine Schwachstelle ohne genannten Fix (17.09.2026).** Die Meldung nannte alle
  Fassungen als betroffen. Erst am Quelltext der benutzten Fassung nachsehen, welcher Zweig
  ungeschützt ist und ob die eigenen Aufrufe ihn erreichen; die Antwort als Test ins
  Repository, dann die Ausnahme nach Abschnitt 4. Der Fall steht in
  [SECURITY.md](SECURITY.md) („Automatische Sicherheitsprüfungen").
- **CodeQL meldet eine Stelle, die mit der Quelle nichts zu tun hat (17.09.2026).** Geht ein
  Aufruf gegen eine Schnittstelle (`io.Writer`), nimmt CodeQL jede Methode dieser Form im
  Programm als Ziel. Den gemeldeten Weg aus der Analyse lesen (`codeFlows` im SARIF, über
  `gh api`), nicht raten.
- **Eine verworfene CodeQL-Meldung ist wieder da (06.10.2026).** Zieht die Stelle im Code um,
  schließt CodeQL die alte Meldung und legt eine mit neuer Nummer an. Vor der Bewertung die
  verworfenen lesen (`gh api "repos/uuuxy/bibliothek/code-scanning/alerts?state=dismissed"`)
  und die Anlagezeit der neuen gegen die Schließzeit der alten halten. Der Kommentar an einer
  verworfenen Meldung fasst 280 Zeichen; die Begründung steht im Code neben der Stelle.
- **Ein grüner Job trägt eine rote Markierung (17.09.2026).** `actions/setup-go` liest jede
  Logzeile der Form `datei.go: Text` als Fehlermeldung des Compilers. Skripte, die in einem
  Workflow laufen, nennen eine Datei deshalb als „Testname in pfad.go";
  `docs/vuln_ausnahmen_form_test.go` hält diese Form für die Sicherheits-Ausnahmen fest.
- **Die Browser-Tests (`e2e`) sind in der CI rot, am Arbeitsplatz war alles grün
  (16.09.2026).** Die Hooks fahren die Browser-Tests nicht. Diese greifen über Beschriftungen
  und Rollen zu. Wer ein sichtbares Wort umbenennt, eine Eingabe zur Pflicht macht oder eine
  Antwort ändert, sucht vorher in `frontend/e2e/` (auch in `helpers.js`) nach dem Wort und dem
  Endpunkt und lässt die Suite am neu gebauten Stack laufen ([SCRIPTS.md](SCRIPTS.md) §4).
- **Am Arbeitsplatz scheitert jede Spec.** Zuerst die Umgebung: Läuft Docker (`docker info`)?
  Nach einem Update von Playwright fehlen die Browser der neuen Fassung
  (`npx playwright install chromium`). Läuft schon eine Suite? Solange ihr Merkzettel
  `frontend/.e2e-hauptlieferant` liegt, bricht der Aufbau jeder weiteren ab
  (`e2e/global-setup.js`), auch der einer einzelnen Spec.
- **Ein Frontend-Test ist rot, und die Meldung zeigt auf die falsche Zeile (06.10.2026).** Gibt
  ein Haken eine Funktion zurück, ruft Vitest sie nach dem Test als Aufräumer.
  `beforeEach(() => mock.mockReset())` gibt den Mock selbst zurück; lehnt er dann ab, kommt
  der Fehler aus dem Aufräumer. Den Rumpf des Hakens in geschweifte Klammern setzen.
- **svelte-check meldet „Invalid character" in einem Kommentar (25.09.2026).** Steht in einer
  JSDoc-Zeile direkt hinter dem Namen des Parameters ein deutsches Anführungszeichen („), liest
  der Parser es als Teil des Namens. Ein Wort davor genügt.
- **Welche Prüfläufe zu einem Stand gehören,** zeigt `gh run list --commit <sha>` mit der
  vollen Kennung des Commits (`git rev-parse <kurz>`); mit der gekürzten bleibt die Liste leer
  (07.10.2026). Die Liste je Zweig zeigte am 05.10.2026 nach einem Push nur ältere Läufe.
- **Ein Lauf steht als „cancelled" (23.08.2026).** Das heißt „nicht geprüft", nicht „nichts
  gefunden". Auf einem Zweig bricht ein neuer Push den laufenden Lauf ab; auf `main` läuft
  jeder zu Ende (`.github/workflows/ci.yml`, `cancel-in-progress` gilt nur außerhalb von
  `main`). Vorher brach bei Pushes im Abstand weniger Minuten jeder Lauf den vorigen ab, und
  `main` war in der Zeit ungeprüft. Der Bau des Images bricht weiter ab
  (`docker-publish.yml`): Dort zählt nur der neueste Stand.
- **Der Job der Browser-Tests endet rot, bevor ein Test läuft (07.10.2026, zweimal).** Im
  Schritt „Install Playwright" blieb das Lesen der Paketlisten an einer Paketquelle von GitHub
  stehen und lief in die Frist von 600 s. Das `apt-get` dahinter läuft unter `sudo` und
  überlebte den Abbruch; es hielt die Sperre der Paketlisten, und die zwei weiteren Versuche
  scheiterten binnen Sekunden an ihr („Could not get lock /var/lib/apt/lists/lock").
  Seit dem 07.10.2026 beendet der Schritt vor jedem neuen Versuch ein übrig gebliebenes
  `apt-get` (`.github/workflows/ci.yml`, `beende_apt`). Am Code liegt ein solcher Lauf nicht;
  die gescheiterten Jobs starten mit `gh run rerun <Nummer des Laufs> --failed` neu.
- **`actionlint` ist rot nach dem Wechsel des Runner-Abbilds (08.10.2026).** actionlint führt
  die Namen der Abbilder von GitHub als feste Liste je Fassung; die jüngste kannte
  `ubuntu-26.04` nicht und meldete jede Zeile `runs-on` damit („label … is unknown"). Der
  Probelauf davor war grün, weil er den Namen als Eingabe übergibt und die Dateien noch den
  alten trugen. Der Name steht seitdem in `.github/actionlint.yaml`; kennt ihn eine neue
  Fassung von actionlint, fällt der Eintrag weg. Am Arbeitsplatz nachstellen: das Abbild aus
  `ci.yml` (`docker run --rm -v "$PWD:/repo:ro" --workdir /repo rhysd/actionlint:…`).

---

## 6. Kann jemand anderes das Programm weiterführen?

Gemessen am 24.09.2026. **Der Code:** außerhalb der Tests keine Go-Funktion über 150 Zeilen
(die längste hat 148),
23 direkte Go-Abhängigkeiten, übliche Bausteine (Go mit `net/http` und `pgx`, Svelte 5,
Tailwind, PostgreSQL, Docker Compose), Tests und CI, Betrieb mit einem Befehl. Die
Architektur ist nach arc42 beschrieben ([ARCHITEKTUR.md](ARCHITEKTUR.md)).

**Was bremst,** liegt um den Code: 2.499 Commits seit dem 29.05.2026, die Zahl der
Projektregeln (`CLAUDE.md`, [invarianten.md](invarianten.md), [sweeps.md](sweeps.md)) und
Wissen außerhalb des Repositorys (Abschnitt 7).

---

## 7. Wissen außerhalb des Repositorys

### 7.1 Quelldokumente

Sie liegen nicht im Repository, weil es öffentlich ist; das Handbuch ist zudem
urheberrechtlich geschützt, die Sicherung enthält Personendaten. Ort auf dem Blatt.

- das Handbuch des bisherigen Bibliotheksprogramms Littera
- die Littera-Sicherung `littera_sav.mdb` (Stand 2010; ein neuerer Stand steht aus,
  [OFFEN.md](OFFEN.md) 7.2)
- die Arbeitshilfe zu Mahnschreiben (Erlass vom 17.12.2014, Az. 674.100.002-00178)
- die Anforderungsliste zum Mahnverfahren („Ablauf Mahnverfahren"); abgeglichen in
  [mittel_konzept.md](mittel_konzept.md) Abschnitt 3

### 7.2 Arbeitsnotizen der Entwicklung

Entwickelt wird mit einem KI-Assistenten. Dessen Arbeitsnotizen zum Projekt — Entscheidungen,
Fallen, Messungen, am 07.10.2026 257 Einträge — liegen außerhalb des Repositorys. Was davon
für die Pflege zählt, steht seit dem 07.10.2026 im Repository, jede Aussage vor dem Eintragen
am Stand des Tages geprüft:

| Was | Wo |
| --- | --- |
| Fälle roter Prüfläufe, Prüfung einer Änderung von außen | Abschnitte 5 und 2 |
| Fehlerarten mit ihrem Gate | [sweeps.md](sweeps.md), Register |
| Handgriffe beim Prüfen | [ARCHITEKTUR.md](ARCHITEKTUR.md) 8.14 |
| Druck, Mail, Fehlerantworten, Regeln der Oberfläche | [ARCHITEKTUR.md](ARCHITEKTUR.md) 8.9, 8.10, 8.6, 5.3 |
| Littera und LUSD | [littera_schema_befund.md](littera_schema_befund.md), [LUSD.md](LUSD.md), [SCRIPTS.md](SCRIPTS.md) |
| Regeln des Fachs, verworfene Vorschläge | [FACHKONZEPT.md](FACHKONZEPT.md), dort §20 |

Draußen bleibt, was Zugänge, Orte oder Namen nennt (Abschnitt 7.3), was nur den Arbeitsplatz
der Entwicklung betrifft, und der Verlauf einzelner Sitzungen; der steht in den
Commit-Nachrichten.

Neue Notizen entstehen weiter. Damit sich nichts wieder ansammelt, gilt seit dem 07.10.2026:
Was für die Pflege zählt, kommt mit dem Commit, der es hervorbringt, in eines der Dokumente
oben (`CLAUDE.md`, Abschnitt 4).

### 7.3 Das Blatt bei der Schule

Auf Papier bei der Schule, nicht im Repository:

- Namen und Erreichbarkeit: Entwicklung, Vertretung, IT des Schulträgers
- Namen und Erreichbarkeit von Schulleitung und Datenschutzbeauftragtem — für eine Datenpanne
  ([Datenschutz-Nachweis](datenschutz/nachweis.md), Abschnitt 8)
- Zugang zum Server (Adresse, Konto) und Programmverzeichnis
- Ort der Kopie von `APP_ENCRYPTION_KEY` und `BACKUP_ENCRYPTION_KEY` außerhalb des Servers
- Ort der Quelldokumente (7.1)
- Zugang zum Repository mit Schreibrecht, falls die Pflege übergeben wird

Vorlage zum Ausdrucken: der [Anhang](#anhang-das-blatt-bei-der-schule--vorlage) am Ende dieses
Dokuments.

---

## 8. Wenn die Pflege endet

1. **Übergabe** an eine andere Stelle. Der Code steht unter EUPL 1.2; jede Stelle darf ihn
   übernehmen, ändern und weitergeben.
2. **Findet sich niemand,** läuft das Programm bis zum Ende des Schuljahres weiter. Die Daten
   kommen aus der nächtlichen Sicherung, und die Schule wechselt auf ein Kaufprogramm. Was es
   dafür gibt: die Sicherung selbst, eine vollständige PostgreSQL-Datenbank, und die
   Bestandsliste als CSV (Einstellungen → Datenverwaltung). Sie trägt je Exemplar Titel,
   Autor, Verlag, ISBN, Jahr, Kategorie, Barcode, Zustand, Signatur, Schlagworte, Eigentum
   und Standort — den Teil des Bestands, der sich nicht neu erfassen lässt. Mehrere
   Schlagworte stehen in einer Zelle, getrennt durch „ | "; das Eigentum ist „Land" oder
   „Schulträger", wie auf dem Etikett; der Standort ist der des Exemplars, ohne Eintrag bleibt
   die Zelle leer. Die Liste führt auch bestellte, noch nicht eingetroffene Exemplare (Zustand
   „Im Zulauf …") und Titel ohne Exemplar (Zeile ohne Barcode). Leser und Ausleihen gibt das
   Programm nicht in einer Form aus, die ein anderes Programm einliest (entschieden am
   01.10.2026): Schüler kommen in jedem Programm aus der LUSD, das Kollegium meldet sich neu
   an, und gewechselt wird zum Schuljahresende, wenn die Lernmittel zurück sind.
3. **Ausgeschlossen** ist ein unbefristeter Weiterbetrieb ohne Sicherheitsupdates. Einen Weg
   zurück zu Littera gibt es nicht: Das Programm gibt keine Daten in Litteras Importform aus.

---

## 9. Offene Stellen

1. **Betriebssystem und Docker auf dem Server:** vorgesehen ist die IT des Schulträgers, mit
   automatischen Sicherheitsupdates in der Nacht (entschieden am 28.09.2026); die Zusage des
   Schulträgers steht aus.
2. **`update.sh` für den Schulserver:** nur Releases einspielen und die Images bei jedem Update
   frisch holen — entschieden am 28.09.2026, nicht gebaut ([OFFEN.md](OFFEN.md) 5.31).
3. **Erreichbarkeit von außen:** entschieden am 28.09.2026 — von außen nur die Seite für die
   Lieferanten, alles andere nur aus dem Schulnetz. Offen sind die Sperre am Eingang und die
   Angaben des Schulträgers dazu: Name im Internet, Freigabe von Port 443, Absenderadresse der
   Schulgeräte ([OFFEN.md](OFFEN.md) 4.23).
4. **Vertretung:** noch nicht benannt.
5. **Betrieb:** Sicherung außer Haus (entschieden am 28.09.2026: zuerst beim Schulträger nach
   einem Speicher fragen), externes Signal bei Ausfall, Probe der Wiederherstellung an einem
   fremden Ziel — [OFFEN.md](OFFEN.md) 7.3, 7.5 und 7.4.

---

## Anhang: Das Blatt bei der Schule — Vorlage

Zum Ausdrucken und Ausfüllen von Hand. Das ausgefüllte Blatt liegt auf Papier bei der Schule,
nicht im Repository: Es trägt Zugänge und den Ort der Schlüssel. Wozu die Angaben dienen, steht
in Abschnitt 7.3.

Ausgefüllt am: ________________ von: ______________________________

### Erreichbarkeit

| Rolle                   | Name                         | E-Mail                         | Telefon              |
| ----------------------- | ---------------------------- | ------------------------------ | -------------------- |
| Entwicklung             | ____________________________ | ______________________________ | ____________________ |
| Vertretung              | ____________________________ | ______________________________ | ____________________ |
| IT des Schulträgers     | ____________________________ | ______________________________ | ____________________ |
| Schulleitung            | ____________________________ | ______________________________ | ____________________ |
| Datenschutzbeauftragter | ____________________________ | ______________________________ | ____________________ |

**Einen Fehler melden:** per E-Mail an die Entwicklung, die Vertretung in Kopie; bei Stillstand
zusätzlich anrufen. Nie über GitHub (Abschnitt 1).

**Bei einer Datenpanne:** sofort Bibliotheksleitung und Admin, dann der Ablauf im
[Datenschutz-Nachweis](datenschutz/nachweis.md), Abschnitt 8.

### Server

| Was                               | Eintrag                                              |
| --------------------------------- | ---------------------------------------------------- |
| Adresse des Servers               | ____________________________________________________ |
| Konto für die Anmeldung am Server | ____________________________________________________ |
| Programmverzeichnis               | ____________________________________________________ |

### Schlüssel

Die beiden Schlüssel aus der `.env` liegen als Kopie außerhalb des Servers. Ohne
`BACKUP_ENCRYPTION_KEY` lässt sich keine Sicherung öffnen, ohne `APP_ENCRYPTION_KEY` bleiben
Schülerfotos und das Mail-Passwort unlesbar (Abschnitt 3.2).

| Schlüssel               | Ort der Kopie außerhalb des Servers                  |
| ----------------------- | ---------------------------------------------------- |
| `APP_ENCRYPTION_KEY`    | ____________________________________________________ |
| `BACKUP_ENCRYPTION_KEY` | ____________________________________________________ |

### Weiteres

| Was                                                                     | Eintrag                          |
| ----------------------------------------------------------------------- | -------------------------------- |
| Ort der Quelldokumente (Abschnitt 7.1)                                  | ________________________________ |
| Zugang zum Repository mit Schreibrecht, falls die Pflege übergeben wird | ________________________________ |
