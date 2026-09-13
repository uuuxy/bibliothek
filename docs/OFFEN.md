# Offene Arbeit

Stand: 13.09.2026

**Die eine Liste.** Hier steht alles, was noch zu tun, zu prüfen oder zu entscheiden ist — Code,
Betrieb und Schule. Einen zweiten Ort gibt es nicht: Das Befund-Register (`docs/befunde.md`) und
die Issues #593, #594, #597, #598, #599 und #600 sind am 13.09.2026 hierher umgezogen. Erledigtes
steht in [erledigt.md](erledigt.md), ältere Stände in `git log -p docs/befunde.md` und in den
geschlossenen Issues.

Andere Dokumente erklären (Konzept, Anleitung, der Katalog der Bugklassen in
[sweeps.md](sweeps.md)), führen aber keine eigene Offen-Liste.

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
2. **Kategorie A wird belegt, nicht behauptet:** ein Test, der am alten Code rot wird. Bis ein
   Fund nachgestellt ist, heißt er „Verdacht".
3. **Neues kommt nur hierher** — Funde, Fragen an Peter, Betriebspunkte. Kein Issue, kein anderes
   Dokument. Eine Frage steht hier, bevor die Antwort kommt.
4. **Erledigt heißt:** hier löschen und in [erledigt.md](erledigt.md) mit Datum und Commit
   eintragen. Eine Antwort bekommt „Entschieden am … (Peter)".
5. **Die Reihenfolge** wird bei jeder Änderung mitgepflegt.

---

## Reihenfolge

1. **1.1** Rechnung ohne Bescheid-Positionen (A). Erst danach die Schulamts- und Schulnummer
   eintragen (8.1).
2. **1.2** und **1.3** am Stack nachstellen, bei Befund beheben.
3. **7.1** Release-Tag für den Stand nach v2.11.0.
4. **3.1–3.3** Theke: was der Offline-Bau voraussetzt.
5. Frage-Runde **4.1–4.3**: Neuladen ohne Netz, Abmelden bei 503, LMF-Frist am Termintag.
6. **Abschnitt 2** Offline-Betrieb der Theke: Plan in drei Stufen vorlegen, je Stufe Nachweis und
   Freigabe.
7. **5.1** Schäden und Benutzer.
8. **5.5–5.9** kleine B-Commits.
9. Vor dem ersten echten Bescheid: **5.2** und **4.5** (E4), dann **4.4** (E6) und **5.3**.
10. Nach der Antwort zu E5 (**8.3**): **5.4**.
11. Übrige Entscheidungen aus Abschnitt 4 gesammelt; **5.10**, **5.11** und Abschnitt 6 nur mit
    Anlass.

**Parallel bei Peter:** Abschnitte 7 und 8 — zuerst S3 (7.3), das Littera-Backup (7.2), die
Anfragen E1, E2, E5 (8.1–8.3), B3 und B4 (8.5) und ein Termin für die Abnahmen (7.7). Einen echten LUSD-Import erst nach der
Littera-Übernahme (7.2).

---

## 1. Sofort (Kategorie A)

### 1.1 Die Rechnung übernimmt Forderungen, die schon auf einem Bescheid stehen

- **Was:** Der Knopf „Ersatzforderung" in der Schülerakte (`StudentProfileActions.svelte`, lädt
  `GET /api/print/rechnung/{schueler_id}`) nimmt alle unbezahlten Schadensfälle des Schülers auf
  und verlangt „bar in der Bibliothek".
  `queryRechnungItems` in `api/print.go` filtert nur `schueler_id` und `ist_bezahlt = false`,
  nicht `bescheid_id IS NULL`.
- **Warum A:** Sobald ein Landes-Bescheid existiert, bekommen Eltern für dieselbe Forderung zwei
  Zahlungsaufforderungen mit zwei verschiedenen Zahlungswegen.
- **Stand:** Am Code geprüft am 13.09.2026. Ohne Wirkung, solange kein Bescheid existiert; ohne
  Schulamts- und Schulnummer (8.1) lehnt der Server jeden Bescheid ab.
- **Nächster Schritt:** Filter und PG-Test, am alten Code rot gesehen. Das Entfernen der
  Altbriefe folgt in 5.4.

### 1.2 Verdacht: Abmelden ohne Antwort des Servers lässt die Sitzung im Browser

