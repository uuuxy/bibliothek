# Offene Arbeit

Stand: 21.10.2026

**Der Fahrplan** steht oben: je Schritt eine Zeile mit Kästchen, in der Reihenfolge der Arbeit.
Die Nummer in Klammern führt zu den Einzelheiten weiter unten. Einen zweiten Ort für Offenes
gibt es nicht; wie die Liste geführt wird, steht am Ende.

---

## Fahrplan

### Jetzt: Fehler und Lücken

Diese Punkte standen als bekannte Grenzen in [ARCHITEKTUR.md](ARCHITEKTUR.md) 11.5; sie lassen
sich lösen. Die zwei der Kategorie A kommen zuerst.

- [x] (A) LMF-Planer: Das Speichern überschreibt still die Änderung eines anderen Platzes. (5.82)
- [x] (A) Theke: Scans, die auf die Antwort des Servers warten, gehen beim Neuladen ohne Meldung
  verloren. (5.83)
- [x] Fünf Fehler an Türen und Abfragen. (5.84)
- [x] Ein Gate hält die Prüfung der UUID-Pfadparameter an jeder Route. (5.85)
- [ ] Theke ohne Netz: Stufe 1 und 3 des Nachweises als Browser-Test. (2.3)
- [ ] Material 3 an fünf Stellen, an denen die Oberfläche davon abweicht. (5.86)
- [ ] Kleinigkeiten an der Oberfläche. (5.87)
- [ ] Listen ordnen Text nach deutscher Regel, nicht nach Codepunkt. (5.88)
- [ ] Aufräumen im Code. (5.89)

### Etappe 1: vor dem Echtstart

Der Echtbetrieb beginnt am Schulserver mit einer leeren Datenbank und der Littera-Übernahme
(entschieden am 28.09.2026). R7, R8 und R9 sind die Risiken aus [ARCHITEKTUR.md](ARCHITEKTUR.md)
11.1, die ein Schritt verkleinert.

- [ ] Generalprobe der Übernahme mit der Sicherung von 2026, sobald sie sich öffnen lässt.
  (7.2, 4.20)
- [ ] `update.sh` für den Schulserver: nur Releases, Images frisch. (5.31)
- [ ] Eingang des Servers: Von außen ist nur die Seite der Lieferanten erreichbar. (4.23)
- [ ] Am Schulserver einrichten: Sicherung außer Haus (7.3, R8), Uptime-Signal (7.5, R9),
  Eigentumsvermerk der Etiketten (4.24), Frist bis zum Sperrbildschirm (9.9). Danach nachsehen:
  Admin-Konten, Verbindung zur DNB (7.8).
- [ ] Abnahmen mit echten Daten. (7.7)
- [ ] Wiederherstellung an einem fremden Ziel proben, allein mit dem Pflegekonzept. (7.4, R7)

### Bei dir

Nur, was kein Test leisten kann; warum, steht jeweils in den Einzelheiten.

