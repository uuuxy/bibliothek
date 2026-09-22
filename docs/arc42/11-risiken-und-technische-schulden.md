# 11. Risiken und technische Schulden

Stand: 17.09.2026

**Dieses Kapitel führt keine Arbeitsliste.** Was zu tun, zu prüfen und zu entscheiden ist —
und in welcher Reihenfolge —, steht an genau einem Ort: [OFFEN.md](../OFFEN.md). Hier
stehen die **architektonischen** Risiken: das, was aus der Bauweise folgt und auch nach
Abarbeiten der Liste bleibt.

Bewertung: **Auswirkung** (was passiert im Ernstfall) × **Sichtbarkeit** (merkt es jemand?).
Ein Risiko mit geringer Sichtbarkeit ist gefährlicher als eines mit hoher Auswirkung —
weil es unbemerkt eintreten kann.

---

## 11.1 Risiken aus der Bauweise

### R1 — Ein Prozess hält Zustand im Speicher (keine zweite Instanz möglich)

| | |
| --- | --- |
| **Auswirkung** | hoch, wenn je skaliert werden soll |
| **Sichtbarkeit** | gering — es funktioniert, bis jemand eine zweite Instanz startet |

SSE-Abonnenten, Rechte-Cache (60 s), Rate-Limit-Zähler und die Idempotenz-Warteschleife
liegen im Prozessspeicher. Zwei Instanzen hinter einem Load Balancer würden: Ereignisse nur
an die Hälfte der Stationen verteilen, Rechteänderungen ungleich wirksam machen und das
Rate-Limit halbieren-verdoppeln. **Das ist kein Konfigurationsschalter, sondern ein Umbau**
(gemeinsamer Bus/Cache). Für eine Schule mit acht Arbeitsplätzen ist die Entscheidung
richtig — sie muss nur bewusst bleiben.

### R2 — Die Sperrreihenfolge ist Konvention ohne Gate

| | |
| --- | --- |
| **Auswirkung** | hoch (Deadlock im Ausleihbetrieb) |
| **Sichtbarkeit** | mittel — ein Deadlock fällt auf, aber erst unter Last |

Jeder heutige Schreibpfad hält Schüler → Ausleihe → Exemplar (A7). Ein neuer Pfad, der die
Reihenfolge tauscht, verklemmt sich gegen die bestehenden. Es gibt **keinen Detektor**;
die Invariante steht als 🟡 im Katalog. Ein Gate wäre schwer, weil die Sperren über mehrere
Funktionen verteilt sind — der ehrliche Zwischenstand ist der Kommentar an jeder Stelle.

### R3 — Die gefilterte Sicht `schueler` erzeugt stille 404

| | |
| --- | --- |
| **Auswirkung** | mittel (ein Vorgang scheitert wortlos) |
| **Sichtbarkeit** | **gering** — genau das ist das Problem |

Ein Schreibpfad, der alle Leser meint und gegen `schueler` schreibt, trifft beim Kollegen
null Zeilen und meldet „nicht gefunden". So sind der Änderungspfad der Stammdaten und das
Zusammenführen aufgefallen — beide erst im Betrieb. Es gibt inzwischen einen Detektor
(`docs/schreibpfade_gegen_sicht_test.go`), und er ist textbasiert: SQL aus Variablen oder
generischen Helfern sieht er nicht.

### R4 — `api/` ist mit 27.810 Zeilen in 153 Dateien das schwerste Paket

| | |
| --- | --- |
| **Auswirkung** | mittel (Änderungsaufwand, Kollisionen) |
| **Sichtbarkeit** | hoch |

Das Paket trägt Router, Middleware, Handler **und** Teile der Fachlogik (LUSD-Parser,
Selbstprüfung, Bestellwesen). `api/schichtung_test.go` hält die Schichtung — aber
**datei-granular**: Eine Bestandsdatei darf beliebig SQL **dazu**bekommen. Der Schnitt
nach `internal/service` ist begonnen, nicht abgeschlossen.

### R5 — Die Rechtematrix ist konfigurierbar und damit verstellbar

| | |
| --- | --- |
| **Auswirkung** | hoch (Datenschutz) |
| **Sichtbarkeit** | mittel |

`role_permissions` entscheidet in `RequirePermission`. Eine Fehlkonfiguration ist damit
keine Anzeigefrage, sondern eine echte Rechteerweiterung — genau das war am 10.08.2026 der
Fall, als ein Kollegiums-Konto zehn von fünfzehn Menüpunkten sah. Gegenmittel: Die Vorgabe
steht in `db/seed.go`, `leitung` wird abgeleitet (A12), Migrationen laufen nur einmal (eine
spätere Handvergabe wird **nicht** zurückgedreht). Es gibt keinen laufenden Wächter, der
eine verstellte Matrix meldet.