- **Was:** `handleLogout` (`frontend/src/lib/stores/authStore.svelte.js`) schickt die Abmeldung
  ab, verwirft jeden Fehler und leert sofort den lokalen Zustand. Das Löschcookie setzt nur die
  Antwort des Servers (`api/logout_handler.go`). Kommt keine an — kein Netz, 502/504 vom Proxy —,
  bleibt das HttpOnly-Cookie im Browser. Beim nächsten Laden fragt `restoreSession` mit diesem
  Cookie `/api/auth/me`.
- **Warum A (Verdacht):** Am geteilten Theken-Rechner wäre nach einem Neuladen die vorige Person
  wieder angemeldet, bis ihre Sitzung abläuft.
- **Stand:** Am Code gelesen am 13.09.2026, nicht nachgestellt. Berührt Offline-Entscheidung (d).
- **Nächster Schritt:** Am Stack nachstellen (Netz weg, abmelden, Netz zurück, neu laden). Bei
  Befund Test rot, dann beheben.

### 1.3 Verdacht: Ein Klick auf „Ausleihe" nimmt dem Scanfeld den Fokus

- **Was:** Die Omnibox fokussiert das Scanfeld nur bei einem Zustandswechsel neu
  (`frontend/src/lib/Omnibox.svelte`), einen globalen Tastenfang gibt es nicht. Das E2E
  `kiosk-scannerfokus.spec.js` vermeidet den Klick auf den Menüpunkt ausdrücklich.
- **Warum A (Verdacht):** Ein folgender Scan ginge ohne Meldung ins Leere, das Buch wäre nicht
  verbucht.
- **Stand:** Notiert am 28.07.2026, am Code plausibel, nie nachgestellt.
- **Nächster Schritt:** Am Stack mit Tastatureingabe nachstellen (Dialoge und Kamera ausgenommen).
  Vor dem Offline-Bau, weil das Band aus Abschnitt 2 dieselbe Fokuslage ändert.

---

## 2. Laufende Arbeit: Offline-Betrieb der Theke

**Ziel (Peter, 13.09.2026):** Bei einem Verbindungsabbruch geht der Betrieb an der Theke normal
weiter.

**Heute, am Code gelesen (13.09.2026):**

- Offline gespeichert werden nur Buch-Barcodes mit `B-` — auf dem Server 55 von rund 34.800
  Exemplaren (30.658 Littera-Mediennummern, 4.065 `LMF-…`). Jeder andere Offline-Scan meldet
  „Netzwerkfehler" und wird nicht gespeichert.
- Ein offline gescannter Schülerausweis wird nicht geladen; die folgenden `B-`-Bücher reihen sich
  als Ausleihe an den vorher geladenen Schüler.
- Mit geladener Lehrkraft wird ein Offline-Buchscan als Rückgabe eingereiht.
- Nach 25 s ohne Server-Signal legt `App.svelte` das Vollbild „VERBINDUNG VERLOREN" über die
  Theke.
- Beim Nachbuchen gilt jede fehlerfreie Antwort als Erfolg; ein zweiter Scan desselben Buchs am
  selben Schüler wird zur Rückgabe; lag das Buch noch bei jemand anderem, wird es dort nur
  zurückgenommen.
- 502/503/504 bei stehendem Netz gelten nicht als offline.
- Laut Gegenprüfung des Plans (am Code gelesen, nicht nachgestellt): Person und Absicht eines
  Scans werden erst nach der hängenden Anfrage gelesen. Wird in diesen bis zu 10 s die Person
  entfernt (Escape), geht das Buch als Rückgabe ohne Person in die Warteschlange.

**Entschieden am 13.09.2026 (Peter):**

- Offline werden alle Buchformen gespeichert: `B-…`, nur Ziffern (auch EAN-13), `LMF-…`.
- Ein offline gescannter Schülerausweis kommt in die Warteschlange; die folgenden Bücher werden
  ihm zugeordnet, der Server löst ihn beim Nachbuchen auf. Keine Schülerdaten auf dem Rechner.
  Sperren prüft der Server beim Nachbuchen und meldet Abgelehntes mit Barcode.
- (a) An der Theke ein nicht blockierendes Band statt des Vollbilds; Scannen geht weiter.
- (b) Lehrerausweis und geladene Lehrkraft werden offline wie ein Schüler zugeordnet; die Bücher
  gehen beim Nachbuchen als Handapparat an die Lehrkraft.
