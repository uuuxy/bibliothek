# Offene Arbeit

Stand: 22.10.2026

Oben der Fahrplan: je Schritt eine Zeile mit Kästchen, in der Reihenfolge der Arbeit. Die Nummer
in Klammern führt zu den Einzelheiten. Einen zweiten Ort für Offenes gibt es nicht; wie die Liste
geführt wird, steht am Ende.

---

## Fahrplan

### Jetzt

- [x] Entscheiden: Breite der Suchleisten auf den Listenseiten. (5.86)
- [x] Drei kleine Funde aus dem Durchgang über den Bestand: die Domain der Schule an einer Stelle,
  Ladefehler der Klassenleitungen, doppelter Scan im Barcode-Editor.
- [x] Entscheiden: Ein Exemplar mit langer LMF-Nummer, das ein neues Etikett braucht, bekommt eine
  kurze `B-`-Nummer, statt den langen Code nachzudrucken.
- [x] Entscheiden: „Neue Barcodes" im Druck-Center entfällt, entschieden am 22.10.2026
  ([FACHKONZEPT.md](FACHKONZEPT.md) §20).
- [ ] Entscheiden: Lehnt der Server einen Etikettendruck mit einer Nummer ohne Exemplar ab, wie
  einen mit einer LMF-Nummer? (5.93)

### Etappe 1: vor dem Echtstart

Der Echtbetrieb beginnt am Schulserver mit einer leeren Datenbank und der Littera-Übernahme
(entschieden am 28.09.2026). R7, R8 und R9 sind Risiken aus [ARCHITEKTUR.md](ARCHITEKTUR.md)
11.1.

- [ ] Generalprobe der Übernahme mit der Sicherung von 2026, sobald sie sich öffnen lässt.
  (7.2, 4.20)
- [ ] Eingang des Servers: Von außen ist nur die Seite der Lieferanten erreichbar. (4.23)
- [ ] Am Schulserver einrichten: Sicherung außer Haus (7.3, R8), Uptime-Signal (7.5, R9),
  Eigentumsvermerk der Etiketten (4.24), Frist bis zum Sperrbildschirm (9.9), Uhr für den
  Klartext-Rest ([DEPLOYMENT.md](DEPLOYMENT.md), Abschnitt 6). Danach nachsehen: Admin-Konten,
  Verbindung zur DNB (7.8).
- [ ] Abnahmen mit echten Daten. (7.7)
- [ ] Wiederherstellung an einem fremden Ziel proben, allein mit dem Pflegekonzept. (7.4, R7)

### Zurückgestellt

- [ ] Vorschläge an sechs Textfeldern als eigene Liste nach Material 3 statt `datalist`. (5.86)

### Bei dir

Nur, was kein Test leisten kann.

