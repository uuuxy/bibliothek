# Offene Arbeit

Stand: 21.09.2026

**Die eine Liste.** Hier steht alles, was noch zu tun, zu prüfen oder zu entscheiden ist — Code,
Betrieb und Schule. Einen zweiten Ort gibt es nicht. Erledigtes wird gelöscht, nicht archiviert:
Die Geschichte steht in den Commit-Nachrichten und in `git log -p docs/OFFEN.md`.

Andere Dokumente erklären (Konzept, Anleitung, der Katalog der Bugklassen in
[sweeps.md](sweeps.md)), führen aber keine eigene Offen-Liste.

---

## Was jetzt dran ist

Dieser Block nennt die Reihenfolge. Alles darunter ist die ausführliche Fassung mit
Begründungen.

**Vorrang hat die Sichtung vom 16.09.2026** (Abschnitt 9): zwölf Punkte, acht davon erledigt.
Offen sind zwei Fragen (9.7) und zwei Bedingungen, die neben der Mängelliste stehen — der
Nachweis der DSGVO-Konformität und ein Hosting- und Pflegekonzept (9.9).

**Was bei dir liegt — der Reihe nach:**

1. **Das Antwortschreiben abschicken.** Es ist fertig (17.09.2026). Daran hängen zwei gestoppte
   Entscheidungen: die Sperre bei offener Forderung (9.3 c) und die Mehrjahresbände (9.6, hält
   4.3 auf). Solange die Antwort aussteht, wird an beiden Stellen nichts gebaut.
2. **Sechs Fragen beantworten** (Abschnitt 4: 4.7 bis 4.9, 4.11, 4.13, 4.14). Ohne sie bleiben
   sechs kleine Bauarbeiten liegen. Jede hat einen Vorschlag danebenstehen.
3. **Drei Zahlen vom Server holen.** Die Befehle stehen fertig in der Liste: Wie viele Leser
   stehen ohne Ausweisnummer da (5.16 C)? Und steht heute ein Kollege in einer Warteschlange,
   in der er nie nachrückt (5.19)? Erst danach werden Nummern nachgetragen — das ändert echte
   Daten. Die Messung aus 4.3 ist gestoppt, bis die Frage nach den Mehrjahresbänden beantwortet
   ist.
4. **Zwei Umbauten freigeben**, die vorbereitet, aber nicht gebaut sind, weil sie die Datenbank
   ändern: die Ausweisnummer schon beim Anlegen eines Kontos (5.16 C) und die eigene Spalte für
   die Karenz-Uhr (4.12). Beide sind entschieden, beide brauchen eine Migration, die Nummern
   bzw. Daten schreibt, die niemand zurücknimmt.
5. **Der Nachweis von Hand für die Theke ohne Netz** (Abschnitt 2): Netz kappen, Bücher aller
   Formen und zwei Ausweise scannen, 20 Minuten warten, Netz zurück, Meldungen ansehen. Dazu
   der Nachweis für den Server.