- (c) Beim Nachbuchen wird die Wirklichkeit gebucht: Lag das Buch noch bei jemand anderem, dort
  zurücknehmen und neu ausleihen; beim selben Schüler nichts umkehren; beides steht im
  Nachbuch-Bericht, nichts verschwindet still.
- (d) 502/503/504 gelten wie ein Netzausfall.
- Arbeitsweise: in Stufen — (1) vorhandene Fehler, (2) Server, (3) Theke, Warteschlange und Band.
  Je Stufe Rot-Test am alten Code, volle Suite mit Postgres, Nachweis am Stack und im Browser,
  dann Peters Freigabe.

**Offen:** Frage 4.1 (Neuladen ohne Netz).

**Stand des Plans (13.09.2026):** Drei Entwürfe, zwei Bewertungen und eine Zusammenführung
liegen vor. Drei Gegenprüfungen am Code (Zuordnung und Verlust, Betrieb und Grenzen, Code und
Ratschen) fanden 21 Lücken, 13 davon wichtig, teils doppelt, keine blockierend. Darunter: Die
Person wird zu spät gelesen (siehe oben). „Buch zurückgeben" im Profil würde offline zur
Ausleihe. Eine Reaktivierung vor dem Wächter könnte eine Forderung stornieren. Eine Rückgabe, die
vor der älteren Ausleihe desselben Buchs nachgebucht wird, ließe das Buch auf dem Schüler. Der
Nachbuch-Bericht nennt den Ausleiher nicht und übersteht das Abmelden mit Personenbezug. Das
Nachschärfen läuft.

**Im Plan festhalten:** Nachgebuchte Rückgaben laufen auch durch `VerbucheRueckkehr`, und der
Hinweis „Schulaufsicht informieren" erreicht jemanden. Das axe-Gate misst auch den Zustand „Band
sichtbar". Beim Nachbuchen vieler Ausleihen die Pool-Abfrage in offener Transaktion beobachten
(6.1).

**Nächster Schritt:** Plan am Code prüfen und in drei Stufen vorlegen.

---

## 3. Theke — vor dem Offline-Bau (Kategorie B)

### 3.1 Die Buch-Ausleihe an eine Lehrkraft meldet jeden Datenbankfehler als 404

`resolveTeacherBorrower` (`internal/service/loan_checkout_validation.go`) macht aus jedem Fehler
„Aktives Lehrerprofil nicht gefunden"; die Geräte-Seite unterscheidet seit `cc9e6c8c`.
**Warum vorher:** Mit (d) muss ein Datenbank-Aussetzer als solcher erkennbar sein, und das
Nachbuchen an Lehrkräfte (b) läuft durch diesen Pfad. **Nächster Schritt:** nur `pgx.ErrNoRows`
wird 404, alles andere geht als Fehler weiter; Vorbild `cc9e6c8c`.

### 3.2 Nach dem Zusammenführen hält die Theke die gelöschte Kennung

Die Omnibox gibt `StudentProfile` kein `onMerged` mit (`frontend/src/lib/Omnibox.svelte`;
`StudentDirectory.svelte` tut es). **Warum vorher:** Eine gelöschte Kennung in der Warteschlange
scheitert beim Nachbuchen. **Nächster Schritt:** nach dem Zusammenführen das Ziel laden, mit
Test.

### 3.3 „Einmalig ignorieren" sieht jede Rolle

Der Knopf im Sperr-Dialog (`OmniboxBlockAlert.svelte`) prüft kein Recht. Der Server lässt ihn nur
mit `edit_students` wirken und verwirft ihn sonst; der Dialog erscheint dann erneut (laut, nicht
still). **Warum vorher:** Sperr-Dialog und Nachbuch-Bericht hängen am selben Merkmal `X-Sperre`.
**Nächster Schritt:** Knopf nur mit Recht zeigen (`hatRecht`).

---

## 4. Entscheidungen (Peter)

### 4.1 Neuladen ohne Netz

Nach einem Neuladen ohne Netz erscheint die Anmeldemaske, und die Anmeldung geht nur mit Server
(`authStore.restoreSession`). Der Service Worker hält nur die Oberfläche vor, keine
API-Antworten. Die Warteschlange bleibt erhalten und wird nach der nächsten Anmeldung mit Netz
nachgebucht.
**Frage:** Soll die Theke nach einem Neuladen ohne Netz weiterarbeiten können — dann bräuchte es
eine Anmeldung ohne Server, eine eigene Sicherheitsfrage —, oder bleibt die Anmeldemaske der
Rückfall? **Wann:** vor dem Offline-Plan; die Antwort bestimmt seinen Umfang.