- [ ] Ausweise aus der Leserdatei am Kartendrucker drucken. (7.10)
- [ ] Die Blätter des Programms einmal auf Papier ansehen. (7.10)
- [ ] Am Testserver nachsehen, ob unter `uploads/fotos` noch Fotos liegen. (7.8)
- [ ] Im Regal nachsehen, ob das Buch 2424 („Mathematik heute", Band 6) da ist; es ist seit dem
  17.09.2026 als Verlust ausgebucht. Liegt es dort: in der Buchakte den Status des Exemplars
  zurückstellen.
- [ ] PR-Pflicht im Regelwerk für `main` entfernen. (7.6)
- [ ] Das Blatt mit den zwei Schlüsseln ausfüllen. (9.9, R7)

### Termin

- [ ] Ab dem 28. Oktober 2026: Node 26, nach der Regel „immer die aktive LTS"
  ([PFLEGEKONZEPT.md](PFLEGEKONZEPT.md), Abschnitt 4).

---

## Einzelheiten

Beleg und nächster Schritt je Punkt. Die Nummern bleiben fest und werden nicht neu vergeben;
Kommentare im Code nennen sie als Herkunft.

### 2.3 Theke ohne Netz: der Nachweis als Browser-Test

Stufe 2, die Nachbuch-Tür (`POST /api/action/nachbuchen`), halten Tests an der Datenbank. Für
Stufe 1 und 3 gibt es keinen Browser-Test; ohne Netz prüft `frontend/e2e` nur das Band
(`band-browser-meldet-offline.spec.js`). Zu bauen, mit `context.setOffline` und angehaltener
Uhr. Wie sich die Theke ohne Verbindung verhält, steht in [FACHKONZEPT.md](FACHKONZEPT.md) 18.4.

- Stufe 1: Antwort des Servers um 20 s verzögert, Schüler laden, Buch, Escape → IndexedDB trägt
  den Schüler. Ohne Netz „zurückgeben" im Profil → Rückgabe. Lehrkraft laden, Buch scannen →
  Ausleihe auf die Lehrkraft. IndexedDB blockiert → „NICHT gespeichert". Zwei Sicherungen
  einspielen → ein Stapel. Zwei parallele Anfragen mit einem Schlüssel gegen `/api/action` →
  eine Ausleihe.
- Stufe 3: Netz aus, Bücher aller Formen und zwei Ausweise scannen → ein Band, kein Vollbild;
  20 Minuten ohne Sperre, Netz an → Sperre; anmelden → nachgebucht, die Meldungen stehen an
  einem zweiten Browser; die Sicherung im Format 1 eines anderen Rechners einspielen.

### 4.20 Littera-Schlagworte: die Verweise

Die Übernahme aus der Sicherung zählt die Verweise zwischen Schlagworten
(`Verweise_Schlagworte`, `Verweis_Zu_Schlagworte`), übernimmt sie aber nicht
(`internal/littera/schlagworte.go`); in der Sicherung von 2010 sind beide leer. Nennt die
Generalprobe mit der Sicherung von 2026 welche, an diesen Daten die Form ablesen und die
Übernahme bauen.

### 4.23 Erreichbarkeit von außen

**Entschieden am 28.09.2026:** Von außen ist nur die Seite für die Lieferanten erreichbar, alles
andere nur aus dem Schulnetz. Lehrkräfte erreichen das Programm vorerst nicht von zu Hause, und
der Katalog ohne Anmeldung ist nur in der Schule durchsuchbar. Caddy holt das Zertifikat bei
Let's Encrypt wie am Testserver; ohne HTTPS ginge die Anmeldung nicht, weil der Browser das
Sitzungs-Cookie nur über HTTPS mitschickt (`ermittleCookieSecure` in `main.go`).

**Zu bauen:** Der Eingang (Caddy) lässt von außen nur den Pfad `/bestellung/<token>` mit den
Dateien der Oberfläche, `/api/public/bestellung/…` und für das Uptime-Signal (7.5) `/health`
durch; alles andere bleibt zu. Mehr braucht die Seite nicht: Das CSRF-Cookie setzt der erste
Abruf (`erneuereCSRFCookie` in `api/csrf.go`), und `/api/auth/me` darf beim Laden scheitern. In
den Einstellungen die öffentliche Adresse setzen, danach [ARCHITEKTUR.md](ARCHITEKTUR.md) 7 und 9
nachziehen.

**Vor dem Bau muss feststehen:** der Name des Servers im Internet (die Geräte der Schule
erreichen ihn unter demselben Namen, denn das Zertifikat gilt für den Namen), die Freigabe von
Port 443 und der Netzabschnitt, in dem der Server steht, und die Absenderadresse, mit der
Anfragen aus dem Schulnetz ankommen; an ihr erkennt der Eingang das Schulnetz.

Nach außen braucht der Server auch im Schulnetz den Mailserver der Schule (die Anmeldung,
`auth/imap.go`), DNB, Google Books und OpenLibrary (`pkg/coverquelle`) und für Updates GitHub,
Docker Hub und die Paketquellen.

### 4.24 Eigentumsvermerk auf neuen Etiketten

Neue Etiketten der Schülerbücherei tragen den Wortlaut der alten Littera-Etiketten (entschieden
am 28.09.2026). Er wird beim Einrichten unter Einstellungen → Schule eingetragen; ohne Eintrag
trägt ein neues Etikett keinen Vermerk. Wie die Übernahme die Vermerke an den Exemplaren
zuordnet, steht in [littera_schema_befund.md](littera_schema_befund.md).

### 5.31 `update.sh` für den Schulserver: nur Releases, Images frisch

Entschieden am 28.09.2026, nicht gebaut:

- **Nur Releases am Schulserver;** der Testserver folgt `main`. Heute holt `update.sh` mit
  `git pull` den neuesten Stand des Zweigs und fragt nicht, ob dessen Prüfläufe grün sind. Ein
  Release entsteht nur mit grünen Pflicht-Prüfungen (`scripts/tag-gate.sh`).
- **Das Datenbank-Image bei jedem Update holen.** Heute bleibt der Container auf der
  Nebenversion von `postgres:18-alpine`, die beim Anlegen vorlag. Eine Nebenversion braucht
  weder Sicherung noch Neuaufbau, und die Datenbank sortiert ohne Sprachregeln
  (`datlocprovider` = c); ein neues Alpine im Image ändert die Reihenfolge der Indizes also
  nicht.
- **`--pull` für die Basis-Images des Backends.** Die Schicht mit `apk upgrade` baut schon jedes
  Update neu, weil `update.sh` danach den Build-Cache leert (Schritt 7).

Nicht entschieden: Alte Sicherungen löschen nur die Läufe von `update.sh` und
`scripts/backup.sh`. Kommt kein Update mehr, bleibt die letzte Vorab-Sicherung liegen, ein
Klartext-Rest nach einem misslungenen Update bis zum nächsten Lauf; die Meldungen der Skripte
sagen das.

### 5.86 Material 3 an fünf Stellen

- **Trefferfläche der Symbolknöpfe:** 32 × 32 px ohne größere Trefferfläche (`.icon-btn`). Das
  Gate `frontend/e2e/icon-trefferflaechen.spec.js` prüft 32 px, CLAUDE.md nennt 48. In der
  Exemplarkarte der Buchakte sind die Knöpfe 14 px groß und fehlen im Gate; ein 32-px-Knopf
  bricht dort die Kopfzeile um. Vier Symbolknöpfe erklären sich mit `title` statt `data-tip`.
- **„Abbrechen" ist gesperrt, solange die Anfrage läuft,** in elf Dialogen und an sieben
  weiteren Stellen (etwa `LeserLoeschenDialog.svelte`). Material 3, Dialogs: „Dismissive actions
  are never disabled." Voraussetzung je Stelle: Erfolg und Ablehnung werden außerhalb des
  Dialogs gemeldet.
- **Die eingeklappte Navigation** zeigt bis zu 18 Symbole ohne Namen. Material 3, Navigation
  rail: „All navigation items require a one word label text".
- **Die Breite der Textfelder** („Text fields shouldn't span the full width of a large
  screen") gilt nur in „Buch bearbeiten" (bis 704 px, `BuchFormular.svelte`); der Wert gehört an
  eine Stelle, die jede Maske nimmt.
- **Vorschläge an Textfeldern** zeichnet der Browser (`datalist`, sechs Stellen), nicht nach
  Material 3.

### 5.87 Kleinigkeiten an der Oberfläche

- **Reste von „Schülerdatei"** nach der Umbenennung in „Leserdatei": das Recht „Schülerdatei
  anzeigen" (`rechteKategorien.js`), `Schuelerdatei.svelte`, `KlassenDruckEinstieg.svelte`,
  „Schülerakte" in drei Erklärtexten und „Gelöschte Schüler (Papierkorb)" (`Papierkorb.svelte`).
  Die Browser-Tests ziehen mit.
- **Die Akte eines Kollegen ohne Ausweisnummer** verweist auf „Benutzer & Rechte", wo er ohne
  Konto keinen Eintrag hat (`LeserakteAktionen.svelte`).
- **Das Druck-Center hängt im Menü an `view_students`** (`menue.js`), seine Buch-Etiketten
  brauchen nur `view_books` und `edit_books`; ebenso verlangt `GET /api/klassen`
  `view_students`, gewählt wird die Klasse auch mit `edit_books` und `manage_settings`. Wer die
  Rechte anders verteilt als ab Werk, erreicht die Etiketten nicht oder sieht „Klassen nicht
  geladen".
- **„Buch bearbeiten" speichert nur per Klick:** Die Eingabetaste löst nichts aus, und in der
  Tab-Reihenfolge steht „Speichern" vor den Feldern.
- **Die Inventur meldet einen Scan nur auf dem Bildschirm,** ohne Ton wie an der Theke.
- **Neun Stellen bauen ihr Cover selbst** statt über `ui/BuchCover` (Liste in
  `frontend/src/lib/frontend-hygiene-cover.test.js`).
- **Wörter:** Im Mahnwesen ist „Neu laden" ein Symbolknopf ohne Wort
  (`MahnwesenAktionen.svelte`), sonst heißt der Knopf „Aktualisieren". In der Leserakte steht
  unter dem Reiter „Stammdaten & Adresse" dieselbe Überschrift noch einmal, und im Reiter
  „Gebühren & Schäden" heißt die Liste „Forderungen".

### 5.88 Listen ordnen nach deutscher Regel

Die Datenbank ordnet Text nach Codepunkt („Mathe 10 < Mathe 5 < apfel < Ägypten"). Die
Titelliste ordnet deshalb in Go; 30 Abfragen ordnen in der Datenbank nach Titel oder Nachname,
und ob die Oberfläche ihre Antwort neu ordnet, ist je Liste nicht durchgesehen (die Mahnliste
tut es). Nächster Schritt: eine Regel für alle Listen, an einer Stelle.

### 5.89 Aufräumen im Code

- `github.com/jung-kurt/gofpdf` ist archiviert und steckt in 19 Dateien außerhalb der Tests;
  gepflegt wird der Ableger `github.com/phpdave11/gofpdf`, den maroto mitbringt und
  `pdf/lmfplan_satz.go` schon nutzt.
- Zwei Normalformen für Namen: `suchnorm` (`repository/suchnorm.go`) und `normierterName`
  (`internal/lusd/paarung.go`).
- Drei Skripte löschen Titel und lassen ihre Werke zurück (`scripts/e2e_altlasten.sql`,
  `scripts/entferne_demo_daten.sql`, `scripts/seed_demo.sql`).
- Ungelesene Felder: `auth.Claims.Ausweisnummer` (JSON `barcode_id`) wird nur beim Erneuern des
  Tokens weitergereicht, `abgaenger_jahr` in der Antwort der Aktivliste hat keinen Leser, und
  `ui/Menue` kann einen Kopf und Gruppen-Überschriften, die kein Aufrufer nutzt.
- Die Grenze `format.Hoehe >= 30` steht zweimal (`pdf/etikett_buch.go`,
  `pdf/etikett_schueler.go`).
- `TestEtikettenkette_ZaehlerFolgtDenFilternDerListe` schickt `?bis=` in der Zone des
  Testprozesses; westlich von UTC ist das der Vortag. Auf `schulzeit.Zone()` umstellen.

### 7.2 Frisches Littera-Backup

`littera_sav.mdb` ist ein Stand von 2010: Ohne die Sicherung von 2026 fehlen die offenen
Ausleihen und alle Ausweisnummern und Exemplare nach 2010, rund 1.350 Titel haben keine
Signatur. Wie die Übernahme läuft, steht in [SCRIPTS.md](SCRIPTS.md) (Abschnitte 0, 1 und 1b),
was Littera speichert, in [littera_schema_befund.md](littera_schema_befund.md). Die teuerste
offene Position vor dem Echtstart.

**Die Sicherung öffnen.** Die zwei Sicherungen vom 01. und 02.09.2026 sind verschlüsselte `.7z`
mit je einer `.bak`. Nach dem Littera-Handbuch („Datensicherung mit SQL Server-Datenbank") ist
das die Sicherung der SQL-Server-Fassung, keine Access-Datei; `mdb-export` liest sie nicht, und
zum Zurückspielen braucht es „das Kennwort welches bei der Installation des Servers erfasst
wurde". Mit Kennwort: die `.bak` lokal in einen SQL Server einspielen, die Tabellen aus
SCRIPTS.md Abschnitt 1 als CSV ausgeben und Spaltennamen und Datumsformate gegen den Importer
prüfen, der bisher nur `mdb-export` kennt. Ohne Kennwort blieben die Auswertungen von Littera
(Leserliste, Medienliste, verliehene Medien); ob sie die nötigen Nummern tragen, ist nicht
geprüft, und sie hießen einen neuen Importer.

**Was die Generalprobe beantworten muss** (lokal und nur lesend, über das CSV-Verzeichnis,
SCRIPTS.md Abschnitt 1b; mit der Sicherung von 2010 bestanden):

1. Ist `FremdLeserNummer` gefüllt? Sie trägt die Nummern, die die Ausweise beim Scannen liefern.
   Ist sie leer, muss jeder Ausweis neu gedruckt werden; der Lauf trägt dann die Lesernummer als
   Ausweisnummer ein.
2. Die offenen Ausleihen, die Ausweisnummern nach 2010 und das Geburtsdatum, die Brücke zum
   späteren LUSD-Abgleich.
3. Die Etiketten: `LITTERA_CSV_DIR=… go test ./internal/littera/` vergleicht den EAN-13 jedes
   Etiketts mit der Rechnung (`pruefeEtiketten`).
4. Lesergruppen ohne Zuordnung halten den Lauf an (2010: „Undefinierte Untergruppe"). Die
   Bücherei ordnet diese Personen in Littera zu, bevor die Sicherung für den Umstieg gezogen
   wird.
5. Standorte: Die Zahl der Vermerke liegt in der Größe der Titelliste vom Juni 2026 (678
   Einträge); bei null hat `Personen_Zuordnung.Flags` in der SQL-Server-Fassung eine andere
   Form. Verfasser ohne Komma und Leser aus einem Zeitschriften-Rundlauf mit `-kein-standort`
   ausnehmen. Entscheiden, ob „Buchbestand Bibliothek" und „Bibliothek" als Standort mitkommen.
6. Sperren und Salden (`Leser.GesperrtAm`, `Leser.Saldo`) übernimmt der Lauf nicht; zählen,
   dann entscheiden, ob Nachtragen von Hand genügt.
7. Die Verweise der Schlagworte (4.20), und ob die Theke die kurze Nummer der Littera-Etiketten
   annehmen muss ([ARCHITEKTUR.md](ARCHITEKTUR.md) 11.5).

**Am Umstiegstag:** Littera-Personen und -Ausleihen im selben Lauf übernehmen, bevor ein echter
LUSD-Import läuft, sonst legt der Lauf Schüler aus der LUSD ein zweites Mal an (SCRIPTS.md
Abschnitt 0). Die Übernahme läuft auf einer neu angelegten Datenbank (entschieden am
28.09.2026). Vorher das Löschkonzept gegenüber Littera
([datenschutz/nachweis.md](datenschutz/nachweis.md), Abschnitt 10, B7).

### 7.3 Sicherung außer Haus

`S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY` und `S3_BUCKET` sind leer; alle Sicherungen
liegen auf einer Platte. Einzutragen ist ein Speicher in der EU; die Regeln dafür (je Server ein
Bucket, keine Versionierung) stehen in [DEPLOYMENT.md](DEPLOYMENT.md), Abschnitt 1. Ein zweiter
Rechner per Kopierbefehl oder eine getauschte Platte leisten dasselbe außerhalb des Programms;
das ist eine Betriebsentscheidung.

### 7.4 Wiederherstellung an einem fremden Ziel

Die Wochenprobe im Container ersetzt sie nicht. Anleitung:
[resilience_and_recovery.md](resilience_and_recovery.md), Abschnitt 2e; dabei die Befehle aus
2a erproben, die bisher nur am Text geprüft sind. Sie setzen `pg_dump`, `dropdb`, `createdb`,
`psql` und ein gebautes `restore-backup` an dem Rechner voraus, an dem sie laufen; die Form über
die Container ist in 2f erprobt. Machen soll die Probe die Vertretung allein mit dem
Pflegekonzept (9.9); sie wartet auf den Schulserver und die benannte Vertretung.

### 7.5 Externes Uptime-Signal

Fällt der Server ganz aus, meldet es niemand. Ein externer Monitor ruft alle 5 Minuten `/health`
ab ([DEPLOYMENT.md](DEPLOYMENT.md), Abschnitt 7.0). Am Schulserver geht das erst, wenn der
Eingang aus 4.23 `/health` von außen durchlässt.

### 7.6 Regelwerk für `main`

Die PR-Pflicht entfernen (Solo-Entscheidung vom 30.07.2026), „Block force pushes" und „Restrict
deletions" anlassen. Am 21.10.2026 trägt das Regelwerk noch `pull_request`; Pushes gehen über
den Admin-Bypass.

### 7.7 Abnahmen

Ablauf in [abnahme_checkliste.md](abnahme_checkliste.md), vorher ein Backup. Die Abnahme prüft
echte Daten, die gehören nur auf den Schulserver, und der LUSD-Import kommt erst nach der
Littera-Übernahme (7.2).

- Flows 1–3 mit dem Sekretariat: LUSD-Import, Versetzung (vor dem Schuljahreswechsel),
  Klassensatz erledigen. Dabei um Geburtsdatum und Eintrittsdatum im LUSD-Bericht bitten.
- Flow 4 (Altbestand-Etiketten) entfällt mit dem Neuaufbau: Die Littera-Übernahme setzt den
  Etikett-Vermerk selbst (`internal/littera/schreiber_bestand.go`).
- Flow 5: Selbstanmeldung einer Lehrkraft.
- Danach: Ergebnis hier eintragen; bei Auffälligkeiten des Parsers die echte LUSD-Datei
  anonymisiert als Testfixture sichern.

### 7.8 Am Server nachsehen (lesend)

- Liegen im Volume noch Fotos aus der Zeit vor dem 13.06.2026? Sie wären unverschlüsselt; das
  betrifft nur ein Volume aus dieser Zeit, also den Testserver.
  `docker compose exec backend ls -la uploads/fotos`; liegen dort Dateien:
  `docker compose exec backend sh -c 'rm -f uploads/fotos/*.jpg'`.
- Sind die Admin-Konten deaktiviert?
- Erreicht der Server die DNB? Ohne sie bringt die ISBN-Abfrage keine Angaben und der
  Cover-Abgleich kein Cover; gespeichert wird trotzdem:
  `docker exec bibliothek-backend sh -c 'wget -q -T 8 -O /dev/null "https://services.dnb.de/sru/dnb?version=1.1&operation=explain" && echo erreichbar || echo nicht erreichbar'`

### 7.10 Was nur ein Mensch am Gerät sieht

Was hier sonst als Probe am Testserver stand, halten Tests: Browser-Tests tippen wie ein
Scanner, und dass der Handscanner der Schule schnell genug tippt, ist am 07.10.2026 an der Theke
belegt.

**Ausweise am Kartendrucker.** In der Leserdatei zwei Leser ankreuzen, „Ausweise drucken".
Erwartet: Jede Karte trägt nur den Ausweis, oben steht keine Reiterzeile, und die Nummer unter
dem Strichcode steht auf der Karte. Dasselbe mit „Testdruck Vorderseite" im Druck-Center,
Reiter „Schülerausweise". Die Druckansicht des Browsers misst
`frontend/e2e/ausweis-druckseite.spec.js`; was der Kartendrucker daraus macht, zeigt nur er.

**Blätter auf Papier.** Die Tests messen Text und Lage auf der PDF-Seite (`pdf/orte_test.go` und
die Tests je Blatt); was der Drucker an Rand und Maßstab daraus macht, zeigt nur das Papier. Je
einmal drucken: einen Mahnbrief für einen Schüler mit mindestens neun Büchern über der Frist
(Anschrift im Fenster des Kuverts, Folgeseite mit „Fortsetzung: …"), die Mahnliste an die
Klassenleitung (Versand an eine eigene Adresse), die Ersatzforderung aus der Leserakte, das
Anschreiben im Anhang der Bestellmail und die DSGVO-Auskunft.

### 9.9 Pflegekonzept und Datenschutz-Nachweis

Beide Entwürfe stehen ([PFLEGEKONZEPT.md](PFLEGEKONZEPT.md),
[datenschutz/nachweis.md](datenschutz/nachweis.md)); den Nachweis beschließt die Schule.

- **Das Blatt bei der Schule** (PFLEGEKONZEPT.md 7.3, Vorlage im Anhang) ausfüllen. Vorschlag:
  die zwei Schlüssel in einem Passwortmanager und als Papier im verschlossenen Umschlag im
  Tresor der Schule, nie per E-Mail.
- **Die Frist bis zum Sperrbildschirm:** Gewünscht sind an der Schule 8 Stunden ohne Bedienung
  (01.10.2026). Am Schulserver unter Einstellungen → Datenschutz & Sitzung 480 Minuten
  eintragen; das Feld nimmt 0 bis 1440, die Vorgabe bleibt 15. Der Nachweis nennt dann die Zahl
  der Schule (Abschnitt 5, „Zugang"; Beschluss B4). Für den unbeaufsichtigten Platz bleibt das
  Leeren der Theke nach 5 Minuten.
- **Die Probe:** Die Vertretung macht die Wiederherstellung an einem fremden Ziel (7.4) allein
  mit dem Pflegekonzept.

---

## So wird die Liste geführt

**Kästchen stehen nur im Fahrplan.** Abgehakt wird im selben Commit, der den Schritt erledigt.
Abgehakte Zeilen fallen bei der nächsten Durchsicht der Liste weg, eine fertige Etappe als
Ganzes. In den Einzelheiten wird Erledigtes gelöscht.

**Hier steht nur Arbeit, die einer von uns beiden tun kann:** am Programm, an den Servern, auf
GitHub. Fragen an Dritte und das Warten auf ihre Antworten stehen hier nicht, auch nicht das
Warten auf eine neue Fassung einer Bibliothek: Die schlägt Dependabot vor, und ein Prüflauf, der
an ihr scheitert, wird rot.

**Auf die Liste „Bei dir" kommt nur, was kein Test leisten kann,** mit dem Satz, warum. Was ein
Test prüfen kann, wird als Test gebaut.

Vor jedem Fund steht dieselbe Frage — nicht „ist das hässlich?", sondern **„kann das still
jemandem schaden?"**

|       | Kategorie                                                                                                                                                                                            | Umgang                                                               |
| ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| **A** | Kann **stillschweigend** ein falsches Ergebnis für einen echten Menschen erzeugen: doppelte Mahnung an Eltern, Daten beim falschen Empfänger, verlorene Eingabe, ein Gate, das nicht rot werden kann | **Sofort**, eigener Commit, eigener Test, eigene Deploy-Entscheidung |
| **B** | Fehler, der sich **laut** meldet, oder Unordnung ohne Wirkung nach außen: totes Codestück, wackeliger Test, Doppelung                                                                                | Hier notieren, gebündelt abarbeiten                                  |
| **C** | „Wenn ich schon mal hier bin" — Umbenennungen, Stilfragen, Refactorings ohne Anlass                                                                                                                  | Nur mit Anlass und Zeit; steht in ARCHITEKTUR.md 11.5, nicht hier    |

1. **Ein Fund = ein Commit.** Was beim Reparieren zusätzlich auffällt, kommt hierher, nicht in
   denselben Commit.
2. **Kategorie A wird belegt, nicht behauptet:** ein Test, der am alten Code rot wird. Bis ein
   Fund nachgestellt ist, heißt er „Verdacht".
3. **Neues kommt nur hierher** — Funde, offene Fragen, Betriebspunkte: als Zeile im Fahrplan,
   wenn es ein Schritt ist, und mit Beleg unter einer Nummer in den Einzelheiten. Kein Issue,
   kein anderes Dokument. Eine Frage steht hier, bevor die Antwort kommt. In
   [ARCHITEKTUR.md](ARCHITEKTUR.md) 11.5 steht nur, was eine Entscheidung trägt oder erst mit
   einem Anlass zum Fehler wird; ein Fehler, der sich lösen lässt, steht hier.
4. **Erledigt heißt:** im Fahrplan abhaken, in den Einzelheiten löschen — Datum und Begründung
   stehen in der Commit-Nachricht, ein Archiv gibt es nicht. Eine Antwort bekommt „Entschieden
   am …" und fällt weg, sobald sie umgesetzt ist. Die Nummer eines gelöschten Punkts wird nicht
   wieder vergeben — Kommentare im Code nennen sie als Herkunft.
5. **Die Reihenfolge** im Fahrplan wird bei jeder Änderung mitgepflegt.