6. **15 Minuten durch die Leserdatei gehen** (5.16 A): Stimmen die Wörter, fehlt etwas?
7. **Schulbücher in neuer Auflage** (4.18). Die Richtung ist entschieden, gebaut ist nichts. Der
   erste Schritt ist klein (das Feld „Auflage"), der dritte ändert das Schema und braucht eine
   Freigabe.
8. **Liegt bei anderen** (Abschnitt 8): die Anfragen an Schule, Schulamt und Schulträger. Hier
   ist nichts zu tun außer nachzufragen, wenn nichts kommt.
9. **Aus dem Abgleich mit Littera** (4.19, 4.20): **Ferienkalender** — heute mahnen wir das Kind,
   dessen Frist in die Herbstferien fiel. Richtung und Form sind am 18.09.2026 entschieden
   (Ferien als iCal-Datei, Fristen rutschen mit); gebaut ist nichts, und eine Frage steht noch
   offen (Feiertage als zweite Datei oder gerechnet). Dazu die Frage, ob die Schülerbücherei eine
   Themensuche bekommt (4.20).

**Beim nächsten Aufspielen erweitert sich die Datenbank** (Migrationen bis 129). Das passiert
beim Start von allein; Daten gehen nicht verloren, nachgetragen wird nichts.

**Was liegen bleiben darf:** die übrigen B-Punkte in Abschnitt 5, die Beobachtungen in 6 und die
Betriebspunkte in 7. Keiner davon schadet still; sie werden gebündelt erledigt.

---

## So wird die Liste geführt

Vor jedem Fund steht dieselbe Frage — nicht „ist das hässlich?", sondern **„kann das still
jemandem schaden?"**

|       | Kategorie                                                                                                                                                                                            | Umgang                                                               |
| ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| **A** | Kann **stillschweigend** ein falsches Ergebnis für einen echten Menschen erzeugen: doppelte Mahnung an Eltern, Daten beim falschen Empfänger, verlorene Eingabe, ein Gate, das nicht rot werden kann | **Sofort**, eigener Commit, eigener Test, eigene Deploy-Entscheidung |
| **B** | Fehler, der sich **laut** meldet, oder Unordnung ohne Wirkung nach außen: totes Codestück, wackeliger Test, Doppelung                                                                                | Hier notieren, gebündelt abarbeiten                                  |
| **C** | „Wenn ich schon mal hier bin" — Umbenennungen, Stilfragen, Refactorings ohne Anlass                                                                                                                  | Nur mit Anlass und Zeit                                              |

1. **Ein Fund = ein Commit.** Was beim Reparieren zusätzlich auffällt, kommt hierher, nicht in
   denselben Commit.
1. **Kategorie A wird belegt, nicht behauptet:** ein Test, der am alten Code rot wird. Bis ein
   Fund nachgestellt ist, heißt er „Verdacht".
2. **Neues kommt nur hierher** — Funde, offene Fragen, Betriebspunkte. Kein Issue, kein anderes
   Dokument. Eine Frage steht hier, bevor die Antwort kommt.
3. **Erledigt heißt:** hier löschen — Datum und Begründung stehen in der Commit-Nachricht, ein
   Archiv gibt es nicht. Eine Antwort bekommt „Entschieden am …" und fällt weg, sobald
   sie umgesetzt ist.
4. **Die Reihenfolge** wird bei jeder Änderung mitgepflegt.

---

## Reihenfolge

1. **Die sechs offenen Fragen** aus Abschnitt 4 (4.7–4.9, 4.11, 4.13, 4.14) — sie halten sechs
   kleine Bauarbeiten auf und kosten zusammen eine halbe Stunde.
2. **Die zwei Messungen am Server**: Ziel-Jahrgang (4.3) und Leser ohne Ausweisnummer
   (5.16 C). Danach die beiden Migrationen, die daran hängen — und die Karenz-Spalte (4.12).
3. **Abschnitt 2** Offline-Betrieb der Theke: gebaut, samt Meldungsliste und Doku. Offen sind
   nur noch die Nachweise am Stack (2.3) — Stufe 1 und 3 von Hand, Stufe 2 über die Tür.
4. **5.16** Leserdatei: gebaut. Offen ist dein Blick auf den Stand und die Ausweisnummer (E).
5. **5.5–5.9**, **5.12** und die B-Punkte aus **5.15** — kleine B-Commits, gebündelt.
6. Mahnverfahren: Vor dem ersten echten Bescheid **5.2**, dann **4.4** (E6) und
   **5.13** Stufe 3 (5.3).
7. Nach der Antwort zu E5 (**8.3**): **5.4**.
8. **5.10** (Gates und Werkzeuge) und Abschnitt 6 nur mit Anlass — mit EINER Ausnahme, die
   vorgezogen gehört: Tests für die Hülle der Littera-Übernahme (`cmd/littera-import`) und
   `cmd/seed`. Sie laufen einmal gegen echte Daten und haben keinen Test; der Kern
   (`internal/littera`) hat sie. Der Schlüsselwechsel ist seit dem 21.09.2026 abgedeckt.

**Parallel auf der Schulseite:** Abschnitte 7 und 8 — zuerst S3 (7.3), das Littera-Backup (7.2), die
Anfragen E1, E2, E5 (8.1–8.3), B3 und B4 (8.5) und ein Termin für die Abnahmen (7.7). Einen echten LUSD-Import erst nach der
Littera-Übernahme (7.2).

---

## 2. Offline-Betrieb der Theke — der Nachweis steht aus

Gebaut sind alle drei Stufen (15./16.09.2026): Die Theke hält die Buch-Barcode-Liste im Browser
und nimmt jede Scan-Form offline an, der Sync schickt an `POST /api/action/nachbuchen`, und was
nicht durchging, steht als Meldungsliste am Band. Wie sich das verhält, steht in
[FACHKONZEPT.md](FACHKONZEPT.md) 18.4 und im [Handbuch](HANDBUCH.md).

Zwei Dinge sind offen.

**Der Nachweis von Hand (2.3).** Gebaut heißt nicht geprüft: Am 16.09.2026 stand zweimal
„gebaut", weil Commits vorlagen — angeschlossen war beide Male nichts. Stufe 1 und 3 gehören
von Hand in den echten Chrome, Stufe 2 über die Tür.

**Der Server nimmt eine Nummer nur in der Schreibweise an, in der sie gespeichert ist.**
`internal/service/omnibox_service.go` vergleicht die Vorsilben mit `strings.HasPrefix` gegen
`"A-"`, `"S-"`, `"L-"`, `"B-"`, `"G-"`, und `GetLeserByBarcode` schlägt mit
`WHERE barcode_id = $1` exakt nach. Die Theke vereinheitlicht seit dem 16.09.2026 selbst
(`frontend/src/lib/scanEinordnen.js`, `normalisiereScan`) — nur wenn hinter der Vorsilbe eine
Ziffer steht, sonst würde aus der Suche nach „s-bahn" ein Ausweis. Ein anderer Aufrufer
(Skript, zweite Oberfläche, direkter Aufruf) läuft weiterhin ins Leere. Ob das am Server
geheilt wird, ist zu entscheiden: Eine Suche über `upper(barcode_id)` nutzt den vorhandenen
Index nicht mehr, und der hält die Eindeutigkeit der Ausweisnummern.

### 2.3 Nachweis am Stack (je Stufe, echter Chrome)

- Stufe 1: DevTools-Drosselung 20 s, Schüler laden, Buch, Escape → IndexedDB trägt den Schüler.
  Offline „zurückgeben" im Profil → Rückgabe. Lehrkraft laden, Buch → Handapparat. IndexedDB
  blockiert → „NICHT gespeichert". Zwei Sicherungen einspielen → ein Stapel. Zwei parallele
  Anfragen mit einem Schlüssel gegen `/api/action` → eine Ausleihe.
- Stufe 2: curl mit Cookie gegen `/nachbuchen`: Umbuchung, Doppelscan, gesperrter Ausweis,
  Rückgabe vor älterer Ausleihe (409 `veraltet`), abgebrochener Online-Versand mit gleichem
  Schlüssel (Ausleihe nachgeholt), verloren gemeldetes Buch (Forderung endet, Hinweis in der
  Meldung), zwei parallele Aufrufe auf ein Exemplar, ein Online-Scan parallel zum Nachbuchen
  desselben Kindes (kein Deadlock); `docker stop` der Datenbank → 503 → Eintrag bleibt;
  `pg_stat_activity` beim Nachbuchen von 200 Einträgen (eine Verbindung je Eintrag).
- Stufe 3: Netz aus, Bücher aller Formen und zwei Ausweise scannen, ein Band, kein Vollbild,
  20 Minuten warten ohne Sperre, Netz an → Sperre; anmelden, nachgebucht, Meldungen an einem
  zweiten Browser; Format-1-Sicherung eines anderen Rechners einspielen.

### 2.4 Nicht im Umfang

Anmeldung ohne Server · Schülerdaten auf dem Rechner · Umbuchen im Online-Weg · Geräte offline ·
Rückbau von `/api/action/batch` (nächste Version) · scannende Person im Protokoll (nachgebucht wird
unter dem beim Sync angemeldeten Konto; steht in der Doku).

## 4. Entscheidungen

Die Nummern bleiben fest; beantwortete Fragen fallen weg, sobald sie umgesetzt sind.

### 4.3 `ziel_jahrgang`: bauen oder streichen

> **Gestoppt am 17.09.2026 (siehe 9.6).** Die Sichtung verlangt ausdrücklich
> Mehrjahresbände. Bis die Frage neu entschieden ist, wird `ziel_jahrgang` nicht angefasst und
> die Messung unten nicht gefahren.

`ziel_jahrgang` (mehrjährige Ausleihe) wird in `internal/service/loan_rules.go` gelesen, aber von
keinem Code geschrieben; die Fristregel verzweigt auf einen Wert, der immer 0 ist. Die Frist am
Rückgabetermin ist entschieden. **Entschieden am 16.09.2026: streichen.** Spalte, die drei
Lesestellen in `repository/book_search.go` und der Zweig in der Fristregel fallen. Der Zweig ist
zugleich das Tor zur LMF-Plan-Frist (sie gilt nur bei `additionalYears == 0`) — ein Wert, den
niemand setzt, darf diese Regel nicht aushebeln können.

**Vor dem Streichen eine Zählung am Server (16.09.2026).** „Wird von keinem Code geschrieben"
gilt für den HEUTIGEN Code; im Juni gab es einen Schreiber (`INSERT INTO buecher_titel (…,
ziel_jahrgang, …)`, Commit `f8dab25a`, mit den Migrationen 029/030). Steht auf dem Server auch
nur ein Titel mit einem Wert, dann ist die mehrjährige Ausleihe dort nicht tot, sondern in
Betrieb — und das Streichen gäbe diesen Büchern beim nächsten Ausleihen eine Frist im selben
Schuljahr statt in einem späteren. Deshalb erst messen, dann bauen:

```sql
SELECT count(*) AS titel_mit_wert, min(ziel_jahrgang), max(ziel_jahrgang)
FROM buecher_titel WHERE ziel_jahrgang <> 0;
```

Ergebnis 0 → streichen wie entschieden. Ergebnis > 0 → die Frage ist eine andere und kommt
zurück auf den Tisch.

### 4.4 E6: Nach der Übergabe an die Schulaufsicht

Bleibt der Schüler gesperrt und die Forderung offen, bis das Sekretariat „bezahlt laut
Finanzbericht" bucht — oder gilt die Übergabe schulseitig als erledigt? **Vorschlag (Konzept):**
Sperre bleibt, Löschblockade fällt. **Wann:** sobald ein erster echter Bescheid absehbar ist;
blockiert 5.3. Einzelheiten in [mittel_konzept.md](mittel_konzept.md), Abschnitt 6.

### 4.7 Fünf Umgebungsvariablen, die Compose nicht durchreicht

`ALLOWED_ORIGIN`, `RATE_LIMIT`, `IMAP_PORT`, `SMTP_ALLOW_INSECURE_TLS` und
`SMTP_ALLOW_PLAINTEXT` liest der Server; `docker-compose.yml` reicht keine davon durch, im
Container gilt also immer die eingebaute Vorgabe. `SENTRY_DSN` wird seit 6125527b (18.09.2026)
durchgereicht, leer heißt aus; gesetzt wird sie in Produktion nur nach Datenschutz A6 (leer oder
EU-Instanz). Die beiden `SMTP_ALLOW_*` führte das Register
als Werkzeug-Variablen — der Server liest sie in `mailservice/versand.go`. Auf dem Server am
13.09.2026 im Container leer.
**Frage:** Je Variable: Ist die Vorgabe gewollt? Sonst durchreichen. **Danach:** Ratsche in 5.10.

### 4.8 Etiketten-Altbestand nachtragen?

30.676 echte Exemplare zählen auf dem Server als „Etikett offen" (13.09.2026). Es gibt zwei Wege
mit verschiedener Bedingung: das Druck-Center („Fehlende Etiketten" → „Altbestand aufräumen",
`POST /api/exemplare/etiketten-altbestand`, mit Vorschau und Stichtag; Abnahme-Flow 4) und
`scripts/repair_altbestand_etiketten.sql` (alles ohne `B-`, also auch jedes `LMF-`-Exemplar). Ob
das Skript schon einmal lief, ist nicht belegt. Beides ist nicht umkehrbar.
**Frage:** Nachtragen, und über welchen Weg? Tragen die LMF-Exemplare ein Etikett? **Wann:** vor
Abnahme-Flow 4; kommt eine neue Littera-Übernahme (7.2), erst danach.

### 4.9 Security-Jobs im Release-Gate

Die Pflichtliste in `.github/workflows/release.yml` enthält keinen der Security-Jobs
(govulncheck, gosec, npm audit, Trivy); ein Tag auf einen Commit mit rotem Trivy erzeugt trotzdem
das Release; für das Image siehe 5.10. **Frage:** aufnehmen oder begründet so lassen? Bei Ja prüft
`docs/umgebung_paritaet_test.go` auch `security-scan.yml`. **Wann:** vor dem nächsten Release.

### 4.11 Topf auf der Bestätigungsseite und den großen Etiketten

Der Händler bekommt am selben Tag zwei gleich aussehende Links; auch zu einer
Schülerbücherei-Bestellung werden die großen Lernmittel-Etiketten „Eigentum des Landes"
angeboten. **Frage:** Nennt die Bestätigungsseite den Topf? Große Etiketten nur bei `land`?

### 4.12 Karenz gegen Lesehistorie

Die Uhr vor der Anonymisierung rechnet ab der letzten Rückgabe über `ausleihen.schueler_id`, die
der Lesehistorie-Lauf trennt. Ist die Karenz länger eingestellt als die Lesehistorie, wird früher
anonymisiert als eingestellt. Es gibt zwei Lesehistorie-Fristen: Schülerbücherei (Vorgabe 90 Tage)
und Lernmittel (`lesehistorie_lernmittel_tage`, Vorgabe 730); mit den Vorgaben (Karenz 90) ohne
Wirkung. **Frage:** Soll die Einstellung Karenz ≤ beide Lesehistorie-Fristen erzwingen, oder
speichert die Uhr ihren Zeitpunkt selbst (eigene Spalte)?

**Entschieden am 16.09.2026: eigene Spalte, per Trigger gepflegt.** Nicht erzwingen — die
Kopplung liefe in die falsche Richtung: Um eine längere Karenz zu bekommen, müsste die Schule
die Lesehistorie verlängern, also mehr Personendaten länger aufbewahren. Die beiden Fristen
beantworten verschiedene Fragen und dürfen sich nicht gegenseitig binden.

**Gebaut wird erst nach deinem Ja (17.09.2026)** — die Umsetzung ist eine Migration mit zwei
Triggern und einer Rückfüllung über den ganzen Bestand. Der Plan steht, der Nachweis ist
beschrieben; was fehlt, ist die Freigabe für einen Eingriff ins Schema.

Stattdessen bekommt die Leserzeile eine Spalte „letzter Vorgang", die kein anderer Löschlauf
anfasst; gepflegt von Triggern auf `ausleihen` (Rückgabe) und `schadensfaelle` (bezahlt oder
storniert), nach dem Muster von `konto_hat_leserzeile` — an der einen Stelle, an der kein
Schreibweg vorbeikommt. Das Prädikat rechnet dann `GREATEST(AbgangSeit, letzter_vorgang_am)`
ohne Unterabfragen. Nachweis: ein PG-Test mit Karenz > Lesehistorie, der beweist, dass der
Lesehistorie-Lauf den Anonymisierungs-Zeitpunkt NICHT verschiebt — am alten Stand rot.

### 4.13 Abgänger-Druck und die Suche

Zwei Kommentare sichern zu „Was auf dem Bildschirm steht, steht auf dem Papier";
`GET /api/abgaenger/pdf` kennt aber nur `klasse`, die Suche filtert im Browser. **Frage:** Folgt
der Druck der Suche (neuer Parameter, dieselbe Auswahl zweimal), oder bleibt er klassenweise
(dann fallen die zwei Kommentare)? Ohne Antwort wird nichts gebaut.

### 4.14 Cover im Kollegiums-Portal (`AnliegenWidget`)

Tabelle und API tragen `isbn` und `titel_id` am Anliegen schon; nur das Wunsch-Formular schickt
sie nicht mit. Am 05.09.2026 entschieden: Arbeitslisten der Bibliothek bleiben ohne Cover.
**Frage:** Gilt das auch für die eigene Liste der Lehrkraft, oder bekommt das Wunsch-Formular
eine Buchauswahl?

### 4.16 Routen ohne Aufrufer

Laut API-Inventar (`docs/api_inventar.md`) ruft weder das Frontend noch ein Skript im Repo diese
Routen auf: `PUT /api/books/{id}/cover`, `POST /api/books/{id}/refresh-cover`,
`POST /api/buecher/exemplare/{id}/schadensnotiz` und `POST /api/buecher/exemplare/{id}/aussondern`.
Ein Grep schließt Aufrufer außerhalb des Repos nicht aus.

**Die vier Befunde (17.09.2026, am Code gelesen) — zu entscheiden ist je Route:**

1. **`PUT /api/books/{id}/cover`** (`inventur/update_cover_handler.go`) setzt eine Cover-Adresse
   von Hand, mit Herkunftsprüfung. Sie ist die Tür, die 5.12 schon nennt: Ein unbekanntes Buch
   ergibt 500 statt 404, und ein altes hochgeladenes Cover bleibt als Datei liegen. In der
   Oberfläche gibt es den Upload (`/cover-upload`) und den Sammellauf (`/admin/sync-covers`) —
   eine Adresse von Hand einzutragen, kann heute niemand. *Vorschlag: streichen.* Wer ein
   bestimmtes Bild will, lädt es hoch; das ist derselbe Zweck ohne Fremd-URL in der Datenbank.
2. **`POST /api/books/{id}/refresh-cover`** (`inventur/cover_aktualisierung.go`) holt das Cover
   EINES Buches neu bei den Katalogdiensten. Sauber gebaut (404 bei unbekanntem Buch,
   unterscheidet Netzausfall von Nicht-Treffer, eigene Tests). In der Oberfläche gibt es nur den
   Sammellauf über den ganzen Bestand. *Vorschlag: anbieten* — ein Knopf „Cover neu holen" in der
   Titel-Akte ist die kleinere Handlung als „alle Cover abgleichen", und die Tür ist fertig.
3. **`POST /api/buecher/exemplare/{id}/schadensnotiz`** (`api/copy_admin_status.go`) schreibt die
   Zustandsnotiz eines Exemplars. Kein Aufrufer; die Notiz entsteht heute nebenbei beim Melden
   eines Schadens und beim Aussondern. *Frage: Soll die Bibliothek eine Notiz am Exemplar von
   Hand ändern können — oder ist die Notiz bewusst nur eine Spur der Vorgänge?*
4. **`POST /api/buecher/exemplare/{id}/aussondern`** (`api/copy_admin_status.go`) sondert ein
   Exemplar aus, mit der richtigen Sperre („noch verliehen" → 409). Ausgesondert wird heute über
   den Status-Editor und über die Schadensmeldung. *Frage: doppelte Tür zum selben Zustand
   (Bugklasse „zwei Türen") — oder der bewusste kurze Weg?* Wenn doppelt: streichen, sonst in der
   Exemplar-Liste anbieten.

`POST /api/buecher/exemplare/{id}/defekt` ist am 16.09.2026 gestrichen — sie war zur Hälfte kaputt
(Zweig ohne Schüler in eine Spalte, die Migration 125 entfernt hat), nicht bloß ungenutzt.

### 4.18 Neue Auflage eines Schulbuchs — ein Werk über den Auflagen

**Der Fall (gefragt am 17.09.2026):** Ein Schulbuch wird nachbestellt, es gibt es aber nur noch in
der nächsten Auflage — neue ISBN, also ein neuer Titel. Es geht ausdrücklich NUR um Schulbücher.

**Was heute passiert.** `buecher_titel.isbn` ist UNIQUE; die neue Auflage wird zwangsläufig eine
zweite Titelzeile, und das ist richtig so — es sind zwei verschiedene Bücher. Nur hängen
`meldebestand` und der Gesamtbestand am TITEL (`api/reorders.go`): Die alte Zeile behält den
Meldebestand 95 und zählt ihre 65 Restexemplare, die neue startet mit der Vorgabe 5 und hat 30.
Die Nachbestell-Liste steht damit für dasselbe Buch zweimal da, und beide Zahlen sind falsch.
Dasselbe gilt für den Klassensatz (`class_books`), Vormerkungen und Reservierungen — sie alle
hängen am Titel.

**Wie Littera es macht** (Handbuch, am 17.09.2026 gelesen): Jede Auflage ist ein eigener Titel mit
eigenem Feld „Auflage" („z. B. 3. Aufl."). Beim Anlegen läuft eine Dublettenkontrolle über die
ISBN — ohne ISBN über Verfasser und Haupttitel — und bietet an, statt eines neuen Titels ein
weiteres EXEMPLAR anzulegen. Verbunden werden Auflagen über eine Verweisung „Früherer Titel",
damit die Recherche beide findet. Ein Zusammenführen von Titeln gibt es nicht, eine Ebene über dem
Titel auch nicht. Das ist allerdings die BIBLIOTHEKS-Software; die Lernmittelverwaltung ist bei
Littera ein eigenes Programm. Littera beantwortet also „finde beide Auflagen", nicht „wie viele
Mathe 7 haben wir".

**Entschieden am 17.09.2026:**

- **Der Bedarf rechnet am Werk**, nicht an der Auflage: „95 Stück Mathe 7, egal welche Auflage."
  Die Suche zeigt einen Treffer mit der Gesamtzahl und darunter die Aufschlüsselung je Auflage.
- **Die Ausgabe warnt**, wenn eine Klasse gemischte Auflagen bekommt. Still darf das nicht
  passieren: verschiedene Auflagen heißen verschiedene Seitenzahlen.
- **Zusammengelegt wird nichts.** Zwei Titelzeilen zu einer zu machen und die Exemplare umzuhängen
  verliert die Auflage — und genau die ist bei Schulbüchern die Information, die zählt. Jedes
  Exemplar bleibt an seiner Auflage.
- **Die Form:** eine schmale Tabelle `werke` (Id, Name) und eine NULLBARE Spalte `werk_id` am
  Titel; gruppiert wird über `COALESCE(werk_id, id)`. Jeder Titel ohne Werk ist damit sein eigenes
  Werk — es gibt keinen Zwischenzustand „halber Bestand gepflegt", und jeder Lesepfad, den die
  Frage nicht betrifft, bleibt unverändert. Am Werk rechnen Meldebestand, Bedarf und
  Nachbestellung; an der Auflage bleiben Exemplar, Etikett, Ausleihe und Ausgabe.

**Offen: Wer legt das Werk an?** (Am 17.09.2026 ohne Meinung geblieben.) **Vorschlag:** Der
Vorschlag entsteht automatisch, das Ja kommt von einem Menschen. Vollautomatisch über den Namen zu
gruppieren verbindet früher oder später zwei „Deutschbuch 7" verschiedener Verlage; rein von Hand
pflegt es niemand, und der Bedarf bleibt falsch. Auf dem Weg über die Nachbestellung ist der
Vorschlag praktisch sicher, weil er von einem konkreten Titel ausgeht.

**Reihenfolge, wenn freigegeben:**

1. **„Auflage" als echtes Feld am Titel.** Heute gibt es das Wort im ganzen System nur als
   Freitext in `erweiterte_eigenschaften` aus dem Listenimport. Ohne das Feld stehen zwei Zeilen
   „Lambacher Schweizer 7" in jeder Liste, die niemand unterscheiden kann — unabhängig davon, was
   danach kommt. Kleinster Schritt, nützt sofort.
2. **Dublettenkontrolle beim Anlegen** wie bei Littera (ISBN, sonst Autor und Titel). Sie löst den
   anderen Fall — dasselbe Buch versehentlich zweimal —, der als Fund schon in 5.5 und 5.12 steht.
3. **Das Werk** samt Migration, Gruppierung im Bedarf und Warnung in der Ausgabe.

**Nicht gebaut.** Schritt 3 ändert das Schema und rechnet die Nachbestell-Liste anders. Das geht
in Stufen mit Nachweis und erst nach deiner Freigabe.

### 4.19 Frist fällt in die Ferien

**Der Fall.** Ausleihe am letzten Schultag vor den Herbstferien, Frist 21 Tage: fällig mitten in
den Ferien. Das Kind kann nicht zurückgeben, steht danach in der Mahnliste und ab einem
überfälligen Medium mit gesperrtem Ausweis an der Theke (`MaxOverdueItems`, Vorgabe 1).

**Was heute passiert.** `calculateDueDate` (`internal/service/loan_rules.go`) rechnet
Kalendertage: Stichtag bei Lernmitteln, feste Tageszahl sonst. Kein Kalender geht ein. Der einzige
Behelf ist der Ferien-Leseclub — ein festes Zieldatum für ALLE Ausleihen, von Hand ein- und
auszuschalten.

**Wie Littera es macht** (Handbuch 5.4.18, Kapitel *Grundeinstellungen für Verleih* und *Verleih*,
gelesen am 17.09.2026): zwei Einstellungen. *Öffnungstage* sind die Wochentage, an denen die
Bibliothek geöffnet hat. *Schließtage* sind einzelne Tage mit Bezeichnung, wahlweise als Zeitraum
eingetragen („Sommerferien, Inventur, Betriebsurlaub"), dazu ein Knopf, der die gesetzlichen
Feiertage eines Landes für ein Kalenderjahr übernimmt — jedes Jahr neu zu drücken. Fällt die Frist
auf einen Schließtag, „errechnet LITTERA automatisch den nächsten möglichen Rückgabetermin, trägt
diesen dann ein und stellt den Zeitraum zwischen errechnetem und tatsächlich möglichem
Rückgabetermin mahn- und gebührenfrei". Einstellbar ist, ob bis zum nächsten Öffnungstag oder bis
zum nächsten gleichen Wochentag verschoben wird.

**Woher kämen die Tage bei uns?** Das ist die eigentliche Frage. Gebaut ist bisher nur, was der
LMF-Plan braucht:

- **Feiertage Hessens:** gerechnet, kein Pflegeaufwand (`pkg/lmfplan/feiertage.go`; die gesetzlichen
  Feiertage Hessens samt Fronleichnam).
- **Sommerferien:** Programmtabelle nach dem KMK-Beschluss bis 2030 plus eigene Einträge der
  Schule (Einstellungen → LUSD & Versetzung → Sommerferien). Läuft die Tabelle aus, warnt die
  Betriebsbereitschaft zwei Jahre vorher.
- **Bewegliche Ferientage, pädagogische Tage, Brückentage:** `lmf_plan_freie_tage` — sie hängen
  aber am einzelnen Plan (`plan_id`), nicht am Schuljahr.
- **Herbst-, Weihnachts- und Osterferien gibt es nirgends**, Öffnungstage der Bibliothek auch
  nicht. Genau diese Ferien sind die, in die eine 21-Tage-Frist fällt.

Eine Tabelle `ferien_schliesszeiten` gab es; Migration 102 hat sie am 06.09.2026 ausgebaut, weil
sie nie einen Schreiber bekam und die automatische Mahnpause vor nichts Realem schützte. Das galt
dem Mahnwesen. Hier geht es um den Schreibpfad der Frist — um die Zahl, die auf dem Kontoauszug
steht und die das Mahnwesen später liest.

**Entschieden am 18.09.2026:**

- **Öffnungstage gibt es nicht** — die Bibliothek hat keine festen Öffnungszeiten. Damit besteht
  der Kalender aus Ferien, Feiertagen und Wochenenden; die Hälfte, die Littera „Öffnungstage"
  nennt, entfällt ersatzlos.
- **Laufende Ausleihen rutschen mit.** Wer im September ausgeliehen hat und im Oktober die
  Herbstferien nachgetragen bekommt, soll nicht gemahnt werden, weil die Eintragung zu spät kam.
  Das Muster dafür steht schon: Beim Veröffentlichen des LMF-Plans folgen die offenen
  Schulbuch-Ausleihen dem Termin ihrer Klasse (`api/lmf_termine_frist.go`), und die Meldung nennt
  die Zahl.
- **Die Ferien kommen als Datei** (iCal), nicht als Tipparbeit: sechs Zeiträume je Schuljahr von
  Hand einzutragen, macht niemand zweimal. Die Datei holt die Schule selbst, z. B. bei
  `schulferien.org/deutschland/ical/`; hochgeladen wird sie in den Einstellungen. Zur Laufzeit
  fragt der Server nichts ab — dieselbe Linie wie bei der Sommerferien-Tabelle.
- **Ein Warner**, wenn der Kalender ausläuft: in der Betriebsbereitschaft, wo schon die
  Ferientabelle steht. Ein Kalender, der still endet, rechnet ab dem ersten fehlenden Tag wieder
  falsch, ohne dass es jemand merkt.

**Beim Mitrutschen gilt dieselbe Ausnahmeliste wie beim LMF-Plan:** nur offene Ausleihen, nur nach
hinten, und nicht angefasst werden von Hand gesetzte Fristen, Lernmittel mit Termin aus dem Plan
und die Fristen des Ferien-Leseclubs. Jede Verschiebung steht im Protokoll, und die Meldung nennt
die Zahl der betroffenen Ausleihen.

**Offen — zwei Dateien oder eine?** Vorgeschlagen war je eine Datei für Ferien und für Feiertage.
Für die Ferien ist das richtig. Bei den Feiertagen rate ich ab: Die rechnet das Programm bereits
exakt für Hessen (`pkg/lmfplan/feiertage.go`, Fronleichnam eingeschlossen), ohne Pflege und ohne
Ablaufdatum. Eine hochgeladene Datei daneben wäre eine zweite Quelle für denselben Zustand — die
teuerste Bugklasse dieses Projekts: Wer die Datei eines anderen Bundeslandes erwischt, verschiebt
Fristen auf einen Tag, an dem die Schule offen hat, und niemand sieht warum. *Vorschlag:* die
zweite Datei annehmen, aber nur als **Gegenprobe** — das Programm vergleicht sie mit seiner
Rechnung und meldet Abweichungen, statt sie zu übernehmen. Das kostet wenig und fängt genau den
Fall, in dem unsere Rechnung falsch wäre.

**Was beim Bauen die Fallen sind** (vorab notiert, weil sie still danebengehen):

1. **`DTEND` ist bei ganztägigen Terminen exklusiv.** Ein Ferienzeitraum „bis 31.10." steht in der
   Datei als `DTEND:20261101`. Wer das direkt übernimmt, hängt einen Tag an oder verliert einen —
   und der erste Schultag wird zum Ferientag. Gate genau darauf.
2. **Eine Vorschau vor dem Übernehmen**, wie beim LUSD-Import: „Diese 6 Zeiträume werden
   eingetragen." Die Quelle ist austauschbar (`schulferien.org` ist eine von mehreren und nicht
   amtlich), und auch der Dateiname ist keine Prüfung — er heißt zwar „Ferien Hessen", ist aber
   in zwei Sekunden umbenannt. Geprüft wird der Inhalt, von einem Menschen an der Vorschau.
   **Die Maschine kann dabei eine harte Probe beisteuern:** Jede Ferien-Datei enthält die
   Sommerferien, und die kennen wir für Hessen bis 2030 aus der Programmtabelle. Weichen sie ab,
   stimmt das Bundesland oder das Jahr nicht — dann warnt der Upload, statt die Fristen der
   ganzen Schule auf fremde Termine zu schieben.
3. **Die Sommerferien gibt es schon** — als JSON-Liste unter dem Schlüssel `sommerferien`
   (`pkg/lmfplan/ferien_einstellung.go`), gelesen von Planer und Selbstprüfung. Der neue Kalender
   muss sie aufnehmen, nicht neben ihnen stehen, sonst gibt es zwei Listen von Sommerferien. Der
   Upload ist dann ein Weg in EINE Tabelle, das Tippen von Hand der andere.
4. **Nur Datei, kein Adressfeld.** Eine URL, die der Server selbst abruft, wäre ein Abruf nach
   außen aus dem Schulnetz heraus — der Grund, aus dem schon die Ferientabelle im Programm steht.

**Reihenfolge, wenn freigegeben:** (1) Kalender-Tabelle samt Übernahme der Sommerferien,
(2) iCal-Upload mit Vorschau, (3) Frist rechnet gegen den Kalender, (4) Mitrutschen bei Nachtrag,
(5) Warner in der Betriebsbereitschaft.

**Nicht gebaut.** Ändert einen Schreibpfad und braucht eine Migration; Gate am Rückweg, vorher rot
gesehen.

### 4.20 „Ich will was Gruseliges" — der Bestand hat kein Thema

**Was heute da ist.** Am Titel stehen Fach (Systematik), Signatur, Jahrgang, Schulzweig und die
Beschreibung, die im Volltext mitsucht. Das Fach ist die Schulfach-Systematik: für Lernmittel
richtig, für die Schülerbücherei nicht — „Jugendliteratur" ist kein Thema.

**Wie Littera es macht:** Schlagworte (frei, mit Verweisen für Schreibvarianten und Pseudonyme)
und daneben Interessenkreise — „eine thematische Gliederung des Belletristikbereiches (z. B.
Krimi, Heimat…)". Beides ist in der Recherche filterbar und als Liste auswertbar.

**Was dabei heute verloren geht:** Der Littera-Altbestand bringt Schlagworte mit (MAB 710). Der
Import liest sie, leitet daraus das Fach ab und wirft sie danach weg
(`internal/service/import_service.go`); gespeichert wird keines. Wer sie haben will, spielt den
Import gegen das frische Backup neu ein (7.2).

**Fragen:** Braucht die Schülerbücherei die Themensuche im öffentlichen Katalog — und wer pflegt
sie bei Neuzugängen? Ein Feld, das nach dem Import nie wieder gefüllt wird, kennt nach zwei Jahren
nur noch die Vergangenheit.

**Vorschlag:** Erst am Backup messen, wie viele Titel überhaupt brauchbare Schlagworte tragen;
darunter erübrigt sich die Frage. Trägt der Altbestand, dann ein geschlossenes Vokabular von
zwölf bis zwanzig Wörtern statt freier Schlagworte — als Auswahl am Titel und als Filter im
Katalog und im Portal. Freie Schlagworte brauchen Normdatenpflege (Littera hat dafür ein eigenes
Modul); die haben wir nicht.

**Nicht gebaut.**

---

## 5. Abarbeitbar (Kategorie B)

### 5.1 Schäden und Benutzer

- **Was „Bezahlt" bedeutet** (17.09.2026, Folge aus 9.3 e): Der Knopf in der Schülerakte
  verbucht heute eine Barzahlung am Tresen. Für Lernmittel des Landes sieht die Arbeitshilfe
  Bargeld nur als Ausnahme vor — mit Quittung und Weiterleitung binnen 14 Tagen. Die Briefe
  nennen inzwischen das Konto; ein eingehender Betrag kommt also in aller Regel als
  Überweisung, und niemand an der Theke sieht ihn. Zu klären mit der Schule: Wer bucht eine
  Zahlung ein, die auf dem Kontoauszug steht? Keine Bauarbeit, bevor das beantwortet ist —
  eine erfundene Antwort steht sonst als Vorgang in der Akte.

### 5.2 Bescheid — vor dem ersten echten Bescheid

- Das Kassenjahr ist das Jahr der Frist (ein Dezember-Brief zählt ins Folgejahr).
- Die Frist ist serverseitig unbegrenzt; eine vergangene Frist macht den Bescheid sofort
  übergabefähig.
- Der Nachdruck liest Bank, Aufsicht, Schulanschrift, Geschäftszeichen und Schulleitung live aus
  den Einstellungen; das Gate `TestBescheidNachdruck_BleibtDerselbeBrief` ändert nur die
  Schüleranschrift. Ein Schnappschuss ist eine Schema-Erweiterung.
- `scripts/tabula_rasa.sql` leert `schadensersatz_nummern` nicht; die Bescheide fallen über
  `TRUNCATE … schueler … CASCADE` mit. Nach Tabula rasa sind alle Bescheide weg, der Nummernkreis
  läuft weiter. Vorher festlegen, ob genau das gewollt ist (Nummern nie recyceln).
- Der Elternbrief je Schadensfall (`GET /api/schadensfaelle/{id}/pdf`, `api/pdf.go`) prüft nicht,
  ob die Forderung auf einem Bescheid steht — dann stünde dieselbe Forderung auf zwei Briefen mit
  zwei Fristen. Den Zahlungsweg nennt er seit dem 17.09.2026 richtig (9.3 e). Die Oberfläche
  öffnet ihn seit dem 15.09.2026 nicht mehr (Mahnverfahren Stufe 1); über die Adresse bleibt er
  erreichbar. Fällt mit dem Entfernen der Altbriefe (5.4) weg, sonst vorher denselben Filter wie
  bei der Ersatzforderung.

### 5.3 Folgen der Übergabe (nach 4.4)

`POST /api/bescheide/{id}/uebergeben` setzt heute nur Status und Zeitpunkt
(`repository/bescheid.go`). Offen:

- Übergabe-PDF (Original und Sammelliste).
- Die Gates der Übergabe-Folgen.

### 5.4 Schadensersatz Teil A, Etappen 3 und 4 (nach 8.3)

Konzept: [mittel_konzept.md](mittel_konzept.md), Abschnitt 4.7.

- **Kreis-Rechnung:** zweite Variante des Renderers (Nummernkreis `SB-Jahr-lfd`, Frist, Tabelle,
  Hinweis auf Ersatzbeschaffung, Zahlungsweg aus den Einstellungen; ohne Referenznummer im
  Landesformat, ohne Rechtsbehelfsbelehrung). Solange die Bankverbindung des Schulträgers fehlt,
  steht sichtbar „(Bankverbindung des Schulträgers nicht hinterlegt)". Warnung in der
  Betriebsbereitschaft. Topf in die Referenznummer, sonst kollidieren Land und Kreis an der
  UNIQUE-Spalte. Heute lehnt der Server jeden Topf außer `land` mit 409 ab.
- **Altbriefe entfernen** (der Zahlungsweg ist seit 17.09.2026 in Ordnung, 9.3 e — es geht nur
  noch darum, ob es sie neben dem Bescheid überhaupt weiter geben soll): Elternbrief
  `pdf/schadensfall.go` ← `api/pdf.go`
  (`GenerateDamagePDFHandler`) ← Route in `api/routes_students.go` ← `useStudentProfile.svelte.js`;
  Rechnung `pdf/rechnung.go` ← `api/print.go` ← `GET /api/print/rechnung/{schueler_id}` in
  `api/routes_system.go` ← Knopf in `StudentProfileActions.svelte`; dazu
  `api/print_rechnung_pg_test.go` und `elternbrief_generiert*`. Der Eltern-Mahnbrief bleibt.
- **`DamageReportModal`:** Staffel-Vorschlag mit Herleitung statt Startwert 15 €, kein
  automatisches PDF-Fenster. **Vorgezogen (17.09.2026, siehe 9.3 a):** Dieser Punkt hängt an
  keiner der offenen Fragen — nur die Kreis-Rechnung hängt an 8.3.
- **Doku:** FACHKONZEPT Abschnitt 3 (Mahnwesen ohne Bescheid) und 14 (PDF-Rechnung, Barzahlung am
  Tresen); SECURITY und VVT-Entwurf mit dem Zweck „Schadensersatz-Bescheid". Den VVT-Satz
  vorziehen, bevor die Schule den Entwurf beschließt (8.5).
- **Release** beim Abschluss.

### 5.5 Bestand, Katalog, Druck

- Die Barcode-Höhe des Ausweises ist im Druck fest (`CardFace.svelte`).
- Die ISBN ist nur je Schreibweise eindeutig (mit oder ohne Bindestrich); ein CHECK auf den
  Jahrgang fehlt, „Jahrgang unbekannt" ist von der Vorgabe nicht zu unterscheiden. Erst Dubletten
  und Jahrgänge am Server messen (Einzeiler dafür), dann Schema.

### 5.6 Schüler und LUSD

- LUSD: Zwei Zeilen mit gleichem Namen und Geburtsdatum, aber verschiedenen Klassen werden still
  zu einer Person; die Meldung „mehrdeutig" fehlt (dokumentierte Grenze). Vor der LUSD-Abnahme als
  Hinweis in der Vorschau.
- **Ein Schüler wird Lehrkraft (oder umgekehrt) — weiter nicht möglich, und das ist Absicht.**
  Seit Migration 123 stehen alle Leser in einer Tabelle, die Art lässt sich aber nur zwischen
  Lehrkraft und LiV umstellen. Über die Schüler-Grenze verbietet es die Datenbank
  (`chk_leser_nur_schueler_werden_abgaenger`), weil ein Schüler aus der LUSD kommt und dort
  wieder auftauchen würde. Der Fall ist selten (ein ehemaliger Schüler kommt als LiV zurück);
  heute legt man dafür einen zweiten Leser an. Wenn er öfter vorkommt, ist es ein eigener,
  kleiner Umzugspfad wie Migration 072 — kein Auswahlfeld.
  Der Littera-Lauf übergeht weiter Praktikanten, Sekretariat und „Im Ausland"
  (`internal/littera/leser.go`).

### 5.7 Bestellwesen

- Der Wareneingang gruppiert nach Datum und dem aus `zustand_notiz` abgeleiteten Lieferanten statt
  nach `bestellung_id`: zwei Töpfe am selben Tag ergeben eine Gruppe, ohne Vorab-Barcode
  „Unbekannter Lieferant", das Datum ohne Schulzeitzone.

### 5.8 LMF und Statistik

- Die Statistik hat keine Sequenznummer (eine langsame Antwort kann eine schnellere überholen)
  und im Browser keinen Fehlerzustand: Ein Query-Fehler ergibt eine leere Liste — protokolliert
  wird er inzwischen (`api/stats.go`), zu sehen ist er nicht.

### 5.9 Oberfläche

- Der Stift der Katalog-Kachel: `BuchKarte.svelte` sagt „öffnet die Akte",
  `e2e/cover-aendern.spec.js` sagt „öffnet die Titel-Verwaltung". Im Browser messen, einen
  Kommentar berichtigen.
- Schülerakte: Scheitert der Abruf des Kopfes (`GET /api/schueler/{id}` in 503 oder Netzfehler),
  bleibt die Akte leer — `StudentProfile.svelte` hat nach `{:else if st.profile}` kein `{:else}`.
  Die drei Listen daneben vermerken ihren Ausfall seit dem 15.09.2026; der Kopf ist der
  verbliebene Eintrag in `fehlerausgang.test.js`. Ein `{:else}` mit `LadeFehler` braucht Platz:
  die Datei steht an der Größen-Ratsche (238 Zeilen).

### 5.10 Gates und Werkzeuge

- Keine Ratsche „Go liest, Compose reicht nicht durch" (nach 4.7); `docs/compose_variablen_test.go`
  prüft nur die Gegenrichtung.
- Die Schema-Gegenrichtung ist blind für UNIQUE, Teilindizes und RESTRICT; die Schema-Parität
  vergleicht Funktionen nur am Namen.
- Mehrere Ratschen haben keine Zeile in der Landkarte von [sweeps.md](sweeps.md), heute noch
  `frontend-hygiene-dialoge/-ladekreis/-schalter.test.js`; Regel 7 hat keine Ratsche.
  (`docs/werkzeuge_im_image_test.go`, `docs/compose_variablen_test.go` und
  `frontend-hygiene-tabellen.test.js` stehen inzwischen drin — am 17.09.2026 nachgezählt.)
  (Die seit dem 17.09.2026 neu gebauten Ratschen tragen sich beim Bauen selbst ein — der
  Rückstand betrifft die älteren.)
- Kein Gate gegen unbegrenzte Listen-Endpunkte.
- **Wiedervorlage 17. November 2026: `GO-2026-6452` (excelize).** Am 16.09.2026 erschien eine
  Schwachstelle ohne heile Fassung — der Eintrag führt alle Versionen ab 0 und nennt keine
  mit Fix. Sie trifft uns nicht: Der gewöhnliche Weg hat den Bereichsschutz seit v2.11.0,
  und der ungeschützte Auslagerungs-Weg ist unerreichbar, weil `xlsxgrenze.Optionen()` beide
  Entpackgrenzen gleich setzt (nachgemessen, vier Tests). Solange das so ist, steht sie als
  begründete Ausnahme in `security/vuln-ausnahmen.json`; das Gate
  (`scripts/govulncheck-gate.sh`) wird von allein rot, sobald die Wiedervorlage abläuft, die
  Ausnahme überflüssig wird oder irgendeine andere Schwachstelle unseren Code trifft. Zu tun:
  nachsehen, ob excelize inzwischen eine Fassung mit Fix hat — dann Ausnahme löschen und
  heben.
- Kein Rückweg für ältere Sicherungen beim Wechsel des `BACKUP_ENCRYPTION_KEY`;
  `cmd/littera-import` und `cmd/seed` haben keine Tests. Vor der Littera-Übernahme.

### 5.11 Doku

Nichts offen (Stand 16.09.2026). Die Nummer bleibt, weil die Reihenfolge oben auf sie verweist.

### 5.13 Mahnverfahren: Stufe 3 (nach 4.4)

Das Modell seit dem 15.09.2026 (Stufen 1 und 2): Die Reiter „Alle ·
Akut fällig · Eskaliert" fragen „Wer hat Bücher zu spät?", der Reiter „Schadensersatz" fragt „Wer
schuldet Geld?". Solange das Buch als ausgeliehen gilt, steht das Kind links; sobald ein Verlust
oder Schaden gebucht ist, rechts — mit genau einem Stand und einem nächsten Schritt je Zeile. Der
Bescheid entsteht direkt aus den überfälligen Büchern; der Brief bucht ihren Verlust.

- **Stufe 3**: Folgen der Übergabe (5.3): Übergabe-PDF und Sammelliste für das Schulamt; E6 (4.4).
- Offen aus Stufe 2: Staffelbetrag als Vorschlag im `DamageReportModal` (5.4) — der Weg über die
  Akte fragt den Betrag weiter ohne Vorschlag.

### 5.12 Offene Nachbarn aus dem Review vom 14.09.2026

- **Wächter „Ehemalige mit offenen Vorgängen" (`fa4a2113`):** Nur der Grundausdruck `AbgangSeit`
  ist mit der Löschuhr vereint; die Löschuhr rechnet zusätzlich `GREATEST(…, max(rueckgabe_am),
  max(Schadensfall))`. Ohne Außenwirkung, aber eine dritte Formulierung derselben Frage.
- **Zweiter Cover-Schreibpfad:** `inventur/update_cover_handler.go` setzt `cover_url`, ohne das
  alte Upload-Cover zu löschen, und meldet ein unbekanntes Buch als 500. Gleiche
  Reihenfolgefrage in `cover_aktualisierung.go`, `endpunkte_cover_retry.go`, `cover_service.go`.
- **ISBN-Dublette:** Der UNIQUE-Constraint fängt nur zeichengleiche Dubletten; geprüft wird auf
  einer bereinigten Kopie, gespeichert der Rohwert. `9783123456789` und `978-3-12-345678-9` sind
  zwei Titel.
- **Selbstanmeldung:** Ein liegengelassener Antrag liest dauerhaft „Zugang beantragt"; nur
  `aktiv = true` räumt `zugang_beantragt_am`. Es geht nur noch um Anträge, die weder
  freigeschaltet noch gelöscht werden. Ob das gewollt ist, steht nirgends.
- **Ratschen mit Umgehungsweg, heute ohne offene Stelle:** Die UUID-Ratsche sieht `[]struct` ohne
  `dive`, `x := T{}`, Typen fremder Pakete und `q.Get(…)` nicht; der Fehlerausgang-Scanner prüft
  Form 2 (`?:`) nur mit `istOkZugriff`; `uuidPfadParameter` hat keinen Test.

### 5.14 Fallengelassene Verdachte (15.09.2026)

Nachgestellt und widerlegt — damit der nächste Durchgang sie nicht noch einmal prüft:
`EmpfaengerFuerBescheid` ohne `deleted_at IS NULL` · Rückkehr storniert die Forderung, das Buch
bleibt ausgesondert · „Pool ODER Tx" nur behauptet · Rechte-Asymmetrie an den Buch-Routen
(`adminH` IST `RequireEditBooks`) · Migration 115 droppt `idx_lmf_termine_plan`.

### 5.15 Offene Punkte aus dem Durchgang vom 15.09.2026

- **Ausweis- und Buchnummer werden nur in der Littera-Übernahme gegeneinander geprüft.** Wer von
  Hand eine Ausweisnummer ändert oder ein Buch umetikettiert (`UpdateCopyBarcode`), wird nicht
  gebremst — und die Theke löst eine Nummer ohne Vorsilbe zuerst als Buch auf. Auf dem Server
  gibt es heute keine Überschneidung.
- **Der Bewegungsstempel folgt einer Auswahl, nicht einer Regel** (`repository/bewegungsstempel_pg_test.go`).
  Aussondern, Bestandskorrektur, Schadensmeldung und Soft-Delete stempeln; „Verloren" und
  Reaktivieren im Status-Editor, der Inventur-Abschluss und `repository/damage.go` nicht. Kein
  Schaden gefunden, aber die Auswahl steht nirgends, und der Test prüft eine feste Liste: Ein
  neuer Schreiber ohne Stempel bliebe grün.
- **Der Stand-Merker der Barcode-Liste rechnet mit dem Beginn der Transaktion.** Ändert eine
  lange Transaktion ein Etikett und committet nach einem kürzeren Schreiber, bleibt es bei 304.
  Nachgestellt hinter dem Build-Tag `raster`
  (`TEST_DATABASE_URL=… go test -tags raster -run TestRaster_ ./repository/`). Heute ändert nur
  `UpdateCopyBarcode` einen Barcode, als kurzer Einzelbefehl — scharf wird es erst mit einem
  Schreiber, der in einer langen Transaktion umetikettiert.
- **Ein Ausweis aus dem Altbestand ohne Vorsilbe** (gemessen `B97601826457`) ist ohne Netz
  „unklar" und wird abgewiesen. Hängt am Ausweis-Neudruck und entscheidet sich mit dem frischen
  Littera-Backup (7.2).

### 5.16 Leserdatei und Rolle Leitung — was noch offen ist

Gebaut ist alles: Lesertabelle, Rolle Leitung, Theke und Leserdatei, eine Maske für jeden, die
Pflicht-Schul-E-Mail und das Zusammenführen von Kollegen.

**A. 15 Minuten: einmal durch die Leserdatei gehen.** Stimmen die Wörter, fehlt etwas? Was ein
Kollege bewusst NICHT in seiner Akte hat: Kontoauszug, Ersatzforderung und DSGVO-Auskunft — sie
gehören der Schülerarbeit und lesen die Sicht `schueler`, ausgeblendet statt kaputt.

**B. Entschieden, damit die Frage nicht wiederkommt:** Die E-Mail eines Kollegen steht am Konto
(`benutzer.email`, `UNIQUE lower(email)`) und nicht in `leser.eltern_email`; in der Akte wird sie
vom Konto gelesen und ist nachtragbar, solange keine da ist. Die Leitung sieht den Menüpunkt
„Einstellungen" und darf darin LUSD & Versetzung, Datenverwaltung, LMF-Aktionen und Lieferanten
bedienen; verschlossen sind Schule, Fristen und Mailversand (`manage_settings`) sowie Benutzer &
Rechte (`manage_users`). Der Nummernkreis bleibt unangetastet: Ohne Netz ist die Vorsilbe die
einzige Information, an der die Theke einen Buchscan von einem Ausweisscan unterscheidet.

**C. Offen: Die Ausweisnummer soll beim Anlegen des Kontos entstehen.** Heute entsteht eine
Nummer an drei Stellen — „Neuer Leser", LUSD-Import, Littera-Übernahme. Ein Konto vergibt keine
(`konto_hat_leserzeile` legt die Leserzeile ohne Ausweis an), und der Druck warnt nicht: Ohne
Nummer kommt ein kaputtes Bild und eine leere Zeile auf die Karte. Entschieden am 16.09.2026,
gebaut wird nach der Freigabe — die Umsetzung braucht eine Migration, die allen Lesern ohne
Nummer eine nachträgt. Vorher die Zählung am Server:

```sql
SELECT count(*) FILTER (WHERE barcode_id IS NULL) AS ohne_nummer,
       count(*)                                   AS leser_gesamt
FROM leser WHERE deleted_at IS NULL;
```

Bei der Umsetzung gilt: **ein Generator** (`GetNextSequence` über `leser.barcode_id`, Vorsilbe
`A-`) — eine zweite Vergabe in SQL wäre der Fehler aus Migration 068 in neuer Form. Der Hinweis
am Feld „Ausweisnummer" („Leer lassen, solange kein Ausweis gedruckt ist") stimmt dann nicht mehr
und fällt.

### 5.17 Ein zweites Konto derselben Person erzeugt eine zweite Leserzeile

Der Wächter `trg_benutzer_hat_leserzeile` hängt jedem Konto ohne Leserzeile eine an, und die
Zuordnung zur vorhandenen läuft über die Schul-E-Mail am Konto. Wird ein Konto gelöscht und
später ein neues angelegt, entsteht wieder eine zweite Zeile. Repariert wird das mit „das ist
dieselbe Person" (Zusammenführen); entstehen lassen sollte man es trotzdem nicht.

### 5.18 Die Dauerleihe eines Kollegen wird als überfällig gefärbt

In der Akte am 16.09.2026 behoben; zwei Ausgänge blieben: `components/BorrowersListe.svelte` und
`utils/ausleiherDruck.js` färben die Frist rot, sobald das Datum vorbei ist. Beide hängen an
derselben Abfrage (`api/copy_admin.go`), und die holt `ist_handapparat` gar nicht erst ab. Das
Gate gehört an den fertigen Inhalt, nicht an die Komponente:
`repository/ueberfaellig_regel_test.go` sucht den Vergleich im SQL, aber diese Abfrage vergleicht
nichts — verglichen wird in JavaScript.

Am Schulserver gemessen (16.09.2026): 0 offene Kollegen-Ausleihen. Es gibt nichts zu reparieren.
Entschieden wird weiter an `ist_handapparat`, nicht an `klasse = 'Lehrer'`.

Aus der Schärfung von Frage 12 bleibt eine Beobachtung: Für `klassen` gibt es im ganzen Go-Code
kein `DELETE` — eine vertippte Klasse steht ab dann in jeder Auswahlliste.

### 5.19 Lesepfade gegen die Sicht `schueler` — eine eigene Achse

Die Ratsche aus Frage 13 führt neun SCHREIBpfade gegen die Sicht, je mit Begründung. Gelesen wird
gegen sie an 32 Stellen, und die zählt niemand. An der Datenbank nachgestellt: Steht eine
Vormerkung für einen Kollegen, findet die Abfrage, die beim Rückgabe-Vorgang den Nächsten
bedient, null Kandidaten — die Zeile ist da, die Warteschlange geht über sie hinweg. Ob es solche
Zeilen heute gibt, ist NICHT gemessen.

Zwei Zahlen vom Server, bevor daran etwas gebaut wird:

```
SELECT count(*) FROM vormerkungen v JOIN leser l ON l.id = v.schueler_id WHERE l.art <> 'schueler';
SELECT count(*) FROM schadensfaelle sf JOIN leser l ON l.id = sf.schueler_id WHERE l.art <> 'schueler';
```

Sind beide 0, ist es Vorsorge (Ratsche für Lesepfade) und keine Reparatur. Ist eine größer als 0,
steht eine Person in einer Warteschlange, die sie nie erreicht.

**Zwei Kleinigkeiten, ebenfalls zum Messen:**

- Titel, die vor dem 17.09.2026 ohne ISBN angelegt wurden, tragen dort einen leeren Text statt
  „nichts"; die Dublettenkontrolle sucht nach „nichts" und findet sie nicht. Gemessen mit
  `SELECT count(*) FROM buecher_titel WHERE isbn = '';` — ist die Zahl 0, erledigt sich der Punkt.
- Ein negativer Listenpreis wird von der Datenbank abgelehnt; die Antwort sagt nicht, was erlaubt
  ist. Das Schwesterfeld derselben Migration nennt seinen Bereich.

---

## 6. Beobachten und Kategorie C (nur mit Anlass)

### 6.1 Beobachtungen

- Die Sperrprüfung liest aus dem Pool, während die Checkout-Transaktion mit `FOR UPDATE` offen ist
  (drei Abfragen über eine zweite Verbindung). Bei `MaxConns = 50` ohne Wirkung; beim Nachbuchen
  vieler Ausleihen (Abschnitt 2) beobachten.
- `pg_dump` und `psql` werden im Backup und in der Restore-Probe per Namen über `PATH` aufgelöst:
  keine Shell, keine Injection, im Container ist `PATH` fest.
- `golang.org/x/crypto/openpgp` (`GO-2026-5932`): kein Fix verfügbar, transitiv, kein Aufrufer im
  eigenen Code.
- Designer: Wer den Browser binnen 800 ms schließt, verliert die letzte Auto-Save-Änderung;
  `sendBeacon` bewusst nicht gebaut.
- Migration 082 (Dedupe der Vormerkungen) verlor den neueren `abholbereit`-Eintrag; auf Prod
  gelaufen, nur relevant für eine weitere gewachsene Datenbank.
- Die Rate-Limiter-Maps räumen erst ab 5.000 Einträgen; nur mit vielen frischen Adressen ein
  CPU-Thema.
- ZAP am 05.09.2026: `style-src 'unsafe-inline'`, `csrf_token` ohne HttpOnly, „Suspicious
  Comments" — alle drei dokumentierte Entscheidungen.
- `startGDPRWorker` (`main.go`) ruft nur die Leihen-Anonymisierung und die Abgänger-Löschung;
  `RunGDPRAnonymizeOldData` läuft allein im Cron. Keine Wirkung erkennbar, die Doku beschreibt
  beide Wege.
- Portal: Das Menü „Mein Portal" verlangt `create_reservations`, `GET /api/lmf-termine` nur eine
  Sitzung — bewusst, der Inhalt ist PII-Stufe 0.
- Ausfallmatrix A3 und B4; A3 erst nach S3 (7.3).

### 6.2 Kategorie C

- Die Ferientabelle 2027–2030 (`pkg/lmfplan/ferien.go`) ist eine ungeprüfte Abschrift; der
  Horizont-Test wird ab 2029 rot. Beim nächsten KMK-Beschluss gegen die Quelle prüfen.
- Browser-Gates: Die M3- und axe-Gates öffnen die Planer-Dialoge nicht, axe misst nur den
  Anfangszustand; kein Screenreader-Durchgang; der Ausweis-Designer geht nur per Maus.
- 17 Bestandsstellen bauen ihr Cover selbst (Liste in `frontend-hygiene-cover.test.js`, darunter
  `KlassenBuchKachel` im Portal). Umstellen beim fachlichen Anfassen, nicht in einem Rutsch.
- 3.000 Titel ohne ISBN: `inventur.SucheTextDNB` nur mit Bestätigung durch einen Menschen
  verdrahten.
- `scripts/backup.sh` exportiert die ganze `.env` — als einziger Punkt hier mit leichtem
  Sicherheitsbezug zuerst.
- `TestEtikettenkette_ZaehlerFolgtDenFilternDerListe` schickt `?bis=time.Now()` in der Zone des
  Testprozesses; unter `TZ=Pacific/Midway` ist das der Vortag und der Zähler nennt 0. Kein
  Produktfehler — im Betrieb kommt dieses Datum aus dem Browser in Berlin. Beim nächsten Anfassen
  auf `schulzeit.Zone()` umstellen (gefunden am 18.09.2026, als die volle Suite einmal unter einer
  fremden Prozesszone lief).
- Tote CSS-Klassen in `altlasten.css`; toter `logout()` in
  `frontend/src/inventur/lib/store.svelte.js`.
- Zwei Regexe für die LMF-Kennung (`pkg/lmf/lmf.go`, `internal/service/import_lmf.go`).
- `migrations/110_schadensersatz_bescheide.sql` nennt drei Barzahlungs-Briefe, es sind zwei;
  eingespielte Migrationen bleiben unverändert.
- Knopfzeile über den Reitern (Mahnwesen): kommt aus dem gemeinsamen Seitengerüst; Anlass wäre ein
  Rundgang über das Gerüst.
- Drei handgebaute Pillen-Gruppen in `StatsDashboard` statt `ui/Segmente.svelte`.
- Etikettenraster doppelt (`api/label_formats.go` und `etikettformate.js`), gehalten von
  `etikettformate-konsistenz.test.js`; am 31.08.2026 entschieden geparkt.
- Reste des Nie-verdrahtet-Sweeps: `inventur_sessions.gestartet_von` wird nie angezeigt;
  `abgaenger_jahr` in der Aktivlisten-Antwort ohne Konsument; bei den Geräten
  `ActionEvent.GeraetID` ohne Broadcast und mit Null-Zeitstempel.
- Cognitive Complexity: 32 Funktionen über 15 ohne Tests (Messung 05.09.2026); lohnend allenfalls
  `OverrideDueDateHandler` und `behandleAbgaenger`.
- `javascript:S6551` und `javascript:S8783`: begründete Dauer-Ausnahmen.
- `docs/docs.go` (Swagger) nennt noch `personenart`, `lehrer_id` und `active_teacher_id`.
  Der Generator `swag` läuft unter Go 1.27 nicht mehr durch (er stolpert über die
  Standardbibliothek); das Drift-Gate prüft nur die Endpunkte, nicht die Feldnamen. Beim
  nächsten Anfassen der API-Doku mit einer neueren `swag`-Fassung erzeugen.
- `auth.Claims.BarcodeID` liest niemand mehr; die Ausweisnummer kommt seit Migration 125
  als LEFT JOIN aus der Leserzeile in die Sitzung, nur damit das Feld gefüllt bleibt.
- Tabellen-Inline-Felder mit 36 px: eine `size="sm"`-Variante von `Feld` erst bei Bedienbefund.
- `LabelHeight >= 30` steht zweimal (`api/label_pdf.go`, `api/schueler_etikett_pdf.go`).
- Zwei Normalformen für Namen (`repository.Suchnorm`, `normName` in `api/lusd_paarung.go`); beim
  Anfassen der Paarung zusammenführen.
- Der Paritätstest vergleicht keine COMMENTs und Seeds.
- Erbe der PR-Zulieferungen: Go-Testdateien über 200 Zeilen, ein schwacher Export-CSV-Test.
- Klone: Go 9 Gruppen (05.09.2026), Frontend 0,41 %.
- 52 Handler-Dateien in `api/` formulieren rohes SQL neben `repository/`; der Bestand ist seit dem
  07.08.2026 eingefroren (`handlerMitSQL` in `api/schichtung_test.go`). Umstellen beim fachlichen
  Anfassen einer Datei, nicht in einem Rutsch.
- Tabellen-Bauteil: gebaut (`4a55c52f`); offen allenfalls eine Sichtabnahme.

### 6.3 Parkdeck (bewusste Nicht-Entscheidungen)

Integer-Cent statt float64 · Bundle-Splitting · TypeScript-Migration · `inventur/` ins Haupt-API
verschmelzen · `cmd/migrate` (MySQL) löschen — seine PG-Tests sichern mit `internal/uebernahme`
geteilten Code · API-Versionierung · Mandantenfähigkeit (RLS) · Trennlinien-Durchgang (26 Dateien
mit `divide-y`, nur als eigener Durchgang mit Messung im Browser) · Zugangsbuch-Ausdruck je
Schulhalbjahr und Topf · Bestandskartei-Ausdruck zum 15.3. und 15.9. (beides nennt
[mittel_konzept.md](mittel_konzept.md), Abschnitt 7.1, als Verfahrensvorgabe; nicht gebaut).

---

## 7. Betrieb (liegt bei der Schulseite)

Geplante Zielumgebung ist der Schulserver; heute ist der Hetzner-Server die einzige Instanz. Beim
Umzug gilt dieser Abschnitt dort erneut — ebenso das, was am Hetzner-Server schon erfüllt ist
(Commit-Geschichte, 13.09.2026).

### 7.2 Frisches Littera-Backup

`littera_sav.mdb` ist ein Stand von 2010. Ohne die offenen Ausleihen startet das System mit „alles
verfügbar"; Ausweisnummern und Exemplare nach 2010 fehlen, rund 1.350 Titel haben keine Signatur.
Anforderungen in [littera_schema_befund.md](littera_schema_befund.md). Vorher B7 (8.5). Die
teuerste offene Position vor dem Echtstart — früh bei der Schule anfragen.
**Reihenfolge:** Littera-Personen und -Ausleihen im selben Lauf übernehmen, **bevor** ein echter
LUSD-Import läuft. Der Littera-Personenlauf erkennt Schüler aus der LUSD nicht und legt sie ein
zweites Mal an ([SCRIPTS.md](SCRIPTS.md), Abschnitt 0). Das Geburtsdatum im Backup ist die Brücke
für den späteren LUSD-Abgleich — vor dem Lauf prüfen.
**Vor dem Personenlauf:** Im frischen Backup nachsehen, ob die Tabelle `FremdLeserNummer` gefüllt
ist (im Stand von 2010 ist sie leer). Sie trägt die Nummern, die die Ausweise beim Scannen liefern;
nur mit ihr funktionieren die vorhandenen Ausweise ohne Neudruck. **Ist sie leer, muss jeder Ausweis
neu gedruckt werden:** Kein Ausweis liefert beim Scannen die Lesernummer, und Littera zählt Bücher
und Leser getrennt, beide ab 1 — fast jede Lesernummer ist auf dem Server schon die Nummer eines
Buchs, der Lauf vergibt dann `L-`-Nummern. Ist sie gefüllt, stehen die einzelnen Personen ohne Karte
mit „keine Karte in FremdLeserNummer" im Protokoll des Laufs.
**Am 17.09.2026 aufgefallen:** In `~/Downloads` und auf dem Schreibtisch liegen seit dem
09.09.2026 zwei Littera-Sicherungen vom 01. und 02.09.2026 (856 KB und 602 KB). Ob das die
Datensicherung aus dem laufenden Littera ist, ist UNGEPRÜFT — für eine Vollsicherung wären
100 MB+ zu erwarten, die Größe spricht eher für einen Teilexport. Zum Nachsehen fehlt auf dem
Rechner ein Entpacker für `.7z`.

**Rückweg zu Littera:** Bücher und Schüler behalten ihre Littera-Nummer, Lehrkräfte nicht. Soll der
Rückweg offen bleiben, vor dem Lauf nachtragen und das Littera-Backup vom Umstiegstag aufheben.

### 7.3 S3-Auslagerung der Backups

`S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY` und `S3_BUCKET` sind leer (13.09.2026); alles
liegt auf einer Platte. Nur EU oder Schulträger. Der Code ist fertig.

### 7.4 Manuelle Restore-Probe an einem fremden Ziel

Die automatische Wochenprobe im Container lief am 13.09.2026 erfolgreich; sie ersetzt die
manuelle Probe nicht. Anleitung: [resilience_and_recovery.md](resilience_and_recovery.md),
Abschnitt 2e — dabei die neuen Befehle erproben, die bisher nur am Text geprüft sind. Sinnvoll
nach S3 oder am Schulserver.

### 7.5 Externes Uptime-Signal

Fällt der Server ganz aus, meldet es niemand. Ein externer Monitor ruft alle 5 Minuten `/health`
ab ([DEPLOYMENT.md](DEPLOYMENT.md), Abschnitt 7.0). Etwa fünf Minuten Aufwand; beim Umzug neu
einrichten.

### 7.6 Ruleset `main`

PR-Pflicht entfernen (Solo-Entscheidung 30.07.2026), „Block force pushes" und „Restrict
deletions" anlassen. Am 13.09.2026 trägt das Ruleset noch `pull_request`; Pushes gehen über den
Admin-Bypass.

### 7.7 Abnahmen

Ablauf in [abnahme_checkliste.md](abnahme_checkliste.md), vorher ein Backup.

- Flows 1–3 mit dem Sekretariat: LUSD-Import, Versetzung (vor dem Schuljahreswechsel),
  Klassensatz erledigen. Dabei um Geburtsdatum und Eintrittsdatum im LUSD-Bericht bitten. Vorher 5.6. Ein
  LUSD-Import mit echten Schülern erst nach der Littera-Übernahme (7.2).
- Flow 4 (Altbestand-Etiketten, nicht umkehrbar) erst nach 4.8.
- Flow 5: Selbstanmeldung einer Lehrkraft.
- Danach: Ergebnis hier eintragen; bei Parser-Auffälligkeiten die echte LUSD-Datei anonymisiert als
  Testfixture sichern.

### 7.8 Am Server nachsehen (lesend, Einzeiler)

- Laut Serverlesung vom 13.09.2026 läuft `a841e6cd` (Container-Start 12:59 UTC); `/health` nennt
  keinen Commit.
- „Neuer Text" im Ausweis-Layout: laut Serverlesung vom 13.09.2026 in keinem Wert von
  `system_einstellungen`. Bestätigen, dann erledigt.
- Sind die Admin-Konten deaktiviert? Ist `/app/uploads/fotos` leer? Gibt es Lehrkräfte mit
  Platzhalter-Mail `@lehrer-umzug.invalid`? Braucht `repair_fach_kategorie.sql` einen zweiten
  Lauf?

## 8. Schule, Schulamt, Schulträger

### 8.1 E1: Schulamts- und Schulnummer

Für die Referenznummer der Bescheide; die Felder stehen in den Einstellungen und sind am
13.09.2026 leer. **Erst nach 5.2 eintragen:** Mit den Nummern entstehen echte Bescheide.

### 8.2 E2: E-Mail-Erlass vom 11.06.2018 und aktuelles Musterschreiben

Die vorliegenden Unterlagen sind von 2014 und nennen eine inzwischen aufgelöste Stelle. Gebaut
ist nach dem Muster von 2014; der Text ist an einer Stelle austauschbar.

### 8.3 E5: Zahlungsweg der Schülerbücherei

Kreiskasse mit Kassenzeichen, Budgetkonto oder bar mit Quittung? Zuständig ist der Fachbereich
Schule und Betreuung des Hochtaunuskreises. **Engpass:** Ohne Antwort kein 5.4.

### 8.4 D2: Getrennte Kundenkonten beim Händler?

Führt der Händler getrennte Kundenkonten für Lernmittel und Schülerbücherei? Das Feld
„Kundennummer Schülerbücherei" am Lieferanten bleibt bis zur Antwort leer.

### 8.5 Datenschutz Teil B

Einzelheiten und Entwürfe in [datenschutz_offene_punkte.md](datenschutz_offene_punkte.md),
Abschnitt B, und in [datenschutz/](datenschutz/).

- **B1/B2** VVT und Datenschutzhinweis beschließen; die Entwürfe haben noch Platzhalter. Vorher den
  Zweck „Schadensersatz-Bescheid" ergänzen (5.4).
- **B3** Foto auf dem Schülerausweis: Erlass oder Einwilligung — vor dem ersten Ausweisdruck mit
  Foto.
- **B4** Schulischen DSB beteiligen, Schwellwertanalyse DSFA schriftlich — vor B1/B2.
- **B5** IT-Sicherheitskonzept mit dem Schulträger, Netzplatzierung.
- **B6** Rolle des Wartenden regeln (AVV oder dienstlich).
- **B7** Löschkonzept gegenüber Littera — vor der Littera-Übernahme (7.2).

Zuerst B3 und B4 anstoßen.

### 8.6 Barrierefreiheit

Gilt für das System die Pflicht zur Barrierefreiheit — mit Erklärung zur Barrierefreiheit und
barrierefreien PDFs (HTML-Druckweg oder begründete Ausnahme)? Bis zur Antwort geparkt; was die
Gates heute prüfen, steht in [FACHKONZEPT.md](FACHKONZEPT.md), Abschnitt 19.

---

## 9. Sichtung vom 16.09.2026

Zwölf Punkte, jeder am Code geprüft. Acht sind erledigt und stehen deshalb nicht mehr hier:
Strichcode-Scan mit Handgerät und Kamera, Littera-Barcodes, alte Ausdrucke, Restwert beim
Melden, Listenpreis am Titel, Zugangs- und Abgangsbuch, Titel ohne Exemplare, Sortierung und
Filter der Leserdatei. Die Arbeit steht in den Commits vom 17.09.2026. Offen ist, was hier
folgt.

Zwei Quellen liegen dem zugrunde: `~/Downloads/Arbeitshilfe_Mahnschreiben.pdf` (Erlass vom
17.12.2014, Az. 674.100.002-00178) und `~/Downloads/Ablauf Mahnverfahren.pdf` (die
Anforderungsliste, abgeglichen in [mittel_konzept.md](mittel_konzept.md) Abschnitt 3).

### 9.3 Vorgaben des Landes (Protokoll 1)

**9.3 c) Sperrung bei offener Bearbeitung — LMF untersagt das.**
`internal/service/loan_checkout_validation.go` (`pruefeOffeneSchaeden`) sperrt bei jedem
unbezahlten Schadensfall jede weitere Ausleihe — **ohne Ausnahme für Lernmittel**. Dasselbe im
Geräte-Pfad (`pruefeGeraetAutomatikSperren`). Übergehbar ist es nur von Hand mit Audit-Eintrag;
der Grundzustand ist die Sperre. Woher diese Regel stammt, ist nicht belegt: In der
Arbeitshilfe zum Erlass vom 17.12.2014 und in der Anforderungsliste „Mahnverfahren" steht
zur Sperre nichts (beide am 17.09.2026 im Original gelesen). Die Zeile im Konzept, die sie
als fremde Praxis auswies, war unbelegt und ist entfernt. **Die Frage ist gestellt,
siehe 9.7.**