### 4.2 Abmelden bei 503

Seit `039145f2` antwortet `POST /api/auth/logout` mit 503, wenn der Widerruf nicht gelingt;
`handleLogout` wertet die Antwort nicht aus. Das Löschcookie geht in beiden Fällen hinaus
(`api/logout_handler.go`), der Theken-Browser hält die Sitzung danach also nicht mehr; gültig
bliebe nur ein anderswo kopiertes Token bis zum Ablauf.
**Frage:** Nur ein Hinweis, oder angemeldet bleiben, bis der Server den Widerruf bestätigt?
**Vorschlag:** Hinweis. **Wann:** mit 1.2 und dem Offline-Plan.

### 4.3 LMF-Frist am Rückgabetermin

Wer am Termintag der Klasse Lernmittel bekommt, erhält den Termin selbst als Frist
(`RueckgabeTerminFuerKlasse` sucht `>= heute`); am Tag danach gilt der Stichtag 31.07. — in den
Ferien. Nach den Ferien wäre die ganze Klasse überfällig und nach 14 Tagen gesperrt. Dazu:
`ziel_jahrgang` (mehrjährige Ausleihe) wird gelesen, aber von keinem Code geschrieben.
**Frage:** Welche Frist gilt für ein Lernmittel, das am oder nach dem Rückgabetermin der Klasse
ausgegeben wird — der Stichtag des nächsten Schuljahres oder der Termin des nächsten Plans? Und:
`ziel_jahrgang` bauen oder streichen? **Wann:** vor dem nächsten Rückgabetermin (Juni 2027) und
vor dem Offline-Bau, denn eine am Termintag offline ausgegebene Ausleihe wird womöglich erst am
Folgetag nachgebucht. Danach 5.8.

### 4.4 E6: Nach der Übergabe an die Schulaufsicht

Bleibt der Schüler gesperrt und die Forderung offen, bis das Sekretariat „bezahlt laut
Finanzbericht" bucht — oder gilt die Übergabe schulseitig als erledigt? **Vorschlag (Konzept):**
Sperre bleibt, Löschblockade fällt. **Wann:** sobald ein erster echter Bescheid absehbar ist;
blockiert 5.3. Einzelheiten in [mittel_konzept.md](mittel_konzept.md), Abschnitt 6.

### 4.5 E4: Feld „Listenpreis" am Titel

Für die Staffel ab dem 2. Verleihjahr. Der Bescheid-Handler übergibt heute als Neupreis 0
(`api/bescheid_handler.go`); die Staffel nimmt dann ersatzweise den Kaufpreis, der Dialog zeigt
„Kaufpreis (kein Neupreis hinterlegt)". Ein Mensch bestätigt den Betrag. **Vorschlag (Konzept):**
optionales Feld; der Vorschlag nimmt den Listenpreis, sonst den Einkaufspreis, und sagt, welchen.
**Wann:** vor dem ersten echten Bescheid — sonst beantwortet der erste Bescheid für ein Buch ab
dem 2. Verleihjahr die Frage still mit „Kaufpreis". Der Staffel-Vorschlag im Schadensdialog (5.4)
übernimmt die Antwort später.

### 4.6 E7 und E8 bestätigen

Kein E-Mail- oder App-Versand der Bescheide; keine Eltern-Namen und keine zweite Anschrift im
Datenbestand. Beides ist so gebaut. **Vorschlag:** bestätigen.

### 4.7 Sechs Umgebungsvariablen, die Compose nicht durchreicht

`ALLOWED_ORIGIN`, `RATE_LIMIT`, `SENTRY_DSN`, `IMAP_PORT`, `SMTP_ALLOW_INSECURE_TLS` und
`SMTP_ALLOW_PLAINTEXT` liest der Server; `docker-compose.yml` reicht keine davon durch, im
Container gilt also immer die eingebaute Vorgabe. Die beiden `SMTP_ALLOW_*` führte das Register
als Werkzeug-Variablen — der Server liest sie in `mailservice/versand.go`. Auf dem Server am
13.09.2026 im Container leer.
**Frage:** Je Variable: Ist die Vorgabe gewollt? Sonst durchreichen. `SENTRY_DSN` bleibt leer
(Datenschutz A6). **Danach:** Ratsche in 5.10.

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