### R6 — Die E-Mail ist die Identität

| | |
| --- | --- |
| **Auswirkung** | hoch (Kontoübernahme) |
| **Sichtbarkeit** | gering, wenn nur auf `rolle` geschaut wird |

Wer `benutzer.email` schreiben darf, übernimmt ein Konto (A2). Abgesichert ist der
gefährlichste Fall (`manage_users` fehlt der Leitung; ein Admin-Konto bleibt ihr
verschlossen). Ein Rechte-Audit, das nur Rollen vergleicht, sieht diesen Weg trotzdem nicht
— er muss in jeder künftigen Rechteänderung mitgedacht werden.

### R7 — Ein Betreiber, ein Wissensstand (Bus-Faktor 1)

| | |
| --- | --- |
| **Auswirkung** | sehr hoch |
| **Sichtbarkeit** | hoch, aber nicht abstellbar |

Entwicklung und Betrieb liegen bei einer Person. Die Gegenmittel sind bewusst gewählt und
Teil der Architektur: Gates statt Checklisten, Selbstprüfung statt Runbook-Gedächtnis,
Begründungen im Code und in der Commit-Historie, ausgedruckter **Rückweg** bei
fehlgeschlagenem Deploy, Wiederherstellungswerkzeuge **im Image**. Das ersetzt keine zweite
Person.

### R8 — Offsite-Backup ist optional und standardmäßig aus

| | |
| --- | --- |
| **Auswirkung** | hoch (Totalverlust des Hosts) |
| **Sichtbarkeit** | mittel — der Job protokolliert das Überspringen |

