# Pflegekonzept und Wartungshandbuch

Stand: 23.10.2026 (Entwurf)

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

1. **Code:** öffentlich auf GitHub (`uuuxy/bibliothek`), Lizenz EUPL 1.2 (`LICENSE`). Nach
   `main` kommt ein Pull Request nur mit der Freigabe des Eigentümers des Repositorys
   (`.github/CODEOWNERS`) und mit allen acht Prüfläufen aus Schritt 3 grün. Ein neuer Push hebt
   die Freigabe auf, und es zählen nur Läufe von GitHub Actions, kein über die Schnittstelle
   gesetzter Status. Die Kollaborateure haben Schreibrecht, kommen aber nur so nach `main`;
   ausgenommen sind die Admins, die direkt auf `main` arbeiten. `main` lässt sich weder löschen
   noch umschreiben, Release-Tags setzen nur Admins (Regelwerke des Repositorys auf GitHub).
2. **Prüfung vor dem Hochladen:** Auf dem Arbeitsplatz der Entwicklung laufen vor jedem Commit
   Formatierung und Lint, vor jedem Push die Tests und die Sicherheitsprüfungen
   (`scripts/install-hooks.sh`, [SCRIPTS.md](SCRIPTS.md) §5).
3. **Prüfung auf GitHub:** vier CI-Jobs und vier Sicherheitsprüfungen je Stand. Ein Release
   (eine Versionsnummer wie `v2.14.0`) entsteht nur, wenn alle acht grün sind
   ([DEPLOYMENT.md](DEPLOYMENT.md) §8, `scripts/tag-gate.sh`). Was beim Einspielen von Hand zu
   tun ist, steht in der Release-Notiz.
4. **Auf den Server** kommt ein Stand nur von Hand, mit `update.sh`: Vorab-Sicherung der
   Datenbank → Bau des neuen Stands → Gesundheitsprüfung → Abgleich, ob der laufende Stand der
   eingespielte ist ([DEPLOYMENT.md](DEPLOYMENT.md) §2.4 und §7). Ein automatisches Update gibt
   es nicht. **Entschieden am 28.09.2026:** Der Schulserver bekommt nur Releases, der
   Testserver folgt `main` als Vorstufe. Welchen Stand `update.sh` holt, sagt `UPDATE_STAND` in
   der `.env`: `release` das neueste Release, `main` (oder kein Eintrag) den Zweig, ohne zu
   fragen, ob seine Prüfläufe grün sind.
5. **Die Datenbank** passt sich beim Start selbst an (Migrationen). Migrationen laufen nur
   vorwärts: Zurück geht es über die Vorab-Sicherung, nicht über den alten Code allein.

**Eine Änderung von außen** — ein Pull Request, auch von Dependabot oder von einem Werkzeug,
das Vorschläge selbst erzeugt — wird vor dem Übernehmen so geprüft:

- Grün in der CI genügt nicht; rot ist ein Befund und kein Rauschen. Ein Job endet am ersten
  roten Schritt: Was dahinter steht, lief in dem Pull Request nicht.
- Die Versionshinweise jeder angehobenen Fassung lesen. Was dort als Änderung am Verhalten
  steht, wird am eigenen Code nachgestellt, mit alter und neuer Fassung, bevor es übernommen
  wird.
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
- Hebt er eine Action der Prüfläufe: `actionlint` prüft die Eingaben (`with:`) nur bei
  Actions, die es unter ihrem Tag kennt. Hier ist jede Action an einen Commit geheftet; ein
  falscher Name einer Eingabe bleibt dort grün (gemessen am 12.10.2026 mit dem Abbild aus
  `ci.yml`). Deshalb `action.yml` zwischen alter und neuer Version lesen
  (`gh api repos/<Action>/compare/<alt>...<neu>`) und den angehefteten Commit gegen das Tag
  halten, das der Kommentar daneben nennt (`gh api repos/<Action>/git/ref/tags/<Tag>`). Ein
  Schritt mit `if: failure()` läuft in einem grünen Lauf nicht mit.
- Ein Zweig auf GitHub ist kein offener Vorschlag: `gh pr list --state all --head <Zweig>`
  zeigt, ob er schon übernommen oder abgelehnt ist.
- Ändert der Vorschlag eine Funktion ohne Test, gehört der Test zur Übernahme.

---

## 3. Die zwei Handgriffe der Vertretung

### 3.1 Ein Update einspielen

1. Auf GitHub nachsehen, ob die Prüfläufe des neuesten Stands grün sind, und die Hinweise
   zum Einspielen lesen (Release-Notiz). Am Schulserver wird nur ein Release eingespielt
   (Abschnitt 2).
2. Am Schulserver im Programmverzeichnis `./update.sh`; es wechselt selbst auf das neueste
   Release. Am Testserver erst `git pull`, dann `./update.sh` — zwei getrennte Befehle, weil
   sonst die alte Fassung des Skripts das Update fährt ([DEPLOYMENT.md](DEPLOYMENT.md) §2.4).