### 4.10 Zwei offene Jules-PRs

#621 (CTE für die Klassensatz-Verfügbarkeit; laut Messung vom 13.09.2026 langsamer als die
bestehende Abfrage, nicht wiederholt) und #620 (`title` an gesperrten Knöpfen des Planers). Dazu
die Remote-Branch des geschlossenen PR #611. **Vorschlag:** beide schließen, Branch löschen.

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
speichert die Uhr ihren Zeitpunkt selbst (eigene Spalte)? **Vorschlag:** erzwingen, der kleinere
Eingriff.

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

### 4.15 Freitext bezahlter Schadensfälle nach der Anonymisierung

Beim Anonymisieren wird `schadensfaelle.beschreibung` nicht geleert. **Frage:** Soll der
Freitext dabei fallen?

### 4.16 Routen ohne Aufrufer

Laut API-Inventar (`docs/api_inventar.md`, erzeugt am 13.09.2026) ruft weder das Frontend noch ein
Skript im Repo diese Routen auf: `PUT /api/books/{id}/cover`, `POST /api/books/{id}/refresh-cover`,
`POST /api/buecher/exemplare/{id}/schadensnotiz`, `POST /api/buecher/exemplare/{id}/aussondern`
und `POST /api/buecher/exemplare/{id}/defekt` (dahinter `MarkCopyDefekt`, siehe 5.1). Ein Grep
schließt Aufrufer außerhalb des Repos nicht aus. **Frage:** je Route streichen oder in der
Oberfläche anbieten?

### 4.17 Echte Schülerdaten auf dem Hetzner-Server?

[datenschutz_offene_punkte.md](datenschutz_offene_punkte.md) nimmt an, der Hetzner-Server trage
nie echte Schülerdaten; er ist aber die einzige laufende Instanz. Ob echte LUSD-Schülerdaten dort
liegen, ist nicht gemessen. **Nächster Schritt:** Messung (7.8), dann Doku oder Datenlage
angleichen. Bezug: 8.5 (B5, B6).

---

## 5. Abarbeitbar (Kategorie B)

### 5.1 Schäden und Benutzer

- `DeleteUser` (`repository/audit_users.go`) prüft offene Schäden nicht: `benutzer_id` steht auf
  `ON DELETE SET NULL`, `check_damage_responsible` erlaubt beide Bezüge leer — eine offene
  Forderung verliert still ihren Verantwortlichen. Nahe A. **Schritt:** Löschen verweigern wie
  bei aktiven Ausleihen.
- `MarkCopyDefekt` (`repository/damage.go`) trägt ohne Schüler die klickende Bearbeiterin als
  Verantwortliche der Forderung ein. **Schritt:** zusammen mit dem Punkt davor klären, wer in
  `benutzer_id` steht; je ein Commit. Die Route dazu
  (`POST /api/buecher/exemplare/{id}/defekt`) ruft im Repo niemand auf (4.16) — dort zuerst
  entscheiden.
- Der Idempotenz-Schlüssel einer Bestellung überlebt eine Änderung des Warenkorbs
  (`orderStore.svelte.js`, `api/order_service.go`). Ging die Antwort verloren, wird der geänderte
  Warenkorb still zur alten Bestellung. Nahe A. **Schritt:** Schlüssel bei jeder Änderung neu
  vergeben.

### 5.2 Bescheid — vor dem ersten echten Bescheid

- Das Kassenjahr ist das Jahr der Frist (ein Dezember-Brief zählt ins Folgejahr).
- Die Frist ist serverseitig unbegrenzt; eine vergangene Frist macht den Bescheid sofort
  übergabefähig.
- Der Nachdruck liest Bank, Aufsicht, Schulanschrift, Geschäftszeichen und Schulleitung live aus
  den Einstellungen; das Gate `TestBescheidNachdruck_BleibtDerselbeBrief` ändert nur die
  Schüleranschrift. Ein Schnappschuss ist eine Schema-Erweiterung.
- Einstieg nur über überfällige Ausleihen im Mahnwesen. Weil `ReportDamage` die Ausleihe beim
  Melden beendet, bekommt jemand mit nur einer Forderung keinen Bescheid. Es braucht einen
  zweiten Einstieg, etwa aus der Schülerakte.