Ohne vollständige S3-Zugangsdaten liegen alle Sicherungen im Volume **desselben Hosts**.
Der Job sagt es („S3 credentials not fully configured – skipping offsite upload"), und die
Selbstprüfung macht ein fehlendes Backup-Geheimnis sichtbar. Ein Host-Verlust ohne Offsite
ist trotzdem ein Datenverlust.

### R9 — Kein externes Uptime-Signal

| | |
| --- | --- |
| **Auswirkung** | mittel |
| **Sichtbarkeit** | gering bei Totalausfall |

`/health`, Selbstprüfung und Bereitschafts-Wächter laufen **im** System. Fällt der Host
aus, meldet sich niemand — der Wächter braucht denselben Prozess, den er überwachen soll.
Ein externes Signal ist als Handgriff notiert ([OFFEN.md](../OFFEN.md) 7.5). Der Vorfall
dazu ist dokumentiert: `unattended-upgrades` startete den Docker-Daemon neu, und der Dienst
lag **zwölf Stunden** still, bis es jemandem auffiel (Gegenmittel seither:
`restart: unless-stopped`).

### R10 — Alte Aufdrucke bleiben ein Sonderfall

| | |
| --- | --- |
| **Auswirkung** | mittel (Theke bleibt stehen) |
| **Sichtbarkeit** | hoch (fällt sofort auf) |

Die Nachsicht für das Code-39-Prüfzeichen (A14) ist eine **Heuristik mit bekannter
Trefferrate**: Bei 43 möglichen Zeichen sieht im Schnitt jeder 43. gültige Code so aus, als
hinge eines dran. Deshalb greift sie nur als zweiter Versuch. Ein falscher Treffer bleibt
dort möglich, wo der gekürzte Wert existiert und der volle nicht — genau der Fall, den sie
auflösen soll. Solange Karten und Etiketten von früher im Umlauf sind, bleibt diese Naht.

---

## 11.2 Technische Schulden

| #  | Schuld                                                                                                                  | Kosten heute                                                     | Warum sie (noch) steht                                                                      |
| -- | ----------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| D1 | **Invarianten auf Ebene 🟡** (Sperren des Lesers, Überfällig-Automatik, Ausleihlimit, Sperrreihenfolge)                   | Ein zweiter Schreibpfad kann sie auslassen                        | Teils Ermessen enthalten (Override mit Audit), teils über Funktionsgrenzen verteilt          |
| D2 | **Swagger deckt 63 von 206 Routen**                                                                                      | Interaktive Doku ist unvollständig                                | Das **vollständige** Verzeichnis ist generiert (`api_inventar.md`); Annotationen sind Handarbeit |
| D3 | **Doppelte Migrationsnummern** (003, 008, 021, 022)                                                                      | Style-Smell; sortiert deterministisch                            | Umnummerieren würde bereits gelaufene Migrationen betreffen — Risiko ohne Nutzen              |
| D4 | **Frontend-Altbestand über 200 Zeilen**                                                                                  | Große Komponenten sind schwer zu ändern                          | Ratsche friert den Bestand ein (darf nicht wachsen); Umbau läuft nebenher                    |
| D5 | **Gemischte Sprache im Code** (`book`/`loan`/`student` neben `leser`/`ausleihen`)                                         | Kognitive Last beim Lesen                                        | Eine Umbenennung quer durch 78 Repository-Dateien wäre ein Risiko ohne fachlichen Gewinn      |
| D6 | **`docs/` ist Go-Paket und Dokumentverzeichnis in einem**                                                                | Verwirrend; Gates liegen bei den Dokumenten                      | Die Gates **wollen** neben ihren Dokumenten liegen (`stand_angaben`, `invarianten_fundstellen`) |
| D7 | **PG-Tests lokal still übersprungen**                                                                                    | Ein Lauf kann grün aussehen, ohne Constraints geprüft zu haben    | Ein Postgres gehört nicht in einen Push; Gegenmittel ist die Skip-Bilanz                     |
| D8 | **Die `Caddyfile` im Repo ist nicht maßgeblich**                                                                         | Zwei Orte, ein Zustand                                           | Der Host betreibt mehrere Dienste in einer Datei; die Repo-Datei sagt das in Zeile 1          |
| D9 | **`scripts/deploy.sh` ist die zweite Tür zum selben Zustand**                                                            | Die falsche Tür geht irgendwann auf (kein Backup, keine Prüfung)  | Sie stammt aus der Einrichtung und trägt den Caddy-Block nach; sie warnt in ihrem Kopf        |
| D11| **Cover/Uploads sind nicht im Backup**                                                                                   | Nach einem Restore müssen Cover neu geholt werden                | Bewusste Entscheidung: reproduzierbar aus ISBN und Quelle (A17)                              |
| D12| **Ratschen sind lexikalisch**                                                                                            | Eine Umformulierung kann sie blind machen                        | Die Blindheit ist **aufgeschrieben** (Landkarte in [sweeps.md](../sweeps.md)) statt behauptet weg |

---

## 11.3 Externe Abhängigkeiten als Risiko

| Abhängigkeit                       | Risiko                                                                                                  | Abfederung                                                                                      |
| ---------------------------------- | ------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| **Schul-Mailserver (IMAP)**        | Ist er weg, kann sich **niemand** anmelden — es gibt keinen zweiten Anmeldeweg                           | Bewusst in Kauf genommen (A2). Laufende Sitzungen (12 h) bleiben gültig                          |
| **Schul-SMTP**                     | Kein Versand von Mahnungen/Bescheiden                                                                    | Der **Druckweg** funktioniert unabhängig; die Mahnstufe hängt ohnehin am Druck (A18)              |
| **LUSD-Bericht (Datei)**           | Format und Spalten liegen nicht in eigener Hand; eine Schüler-ID kommt in der Praxis nicht              | Kopfzeile wird gesucht statt vorausgesetzt; drei Zuordnungsstufen; Umbenennungspaare; Karenz      |
| **DNB / OpenLibrary / Google Books** | Cover und Metadaten können fehlen oder sich ändern                                                     | Allowlist, URL-Neuaufbau, Retry, manuelles Nachpflegen möglich; Cover sind nie betriebskritisch    |
| **PostgreSQL-Major-Upgrade**       | Ein Volume aus einer älteren Major-Version ist **nicht** startbar; ein zu alter Client verweigert den Server | Umzug nur per Dump/Restore (dokumentiert); der `pg_dump`-Client im Image **muss** mitziehen, sonst schlägt die Restore-Probe sonntags Alarm |
| **CGO/WebP**                       | Der Hauptbuild braucht CGO und `build-base`                                                              | Auf die eine Stufe begrenzt; alle CLI-Werkzeuge bauen mit `CGO_ENABLED=0`                         |
| **`ghcr.io`**                      | Registry-Ausfall beim Deploy                                                                             | `update.sh` baut lokal auf dem Server, die Registry ist nicht der kritische Pfad                   |

---

## 11.4 Das aktuelle Abnahme-Gate

Über allem steht derzeit ein fachliches, nicht ein technisches Risiko: Das
**Anforderungsprotokoll vom 16.09.2026** nennt zwölf Punkte und schließt
damit, dass das Programm nach deren Abstellen für Schulen nutzbar wäre. Architektonisch
wiegen zwei:

1. **Alte Aufdrucke** — Ausweise und Etiketten, die vor dem 17.09.2026 gedruckt wurden
   (siehe R10 und A14).
2. **Mehrjährige Ausleihen** an dasselbe Kind — eine Anforderung, die einer erst am
   15.09.2026 getroffenen Streichung entgegensteht; die Entscheidung ist gestoppt und hängt
   an vier offenen Fragen.

Stand, Reihenfolge und Fragen: [OFFEN.md](../OFFEN.md), Abschnitt 9. Hier steht das nur,
damit niemand die Architekturdokumentation liest und das größte Risiko darin nicht findet.
