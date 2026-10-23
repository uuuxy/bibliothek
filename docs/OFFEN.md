# Offene Arbeit

Stand: 23.10.2026

Oben der Fahrplan: je Schritt eine Zeile mit Kästchen, in der Reihenfolge der Arbeit. Die Nummer
in Klammern führt zu den Einzelheiten. Einen zweiten Ort für Offenes gibt es nicht; wie die Liste
geführt wird, steht am Ende.

---

## Fahrplan

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

### Bei dir

Nur, was kein Test leisten kann.

- [ ] Die Datei anlegen, die Vorschläge der Kollaborateure an deine Freigabe bindet, oder ihr
  Anlegen ausdrücklich freigeben. (7.11)
- [ ] Im Ausweis-Designer eine Vorlage neu anwenden oder das gespeicherte Design hinter die
  gestrichelte Linie schieben und an beiden Kartendruckern eine Karte drucken. (7.10)
- [ ] Die Blätter des Programms einmal auf Papier ansehen. (7.10)
- [ ] Am Testserver nachsehen, ob unter `uploads/fotos` noch Fotos liegen. (7.8)
- [ ] Am Testserver nachsehen, gegen welchen Mailserver die Anmeldung prüft (`IMAP_HOST` in der
  `.env`) und ob dort echte Daten von Schülern liegen. (7.12)
- [ ] Im Regal nachsehen, ob das Buch 2424 („Mathematik heute", Band 6) da ist; es ist seit dem
  17.09.2026 als Verlust ausgebucht. Liegt es dort: in der Buchakte den Status des Exemplars
  zurückstellen.
- [ ] Das Blatt mit den zwei Schlüsseln ausfüllen. (9.9, R7)

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
   „Natur und Technik - Biologie 7/10" (ISBN 9783060158423, nur das Exemplar mit dem Etikett
   `LMF-9783060158423-001`) nebeneinander; die ISBN-Suche findet nur den zweiten. Die 4.065
   LMF-Exemplare kamen dort in einem einzigen Lauf an 130 Titel. Zeilen, deren Zustand
   „Sammelbestand, kein Einzeletikett" nennt (dort bei „Red line"), legt der Bestands-Import als
   etikettiert an; solche Bücher fehlen dann unter „Fehlende Etiketten". Klären, ob sie über den
   Listenimport kommen, der sie ohne Etikett anlegt.
9. Etiketten mit `B-`-Nummern aus der Zeit am Testserver: Die neue Datenbank vergibt ab `B-10001`
   neu. Klebt ein solches Etikett an einem Buch, das am Schulserver nicht mit dieser Nummer
   hereinkommt, bekäme ein anderes Buch dieselbe Nummer. Am Testserver zählen:
   `SELECT count(*), max(substr(barcode_id, 3)::bigint) FROM buecher_exemplare WHERE barcode_id ~ '^B-[0-9]{1,15}$' AND etikett_gedruckt;`
   Gibt es welche, kommen diese Bücher vor jeder neuen Nummer mit ihrer Nummer herein, oder
   `barcode_seq` beginnt über der höchsten.

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

**Ausweise am Kartendrucker.** Der Drucker druckt; was näher als 3 mm am Rand stand, fehlte
unten. Im Druck-Center, Reiter „Schülerausweise", unter „Vorlage wählen …" eine Vorlage oder
„Standardwerte wiederherstellen" nehmen (beide stehen bündig an der gestrichelten Linie) oder
Text, Bilder und Strichcode des gespeicherten Designs selbst dahinter schieben (Farbflächen
dürfen bis an den Rand), dann in der Leserdatei zwei Leser ankreuzen, „Ausweise drucken", an
jedem der zwei Drucker. Erwartet: Nichts fehlt am Rand, die Nummer unter dem Strichcode steht
ganz auf der Karte. Die Lage messen
`frontend/src/lib/designer/kartenRand.test.js` (Standard-Design und Vorlagen) und
`frontend/e2e/ausweis-druckseite.spec.js` (Druckansicht), den Drucker nicht.

**Blätter auf Papier.** Die Tests messen Text und Lage auf der PDF-Seite (`pdf/orte_test.go`),
Rand und Maßstab des Druckers nicht. Je einmal drucken: einen Mahnbrief für einen Schüler mit
mindestens neun Büchern über der Frist (Anschrift im Fenster des Kuverts, Folgeseite mit
„Fortsetzung: …"), die Mahnliste an die Klassenleitung (Versand an eine eigene Adresse), die
Ersatzforderung aus der Leserakte, das Anschreiben im Anhang der Bestellmail und die
DSGVO-Auskunft.

### 7.11 Freigabe für Vorschläge der Kollaborateure

Entschieden am 23.10.2026: Ein Pull Request nach `main` braucht die Freigabe des Eigentümers des
Repositorys. Dazu nennt `.github/CODEOWNERS` den Eigentümer für alle Dateien, und das Regelwerk
für `main` verlangt eine Freigabe, die des Code-Eigentümers, aufgehoben bei neuem Push. Die Datei
wird von Hand angelegt, oder ihr Anlegen wird ausdrücklich freigegeben; danach das Regelwerk
umstellen und [PFLEGEKONZEPT.md](PFLEGEKONZEPT.md) Abschnitt 2 nachziehen.

Heute verlangt das Regelwerk für `main` einen Pull Request, aber keine Freigabe
(`required_approving_review_count: 0`) und keinen bestandenen Lauf der CI; vier Kollaborateure
haben Schreibrecht. Wer eines dieser Konten übernimmt, bringt Code ohne Freigabe nach `main`, und
der Testserver übernimmt ihn beim nächsten Update. Mit der Freigabe zugleich die Läufe der CI als
Pflicht für Pull Requests eintragen.

### 7.12 Testserver von außen

Der Testserver ist ganz aus dem Internet erreichbar, mit Anmeldung und Schnittstelle
(`update_caddy.sh`; `/health` und die Anmeldeseite antworten von außen). Prüft die Anmeldung dort
gegen den Mailserver der Schule, kann jeder aus dem Netz Passwörter der Lehrkräfte durchprobieren,
gebremst je Konto und Adresse (5) und je Adresse (50 in 15 Minuten); mit vielen Adressen ist das
keine Grenze. Am Server nachsehen: `IMAP_HOST` in der `.env`, und ob dort echte Daten von Schülern
liegen. Ist eins davon so, bekommt der Testserver einen Eingang wie in 4.23 oder eine Sperre nach
Adresse im Caddy.

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