- `scripts/tabula_rasa.sql` leert `schadensersatz_nummern` nicht; die Bescheide fallen über
  `TRUNCATE … schueler … CASCADE` mit. Nach Tabula rasa sind alle Bescheide weg, der Nummernkreis
  läuft weiter. Vorher festlegen, ob genau das gewollt ist (Nummern nie recyceln).

### 5.3 Folgen der Übergabe (nach 4.4)

`POST /api/bescheide/{id}/uebergeben` setzt heute nur Status und Zeitpunkt
(`repository/bescheid.go`). Offen:

- Exemplare auf `VERLUST` umstellen. Heute bleibt `aussonderung_grund` bei `BESCHAEDIGUNG`, auch
  bei Verlust; der Fehlbestandsbericht findet diese Exemplare nicht.
- Übergabe-PDF (Original und Sammelliste).
- „Buch doch gefunden" im Fehlbestandsbericht (`MarkiereVerlustAlsGefunden`) muss
  `repository.VerbucheRueckkehr` rufen, sonst endet die Forderung nur an der Theke. `VERLUST` und
  Aufruf gehören in denselben Strang — getrennt entstünde ein A.
- Die Gates der Übergabe-Folgen.

„Ausleihen beenden" entfällt: `ReportDamage` beendet die Ausleihe schon beim Melden.

### 5.4 Schadensersatz Teil A, Etappen 3 und 4 (nach 8.3)

Konzept: [mittel_konzept.md](mittel_konzept.md), Abschnitt 4.7.

- **Kreis-Rechnung:** zweite Variante des Renderers (Nummernkreis `SB-Jahr-lfd`, Frist, Tabelle,
  Hinweis auf Ersatzbeschaffung, Zahlungsweg aus den Einstellungen; ohne Referenznummer im
  Landesformat, ohne Rechtsbehelfsbelehrung). Solange die Bankverbindung des Schulträgers fehlt,
  steht sichtbar „(Bankverbindung des Schulträgers nicht hinterlegt)". Warnung in der
  Betriebsbereitschaft. Topf in die Referenznummer, sonst kollidieren Land und Kreis an der
  UNIQUE-Spalte. Heute lehnt der Server jeden Topf außer `land` mit 409 ab.
- **Altbriefe entfernen:** Elternbrief `pdf/schadensfall.go` ← `api/pdf.go`
  (`GenerateDamagePDFHandler`) ← Route in `api/routes_students.go` ← `useStudentProfile.svelte.js`;
  Rechnung `pdf/rechnung.go` ← `api/print.go` ← `GET /api/print/rechnung/{schueler_id}` in
  `api/routes_system.go` ← Knopf in `StudentProfileActions.svelte`; dazu
  `api/print_rechnung_pg_test.go` und `elternbrief_generiert*`. Der Eltern-Mahnbrief bleibt.
- **`DamageReportModal`:** Staffel-Vorschlag mit Herleitung statt Startwert 15 €, kein
  automatisches PDF-Fenster.
- **Doku:** FACHKONZEPT Abschnitt 3 (Mahnwesen ohne Bescheid) und 14 (PDF-Rechnung, Barzahlung am
  Tresen); SECURITY und VVT-Entwurf mit dem Zweck „Schadensersatz-Bescheid". Den VVT-Satz
  vorziehen, bevor die Schule den Entwurf beschließt (8.5).
- **Release** beim Abschluss.

### 5.5 Bestand, Katalog, Druck

- Massenlöschen `DELETE /api/books` hängt an `edit_books`, Einzellöschen an `delete_books`.
- `DeleteBooks` liest die Spuren vor der Transaktion.
- Titel-Etiketten drucken ausgesonderte Exemplare mit (`api/labels.go`).
- Buchetiketten haben keinen Ersatz für Zeichen außerhalb cp1252; ş und ł werden zum Punkt
  (`api/label_pdf.go`).
- Die Barcode-Höhe des Ausweises ist im Druck fest (`CardFace.svelte`).
- Die ISBN ist nur je Schreibweise eindeutig (mit oder ohne Bindestrich); ein CHECK auf den
  Jahrgang fehlt, „Jahrgang unbekannt" ist von der Vorgabe nicht zu unterscheiden. Erst Dubletten
  und Jahrgänge am Server messen (Einzeiler für Peter), dann Schema.