**9.3 e) Mahnwesen — drei begründete Abweichungen.** Die Punkte 1 bis 3 der Anforderungsliste
(Abwertung, Beschädigungsgrad, wählbare Preisgrundlage) sind am 17.09.2026 gebaut. Es bleiben
drei Stellen, an denen das Programm bewusst etwas anderes tut:

| Nr. | Verlangt                                  | Stand                                                                                     |
| --- | ----------------------------------------- | ----------------------------------------------------------------------------------------- |
| 5   | Versand per Post, E-Mail oder App         | Nur Post — Schriftform, dazu die Datenschutz-Entscheidung A3 vom 22.08.2026.               |
| 6   | Zahlung ohne Rückgabe → Buch **gelöscht** | Wird ausgesondert statt gelöscht — die Bestandskartei muss den Abgang nachweisen.          |
| 7   | Mahnfrist **sechs** Wochen                | Vier Wochen — Arbeitshilfe und Verfahrensbeschreibung nennen vier, mit Datum.              |

Offen bleibt daneben: Der Knopf „Bezahlt" in der Schülerakte bedeutet heute Barzahlung am
Tresen; für Lernmittel ist das die Ausnahme mit Quittung, nicht der Regelweg (5.1). Und der
Zahlungsweg für ein Buch der Schülerbücherei ist nicht entschieden — die Briefe schreiben
dort „(Bankverbindung des Schulträgers nicht hinterlegt)" (E5, 8.3).