- [ ] Ausweise aus der Leserdatei am Kartendrucker drucken. (7.10)
- [ ] Die Blätter des Programms einmal auf Papier ansehen. (7.10)
- [ ] Am Testserver nachsehen, ob unter `uploads/fotos` noch Fotos liegen. (7.8)
- [ ] Im Regal nachsehen, ob das Buch 2424 („Mathematik heute", Band 6) da ist; es ist seit dem
  17.09.2026 als Verlust ausgebucht. Liegt es dort: in der Buchakte den Status des Exemplars
  zurückstellen.
- [ ] PR-Pflicht im Regelwerk für `main` entfernen. (7.6)
- [ ] Das Blatt mit den zwei Schlüsseln ausfüllen. (9.9, R7)
- [x] Am Testserver lesen, an welchem Titel das Exemplar 101634 und die LMF-Exemplare derselben
  ISBN hängen.

### Termin

- [ ] Ab dem 28. Oktober 2026: Node 26, nach der Regel „immer die aktive LTS"
  ([PFLEGEKONZEPT.md](PFLEGEKONZEPT.md), Abschnitt 4).

---

## Einzelheiten

Beleg und nächster Schritt je Punkt. Die Nummern bleiben fest; Kommentare im Code nennen sie als
Herkunft.

### 4.20 Littera-Schlagworte: die Verweise

Die Übernahme zählt die Verweise zwischen Schlagworten (`Verweise_Schlagworte`,
`Verweis_Zu_Schlagworte`), übernimmt sie aber nicht (`internal/littera/schlagworte.go`); in der
Sicherung von 2010 sind beide leer. Hat die Sicherung von 2026 welche: an ihnen die Form ablesen
und die Übernahme bauen.

### 4.23 Erreichbarkeit von außen

Entschieden am 28.09.2026: Von außen ist nur die Seite der Lieferanten erreichbar, alles andere
nur aus dem Schulnetz, also auch nicht für Lehrkräfte von zu Hause.

**Zu bauen:** Caddy lässt von außen nur `/bestellung/<token>` mit den Dateien der Oberfläche,
`/api/public/bestellung/…` und `/health` (7.5) durch. Mehr braucht die Seite nicht: Das
CSRF-Cookie setzt der erste Abruf (`erneuereCSRFCookie` in `api/csrf.go`), und `/api/auth/me`
darf beim Laden scheitern. Das Zertifikat kommt von Let's Encrypt; ohne HTTPS schickt der Browser
das Sitzungs-Cookie nicht (`ermittleCookieSecure` in `main.go`). Danach in den Einstellungen die
öffentliche Adresse setzen und [ARCHITEKTUR.md](ARCHITEKTUR.md) 7 und 9 nachziehen.

**Vorher muss feststehen:** der Name des Servers im Internet (auch die Geräte der Schule nutzen
ihn, das Zertifikat gilt für den Namen), die Freigabe von Port 443, der Netzabschnitt des Servers
und die Absenderadresse, an der der Eingang das Schulnetz erkennt. Ausgehend braucht der Server
den Mailserver der Schule (`auth/imap.go`), DNB, Google Books und OpenLibrary
(`pkg/coverquelle`) und für Updates GitHub, Docker Hub und die Paketquellen.

### 4.24 Eigentumsvermerk auf neuen Etiketten

Neue Etiketten der Schülerbücherei tragen den Wortlaut der alten Littera-Etiketten (entschieden
am 28.09.2026). Er wird beim Einrichten unter Einstellungen → Schule eingetragen; ohne Eintrag
trägt ein neues Etikett keinen Vermerk.

### 5.86 Material 3, wo die Oberfläche noch abweicht

- **Vorschläge an Textfeldern** zeichnet der Browser (`datalist`, sechs Stellen). Entschieden am
  21.10.2026: eine eigene Vorschlagsliste, mit der Tastatur bedienbar wie die Auswahllisten; am
  selben Tag zurückgestellt.

### 5.93 Etikett für eine Nummer ohne Exemplar

`POST /api/print/labels` druckt jede Nummer, die ein Auftrag nennt; `buchetikett.ErgaenzeServerfelder`
lässt eine unbekannte ohne Jahr, Signatur und Topf. Die Oberfläche schickt nur Exemplare aus der
Datenbank. Eine Nummer ohne Exemplar kommt aus einem Browserfenster, das noch „Neue Barcodes"
zeigt, oder von einem Exemplar, das zwischen Laden und Druck gelöscht oder umnummeriert wurde;
`api/druck/etiketten_topf_pg_test.go` druckt eine solche Nummer mit Absicht. Nächster Schritt nach
der Entscheidung: solche Aufträge mit 409 ablehnen und die Nummern nennen wie bei LMF-Nummern
(`buchetikett.NichtDruckbare`), oder es bleibt so.

### 7.2 Frisches Littera-Backup

`littera_sav.mdb` ist ein Stand von 2010: Es fehlen die offenen Ausleihen und alle Ausweisnummern
und Exemplare nach 2010, rund 1.350 Titel haben keine Signatur. Die Übernahme beschreibt
[SCRIPTS.md](SCRIPTS.md) (Abschnitte 0, 1 und 1b), was Littera speichert,
[littera_schema_befund.md](littera_schema_befund.md).

**Die Sicherung öffnen.** Die Sicherungen vom 01. und 02.09.2026 sind verschlüsselte `.7z` mit je
einer `.bak` der SQL-Server-Fassung (Littera-Handbuch, „Datensicherung mit SQL Server-Datenbank");
`mdb-export` liest sie nicht. Zum Zurückspielen braucht es „das Kennwort welches bei der
Installation des Servers erfasst wurde". Mit Kennwort: die `.bak` lokal in einen SQL Server
einspielen, die Tabellen aus SCRIPTS.md Abschnitt 1 als CSV ausgeben, Spaltennamen und
Datumsformate gegen den Importer prüfen. Ohne Kennwort blieben die Auswertungen von Littera; ob
sie die nötigen Nummern tragen, ist nicht geprüft, und sie bräuchten einen neuen Importer.

**Was die Generalprobe beantworten muss** (SCRIPTS.md Abschnitt 1b; mit der Sicherung von 2010
bestanden):

1. Ist `FremdLeserNummer` gefüllt? Sonst muss jeder Ausweis neu gedruckt werden; der Lauf trägt
   dann die Lesernummer als Ausweisnummer ein.
2. Kommen die offenen Ausleihen, die Ausweisnummern nach 2010 und das Geburtsdatum mit, die
   Brücke zum LUSD-Abgleich?
3. Etiketten: `LITTERA_CSV_DIR=… go test ./internal/littera/` vergleicht den EAN-13 jedes
   Etiketts mit der Rechnung (`pruefeEtiketten`).
4. Lesergruppen ohne Zuordnung halten den Lauf an; die Bücherei ordnet sie in Littera zu, bevor
   die Sicherung für den Umstieg gezogen wird.
5. Standorte: Die Zahl der Vermerke liegt in der Größe der Titelliste vom Juni 2026 (678); bei
   null hat `Personen_Zuordnung.Flags` eine andere Form. Verfasser ohne Komma und Leser aus einem
   Zeitschriften-Rundlauf mit `-kein-standort` ausnehmen. Entscheiden, ob „Buchbestand
   Bibliothek" und „Bibliothek" als Standort mitkommen.
6. Sperren und Salden (`Leser.GesperrtAm`, `Leser.Saldo`) übernimmt der Lauf nicht: zählen, dann
   entscheiden, ob Nachtragen von Hand genügt.
7. Die Verweise der Schlagworte (4.20), und ob die Theke die kurze Nummer der Littera-Etiketten
   annehmen muss ([ARCHITEKTUR.md](ARCHITEKTUR.md) 11.5).
8. Lernmittel mit LMF-Etikett: Hängt der Bestands-Import (Kombi-CSV) ihre Exemplare an den Titel
   aus Littera, oder entsteht ein zweiter Titel desselben Buchs? Am Testserver stehen
   „Natur und Technik - Biologie 7 - 10 Berl…" (Liste vom Juni, ohne ISBN, mit 101634) und
   „Natur und Technik - Biologie 7/10" (ISBN 9783060158423, nur `LMF-9783060158423-001`)
   nebeneinander; die ISBN-Suche findet nur den zweiten. Die 4.065 LMF-Exemplare kamen dort in
   einem einzigen Lauf an 130 Titel.

**Am Umstiegstag:** in der Reihenfolge aus SCRIPTS.md Abschnitt 0, auf einer neu angelegten
Datenbank (entschieden am 28.09.2026); vorher das Löschkonzept gegenüber Littera
([datenschutz/nachweis.md](datenschutz/nachweis.md), Abschnitt 10, B7).

### 7.3 Sicherung außer Haus

`S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY` und `S3_BUCKET` sind leer; alle Sicherungen
liegen auf einer Platte. Einzutragen ist ein Speicher in der EU, Regeln in
[DEPLOYMENT.md](DEPLOYMENT.md), Abschnitt 1. Ein zweiter Rechner per Kopierbefehl oder eine
getauschte Platte leisten dasselbe außerhalb des Programms.

### 7.4 Wiederherstellung an einem fremden Ziel

Die Anleitung ([resilience_and_recovery.md](resilience_and_recovery.md), 2a, 2e und 2f) ist
erprobt. Offen ist die Probe durch die Vertretung allein mit dem Pflegekonzept (9.9); die
Wochenprobe im Container ersetzt sie nicht. Sie wartet auf den Schulserver und die benannte
Vertretung.

### 7.5 Externes Uptime-Signal

Fällt der Server ganz aus, meldet es niemand. Ein externer Monitor ruft alle 5 Minuten `/health`
ab ([DEPLOYMENT.md](DEPLOYMENT.md), Abschnitt 7.0); am Schulserver erst, wenn der Eingang aus 4.23
`/health` durchlässt.

### 7.6 Regelwerk für `main`

Die PR-Pflicht entfernen (Solo-Entscheidung vom 30.07.2026), „Block force pushes" und „Restrict
deletions" anlassen. Am 21.10.2026 trägt das Regelwerk noch `pull_request`; Pushes gehen über
den Admin-Bypass.

### 7.7 Abnahmen

Ablauf in [abnahme_checkliste.md](abnahme_checkliste.md). Sie prüft echte Daten, also nur am
Schulserver, und erst nach der Littera-Übernahme (7.2).

- Flows 1–3 mit dem Sekretariat: LUSD-Import, Versetzung (vor dem Schuljahreswechsel),
  Klassensatz erledigen. Dabei um Geburtsdatum und Eintrittsdatum im LUSD-Bericht bitten.
- Flow 4 entfällt: Die Littera-Übernahme setzt den Etikett-Vermerk selbst
  (`internal/littera/schreiber_bestand.go`).
- Flow 5: Selbstanmeldung einer Lehrkraft.

### 7.8 Am Server nachsehen

- Fotos aus der Zeit vor dem 13.06.2026 lägen unverschlüsselt im Volume; das betrifft nur den
  Testserver. `docker compose exec backend ls -la uploads/fotos`; liegen dort Dateien:
  `docker compose exec backend sh -c 'rm -f uploads/fotos/*.jpg'`.
- Sind die Admin-Konten deaktiviert?
- Erreicht der Server die DNB? Ohne sie bringt die ISBN-Abfrage keine Angaben und der
  Cover-Abgleich kein Cover:
  `docker exec bibliothek-backend sh -c 'wget -q -T 8 -O /dev/null "https://services.dnb.de/sru/dnb?version=1.1&operation=explain" && echo erreichbar || echo nicht erreichbar'`

### 7.10 Was nur ein Mensch am Gerät sieht

**Ausweise am Kartendrucker.** In der Leserdatei zwei Leser ankreuzen, „Ausweise drucken";
dasselbe mit „Testdruck Vorderseite" im Druck-Center, Reiter „Schülerausweise". Erwartet: Jede
Karte trägt nur den Ausweis, oben steht keine Reiterzeile, die Nummer unter dem Strichcode steht
auf der Karte. Die Druckansicht misst `frontend/e2e/ausweis-druckseite.spec.js`, den Drucker
nicht.

**Blätter auf Papier.** Die Tests messen Text und Lage auf der PDF-Seite (`pdf/orte_test.go`),
Rand und Maßstab des Druckers nicht. Je einmal drucken: einen Mahnbrief für einen Schüler mit
mindestens neun Büchern über der Frist (Anschrift im Fenster des Kuverts, Folgeseite mit
„Fortsetzung: …"), die Mahnliste an die Klassenleitung (Versand an eine eigene Adresse), die
Ersatzforderung aus der Leserakte, das Anschreiben im Anhang der Bestellmail und die
DSGVO-Auskunft.

### 9.9 Pflegekonzept und Datenschutz-Nachweis

Beide Entwürfe stehen ([PFLEGEKONZEPT.md](PFLEGEKONZEPT.md),
[datenschutz/nachweis.md](datenschutz/nachweis.md)).

- **Das Blatt bei der Schule** (PFLEGEKONZEPT.md 7.3, Vorlage im Anhang): die zwei Schlüssel in
  einem Passwortmanager und als Papier im verschlossenen Umschlag im Tresor der Schule, nie per
  E-Mail.
- **Die Frist bis zum Sperrbildschirm:** Gewünscht sind 8 Stunden ohne Bedienung (01.10.2026).
  Am Schulserver unter Einstellungen → Datenschutz & Sitzung 480 Minuten eintragen (Vorgabe 15);
  der Nachweis nennt dann die Zahl der Schule (Abschnitt 5, „Zugang"; Beschluss B4). Die Theke
  leert sich weiter nach 5 Minuten.
- **Die Probe** der Vertretung: 7.4.

---

## So wird die Liste geführt

1. **Kästchen stehen nur im Fahrplan,** abgehakt im Commit, der den Schritt erledigt; abgehakte
   Zeilen fallen bei der nächsten Durchsicht weg. In den Einzelheiten wird Erledigtes gelöscht.
   Datum und Begründung stehen in der Commit-Nachricht, ein Archiv gibt es nicht. Die Nummer
   eines gelöschten Punkts wird nicht wieder vergeben. Die Reihenfolge im Fahrplan wird bei jeder
   Änderung mitgepflegt.
2. **Hier steht nur Arbeit, die einer von uns beiden tun kann:** am Programm, an den Servern, auf
   GitHub. Keine Fragen an Dritte und kein Warten auf ihre Antworten oder auf eine neue Fassung
   einer Bibliothek (die schlägt Dependabot vor). „Bei dir" nur, was kein Test leisten kann.
3. **Neues kommt nur hierher:** als Zeile im Fahrplan und mit Beleg unter einer Nummer, kein
   Issue, kein anderes Dokument. Eine Frage steht hier, bevor die Antwort kommt; die Antwort
   bekommt „Entschieden am …" und fällt weg, sobald sie umgesetzt ist.
   [ARCHITEKTUR.md](ARCHITEKTUR.md) 11.5 nimmt nur, was eine Entscheidung trägt oder erst mit
   einem Anlass zum Fehler wird; ein Fehler, der sich lösen lässt, steht hier.
4. **Vor jedem Fund die Frage: „Kann das still jemandem schaden?"**
   - **A:** Kann still ein falsches Ergebnis für einen echten Menschen erzeugen (doppelte
     Mahnung, Daten beim falschen Empfänger, verlorene Eingabe, ein Gate, das nicht rot werden
     kann). Sofort, mit eigenem Commit und einem Test, der am alten Code rot wird; bis dahin
     heißt der Fund „Verdacht".
   - **B:** Meldet sich laut oder ist Unordnung ohne Wirkung nach außen. Hier notieren, gebündelt
     abarbeiten.
   - **C:** Umbenennungen, Stilfragen, Umbauten ohne Anlass. Nur mit Anlass; steht in
     ARCHITEKTUR.md 11.5.
5. **Ein Fund, ein Commit.** Was beim Reparieren zusätzlich auffällt, kommt hierher.