### 5.6 Schüler und LUSD

- Das Bearbeiten-Formular schickt das alte `abgaenger_jahr` mit, ein Klassenwechsel rechnet es
  nie neu. Vor der Versetzungs-Abnahme (7.7).
- Das Schülerfoto per Barcode wird auch für Schüler im Papierkorb ausgeliefert
  (`api/photo_serve.go`).
- Purge-Fehler kommen immer als 409 (`api/student_deleted.go`).
- LUSD: Zwei Zeilen mit gleichem Namen und Geburtsdatum, aber verschiedenen Klassen werden still
  zu einer Person; die Meldung „mehrdeutig" fehlt (dokumentierte Grenze). Vor der LUSD-Abnahme als
  Hinweis in der Vorschau.

### 5.7 Bestellwesen

- Der Wareneingang gruppiert nach Datum und dem aus `zustand_notiz` abgeleiteten Lieferanten statt
  nach `bestellung_id`: zwei Töpfe am selben Tag ergeben eine Gruppe, ohne Vorab-Barcode
  „Unbekannter Lieferant", das Datum ohne Schulzeitzone.
- Mail-Datum und Link-Frist stehen in Serverzeit.

### 5.8 LMF und Statistik

- Steht eine Klasse zweimal im Plan, nennen Ausleihe und Massenabgleich zwischen den Terminen
  verschiedene Fristen (nach 4.3).
- Die Statistik hat keine Sequenznummer (eine langsame Antwort kann eine schnellere überholen) und
  keinen Fehlerzustand: Ein Query-Fehler ergibt eine leere Liste ohne Logzeile (`api/stats.go`).

### 5.9 Oberfläche

- Escape in einem offenen Select schließt den ganzen Dialog: `ui/Select.svelte` ruft nur
  `preventDefault`, `escapeSchliesst.js` prüft das nicht.
- Der Stift der Katalog-Kachel: `BuchKarte.svelte` sagt „öffnet die Akte",
  `e2e/cover-aendern.spec.js` sagt „öffnet die Titel-Verwaltung". Im Browser messen, einen
  Kommentar berichtigen.
- Eine Sperre am Gerät meldet „ausleihe für diese/n Schüler/in ist gesperrt: Gerät ist aktuell
  gesperrt", obwohl kein Schüler betroffen ist: `ErrBlocked` (`internal/service/loan.go`) trägt den
  Schülertext, `internal/service/device_service.go` hängt die Gerätemeldung an. Offen seit
  `3899cf18`.

### 5.10 Gates und Werkzeuge

- Keine Ratsche „Go liest, Compose reicht nicht durch" (nach 4.7); `docs/compose_variablen_test.go`
  prüft nur die Gegenrichtung.
- Die Schema-Gegenrichtung ist blind für UNIQUE, Teilindizes und RESTRICT; die Schema-Parität
  vergleicht Funktionen nur am Namen.
- Mindestens zehn Ratschen haben keine Zeile in der Landkarte von [sweeps.md](sweeps.md), u. a.
  `docs/werkzeuge_im_image_test.go`, `docs/compose_variablen_test.go`,
  `frontend-hygiene-dialoge/-ladekreis/-schalter/-tabellen.test.js`; Regel 7 hat keine Ratsche.
- Kein Gate gegen unbegrenzte Listen-Endpunkte.
- `api/search_debug_test.go` hat eine feste DSN auf die Entwicklungs-DB und einen Skip ohne Guard.
- Kein Rückweg für ältere Sicherungen beim Wechsel des `BACKUP_ENCRYPTION_KEY`;
  `cmd/rotate-encryption-key`, `cmd/littera-import` und `cmd/seed` haben keine Tests. Vor einem
  Schlüsselwechsel oder der Littera-Übernahme.
- `prettier --check` und `gofmt -l` laufen nur im pre-commit-Hook, nicht in CI.
- Das versionierte Image fragt keinen CI-Check ab: `.github/workflows/docker-publish.yml` prüft beim
  v-Tag nur das Muster und ob der Commit auf `main` liegt; ein Tag auf roter CI erzeugt
  `ghcr.io/uuuxy/bibliothek:x.y.z`, nur das Release (`release.yml`) bleibt aus. `DEPLOYMENT.md`
  (Abschnitt Release) behauptet, beide Tag-Workflows prüften die CI. Die Produktion baut heute
  selbst (`update.sh`, `docker compose up -d --build`); ein Deploy aus dem Image wäre betroffen.
  Am Code geprüft am 13.09.2026. **Schritt:** das Image-Workflow dieselbe Pflichtliste abfragen
  lassen, mit Ratsche in `docs/umgebung_paritaet_test.go`.