### 9.6 A: Mehrjahresbände — die Entscheidung von 4.3 kippt

Protokoll 5, zweiter Spiegelstrich: „Bei den Ausleihfristen fehlt das Jahr. Mehrjahresbände
lassen sich nicht abbilden."

Die Mechanik ist vollständig da: `ziel_jahrgang` → `AdditionalYears` →
`stichtag.AddDate(jahre, 0, 0)` in `internal/service/loan_rules.go`. Es fehlt allein die Tür,
die den Wert setzt. **4.3 hält fest: „Entschieden am 16.09.2026: streichen."** Das ist genau das
Gegenteil dessen, was die prüfende Stelle verlangt — und mit der Spalte fiele auch die Rechnung,
die es dafür schon gibt. **Die Entscheidung gehört zurück auf den Tisch, bevor die Messung aus
4.3 läuft.** Solange sie offen ist, wird `ziel_jahrgang` nicht angefasst.

### 9.7 Offene Fragen

1. **Sperre bei offener Forderung** (9.3 c): Die Frage ist gestellt (Text am 17.09.2026
   formuliert): Gilt das Verbot nur für Lernmittel oder für jede Ausleihe, worauf stützt es
   sich, und zählt eine übergehbare Abweisung schon als „Sperrung"? Bis zur Antwort bleibt es,
   wie es ist.
