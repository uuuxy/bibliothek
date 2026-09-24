# Pflegekonzept und Wartungshandbuch

Stand: 24.09.2026 (Entwurf)

Dieses Dokument beantwortet zwei Fragen. Für die Schule und den Schulträger: Wer betreibt und
pflegt das Programm, wie kommt eine Änderung auf den Server, und was geschieht, wenn die Pflege
endet? Für die Vertretung: Wie spielt man ohne die Entwicklung ein Update ein und holt eine
Sicherung zurück?

Befehlsfolgen stehen hier nicht. Sie stehen jeweils an einer Stelle, auf die dieses Dokument
verweist, und dort halten Tests sie mit dem Code in Übereinstimmung
(`docs/rueckweg_anleitungen_test.go`, `docs/deployment_anleitung_test.go`).

**Nicht in diesem Dokument**, weil das Repository öffentlich ist: Zugänge, die Adresse des
Servers, der Ort der Schlüssel und der Quelldokumente, Namen und Telefonnummern. Das alles
steht auf einem Blatt, das bei der Schule liegt (Abschnitt 7.3).

---

## 1. Zuständigkeiten

| Rolle             | Wer                                       | Aufgabe                                                                                    |
| ----------------- | ----------------------------------------- | ------------------------------------------------------------------------------------------ |
| Betrieb           | die Schule, auf eigenem Server            | nutzt das Programm; Ansprechpartner für Einstellungen, Rechte und Fragen der Bibliothek    |
| Hardware und Netz | die IT des Schulträgers                   | Server, Netzanbindung                                                                      |
| Pflege            | die Entwicklung des Programms             | Fehler beheben, Sicherheitsupdates, Termine und Vorgaben nachziehen, Updates bereitstellen |
| Vertretung        | eine benannte Person (Name auf dem Blatt) | ein Update einspielen (3.1), eine Sicherung zurückholen (3.2)                              |

Von der Vertretung wird nicht mehr verlangt als die zwei Handgriffe in Abschnitt 3. Änderungen am Code
bleiben bei der Entwicklung oder bei einer Stelle, die die Pflege übernimmt (Abschnitt 8).

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
4. **Auf den Server** kommt ein Stand nur von Hand, mit `update.sh`: Vorab-Sicherung → neuer
   Code → Bau → Gesundheitsprüfung → Abgleich, ob der laufende Stand der eingespielte ist
   ([DEPLOYMENT.md](DEPLOYMENT.md) §2.4 und §7). Ein automatisches Update gibt es nicht.
5. **Die Datenbank** passt sich beim Start selbst an (Migrationen). Migrationen laufen nur
   vorwärts: Zurück geht es über die Vorab-Sicherung, nicht über den alten Code allein.

---

## 3. Die zwei Handgriffe der Vertretung

### 3.1 Ein Update einspielen

1. Die Release-Notiz lesen: Sie nennt, was von Hand zu tun ist.
2. Am Server im Programmverzeichnis erst `git pull`, dann `./update.sh` — zwei getrennte
   Befehle, weil das Skript sich sonst während des Laufs selbst ersetzt
   ([DEPLOYMENT.md](DEPLOYMENT.md) §2.4).
3. **Erfolg** heißt: Das Skript endet mit „UPDATE ERFOLGREICH ABGESCHLOSSEN", und unter
   System → Einstellungen → Betriebsbereitschaft steht kein kritischer Befund.
4. **Misserfolg:** Das Skript bricht ab und gibt eine Anleitung zum Zurückgehen samt Pfad der
   Vorab-Sicherung aus. Ihr folgen; die Einzelheiten stehen in
   [resilience_and_recovery.md](resilience_and_recovery.md) §2a („Nach einem fehlgeschlagenen
   Deploy") und §2c.

### 3.2 Eine Sicherung zurückholen

**Was es gibt:** Jede Nacht um 02:30 UTC eine verschlüsselte Sicherung der Datenbank; die
letzten 14 bleiben. Sie liegen auf demselben Server — eine Kopie außer Haus ist vorbereitet,
aber nicht eingerichtet ([OFFEN.md](OFFEN.md) 7.3). Sonntags um 03:30 UTC spielt das Programm
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
im Klartext). Danach die Betriebsbereitschaft ansehen: „Schlüssel und Bestand" meldet, ob der
laufende Schlüssel zu den Daten passt. Buchcover sind nicht in der Sicherung; das Programm lädt
sie nach ([DEPLOYMENT.md](DEPLOYMENT.md) §6, mit dem Befehl dafür).

**Üben:** Die Probe von Hand an einem fremden Ziel ([resilience_and_recovery.md](resilience_and_recovery.md)
§2e, [OFFEN.md](OFFEN.md) 7.4) ist zugleich die Probe dieses Dokuments: Die Vertretung macht sie
einmal allein, nur mit diesen Seiten.