3. **Erfolg** heißt: Das Skript endet mit „UPDATE ERFOLGREICH ABGESCHLOSSEN", und unter
   System → Einstellungen → Betriebsbereitschaft steht kein kritischer Befund.
4. **Misserfolg:** Das Skript bricht ab und gibt eine Anleitung zum Zurückgehen aus: den
   Commit, der vor dem Update lief (aus dem laufenden Image gelesen), und den Pfad der
   Vorab-Sicherung. Ihr folgen. Die Vorab-Sicherung ist in diesem Fall nicht verschlüsselt;
   die Einzelheiten stehen in [resilience_and_recovery.md](resilience_and_recovery.md) §2b
   („Der Klartext-Fall") und §2c.

### 3.2 Eine Sicherung zurückholen

**Was es gibt:** Jede Nacht um 02:30 UTC eine verschlüsselte Sicherung der Datenbank; die
jüngsten 14 bleiben, dazu von den älteren je Kalenderwoche eine für 12 Wochen, damit ein
Fehler, der erst nach den Sommerferien auffällt, noch eine Sicherung von
davor vorfindet. Ein älterer Stand bringt Personen zurück, die seitdem von Hand endgültig
gelöscht wurden; wie sie nachgeholt werden, steht in
[resilience_and_recovery.md](resilience_and_recovery.md), Abschnitt 2a, Schritte 5b und 8. Dazu legt `update.sh` vor jedem Update eine
Sicherung an, die beim ersten Update nach 30 Tagen gelöscht wird; eine Sicherung von Hand mit
`scripts/backup.sh` wird beim ersten Lauf nach 7 Tagen gelöscht. Die Nachtsicherungen liegen im
Container, diese beiden in `backups/` im Programmverzeichnis
([SCRIPTS.md](SCRIPTS.md) §2). Alle Sicherungen liegen auf demselben Server —
eine Kopie außer Haus ist vorbereitet, aber nicht eingerichtet ([OFFEN.md](OFFEN.md) 7.3).
Wird sie als S3-Speicher eingerichtet, löscht das Programm dort nach derselben Regel wie bei den
Nachtsicherungen auf dem Server; die Versionierung am Speicher bleibt aus, sonst behielte er
jede gelöschte Sicherung als ältere Fassung. Sonntags um 03:30 UTC spielt das Programm
die jüngste Sicherung probeweise in eine Wegwerf-Datenbank ein; misslingt das, meldet es die
Betriebsbereitschaft als kritisch, und die tägliche Alarm-Mail geht hinaus.

**Was man braucht:** die zwei Schlüssel aus der `.env`, beide außerhalb des Servers verwahrt
(Ort auf dem Blatt). Die `.env` steht in keiner Sicherung.

- Ohne `BACKUP_ENCRYPTION_KEY` — den, der zur Zeit der Sicherung galt — lässt sich keine
  Sicherung öffnen. Nach einem Wechsel bleibt der alte Schlüssel deshalb in der Kopie, bis die
  letzte Sicherung von davor gelöscht ist; der Wechsel und das Öffnen einer älteren Sicherung
  stehen in [SECURITY.md](SECURITY.md#backup_encryption_key-wechseln).
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
| Abhängigkeiten               | wöchentlich, montags 06:00 Berliner Zeit                                                                                    | Dependabot öffnet Pull Requests; die CI prüft sie                                                                                                | einzeln nach Abschnitt 2 prüfen, dann übernehmen; grün in der CI allein genügt nicht. Kein automatisches Übernehmen; TypeScript 7 ist bewusst zurückgehalten (`.github/dependabot.yml`)                                                                                         |
| Sicherheitsprüfung           | bei jedem Push und montags 07:00 UTC                                                                                        | `.github/workflows/security-scan.yml` wird rot                                                                                                   | Abhängigkeit heben. Gibt es keinen Fix und trifft die Lücke den Code nicht: Ausnahme nach den Regeln in `security/vuln-ausnahmen.json` — mit Nachweis als Test und Wiedervorlage                                                     |
| Fuzz-Suche                   | nachts 02:17 UTC, je Fuzz-Test 15 Minuten                                                                                   | `.github/workflows/fuzz.yml` wird rot; die Meldung nennt die Eingabe                                                                             | Eingabe als Startwert in den Fuzz-Test, Ursache beheben ([ARCHITEKTUR.md](ARCHITEKTUR.md) 8.14). Ein Fund heißt: Go und Datenbank rechnen bei dieser Eingabe verschieden, oder eine Normalform schreibt beim zweiten Durchgang noch um |
| Wiedervorlage einer Ausnahme | je Eintrag in `security/vuln-ausnahmen.json`; zurzeit ist die Liste leer                                                    | ab dem Tag nach der Wiedervorlage ist die Sicherheitsprüfung rot, bei jedem Push und im Wochenlauf                                               | nachsehen, ob es einen Fix gibt; dann die Abhängigkeit heben und die Ausnahme löschen                                                                                                                                                |
| Go                           | halbjährlich (Februar, August); unterstützt sind die zwei neuesten Linien, zurzeit 1.26 und 1.27                            | Dependabot schlägt die neue Docker-Basis vor; ein Test verlangt dieselbe Version in `go.mod` und `Dockerfile` (`docs/umgebung_paritaet_test.go`) | `go.mod`, `go.work` und `Dockerfile` gemeinsam heben; `golangci-lint` und `govulncheck` am Arbeitsplatz mitziehen, sonst verweigern die Hooks; die feste Version von govulncheck in `.github/workflows/security-scan.yml` und die von gosec in `scripts/gosec-gate.sh` mitheben |
| Node                         | Node 26 ist bis zum 20. Oktober 2027 aktive LTS, danach in Wartung bis 30. April 2029; Node 27 wird am 20. Oktober 2027 LTS | Projektregel: immer die aktive LTS; ein Test verlangt dieselbe Hauptversion an allen Stellen                                                     | `Dockerfile` und `.github/workflows/ci.yml` gemeinsam heben                                                                                                                                                                          |
| Runner-Abbild der Prüfläufe  | wenn GitHub ein neues Ubuntu-Abbild bereitstellt oder das eingesetzte abkündigt; eingesetzt ist `ubuntu-26.04` | Ankündigung von GitHub; alle fünf Workflows nennen das Abbild fest, `ubuntu-latest` nimmt keiner | Probelauf mit `gh workflow run ci.yml -f runner=<Abbild>`, dann `runs-on` in allen fünf Workflows umstellen und den Namen in `.github/actionlint.yaml` eintragen, solange actionlint ihn nicht kennt; den ersten Lauf je Workflow ansehen |
| PostgreSQL, Hauptversion     | 18 wird bis 14. November 2030 gepflegt                                                                                      | ein Test verlangt eine Hauptversion an allen Stellen (`docs/umgebung_paritaet_test.go`)                                                          | nur über Sicherung und Wiederherstellung ([DEPLOYMENT.md](DEPLOYMENT.md) §5); den `pg_dump`-Client im `Dockerfile` mitziehen, sonst schlägt die sonntägliche Probe Alarm                                                             |
| PostgreSQL, Nebenversion     | vierteljährlich                                                                                                             | bringt das Image eine neue ICU-Fassung, steht im Log der Datenbank `collation "deutsch" has version mismatch`                                    | `update.sh` holt das Datenbank-Image bei jedem Update neu (Schritt 3); eine Nebenversion braucht weder Sicherung noch Neuaufbau, und die Datenbank sortiert ohne Sprachregeln (`datlocprovider` = c), ein neues Alpine ändert also keine Indizes. Nach der Warnung einmal `ALTER COLLATION deutsch REFRESH VERSION`; kein Index benutzt die Kollation, neu aufzubauen ist nichts (Migration 165) |
| Pakete im Backend-Image      | laufend (Sicherheitskorrekturen von Alpine)                                                                                 | —                                                                                                                                                | `update.sh` baut mit `--pull` und leert danach den Build-Cache: Das nächste Update holt die Basis-Images neu und baut die Schicht mit `apk upgrade` neu                        |
| Update einspielen            | mindestens einmal im Monat, auch ohne neue Funktionen (entschieden am 28.09.2026)                                           | —                                                                                                                                                | ein Update nach 3.1; es holt dabei die Sicherheitskorrekturen von Datenbank und Paketen                                                                                                                      |
| Ferien                       | Sommerferien Hessens stehen bis 2030 im Programm, die übrigen Ferien bis zum Schuljahr 2029/30                              | ab Januar 2029: Test rot, Warnung in der Betriebsbereitschaft                                                                                    | neue Termine von KMK und Kultusministerium eintragen (`pkg/lmfplan/sommerferien.go`, `uebrige_ferien.go`). Sommerferien kann die Schule selbst eintragen: Einstellungen → LUSD & Versetzung → Sommerferien                                 |
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

**Fälle, die vorkamen:**

- **Die Sicherheitsprüfung ist rot, am Stand hat sich nichts geändert (npm).** Zu einem
  Paket, das nur über ein anderes hereinkommt, ist eine Schwachstelle veröffentlicht. Im Ordner
  `frontend` nennt `npm ls <Paket>`, wer es hereinzieht. Geprüft wird ohne die
  Entwicklungspakete (`npm audit --audit-level=high --omit=dev` in `scripts/npm-audit-gate.sh`, das
  auch der Hook vor dem Push ruft: Ein Befund hält dort den Push an); `@tailwindcss/vite` steht
  unter `dependencies`, sein Unterbaum zählt deshalb mit. Abhilfe: `npm update` mit genau den
  gemeldeten Paketnamen; das ändert nur `package-lock.json`. Die Sammel-PRs von Dependabot
  heben direkte Pakete; ein gemeldetes Paket aus dem Unterbaum ist darin nicht unbedingt
  enthalten.
- **Die Sicherheitsprüfung ist rot, am Stand hat sich nichts geändert (Go-Bibliothek).** Der
  Job „Docker – Trivy image scan" meldet Schwachstellen in einer Bibliothek, etwa
  `xuri/excelize`; govulncheck und Dependabot melden dabei nicht unbedingt etwas. Gibt es keine
  veröffentlichte Fassung mit den Korrekturen und nennt jede Meldung einen Entwicklungsstand
  der Bibliothek: auf den Stand heben, den die jüngste Meldung nennt
  (`go get github.com/xuri/excelize/v2@<Commit>`, dann `go mod tidy`). Am Arbeitsplatz zeigt
  `trivy fs --scanners vuln --severity HIGH,CRITICAL --ignore-unfixed --exit-code 1 <Ordner>`
  an einem Ordner mit dem alten `go.mod` und `go.sum` die Meldungen und am neuen keine.
  Danach die Go-Suite, bei excelize auch ein Lesevergleich an echten Excel-Dateien: ein
  kleines Programm, einmal mit jedem Stand gebaut, das je Blatt die Zahl der Zeilen und Zellen
  und eine Prüfsumme über die Werte ausgibt, roh und formatiert. Trifft eine Meldung einen
  Aufruf des Programms (`OpenReader`, `GetRows`), kommt ein Test dazu, der am alten Stand rot
  ist (`pkg/xlsxgrenze/`); den alten Stand nimmt
  `GOWORK=off go test -modfile=<altes go.mod> ./pkg/xlsxgrenze/`. Auf die veröffentlichte
  Fassung hebt der Vorschlag von Dependabot, sobald es sie gibt.
- **Der Push bricht an der Schwachstellen-Prüfung ab, am Stand hat sich nichts geändert
  (Go-Standardbibliothek).** govulncheck meldet Schwachstellen in der Standardbibliothek, die
  eine neue Nebenversion von Go behebt. Abhilfe: die Fassung an drei Stellen gemeinsam heben
  (`go.mod`, `go.work`, `FROM golang:…` im `Dockerfile`; `docs/umgebung_paritaet_test.go`
  hält sie gleich) und die Module aus `golang.org/x/…` auf die Fassung, die die Meldung nennt
  (`go get golang.org/x/net@<Fassung>`). Vorher nachsehen, dass es die Fassung gibt:
  `GOTOOLCHAIN=go<Fassung> go version` und `docker manifest inspect golang:<Fassung>-alpine`.
  Liest ein Werkzeug die Paketdaten der neuen Go-Fassung nicht, meldet golangci-lint an jeder
  Datei „could not import … export data version … is greater than maximum supported version
  …", und gosec beendet jedes Paket mit „has type errors". Dann braucht das Werkzeug eine
  neuere Fassung: golangci-lint am Arbeitsplatz (`brew upgrade golangci-lint`) und in
  `.github/workflows/ci.yml` (`version:`), sonst scheitert schon der Hook vor dem Commit und
  in der CI der Lint-Schritt vor den Tests; gosec in `scripts/gosec-gate.sh`, das der
  Sicherheits-Prüflauf und der Hook vor dem Push nehmen. Die Werkzeuge der CI vor dem Push am
  Arbeitsplatz nachstellen, mit `GOBIN=<Ordner> go install …@<Fassung>` und dem Aufruf
  wörtlich aus dem Workflow: Mit dem zusätzlichen Schalter `-quiet` endet gosec ohne Fund mit
  Exit 0, auch wenn es kein Paket laden konnte. Danach die Go-Suite und ein Bau des Images
  unter einem Probe-Namen (`docker build -t <Name> .`). Weist die neue Fassung Zertifikate ab
  (die Liste `distrustedSPKIHashes` im Paket `crypto/x509`, so seit Go 1.27.3), vorher
  die Kette des Mailservers der Schule dagegen halten, vom Server aus:
  `openssl s_client -showcerts` gegen `IMAP_HOST` auf Port 993, je Zertifikat
  `openssl x509 -pubkey -noout | openssl pkey -pubin -outform der | sha256sum`. Steht ein
  Schlüssel in der Liste, sperrte das Update die Anmeldung aus. Vom Arbeitsplatz aus verfälscht
  ein Virenschutz, der TLS aufbricht, die Probe.
- **gosec meldet eine Stelle, an der sich nur der Weg der Daten geändert hat.** Zieht eine
  Abfrage aus einer Tür nach `repository/`, kommt die Zeile als Rückgabe einer Funktion, die
  die Kennung aus der Anfrage als Argument bekommt, und die Analyse wertet die Rückgabe als von
  der Anfrage abhängig. Ebenso, wenn der Erzeuger eines PDFs aus `api/` in ein anderes Paket
  zieht: Die Tür schreibt dann die Rückgabe einer fremden Funktion in die Antwort. Der Job
  „Go – gosec static analysis" meldet dort die Regel G705 (XSS über Taint-Analyse), so am
  Bescheid in `api/schaden/` beim Schreiben des PDFs in die Antwort. Ist der Körper wie dort
  das erzeugte PDF, ausgeliefert als `application/pdf`, als Anhang und mit `nosniff`, kommt
  der Vermerk `#nosec G705` mit Grund an die Stelle. Der Hook vor dem Push fährt gosec über
  dasselbe Skript wie der Prüflauf (`scripts/gosec-gate.sh`); nach einem solchen Umzug das
  Skript schon vor dem Commit laufen lassen.
- **Nach einem Update der Pakete baut das Frontend nicht mehr.** Ein `npm update` ohne
  Paketnamen hebt auch den Bundler; `npm run build` kann dann an gültigem Code brechen, während
  svelte-check, ESLint und Vitest grün bleiben. Pakete deshalb einzeln heben und danach
  `npm run build` laufen lassen; der Hook vor dem Push baut das Frontend nicht.
- **Nach einem Update von axe-core ist der Browser-Test zur Barrierefreiheit rot.** Eine neue
  Fassung kann eine Regel von sich aus einschalten, so axe-core 4.14.0 mit
  `label-content-name-mismatch` (der sichtbare Text eines Knopfs steht in seinem Namen, WCAG
  2.5.3). Gemeldet werden dann Verstöße, die schon im Code standen; behoben wird im Code, die
  Regel bleibt an. Welche Regel neu ist, sagen die Versionshinweise
  (`gh api repos/dequelabs/axe-core/releases/tags/v<Fassung>`). `e2e/barrierefreiheit-axe.spec.js`
  sieht nur die Hauptansichten im Grundzustand und in der CI nur die Testdaten: Gegen die
  Entwicklungsdatenbank (`E2E_BASE_URL`, siehe `playwright.config.js`) zeigte dieselbe Spec eine
  Stelle mehr, die ISBN-Knöpfe im Katalog. Die Namensregel hält am Quelltext
  `frontend-hygiene-namen.test.js`.
- **Go, eine Schwachstelle ohne genannten Fix.** Nennt die Meldung alle Fassungen als
  betroffen: erst am Quelltext der benutzten Fassung nachsehen, welcher Zweig ungeschützt ist
  und ob die eigenen Aufrufe ihn erreichen; die Antwort als Test ins Repository, dann die
  Ausnahme nach Abschnitt 4. Ein Beispiel ist GO-2026-6452 mit den Tests in
  `pkg/xlsxgrenze/negativer_sharedstring_test.go` ([SECURITY.md](SECURITY.md), „Automatische
  Sicherheitsprüfungen").
- **CodeQL meldet eine Stelle, die mit der Quelle nichts zu tun hat.** Geht ein Aufruf gegen
  eine Schnittstelle (`io.Writer`), nimmt CodeQL jede Methode dieser Form im Programm als
  Ziel. Den gemeldeten Weg aus der Analyse lesen (`codeFlows` im SARIF, über `gh api`), nicht
  raten.
- **Eine verworfene CodeQL-Meldung ist wieder da.** Zieht die Stelle im Code um, schließt
  CodeQL die alte Meldung und legt eine mit neuer Nummer an. Vor der Bewertung die verworfenen
  lesen (`gh api "repos/uuuxy/bibliothek/code-scanning/alerts?state=dismissed"`) und die
  Anlagezeit der neuen gegen die Schließzeit der alten halten. Der Kommentar an einer
  verworfenen Meldung fasst 280 Zeichen; die Begründung steht im Code neben der Stelle.
- **Ein grüner Job trägt eine rote Markierung.** `actions/setup-go` liest jede Logzeile der
  Form `datei.go: Text` als Fehlermeldung des Compilers. Skripte, die in einem Workflow laufen,
  nennen eine Datei deshalb als „Testname in pfad.go"; `docs/vuln_ausnahmen_form_test.go` hält
  diese Form für die Sicherheits-Ausnahmen fest.
- **Die Browser-Tests (`e2e`) sind in der CI rot, am Arbeitsplatz war alles grün.** Die Hooks
  fahren die Browser-Tests nicht. Diese greifen über Beschriftungen und Rollen zu. Wer ein
  sichtbares Wort umbenennt, eine Eingabe zur Pflicht macht oder eine Antwort ändert, sucht
  vorher in `frontend/e2e/` (auch in `helpers.js`) nach dem Wort und dem Endpunkt und lässt die
  Suite am neu gebauten Stack laufen ([SCRIPTS.md](SCRIPTS.md) §3).
- **Am Arbeitsplatz scheitert jede Spec.** Zuerst die Umgebung: Läuft Docker (`docker info`)?
  Nach einem Update von Playwright fehlen die Browser der neuen Fassung
  (`npx playwright install chromium`). Läuft schon eine Suite? Solange ihr Merkzettel
  `frontend/.e2e-hauptlieferant` liegt, bricht der Aufbau jeder weiteren ab
  (`e2e/global-setup.js`), auch der einer einzelnen Spec.
- **Ein Frontend-Test ist rot, und die Meldung zeigt auf die falsche Zeile.** Gibt ein Haken
  eine Funktion zurück, ruft Vitest sie nach dem Test als Aufräumer.
  `beforeEach(() => mock.mockReset())` gibt den Mock selbst zurück; lehnt er dann ab, kommt
  der Fehler aus dem Aufräumer. Den Rumpf des Hakens in geschweifte Klammern setzen.
- **svelte-check meldet „Invalid character" in einem Kommentar.** Steht in einer JSDoc-Zeile
  direkt hinter dem Namen des Parameters ein deutsches Anführungszeichen („), liest der Parser
  es als Teil des Namens. Ein Wort davor genügt.
- **Welche Prüfläufe zu einem Stand gehören,** zeigt `gh run list --commit <sha>` mit der
  vollen Kennung des Commits (`git rev-parse <kurz>`); mit der gekürzten bleibt die Liste
  leer. Die Liste je Zweig kann direkt nach einem Push noch ohne den neuen Lauf sein.
- **Ein Lauf steht als „cancelled".** Das heißt „nicht geprüft", nicht „nichts gefunden". Auf
  einem Zweig bricht ein neuer Push den laufenden Lauf ab; auf `main` läuft jeder zu Ende
  (`.github/workflows/ci.yml`, `cancel-in-progress` gilt nur außerhalb von `main`), sonst
  bliebe `main` bei Pushes im Abstand weniger Minuten ungeprüft. Der Bau des Images bricht ab
  (`docker-publish.yml`): Dort zählt nur der neueste Stand.
- **Der Job der Browser-Tests endet rot, bevor ein Test läuft.** Im Schritt „Install
  Playwright" kann das Lesen der Paketlisten an einer Paketquelle von GitHub stehen bleiben
  und in die Frist von 600 s laufen. Das `apt-get` dahinter läuft unter `sudo` und überlebt
  den Abbruch; es hält die Sperre der Paketlisten, an der jeder weitere Versuch binnen Sekunden
  scheitern würde („Could not get lock /var/lib/apt/lists/lock"). Deshalb beendet der Schritt
  vor jedem neuen Versuch ein übrig gebliebenes `apt-get` (`.github/workflows/ci.yml`,
  `beende_apt`). Am Code liegt ein solcher Lauf nicht; die gescheiterten Jobs starten mit
  `gh run rerun <Nummer des Laufs> --failed` neu.
- **Der Linter der Oberfläche ist rot nach einem Sammel-Update, am Code hat sich nichts
  geändert.** Welches Paket eine Meldung bringt, zeigt ein eigenes Arbeitsverzeichnis mit dem
  Stand des Pull Requests: nach `npm ci` je ein angehobenes Paket auf die alte Fassung
  zurücksetzen (`npm install --no-save --ignore-scripts <Paket>@<Fassung>`) und den Linter
  fahren. Meldet ESLint eine Eigenschaft als unbenutzt, die ihr Bauteil nur setzt und über
  `bind:` an den Aufrufer gibt (`$bindable()`), ist die Meldung falsch; sie kam mit
  typescript-eslint 8.71.1, 8.71.2 behebt sie.
- **`actionlint` ist rot nach dem Wechsel des Runner-Abbilds.** actionlint führt die Namen der
  Abbilder von GitHub als feste Liste je Fassung; kennt die eingesetzte Fassung einen Namen
  nicht, meldet sie jede Zeile `runs-on` damit („label … is unknown"). Ein Probelauf mit dem
  Namen als Eingabe bleibt grün, solange die Dateien den alten tragen. Der Name kommt in
  `.github/actionlint.yaml`; kennt ihn eine neue Fassung von actionlint, fällt der Eintrag weg.
  Am Arbeitsplatz nachstellen: das Abbild aus `ci.yml`
  (`docker run --rm -v "$PWD:/repo:ro" --workdir /repo rhysd/actionlint:…`).
- **Ein Browser-Test scheitert an „no space left on device“, die Platte des Arbeitsplatzes
  ist voll.** Am Code liegt es nicht. Den Platz belegen vor allem der Bauzwischenspeicher von
  Go und der Build-Cache von Docker. Go legt je Pfad einer Arbeitskopie eigene Einträge an und
  räumt sie erst nach Tagen; mehrere Wegwerf-Arbeitskopien am Tag und Proben per Overlay
  füllen ihn schnell. Docker hält den Build-Cache an der eingestellten Grenze (`builder.gc` in
  der Einstellungsdatei des Dienstes), zählt dabei aber keine Schicht, die auf einem
  Basis-Image aufbaut, das auch als Image im lokalen Speicher liegt. Nachgestellt: Nach
  `docker pull golang:1.27.2-alpine` führte `docker buildx du` 5,7 GB mehr als „Shared“, nach
  `docker rmi` wieder nicht. Abhilfe: `go clean -cache` und
  `docker builder prune --all --force`. Ein Zeitfilter (`--filter until=168h`) hilft nicht: Der
  Lauf löscht je Kette nur die äußerste Schicht und führt ihre Eltern danach als eben benutzt
  (gemessen an `docker buildx du --verbose`: 33 von 159 GB gelöscht, 336 von 337 Eltern trugen
  danach die neue Zeit). Volumes bleiben unberührt, dort liegen Datenbanken. Vorbeugen: Eine
  Wegwerf-Arbeitskopie baut in einen eigenen Zwischenspeicher (`GOCACHE=<Ordner>`), der mit
  ihr gelöscht wird; was Docker belegt, zeigen `docker system df` und die letzten Zeilen von
  `docker buildx du`. Am Server leert `update.sh` den Build-Cache nach jedem Update ganz
  (Schritt 7).
- **Nach einem abgebrochenen `rundgang-rechte.spec.js` sind viele Browser-Tests rot.** Der Test stellt
  die Rolle Mitarbeiter nacheinander auf einzelne Rechte und erst am Ende zurück. Bricht der Lauf ab,
  bleibt sie auf einem Recht stehen, und ein Neubau ändert das nicht: Die Vorgabe (`db/rechte_vorgabe.go`)
  ergänzt nur fehlende Zeilen. Abhilfe: die Haken der Rolle unter Benutzer & Rechte nach der Vorgabe
  setzen. Ab Werk hält sie 18 von 24 Rechten.
- **Das Seitenbild eines roten Browser-Tests zeigt eine Seite, die der Test nie geöffnet hat.**
  Zu einem roten Test legt Playwright Abzüge aller offenen Seiten des Browsers ab, und das
  Seitenbild in `error-context.md` kann von einer davon stammen. Ein Kontext aus
  `browser.newContext()` bleibt offen, bis ihn jemand schließt; Playwright tut es erst mit dem
  Ende des Workers, und der läuft nach grünen Tests weiter. Jede Spec schließt deshalb am Ende
  des Tests die Kontexte, die sie selbst geöffnet hat.
- **Ein Browser-Test, der die Anmeldung in der Datenbank ändert, ist am Arbeitsplatz manchmal
  rot, in der CI grün.** Eine Anfrage der Seite, die erst nach der Änderung beim Server ankommt,
  bekommt 423. Der Sperrbildschirm kommt dann vor dem nächsten Schritt des Tests, und dessen
  Klick endet mit „element is not visible“: Hinter dem Sperrbildschirm ist die Anwendung
  ausgeblendet. Am Arbeitsplatz antworten Listen bei den Resten vieler Testläufe langsamer; in
  der Titel-Verwaltung kam die zweite gleichlautende Abfrage der Titelliste erst beim Server
  an, als die Antwort auf die erste begann, rund 180 ms später. Der Test wartet deshalb vor der
  Änderung, bis keine Anfrage der Seite mehr offen ist (`beobachteDatenanfragen` in
  `e2e/messhilfe.js`). Wann welche Anfrage ankam, zeigt das Protokoll des Backends
  (`docker logs`).
- **Wer einen roten Fall der ganzen Suite allein wiederholt, verliert dessen Spur.** Playwright
  leert seinen Ordner für Ergebnisse (`test-results/`) zu Beginn jedes Laufs, samt Trace und
  Seitenabzug des roten Falls. Die ganze Suite läuft deshalb mit `--output <Zwischenordner>`,
  oder der Ordner wird vor der Wiederholung gesichert.

---

## 6. Kann jemand anderes das Programm weiterführen?

Gemessen am 21.10.2026 am Go-Code ohne Tests: 603 Dateien, 2.619 Funktionen.

**Was trägt:**

- Die Stelle findet sich über den Menüpunkt. Die Türen stehen in zwölf Ordnern nach den
  Menüpunkten der Oberfläche, je 6 bis 26 Dateien, und tragen kein SQL. Den Einstieg gibt
  [WEGWEISER.md](WEGWEISER.md) auf einer Seite.
- Eine Datei lässt sich in einem Zug lesen: Die Hälfte hat höchstens 107 Zeilen, neun von
  zehn höchstens 267. 17 haben über 400, eine über 600.
- Funktionen sind kurz: Die Hälfte hat höchstens 13 Zeilen, 18 haben über 80, die längste 148
  (eine Liste von Adressen).
- Namen und Kommentare folgen einem Regelwerk ([STIL.md](STIL.md)): Fachwörter deutsch, je
  Sache ein Wort; ein Datum steht in einem Kommentar nur zu einer Quelle.
  `stil_ratsche_test.go` meldet jede Abweichung im Go-Code,
  `frontend-hygiene-vorgeschichte.test.js` jedes Datum in einem Kommentar der Oberfläche,
  `docs/kommentare_ohne_datum_test.go` jedes in Skripten, Workflows und Konfiguration.
- 24 direkte Go-Abhängigkeiten, übliche Bausteine (Go mit `net/http` und `pgx`, Svelte 5,
  Tailwind, PostgreSQL, Docker Compose), Tests und CI, Betrieb mit einem Befehl. Die
  Architektur ist nach arc42 beschrieben ([ARCHITEKTUR.md](ARCHITEKTUR.md)).

**Was bremst:**

- An der Grenze mischen sich die Sprachen: Adressen, Felder in JSON, Tabellen und Spalten
  behalten ihre Namen, deutsche wie englische ([STIL.md](STIL.md), Abschnitt 9).
- Die Datenbankschicht ist ein Ordner mit 168 Dateien; gefunden wird dort über den Namen der
  Datei. Warum sie ein Paket bleibt, steht in [ARCHITEKTUR.md](ARCHITEKTUR.md) 5.2.2.
- Die größte Datei hat 629 Zeilen (`repository/leser_zusammenfuehren.go`).
- Um den Code: über 3.700 Commits seit dem 29.05.2026, die Zahl der Projektregeln (`CLAUDE.md`,
  [invarianten.md](invarianten.md), [sweeps.md](sweeps.md)), ein Architektur-Dokument von
  rund 3.600 Zeilen und Wissen außerhalb des Repositorys (Abschnitt 7).

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
Fallen, Messungen — liegen außerhalb des Repositorys. Was davon für die Pflege zählt, steht im
Repository, jede Aussage vor dem Eintragen am Stand des Tages geprüft:

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

Neue Notizen entstehen weiter. Damit sich nichts ansammelt, gilt: Was für die Pflege zählt,
kommt mit dem Commit, der es hervorbringt, in eines der Dokumente oben (`CLAUDE.md`,
Abschnitt 4).

Die Dokumente nennen, was gilt; wie es dazu kam, steht in der Commit-Nachricht. Ein Datum
steht nur bei einer Entscheidung, einer Quelle, einer Frist oder einer Messung, deren
Gültigkeit es begrenzt. `docs/vorgeschichte_ratsche_test.go` zählt je Dokument die Zeilen mit
Datum und lässt die Zahl nur sinken.

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
   Autor, Verlag, ISBN, Jahr, Kategorie, Barcode, Zustand, Signatur, Schlagworte, Eigentum,
   Standort und LMF-Nummer — den Teil des Bestands, der sich nicht neu erfassen lässt. Mehrere
   Schlagworte stehen in einer Zelle, getrennt durch „ | "; das Eigentum ist „Land" oder
   „Schulträger", wie auf dem Etikett; der Standort ist der des Exemplars, ohne Eintrag bleibt
   die Zelle leer. Die LMF-Nummer ist die des alten Etiketts eines Lernmittels: Bis es
   überklebt ist, findet ein anderes Programm das Buch nur über sie. Die Liste führt auch
   bestellte, noch nicht eingetroffene Exemplare (Zustand „Im Zulauf …") und Titel ohne
   Exemplar (Zeile ohne Barcode). Leser und Ausleihen gibt das
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
2. **Erreichbarkeit von außen:** entschieden am 28.09.2026 — von außen nur die Seite für die
   Lieferanten, alles andere nur aus dem Schulnetz. Die Sperre steht in der Caddy-Konfiguration
   ([DEPLOYMENT.md](DEPLOYMENT.md), Abschnitt 3). Offen sind die Angaben des Schulträgers dazu:
   Name im Internet, Freigabe von Port 443 und 80, Netze der Schulgeräte.
3. **Vertretung:** noch nicht benannt.
4. **Betrieb:** Sicherung außer Haus (entschieden am 28.09.2026: zuerst beim Schulträger nach
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

Nach einem Wechsel des `BACKUP_ENCRYPTION_KEY` bleibt der frühere Schlüssel in der Kopie, bis
keine Sicherung von davor mehr liegt ([SECURITY.md](SECURITY.md#backup_encryption_key-wechseln)):

| Früherer `BACKUP_ENCRYPTION_KEY` | Eintrag                                       |
| -------------------------------- | --------------------------------------------- |
| Tag des Wechsels                 | _____________________________________________ |
| Ausgetragen am                   | _____________________________________________ |

### Weiteres

| Was                                                                     | Eintrag                          |
| ----------------------------------------------------------------------- | -------------------------------- |
| Ort der Quelldokumente (Abschnitt 7.1)                                  | ________________________________ |
| Zugang zum Repository mit Schreibrecht, falls die Pflege übergeben wird | ________________________________ |
