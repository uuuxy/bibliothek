# Befund-Register

**Zweck:** Ein Ort für Dinge, die beim Arbeiten auffallen, aber nicht sofort angefasst
werden. Ohne diese Liste gibt es nur zwei Umgangsformen mit einem Fund — sofort
reparieren oder vergessen — und beide sind falsch.

Ergänzt [`invarianten.md`](invarianten.md): Dort steht, was immer wahr sein _muss_.
Hier steht, was aufgefallen ist und noch nicht entschieden oder erledigt wurde.

**Erledigtes wird gelöscht, nicht abgehakt** — es steht vollständig in
`git log -p docs/befunde.md`. Stände vor früheren Kürzungen: `2e09ec14` (31.08.),
`47a09f70` (01.09.), `9f91784b` (04.09.), `e2bd4b72` (05.09.), `36c7bdce` (05.09. abends: Abgänger + LMF-Plan gebaut).

---

## Die Einordnung

Vor jedem Fund steht dieselbe Frage — **nicht** „ist das hässlich?", sondern
**„kann das still jemandem schaden?"**:

|       | Kategorie                                                                                                                                                                                            | Umgang                                                               |
| ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| **A** | Kann **stillschweigend** ein falsches Ergebnis für einen echten Menschen erzeugen: doppelte Mahnung an Eltern, Daten beim falschen Empfänger, verlorene Eingabe, ein Gate, das nicht rot werden kann | **Sofort**, eigener Commit, eigener Test, eigene Deploy-Entscheidung |
| **B** | Fehler, der sich **laut** meldet, oder Unordnung ohne Wirkung nach außen: totes Codestück, wackeliger Test, Doppelung                                                                                | **Hier notieren**, gebündelt abarbeiten                              |
| **C** | „Wenn ich schon mal hier bin" — Umbenennungen, Stilfragen, Refactorings ohne Anlass                                                                                                                  | Nur mit Anlass und Zeit                                              |

Zwei Regeln dazu:

1. **Ein Fund = ein Commit.** Kein Anhängen verwandter Aufräumarbeiten. Was beim
   Reparieren zusätzlich auffällt, kommt in diese Liste, nicht in denselben Commit.
2. **Kategorie A wird belegt, nicht behauptet.** Ein Fund dieser Klasse braucht einen
   Test, der mit dem alten Code rot wird. Ohne diese Gegenprobe ist unklar, ob
   überhaupt etwas repariert wurde.