2. **Mehrjahresbände** (9.6): Wörtlich steht im Protokoll nur der eine Satz „Bei den
   Ausleihfristen fehlt das Jahr. Mehrjahresbände lassen sich nicht abbilden." Die Deutung geht
   mit dem Schreiben zurück. **4.3 bleibt bis dahin gestoppt.**

### 9.9 Zwei Bedingungen neben der Mängelliste

Die Einschätzung am Ende des Protokolls nennt zwei Punkte, die in keinem der zwölf Mängel
stehen:

> „Ein Nachweis der DSVGO-Konformität liegt nicht vor.
> Hosting- und Programmpflegekonzepte sind nicht geplant. Dies könnte ein Ausschlusskriterium
> sein."

- **Nachweis der DSGVO-Konformität.** Das Material liegt vor und ist vollständiger, als der Satz
  vermuten lässt: VVT-Entwurf, Datenschutzhinweis-Entwurf, die PII-Matrix über jede Route, die
  Löschfristen samt nächtlichem Job. Was fehlt, ist ein Dokument, das man einer prüfenden Stelle
  GIBT — und die Beschlussfassung der Schule (8.5, B1–B7). Das Zusammenstellen ist Arbeit an der
  Doku, keine Bauarbeit.
- **Hosting- und Programmpflegekonzept.** Hier fehlt wirklich etwas. Betrieb, Sicherung,
  Wiederherstellung und Aktualisierung sind beschrieben — aber als Anleitung für den Betreiber
  (DEPLOYMENT.md, resilience_and_recovery.md, SCRIPTS.md), nicht als Konzept, das jemand prüft:
  Wer betreibt das Programm, wer pflegt es, was gilt, wenn die Pflege endet, wie kommen
  Aktualisierungen zu einer Schule, die es einsetzt. Das ist zugleich die Frage, ob das Programm
  über die eigene Schule hinaus verwendbar ist — sie lässt sich nicht aus dem Code beantworten.

**Beides ist Voraussetzung für ein „nutzbar", nicht Beiwerk** — und beides steht in keinem
der vier Punkte darüber. Wer die Mängelliste abarbeitet und diese zwei Sätze überliest, hat das
Gate nicht bestanden.