### 5.11 Doku

- `abnahme_checkliste.md` spricht im Kopf von vier Flows, es sind fünf.
- Der Kopfkommentar von `frontend/e2e/m3-bauform.spec.js` nennt unter „Was dieses Gate nicht
  sieht" elf Overlays, die `Modal.svelte` nicht benutzen (StudentLockModal, DamageReportModal,
  WebcamCapture, OmniboxBlockAlert, …). Seit dem Durchgang bis zum 07.09.2026 (`dc99bad2`)
  nutzen mindestens `StudentLockModal` und `DamageReportModal` das Bauteil. Die Liste am Code
  nachziehen.
- HANDBUCH: Hinweis, dass im Vermerk des LMF-Plans keine Schülernamen stehen — er erscheint im
  Portal des ganzen Kollegiums und im PDF.

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
- ZAP am 13.09.2026 (angemeldet): 44× „SQL Injection" am Stack widerlegt; der echte Fund daraus
  (UUID ergab 500) ist erledigt.
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
- Tabellen-Inline-Felder mit 36 px: eine `size="sm"`-Variante von `Feld` erst bei Bedienbefund.
- `LabelHeight >= 30` steht zweimal (`api/label_pdf.go`, `api/schueler_etikett_pdf.go`).
- Zwei Normalformen für Namen (`repository.Suchnorm`, `normName` in `api/lusd_paarung.go`); beim
  Anfassen der Paarung zusammenführen.
- Der Paritätstest vergleicht keine COMMENTs und Seeds.
- Jules-Erbe: Go-Testdateien über 200 Zeilen, ein schwacher Export-CSV-Test.
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

## 7. Betrieb (liegt bei Peter)

Geplante Zielumgebung ist der Schulserver; heute ist der Hetzner-Server die einzige Instanz. Beim
Umzug gilt dieser Abschnitt dort erneut — ebenso das, was am Hetzner-Server schon erfüllt ist
(siehe [erledigt.md](erledigt.md), 13.09.2026).

### 7.1 Release-Tag

Nach v2.11.0 liegen 25 Commits ohne Tag auf `main` (13.09.2026). Tag nach grüner CI, ohne
Rückfrage — damit der Offline-Bau auf einem benannten Stand beginnt.

### 7.2 Frisches Littera-Backup

`littera_sav.mdb` ist ein Stand von 2010. Ohne die offenen Ausleihen startet das System mit „alles
verfügbar"; Ausweisnummern und Exemplare nach 2010 fehlen, rund 1.350 Titel haben keine Signatur.
Anforderungen in [littera_schema_befund.md](littera_schema_befund.md). Vorher B7 (8.5). Die
teuerste offene Position vor dem Echtstart — früh bei der Schule anfragen.
**Reihenfolge:** Littera-Personen und -Ausleihen im selben Lauf übernehmen, **bevor** ein echter
LUSD-Import läuft. Der Littera-Personenlauf erkennt Schüler aus der LUSD nicht und legt sie ein
zweites Mal an ([SCRIPTS.md](SCRIPTS.md), Abschnitt 0). Das Geburtsdatum im Backup ist die Brücke
für den späteren LUSD-Abgleich — vor dem Lauf prüfen.

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
- Messung zu 4.17: Liegen echte Schülerdaten auf dem Server?

### 7.9 Bis zum Offline-Bau: Hinweis an die Theke

Ohne Netz speichert die Theke heute nur `B-`-Bücher (Abschnitt 2). Während eines Netzausfalls
keinen neuen Ausweis scannen — die folgenden Bücher gingen an den vorher geladenen Schüler, bei
geladener Lehrkraft würden sie zur Rückgabe. Vorgänge auf Papier notieren und nach der Rückkehr
der Verbindung scannen.

---

## 8. Schule, Schulamt, Schulträger

### 8.1 E1: Schulamts- und Schulnummer

Für die Referenznummer der Bescheide; die Felder stehen in den Einstellungen und sind am
13.09.2026 leer. **Erst nach 1.1 eintragen.**

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