---

## 4. Wiederkehrende Aufgaben

| Was                          | Wann                                                                                                                        | Woran man es merkt                                                                      | Was zu tun ist                                                                                                                                                                                    |
| ---------------------------- | --------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Abhängigkeiten               | wöchentlich, montags 06:00 Berliner Zeit                                                                                    | Dependabot öffnet Pull Requests; die CI prüft sie                                       | einzeln ansehen, bei grüner CI übernehmen. Kein automatisches Übernehmen; TypeScript 7 ist bewusst zurückgehalten (`.github/dependabot.yml`)                                                      |
| Sicherheitsprüfung           | bei jedem Push und montags 07:00 UTC                                                                                        | `.github/workflows/security-scan.yml` wird rot                                          | Abhängigkeit heben. Gibt es keinen Fix und trifft die Lücke den Code nicht: Ausnahme nach den Regeln in `security/vuln-ausnahmen.json` — mit Nachweis als Test und Wiedervorlage                  |
| Wiedervorlage einer Ausnahme | nächste am 17. November 2026 (`GO-2026-6452`)                                                                               | das Gate wird am Tag selbst rot                                                         | nachsehen, ob es einen Fix gibt ([OFFEN.md](OFFEN.md) 5.10)                                                                                                                                       |
| Go                           | halbjährlich (Februar, August); unterstützt sind die zwei neuesten Linien, zurzeit 1.26 und 1.27                            | Dependabot (Docker-Basis)                                                               | `go.mod` und `Dockerfile` gemeinsam heben; `golangci-lint` und `govulncheck` am Arbeitsplatz mitziehen, sonst verweigern die Hooks                                                                |
| Node                         | Node 24 ist bis zum 20. Oktober 2026 aktive LTS, danach in Wartung bis 30. April 2028; Node 26 wird am 28. Oktober 2026 LTS | Projektregel: immer die aktive LTS                                                      | `Dockerfile` und `.github/workflows/ci.yml` gemeinsam heben                                                                                                                                       |
| PostgreSQL, Hauptversion     | 18 wird bis 14. November 2030 gepflegt                                                                                      | ein Test verlangt eine Hauptversion an allen Stellen (`docs/umgebung_paritaet_test.go`) | nur über Sicherung und Wiederherstellung ([DEPLOYMENT.md](DEPLOYMENT.md) §5); den `pg_dump`-Client im `Dockerfile` mitziehen, sonst schlägt die sonntägliche Probe Alarm                          |
| PostgreSQL, Nebenversion     | vierteljährlich                                                                                                             | —                                                                                       | `update.sh` holt das Datenbank-Image nicht neu ([OFFEN.md](OFFEN.md) 7.8)                                                                                                                         |
| Ferien                       | Sommerferien und übrige Ferien Hessens stehen bis 2030 im Programm                                                          | ab Januar 2029: Test rot, Warnung in der Betriebsbereitschaft                           | neue Termine von KMK und Kultusministerium eintragen (`pkg/lmfplan/ferien.go`, `schulferien.go`). Sommerferien kann die Schule selbst eintragen: Einstellungen → LUSD & Versetzung → Sommerferien |
| Betriebsbereitschaft         | täglich                                                                                                                     | Alarm-Mail bei kritischem Befund                                                        | Befund, Folge und Abhilfe stehen in der Mail ([FACHKONZEPT.md](FACHKONZEPT.md) §15)                                                                                                               |

Feiertage rechnet das Programm selbst aus (`pkg/lmfplan/feiertage.go`); sie brauchen keine
Pflege. Die beweglichen Ferientage legt jede Schule selbst; sie stehen nicht im Programm.

Die Alarm-Mails gehen an die Adressen unter Einstellungen → Erreichbarkeit & Alarme, ist das
Feld leer, an alle aktiven Admin-Konten. **Vorschlag:** Entwicklung und Vertretung stehen dort.

---

## 5. Wenn ein Prüflauf rot ist

Für die Entwicklung und für jeden, der sie übernimmt.