**Reihenfolge:** steht seit 09.09.2026 auf GitHub — angeheftetes Issue [#594](https://github.com/uuuxy/bibliothek/issues/594) (Pakete 1–5 = #595, #596, #597, #598, #593; Betrieb #599, Schule #600; Milestones sind seit v2.8.0 nur noch Namen). Alles Weitere beim nächsten fachlichen Anfassen.

---

## Offen — abarbeitbar

- **Trennlinien in Tabellen (C, Design-Frage).** M3 Lists: „Limit dividers to
  uncontained or complex lists, only when a stronger visual separation is necessary."
  Der LMF-Planer kommt seit 06.09.2026 ohne Zeilen-Trennlinie aus (48-px-Zeilen,
  Hover-Fläche); 26 Dateien tragen `divide-y` (Stand 13.09.2026), meist `divide-slate-100`. Ob die
  ganze Anwendung nachzieht, ist eine Gestaltungsentscheidung mit sichtbaren Folgen —
  nicht nebenbei, sondern als eigener Durchgang mit Messung im Browser.

- **Rasterdurchgang 06.09.2026 über die Änderungen desselben Tages** (Peter: „lass bitte
  die Schemata komplett über die heutigen Änderungen laufen"). Elf Fragen plus
  Sweeps-Achse über 97 geänderte Dateien (+4525/−985), sechs Prüfer. Vierzehn Funde, alle
  im selben Durchgang behoben und je am Rückbau rot gesehen (06116d38 bis 4ef052f1) —
  darunter ein rotes `main` seit 14:40 (der Postgres-Test kannte die zwei Schlusszeilen
  „Nachzügler"/„Aufräumen" nicht, und der Pre-Push-Hook lässt die `*_pg_test.go` aus).
  Hier bleibt nur, was offen ist:

  - **Portal-Menü und Portal-Route messen verschieden (C).** Das Menü „Mein Portal"
    verlangt `create_reservations`, `GET /api/lmf-termine` und dessen PDF nur eine
    Sitzung. Ein HELFER sieht den Menüpunkt nie, kann den veröffentlichten Plan aber
    abrufen. Inhalt ist PII-Stufe 0 und so dokumentiert; die Asymmetrie ist bewusst.
  - **Der Vermerk ist Freitext, die PII-Matrix nennt ihn „kein Schülerbezug" (C).** Tippt
    die Bibliothek dort einen Schülernamen („Nachzügler: …"), steht er im Portal des
    ganzen Kollegiums und im PDF. Kein Code-Fehler; gehört ins Handbuch.
  - **Die Ferientabelle 2027–2030 ist eine ungeprüfte Abschrift (C).** Extern verifiziert
    ist nur 2026 (gegen Peters Excel). Ein Zahlendreher in Tabelle UND Test fiele erst im
    Juni des betroffenen Jahres auf. Beim nächsten KMK-Beschluss gegen die Quelle prüfen.
  - **Kleinkram (C):** Startstunde, Vorschau-Form und die Wartezeit der Entprellung sind
    am 12.09.2026 erledigt (`a5d2a6f6`, `cc7bc392`, `c672183e`) — je mit einem Gate, das
    die Zahl nicht wieder auseinanderlaufen lässt. Offen: die drei Browser-Gates öffnen
    die neuen Planer-Dialoge nicht. Nachgesehen am 13.09.2026: Welche drei gemeint waren,
    steht nirgends; `m3-bauform.spec.js` und `barrierefreiheit-dialog.spec.js` öffnen keinen
    Planer-Dialog, nur der Funktionstest `lmf-plan.spec.js` tut es. Beim nächsten Anfassen
    entscheiden, ob die M3- und axe-Gates die Planer-Dialoge mitöffnen sollen.

- **Buchcover: Rest nach dem Bestellbedarf** (04.09.2026). Der Bestellbedarf zeigt das
  Cover in der Zeile (`ui/BuchCover.svelte`); auf dem Zielsystem tragen 5.724 von 8.706
  Titeln ohne Exemplar ein Cover — die Anzeige lohnt. Offen:

  - **17 Bestandsstellen** bauen ihr Cover noch selbst (Liste in
    `frontend-hygiene-cover.test.js`, eingefroren). Umstellen beim nächsten fachlichen
    Anfassen — nicht in einem Rutsch, das sind täglich benutzte Bildschirme.
  - **Arbeitslisten bleiben ohne Cover** (entschieden 05.09.2026): Das führende Element der
    `ArbeitsZeile` ist der Klassenkreis, nach dem Klassensatz-Reservierungen und Wünsche &
    Meldungen überflogen werden — M3 kennt genau eines. Die zwei fehlenden Backend-Felder
    (`isbn` in `GetKlassensatzReservierungen`, `cover_url` in `anliegen.go`) werden deshalb
    nicht gebaut.
  - **Portal:** `AnliegenWidget` zeigt kein Cover (Entscheidung unten); `PortalSchulbuecher`
    und `PortalLernmittel` zeigen Cover, aber über den Eigenbau `KlassenBuchKachel` statt
    `ui/BuchCover` — die Kachel steht unter den 17 Bestandsstellen.
  - **3.000 Titel ohne ISBN** (`PENDING`, werden vom Sync nie versucht): Datenfrage, keine
    Reparatur — zehn Jahre alte Littera-Daten. `inventur.SucheTextDNB` (Freitext) **nur mit
    Bestätigung durch einen Menschen** verdrahten: Freitext auf „Mathematik" liefert
    hunderte Treffer, den ersten zu nehmen hängt ein falsches Cover an ein Buch. Vorbild:
    DNB-Signaturvorschlag (22a10b1).

- **Bestands-Durchgang 10.09.2026** (Peter: „lass alle Schemata nochmal über alles laufen,
  keine Ausnahmen"). Alle zwölf Fragen über jede Datei (neun Prüfer, je ein Bereich), die
  „sieht nicht"-Spalte der Landkarte als Suchliste, dazu die volle Gate-Batterie (Go mit
  Postgres 2341 grün, Vitest, svelte-check, Lint, deadcode, govulncheck, Trivy, Druck-Gate,
  Playwright 183 grün). **22 Funde in 20 Commits behoben** (9056f2d0 … f67078f3), jeder
  am alten Code rot gesehen; neue Klassen in `sweeps.md`. Hier nur, was offen ist:

  - **Entschieden, bleibt so (Peter, 11.09.2026):** Geisterbuch-Fall bei der Ausleihe
    (`entferneErfuellteVormerkung`). Nimmt ein Kind statt des bereitgelegten ein anderes
    Exemplar, geht das bereitgelegte ins Regal zurück und nicht an den Nächsten auf der
    Warteliste. Es geht nichts verloren; der Nächste wird bei der nächsten Rückgabe
    bedient. Die strenge Reihenfolge der Warteliste ist im Betrieb nicht nötig.
  - **Schäden/Benutzer (B):** `MarkCopyDefekt` trägt ohne Schüler den klickenden
    BEARBEITER als Verantwortlichen ein, zwei Leser nennen ihn „Schuldner"; `DeleteUser`
    prüft offene Schäden nicht.
  - **Bescheid (B):** Kassenjahr = Jahr der Frist (Dezember-Brief zählt ins Folgejahr); Frist serverseitig
    unbegrenzt (Vergangenheit = sofort übergabefähig); Nachdruck liest Bank, Aufsicht,
    Schulanschrift, Geschäftszeichen und Schulleitung live aus den Einstellungen; die Rechnung
    (Knopf in der Schülerakte, `api/print.go`) führt Bescheid-Positionen weiter mit „bar in der
    Bibliothek" (zwei Zahlungswege für eine Forderung, Etappe 4; der Elternbrief nur noch über
    die direkte URL); Einstieg nur über überfällige Ausleihen — und weil `ReportDamage` die
    Ausleihe beim Melden beendet, fällt jede gemeldete Forderung aus dem Mahnwesen: Wer „nur"
    eine Forderung hat, bekommt keinen Bescheid;
    `aussonderung_grund` bleibt BESCHAEDIGUNG auch bei Verlust (Nachtrag 12.09.2026: Damit
    findet der Fehlbestandsbericht diese Exemplare nicht — sein „Buch doch gefunden" sucht
    `VERLUST`. Solange die Rückkehr über die Theke läuft, fällt es nicht auf); `tabula_rasa.sql` leert
    `schadensersatz_nummern` nicht.
  - **Bestand/Katalog (B):** Massenlöschen `DELETE /api/books` hängt an `edit_books`,
    Einzellöschen an `delete_books`; ISBN-Eindeutigkeit nur je Schreibweise (mit/ohne
    Bindestrich); `DeleteBooks` liest die Spuren vor der Transaktion; Titel-Etiketten
    drucken ausgesonderte Exemplare mit; Jahrgangs-CHECK erst nach Messung der Prod-Daten.
  - **Bestellwesen (B):** Wareneingang gruppiert nach Datum und dem aus
    `zustand_notiz` abgeleiteten Lieferanten (`internal/service/order_service.go`) statt
    `bestellung_id` (zwei Töpfe am selben Tag = eine Gruppe; ohne Vorab-Barcode
    „Unbekannter Lieferant"; Datum ohne Schulzeitzone); Mail-Datum und Link-Frist in
    Serverzeit; Idempotenz-Schlüssel überlebt eine Änderung des Warenkorbs.
  - **Theke (B):** verliehenes, als defekt gemeldetes Gerät nicht rückgebbar; Sperr-Dialog hängt am
    Fehlertext (Schadens-Sperre ohne Übergehen-Dialog); Geräte-Ausleihe übernimmt
    `active_teacher_id` ungeprüft; Offline-Warteschlange nur für `B-`-Barcodes.
  - **Schüler/Frontend (B):** Theke hält nach dem Zusammenführen die gelöschte Quell-ID;
    Bearbeiten-Formular schickt das alte `abgaenger_jahr` mit (Klassenwechsel rechnet nie
    neu); Foto per Barcode ohne `deleted_at`; Purge-Fehler immer 409.

    Erledigt am 12.09.2026: die beiden Suchen und die vier Listen samt erweitertem
    Fehlerausgang-Scanner (`05391fe8`, `dfc9913a`, `5882d0cd`), der Mail-Knopf ohne Recht
    (`143926df`), der Ausweis 12T (`59beb423`), die zurückgestellte 180-Tage-Uhr
    (`473bea94`) und die Wiederherstellung am Namensindex (`39b58be8`). Der
    Abgänger-Druck steht jetzt unten unter „Entscheidung nötig".

  - **LMF/Statistik (B):** Klasse zweimal im Plan — Ausleihe und Massenabgleich nennen
    zwischen den Terminen verschiedene Fristen; Statistik ohne Sequenznummer und ohne
    Fehlerzustand, Query-Fehler ergeben leere Listen ohne Logzeile.
  - **Gates/Betrieb (B):** Gegenrichtungs-Ratsche blind für UNIQUE/Teilindizes und
    RESTRICT-FKs, Schema-Parität vergleicht Funktionen nur am Namen; mindestens zehn
    Ratschen ohne Landkarten-Zeile (u. a. `docs/werkzeuge_im_image_test.go`,
    `docs/compose_variablen_test.go`, `frontend-hygiene-dialoge/-ladekreis/-schalter/-tabellen.test.js`;
    Regel 7 hat keine Ratsche); `api/search_debug_test.go`
    mit fester DSN auf die Entwicklungs-DB und Skip ohne Guard; kein Rückweg beim Wechsel
    des `BACKUP_ENCRYPTION_KEY`; Escape in einem offenen Select schließt den ganzen Dialog.

    Erledigt am 12.09.2026: `migrate-fotos` ist aus dem Repo und wird im Image gebaut
    (`357d28f2`) — beim Nachsehen fiel auf, dass docs/SCRIPTS.md seit jeher einen Aufruf
    nennt, den es im Container nie gab; ein Gate hält jetzt fest, dass jedes Werkzeug
    einer Anleitung auch im Image liegt. Dass das Release-Gate die vier Security-Jobs
    nicht verlangt, stimmt weiterhin — es ist eine Entscheidung (siehe unten), kein
    Versehen; die falsche Begründung stand in DEPLOYMENT §8 und ist berichtigt.

  - **Kleinkram (C):** `resilience_and_recovery.md` nennt `bibliothek-db` als Dienst;
    `scripts/backup.sh` exportiert die ganze `.env`; tote CSS-Klassen in `altlasten.css`;
    zwei Regexe für die LMF-Kennung. (DEPLOYMENT §8/§2.3 und die toten
    Compose-Variablen: erledigt am 12.09.2026, `e5d8e21a` — je mit Ratsche.)

- **Rasterdurchgang 12.09.2026 über die Änderungen vom 11.09.** (Peter: „wir haben die
  ganzen Schemata komplett drüberlaufen lassen — überprüfe alles sorgfältig und fahre
  gegebenenfalls fort"). Anlass ist dieselbe Lücke wie am 05.09.: Der Bestands-Durchgang
  vom 10.09. endete bei `f67078f3`, danach kamen **sieben Commits** (`4fd4bc42` …
  `214a5ecd`, sechs davon mit Code, 29 Dateien) — und die waren nie gerastert. Ein
  Durchgang „über alles" altert ab dem nächsten Commit. Zwölf Fragen über die sechs
  Code-Commits, dazu die Zwillingsfrage je Fix. Drei Funde, alle behoben und am alten Code
  rot gesehen:

  - **Fremdrückgabe an der Sperre (`395551f9`).** Der Fix vom 11.09. hängte die
    Sperrprüfung an `!isReturningThis` — der dritte Fall fiel damit auf die falsche Seite.
    Wer gesperrt oder am Ausleihlimit war, konnte das Buch eines Mitschülers nicht abgeben,
    obwohl dabei für ihn keine Ausleihe entsteht; ohne offene Sitzung ging dieselbe
    Rückgabe durch. Neue Klasse in `sweeps.md`. Der Umbau hätte fast die
    Reservierungs-Schranke des Handapparats mitgenommen (zweiter Aufrufer derselben
    Prüfung) — gehalten hat sie `TestHandleLehrerHandapparat_SchranktEin`.
  - **Boot-Restore gegen den Datenbank-Aussetzer (`8750b897`).** Der Fix vom 11.09. gab dem
    Aussetzer den eigenen Statuscode 503, damit kein Arbeitsplatz abgemeldet wird — gelernt
    hatten ihn aber nur die Leser im laufenden Betrieb. Der Boot-Restore las weiter bloß
    `res.ok`: Wer währenddessen neu lud, stand trotz gültigem Cookie am Login.
    **Prüfmuster:** Wer einen Statuscode neu einführt, zählt seine Leser — jede Stelle, die
    `res.ok` oder `status ===` auswertet, nicht nur die, die den Fehler gemeldet hat.
  - **Zahl der geprüften Bereiche (`a17fc517`).** „Geprüft werden fünfzehn Bereiche" stand
    im FACHKONZEPT, während `Pruefe` sechzehn lieferte; aufgefallen ist es nur, weil
    `387a9949` den siebzehnten baute. Jetzt eine Ratsche, in drei Richtungen rot gesehen.

  Offen aus demselben Durchgang:

  - Erledigt am 12.09.2026: Der 503 der Sitzungsprüfung nennt nur noch den Satz, nicht die
    gewrappte Ursache (`2c8fe70d`, neue Funktion `apierrors.SendHTTPErrorMitMeldung`), und
    das Abmelden meldet, was wirklich widerrufen wurde — 503 statt `{"status":"ok"}`, wenn
    die Sperrliste nicht antwortet (`039145f2`).
  - **Beobachtung, kein Fund (C):** Die Sperrprüfung liest seit dem 11.09. aus dem Pool,
    während die Transaktion des Checkouts offen ist (drei Abfragen über eine zweite
    Verbindung, `FOR UPDATE` gehalten). Bei `MaxConns = 50` und einer Handvoll
    Arbeitsplätzen ohne Wirkung; die Form — Pool-Abfrage in offener Transaktion — ist die,
    die bei kleinem Pool in den Stillstand läuft.

- **Prüflauf 13.09.2026 über Issues, Register und Dokumentation** (Peter: „überprüfe alles
  genau … auch die Dokumentation"). Fünf Prüfer, jede Aussage mit Fundstelle, Stichproben am
  Code nachgesehen. Behoben: die doppelte ISBN wurde nie als Dublette erkannt (Constraint
  hieß noch `books_isbn_key`, `260b1436`), die Ausnahme für `ziel_jahrgang` in
  `inventur/schema_paritaet_test.go` nannte einen Schreiber, den es nie gab, und zehn
  Commit-Nummern in dieser Datei waren im letzten Zeichen falsch. Die Doku-Korrekturen
  stehen in eigenen Commits. Offen aus demselben Lauf:

  - Erledigt am 13.09.2026: Das API-Inventar liest keine Testdateien mehr (`ed4c6c10`), die
    Swagger-Spezifikation nennt keinen festen Host (`e502c3c6`), und `security-scan.sh` erreicht
    den lokalen Stack. Beim Nachsehen fiel mehr auf als der Port: Das Token ging als Bearer mit,
    die Anwendung liest aber nur das Cookie (jeder Lauf war unangemeldet), und ein Lauf ohne
    importierte URL meldete „Scan komplett" (`61ea1357`, am Stack gelaufen).
  - **UUID aus Query oder Body ungeprüft an die Datenbank → 500 (B).** Aus dem ZAP-Lauf vom
    13.09.2026, am Stack nachgestellt: `GET /api/inventur/fehlbestand?session_id=x`,
    `GET /api/vormerkungen?titel_id=x` bzw. `?schueler_id=x`, `POST /api/vormerkungen` mit
    `{"titel_id":"x"}`, `POST /api/inventur/finish` und `/abort` mit `{"session_id":"x"}` — alle
    500 statt 400. `ValidateUUIDParamsMiddleware` prüft nur Pfad-Parameter. Gemessen: 82
    `string`-Felder auf `…id` in Request-Structs (`api/`, `inventur/`), keines mit `uuid`-Tag.
    Nicht pauschal taggen: `barcode_id` (16) und `lusd_id` (4) sind keine UUIDs — dieselbe
    Verwechslung, vor der der Kommentar an `uuidPfadParameter` warnt. Eigener Durchgang mit
    namentlicher Liste und Ratsche.
  - **Beobachtung (C):** `startGDPRWorker` (`main.go`) ruft beim Start und im eigenen
    24-h-Takt nur die Leihen-Anonymisierung und die Abgänger-Löschung; `RunGDPRAnonymizeOldData`
    läuft allein im Cron (`jobs/cron.go`), täglich. Keine Wirkung erkennbar, die Doku
    beschreibt jetzt beide Wege.
  - **Kommentar (C):** `migrations/110_schadensersatz_bescheide.sql` nennt drei
    Barzahlungs-Briefe, es sind zwei. Eingespielte Migrationen werden nicht nachträglich
    geändert.

## Offen — Entscheidung nötig (Peter)

Was einem Menschen zur Entscheidung vorgelegt wird, gehört HIER hin, bevor die Antwort
kommt — ein Vorschlag, der nur im Gespräch steht, überlebt die Sitzung nicht.

| Fund                                                                                                                                                                                                                                                                                                                                                                                                                                                                         | Frage                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Landesmittel/Kreismittel: Schadensersatz (Teil A) + Beschaffung (Teil B)** — Konzept in [mittel_konzept.md](mittel_konzept.md) (09.09.2026). Teil A: Etappe 1 gebaut, aus Etappe 2 fehlen nur die Folgen der Übergabe (E6); Etappen 3–4 (Kreis-Rechnung, Altbriefe) offen — Stand in #597/#598. Die zwei Altbriefe (`pdf/schadensfall.go`, `pdf/rechnung.go`) verlangen weiter Barzahlung „in der Bibliothek"; im Schadensdialog ist der Betrag Freitext mit Vorgabe 15 €. | **E1** Nummern für die Referenznummer (Sekretariat) · **E2** aktuelles Musterschreiben anfordern (die vorliegenden Unterlagen sind von 2014 und nennen eine inzwischen aufgelöste Stelle) · **E3** Frist 28 Tage — umgesetzt (Einstellung, Vorgabe 28) · **E4** Feld „Listenpreis" am Titel für die Staffel ab dem 2. Verleihjahr · **E5** Zahlungsweg der Schülerbücherei mit dem Schulträger klären · **E6** nach Übergabe: Sperre bleibt, Löschblockade fällt? · **E7** kein E-Mail-/App-Versand · **E8** keine Eltern-Namen/zweite Anschrift · **Teil B Beschaffung — erster Schnitt GEBAUT 10.09.2026** (Migration 109: eine Bestellung = ein Topf, gemischter Warenkorb → zwei Bestellungen, Vermerk auf Anschreiben und Mail, zweite Kundennummer, Backfill, Lernmittel-Frage im Staging, Topf nachträglich korrigierbar; D1/D3/D5 von der EDV-Servicestelle für Schulbibliotheken beantwortet, siehe Konzept 7.4). **Zweiter Schnitt GEBAUT 12.09.2026** (`14e6a56d`, `9f0fca8f`): Berichte in Blöcken je Topf mit eigener Summe, Topf-Filter in der Historie, Kennzahlen je Topf. |

| **LMF-Frist am Rückgabetermin** (Bestands-Durchgang 10.09.2026). Wer am Termintag der Klasse neue Schulbücher bekommt, erhält den Termin selbst als Frist (heute 23:59, `RueckgabeTerminFuerKlasse` sucht `>= heute`); am Tag danach gilt der Stichtag 31.07. — in den Ferien. Nach den Ferien wäre die ganze Klasse überfällig und nach 14 Tagen gesperrt. `ziel_jahrgang` (mehrjährige Ausleihe) hat keinen Schreiber. | Welche Frist gilt für ein Lernmittel, das am oder nach dem Rückgabetermin der Klasse ausgegeben wird — der Stichtag des NÄCHSTEN Schuljahres, oder der Termin des nächsten Plans? |
| **Karenz gegen Lesehistorie** (10.09.2026). Die Uhr vor der Anonymisierung rechnet ab der letzten Rückgabe — über `ausleihen.schueler_id`, die der Lesehistorie-Lauf nach `lesehistorie_tage` trennt. Ist die Karenz länger eingestellt als die Lesehistorie, wird früher anonymisiert als eingestellt. Mit den Vorgaben (90/90) ohne Wirkung. Für Lernmittel gilt eine eigene Lesehistorie-Frist (`lesehistorie_lernmittel_tage`, Vorgabe 730). | Soll die Einstellung Karenz ≤ Lesehistorie erzwingen, oder soll die Uhr ihren Zeitpunkt selbst speichern (eigene Spalte)? |
| **Topf auf Bestätigungsseite und großen Etiketten** (10.09.2026). Der Händler bekommt am selben Tag zwei gleich aussehende Links; auch für eine Schülerbücherei-Bestellung werden die großen Lernmittel-Etiketten mit „Eigentum des Landes" angeboten und angehängt. | Nennt die Bestätigungsseite den Topf? Große Etiketten nur bei `land`? |
| **Security-Jobs im Release-Gate** (10.09.2026). Die Pflichtliste enthält keinen der vier Security-Jobs (govulncheck, gosec, npm audit, Trivy-Image); ein Tag auf einen Commit mit rotem Trivy erzeugt trotzdem Release und Image. | Gehören sie in die Pflichtliste, oder bleibt es bewusst so (dann begründet festhalten)? |
| **Abgänger-Druck und die Suche** (10.09.2026, beim Abarbeiten am 12.09. vorgelegt). Zwei Kommentare sichern zu „Was auf dem Bildschirm steht, steht auf dem Papier" — der Druck folgt aber nur dem Klassenfilter. `GET /api/abgaenger/pdf` kennt nur `klasse`; die Suche filtert im Browser über Vorname, Nachname, Klasse und Barcode. | Soll der Druck der Suche folgen (dann bekommt der Endpunkt einen `suche`-Parameter, und dieselbe Auswahl steht zweimal — im Browser und in SQL), oder bleibt er klassenweise (dann fallen die zwei Kommentare)? Ohne Antwort wird hier nichts gebaut: Beides ist vertretbar, und es geht um das, was aus dem Drucker kommt. |
| **Portal ohne Cover: `AnliegenWidget`** (Paket 5, #593). Die eigenen Wünsche und Meldungen im Kollegiums-Portal zeigen kein Buchcover. Tabelle und API tragen `isbn` und `titel_id` am Anliegen schon (seit 18.08.2026, `0ddb0fc0`); nur das Wunsch-Formular im Widget schickt sie nicht mit. Am 05.09.2026 abgelehnt wurde `cover_url` für die Arbeitslisten der Bibliothek („Arbeitslisten bleiben ohne Cover"). Das Portal ist aber keine Arbeitsliste der Bibliothek. | Gilt die Entscheidung vom 05.09. auch für die eigene Liste der Lehrkraft — oder bekommt das Wunsch-Formular eine Buchauswahl, die `isbn`/`titel_id` mitschickt, damit dort ein Cover stehen kann? |

## Beobachten (nichts zu tun)

| Fund                                                                                                                                                                              | Warum nur beobachten                                                                                                                                                                                                                                                                                                                                                             |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pg_dump`/`psql` werden im Backup und in der Restore-Probe per Namen über `PATH` aufgelöst (`jobs/backup.go`, `jobs/restore_probe_hilfen.go`; Sicherheits-Audit 07.09.2026, Info) | Argumente kommen aus der Server-DSN als getrennte argv-Elemente, keine Shell — keine Injection. Im Container ist `PATH` fest; absolute Pfade wären robuster, lohnen aber keinen Umbau ohne Anlass.                                                                                                                                                                               |
| `golang.org/x/crypto/openpgp` gilt als unwartbar (`GO-2026-5932`)                                                                                                                 | Kein Fix verfügbar (`Fixed in: N/A`), transitiv, **kein** Aufrufer im eigenen Code (`govulncheck`, zuletzt 31.08. bestätigt). Dependabot meldet sich, falls sich das ändert.                                                                                                                                                                                                     |
| Designer-Restlücke: Browser-Schließen binnen 800 ms verliert die letzte Auto-Save-Änderung                                                                                        | `onDestroy` schickt seit `37abcbe0` sofort; `sendBeacon` bewusst nicht gebaut (Randlage).                                                                                                                                                                                                                                                                                        |
| 082-Dedupe der Vormerkungen verlor den neueren `abholbereit`-Eintrag                                                                                                              | Auf Prod gelaufen, nicht rückholbar; nur relevant, falls je eine weitere **gewachsene** DB migriert wird.                                                                                                                                                                                                                                                                        |
| Rate-Limiter-Maps (`api/rate_limit.go`, `api/middleware_ratelimit.go`) räumen erst ab 5.000 Einträgen und dann nur Stale > 5 min; jeder neue Eintrag iteriert die ganze Map       | Sicherheits-Audit 05.09.: Nur mit vielen frischen Adressen (IPv6-Rotation) ein CPU-Thema; hinter Caddy trifft es die Schule nicht von innen. Ohne Anlass nicht anfassen.                                                                                                                                                                                                         |
| ZAP-Scan 05.09. (localhost:8084, unangemeldet, 17 GET-Endpunkte): `style-src 'unsafe-inline'`, `csrf_token` ohne HttpOnly, „Suspicious Comments"                                  | Alle drei sind dokumentierte Entscheidungen (SECURITY.md: CSP-Begründung; Double-Submit-Cookie muss JS-lesbar sein; der Kommentar ist `//scanapp.org` aus der QR-Bibliothek). HSTS/Cache-Meldungen betrafen `content-autofill.googleapis.com` (Chrome), nicht die Anwendung.                                                                                                     |
| ZAP-API-Scan 13.09. (localhost:8084, angemeldet, 74 URLs): 0 FAIL, 7 WARN, darunter 44× „SQL Injection" (40018)                                                                   | Am Stack nachgestellt: `bis` wird bei jeder Injektion mit 400 abgewiesen (Datumsformat); bei `q` liefern Wahr- und Falsch-Paar dieselbe Antwort (0/0) — ZAP verglich den Suchbegriff „q" (12 Treffer) mit einem anderen Suchbegriff. Viele Anfragen liefen in 429 (lokal `RATE_LIMIT=500`), der Scan ist also gedrosselt. Der einzige echte Fund daraus steht oben (UUID → 500). |

## Kategorie C — bewusst nicht ohne Anlass

Einzeiler; die ausführlichen Begründungen stehen in der Git-Historie dieser Datei.

| Posten                                                                                                                                                                                                             | Warum geparkt                                                                                                                                                                                                                                                                                                    |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Knopfzeile über den Reitern (Mahnwesen, 04.09.)                                                                                                                                                                    | Sie kommt aus dem gemeinsamen Seitengerüst. Daran zu drehen entscheidet über ALLE Seiten — Anlass wäre ein Rundgang über das Gerüst, nicht eine Seite.                                                                                                                                                           |
| 3 handgebaute Pillen-Gruppen in `StatsDashboard` (`pills`)                                                                                                                                                         | Zweite Fassung von `ui/Segmente.svelte`. Tauschen beim nächsten fachlichen Anfassen der Statistik.                                                                                                                                                                                                               |
| Etikettenraster doppelt (`api/label_formats.go` ↔ `src/lib/etikettformate.js`)                                                                                                                                     | 31.08. ENTSCHIEDEN geparkt: Server-Umbau bräuchte neuen Endpunkt + async-Ladezustand in drei Verbrauchern des täglichen Druckbildschirms — neuer Ausfallmodus gegen null Risiko heute. `etikettformate-konsistenz.test.js` hält die Drift. Wieder anfassen bei viertem Verbraucher oder konfigurierbarem Format. |
| Reste des Nie-verdrahtet-Sweeps (01.09.)                                                                                                                                                                           | `inventur_sessions.gestartet_von` nie angezeigt (Produktfrage); `abgaenger_jahr` in der Aktivlisten-Antwort ohne Konsument; Geräte-Torso (`ActionEvent.GeraetID`, kein Broadcast, Null-Zeitstempel) wartet auf den Geräte-Ausbau.                                                                                |
| Cognitive Complexity: `gocognit -over 15` ohne Tests = 32 Funktionen (Messung 05.09.2026)                                                                                                                          | Zählweise rechnet den Handler-Closure als Ebene. Lohnend allenfalls `OverrideDueDateHandler` (30), `behandleAbgaenger` (25).                                                                                                                                                                                     |
| `javascript:S6551`, `javascript:S8783`                                                                                                                                                                             | Begründete Dauer-Ausnahmen.                                                                                                                                                                                                                                                                                      |
| Tabellen-Inline-Felder (Rückgabedatum, Exemplar-Barcode) 36 px                                                                                                                                                     | `size="sm"`-Variante von `Feld` erst bei Bedienbefund.                                                                                                                                                                                                                                                           |
| `LabelHeight >= 30` steht 2× in zwei Dateien (`api/label_pdf.go`, `api/schueler_etikett_pdf.go`; Messung 05.09.2026)                                                                                               | Schwellwert-Doppelung ohne Wirkung.                                                                                                                                                                                                                                                                              |
| Zwei Go-Normalformen für Namen: `repository.Suchnorm` (Zwilling von SQL `suchnorm`, LUSD-Schlüssel) und `normName` in `api/lusd_paarung.go` (eigene Umlaut-Tabelle, zusätzlich ohne Bindestrich/Leerzeichen/Punkt) | Die Paarung ist ein Vorschlag mit menschlicher Bestätigung, Unschärfe dort harmlos. Beim nächsten Anfassen der Paarung: `normName` auf `Suchnorm` aufsetzen und nur die Zeichen-Tilgung behalten.                                                                                                                |
| Paritätstest vergleicht keine COMMENTs/Seeds                                                                                                                                                                       | Kosmetik; Struktur ist gedeckt.                                                                                                                                                                                                                                                                                  |
| Jules-Erbe                                                                                                                                                                                                         | Go-Testdateien > 200 Zeilen aus Jules-PRs; Export-CSV-„breaks stream"-Test schwach.                                                                                                                                                                                                                              |
| Klone: Go `dupl -t 100` ohne Tests = 9 Gruppen (05.09.2026); Frontend `jscpd` 0,41 % (ältere Messung, Werkzeug lokal nicht installiert)                                                                            | Unter der Schwelle.                                                                                                                                                                                                                                                                                              |

## Außerhalb dieses Registers (Betrieb, liegt bei Peter)

Die EINE Liste der offenen Betriebs-Punkte. Littera-Details in
[littera_schema_befund.md](littera_schema_befund.md).

- **Frisches Littera-Backup** — littera_sav.mdb ist ein 2010er-Stand; Anforderungen
  (FremdLeserNummer/FremdBarcode) in [littera_schema_befund.md](littera_schema_befund.md).
- **Drei Sekretariats-Abnahmen** à ~10 Minuten (LUSD-Import, Versetzung ⏰ vor dem
  Schuljahreswechsel, Klassensatz-Erledigen) — Ablauf in
  [abnahme_checkliste.md](abnahme_checkliste.md).
- **Zielumgebung** (die Seite System → Betriebsbereitschaft zeigt den Ist-Zustand):
  Prod-Secrets mit `ENFORCE_PROD_SECRETS=true` (ohne `BACKUP_ENCRYPTION_KEY` läuft
  KEIN Backup) · Schul-IMAP/SMTP-Zugangsdaten · **`SELBSTANMELDUNG_DOMAIN`** setzen ·
  einmalige manuelle Restore-Probe am echten Ziel ([DEPLOYMENT.md](DEPLOYMENT.md) §6) ·
  `SENTRY_DSN` leer lassen (A6 in
  [datenschutz_offene_punkte.md](datenschutz_offene_punkte.md)) ·
  **S3-Auslagerung der Backups**.
  Lesend nachgesehen am 13.09.2026 auf dem heutigen Server (Hetzner, derzeit einzige
  Instanz, Stand 223b98f7): Prod-Secrets gesetzt, `ENFORCE_PROD_SECRETS=true`,
  `BACKUP_ENCRYPTION_KEY` gesetzt (nächtliche Backups laufen, Restore-Probe vom 13.09.
  erfolgreich), IMAP gesetzt, SMTP in der Datenbank eingerichtet, `SELBSTANMELDUNG_DOMAIN`
  gesetzt, `SENTRY_DSN` leer. **Offen bleiben dort S3 und die manuelle Restore-Probe am
  fremden Ziel.** Noch im Bestand: 2.000 `DEMO-S-*` und 2.500 `DEMO-B-*`; 30.676 echte
  Exemplare zählen als „Etikett offen".
- **GitHub**: PR-Pflicht abschaffen (Solo-Entscheidung 30.07.; das Ruleset `main` trägt am
  05.09. noch die `pull_request`-Regel), „Block force pushes" und „Restrict deletions"
  anlassen.
- Datenwert Schulname/„Neuer Text" auf Live korrigieren — am 13.09.2026 lesend nachgesehen:
  `schule_name`, `schule_strasse`, `schule_plz`, `schule_ort` sind befüllt, keiner enthält
  „Neuer Text". Wenn nichts anderes gemeint war, ist der Posten erledigt.
- **Restore-Probe 2e einmal mit dem neuen Weg** (`docs/resilience_and_recovery.md`): Die
  alte Anleitung griff die Deploy- statt der Nachtsicherung (e490960e) — ob die neuen
  Befehle am Server so laufen, ist nur am Text geprüft.
- **Vier Umgebungsvariablen, die Compose nicht durchreicht** (gemessen 08.09.2026):
  `ALLOWED_ORIGIN`, `RATE_LIMIT`, `SENTRY_DSN`, `IMAP_PORT` werden von Go gelesen, stehen
  aber in keinem `environment:`-Block von `docker-compose.yml` — im Container gilt also
  immer der eingebaute Vorgabewert, auch wenn die `.env` etwas anderes sagt. Dieselbe
  Klasse wie der zweimal aufgetretene `BACKUP_ENCRYPTION_KEY`-Fall. Je Variable zu klären,
  ob die Vorgabe gewollt ist; sonst durchreichen. Die übrigen neun ungelisteten (`PATH`,
  `TEST_DATABASE_URL`, `PG_DSN`, `MYSQL_DSN`, `BE_CRASHER`, `LITTERA_CSV_DIR`,
  `FOTOS_BEHALTEN`, `SMTP_ALLOW_INSECURE_TLS`, `SMTP_ALLOW_PLAINTEXT`) gehören Werkzeugen
  außerhalb des Servers und sind richtig so. Ein Gate dagegen gibt es nicht — es bräuchte
  diese Ausnahmeliste. `docs/compose_variablen_test.go` prüft nur die Gegenrichtung
  (Compose reicht etwas durch, das Go nicht liest). Auf dem Server am 13.09.2026 nachgesehen:
  `ALLOWED_ORIGIN`, `RATE_LIMIT`, `IMAP_PORT` sind im Container leer, es gelten die Vorgaben.

**Parkdeck** (bewusste Nicht-Entscheidungen, nur mit Anlass wieder anfassen):
Integer-Cent-Refactor (float64/NUMERIC) · Bundle-Splitting (720-kB-Chunk) ·
TypeScript-Migration (null TS-Dateien) · Verschmelzung `inventur/` ins Haupt-API ·
`cmd/migrate` (MySQL) löschen — hat keine Datenquelle mehr, aber seine PG-Tests
sichern mit `internal/uebernahme` geteilten Code · Zukunftsideen API-Versionierung
(`/api/v1`) und Mandantenfähigkeit (RLS).

Stand: 2026-09-13