- **Auf `main` rot:** nicht einspielen. Ein Release entsteht ohnehin nicht (Abschnitt 2).
- **Eine Ratsche** (ein Test, der eine bekannte Fehlerart im ganzen Code sucht) nennt im
  Kopfkommentar ihren Anlass, was sie nicht sieht und was bei Rot zu tun ist; die Übersicht
  steht in [sweeps.md](sweeps.md) („Landkarte der Ratschen"). Repariert wird die Ursache. Eine
  Liste erlaubter Ausnahmen zu verlängern oder eine Zahl zu erhöhen, lockert die Prüfung und
  braucht eine Begründung in der Commit-Nachricht.
- **Rot ohne Änderung am Code** ist bei zwei Arten von Tests gewollt: Die Horizont-Tests der
  Ferien und die Wiedervorlagen der Sicherheits-Ausnahmen werden an einem Datum rot. Dann ist
  die Aufgabe aus Abschnitt 4 fällig.
- **Wo die Begründungen stehen:** in den Commit-Nachrichten (`git log --grep`), in
  [invarianten.md](invarianten.md) (was immer gelten muss, und das Raster aus vierzehn Fragen,
  wenn ein Schreibpfad seine Form wechselt), in [arc42/09](arc42/09-architekturentscheidungen.md)
  (Entscheidungen) und in [OFFEN.md](OFFEN.md) (alles Offene).

---

## 6. Kann jemand anderes das Programm weiterführen?

Gemessen am 24.09.2026. **Der Code:** keine Go-Funktion über 150 Zeilen (die längste hat 148),
23 direkte Go-Abhängigkeiten, übliche Bausteine (Go mit `net/http` und `pgx`, Svelte 5,
Tailwind, PostgreSQL, Docker Compose), Tests und CI, Betrieb mit einem Befehl. Die
Architektur ist nach arc42 beschrieben ([arc42/](arc42/README.md)).

**Was bremst,** liegt um den Code: 2.499 Commits seit dem 29.05.2026, die Zahl der
Projektregeln (`CLAUDE.md`, [invarianten.md](invarianten.md), [sweeps.md](sweeps.md)) und
Wissen außerhalb des Repositorys (Abschnitt 7).

---

## 7. Wissen außerhalb des Repositorys

### 7.1 Quelldokumente

Sie liegen nicht im Repository, weil es öffentlich ist: Das Handbuch ist urheberrechtlich
geschützt, die Sicherung enthält Personendaten. Ort auf dem Blatt.

- das Handbuch des bisherigen Bibliotheksprogramms Littera
- die Littera-Sicherung `littera_sav.mdb` (Stand 2010; ein neuerer Stand steht aus,
  [OFFEN.md](OFFEN.md) 7.2)
- die Arbeitshilfe zu Mahnschreiben (Erlass vom 17.12.2014, Az. 674.100.002-00178)
- die Anforderungsliste zum Mahnverfahren („Ablauf Mahnverfahren"); abgeglichen in
  [mittel_konzept.md](mittel_konzept.md) Abschnitt 3

### 7.2 Arbeitsnotizen der Entwicklung

Entwickelt wird mit einem KI-Assistenten. Dessen Arbeitsnotizen zum Projekt — Entscheidungen,
Fallen, Messungen, am 24.09.2026 219 Einträge — liegen außerhalb des Repositorys. Was davon
für die Pflege zählt, kommt entlang der Gliederung dieses Dokuments ins Repository.

### 7.3 Das Blatt bei der Schule

Auf Papier bei der Schule, nicht im Repository:

- Namen und Erreichbarkeit: Entwicklung, Vertretung, IT des Schulträgers
- Zugang zum Server (Adresse, Konto) und Programmverzeichnis
- Ort der Kopie von `APP_ENCRYPTION_KEY` und `BACKUP_ENCRYPTION_KEY` außerhalb des Servers
- Ort der Quelldokumente (7.1)
- Zugang zum Repository mit Schreibrecht, falls die Pflege übergeben wird

---

## 8. Wenn die Pflege endet

1. **Übergabe** an eine andere Stelle. Der Code steht unter EUPL 1.2; jede Stelle darf ihn
   übernehmen, ändern und weitergeben.
2. **Findet sich niemand,** läuft das Programm bis zum Ende des Schuljahres weiter. Die Daten
   kommen aus der nächtlichen Sicherung, und die Schule wechselt auf ein Kaufprogramm.
3. **Ausgeschlossen** ist ein unbefristeter Weiterbetrieb ohne Sicherheitsupdates. Einen Weg
   zurück zu Littera gibt es nicht: Das Programm gibt keine Daten in Litteras Importform aus.

---

## 9. Offene Stellen

1. **Einsatz über die eigene Schule hinaus:** offen gelassen am 24.09.2026.
2. **Meldeweg und Reaktionszeit:** Wie die Schule einen Fehler meldet und wann sie Antwort
   bekommt, ist nicht vereinbart. Littera regelte das über einen Pflegevertrag mit Hotline und
   Fernwartung.
3. **Betriebssystem und Docker auf dem Server:** Wer sie aktualisiert, ist nicht geregelt; die
   IT des Schulträgers ist für Hardware und Netz genannt.
4. **Release oder `main`:** `update.sh` spielt den neuesten Stand von `main` ein, nicht das
   letzte Release. Ob am Schulserver nur Releases eingespielt werden, ist nicht entschieden.
5. **Betrieb:** Sicherung außer Haus, externes Signal bei Ausfall, Probe der Wiederherstellung
   an einem fremden Ziel — [OFFEN.md](OFFEN.md) 7.3, 7.5 und 7.4.
