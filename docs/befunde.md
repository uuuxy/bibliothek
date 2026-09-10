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

**Reihenfolge:** steht seit 09.09.2026 auf GitHub — angeheftetes Issue [#594](https://github.com/uuuxy/bibliothek/issues/594) (Pakete 1–5 = #595, #596, #597, #598, #593; Betrieb #599, Schule #600; Milestones v2.5.0 → v2.6.0 → v2.7.0). Alles Weitere beim nächsten fachlichen Anfassen.

---

## Offen — abarbeitbar

- **Trennlinien in Tabellen (C, Design-Frage).** M3 Lists: „Limit dividers to
  uncontained or complex lists, only when a stronger visual separation is necessary."
  Der LMF-Planer kommt seit 06.09.2026 ohne Zeilen-Trennlinie aus (48-px-Zeilen,
  Hover-Fläche); 44 andere Dateien tragen `divide-y`, meist `divide-slate-100`. Ob die
  ganze Anwendung nachzieht, ist eine Gestaltungsentscheidung mit sichtbaren Folgen —
  nicht nebenbei, sondern als eigener Durchgang mit Messung im Browser.

- **Rasterdurchgang 05.09.2026 abends über die Änderungen vom 04./05.09.** (Peter: „lass
  alle Schemata nochmal über die heutigen und gestrigen Änderungen laufen"). Elf Fragen
  plus Sweeps-Achse über 247 geänderte Dateien; vier Funde, alle im selben Durchgang
  behoben und rot bewiesen — hier bleibt nur, was offen ist:

  Nachgetragen am selben Abend (Peters Frage „die Schemata hast du laufen lassen,
  richtig?"): Der Durchgang vom 04.09. lief um 22:17, **acht Code-Commits kamen danach**
  und waren nie gerastert — darunter die beiden Refactorings 1359408b (Parameterlisten →
  Strukturen) und d267e708 (doppelte Abfrage zusammengelegt). Beide jetzt geprüft: Die
  zusammengelegte Sperr-Abfrage und die drei herausgezogenen Helfer sind
  verhaltensgleich (`persistBooksFallback` startet mit `imported = 0`, und das war es
  vorher auch); kein Literal der zehn neuen Parameter-Strukturen lässt ein Feld offen.
  Die Bugklasse hat jetzt eine Ratsche (`parameter_strukturen_test.go`, sweeps.md).

  - **Beobachtung, kein Fund (C):** Der Wächter „Ehemalige mit offene Vorgängen"
    (`ZaehleEhemaligeMitOffenenVorgaengen`) verlangt `abgaenger_seit IS NOT NULL`, das
    Löschprädikat rechnet mit `COALESCE(abgaenger_seit, aktualisiert_am)`. Zwei
    Formulierungen derselben Frage „wie lange ist der weg?" — heute ohne Wirkung, weil
    Migration 094 die Spalte nachgetragen hat und alle drei Schreiber (LUSD, Versetzung,
    Zusammenführen) sie stempeln; nachgezählt am Code. Beim nächsten Anfassen des
    Wächters angleichen.

- **Rasterdurchgang 06.09.2026 über die Änderungen desselben Tages** (Peter: „lass bitte
  die Schemata komplett über die heutigen Änderungen laufen"). Elf Fragen plus
  Sweeps-Achse über 97 geänderte Dateien (+4525/−985), sechs Prüfer. Vierzehn Funde, alle
  im selben Durchgang behoben und je am Rückbau rot gesehen (06116d38 bis 4ef052f1) —
  darunter ein rotes `main` seit 14:40 (der Postgres-Test kannte die zwei Schlusszeilen
  „Nachzügler"/„Aufräumen" nicht, und der Pre-Push-Hook lässt die `*_pg_test.go` aus).
  Hier bleibt nur, was offen ist:

  - **`ohne_rueckgabe_termin` geht ans Kollegium, das es nie zeigt (C).**
    `GET /api/lmf-termine` füllt das Feld für jeden Aufrufer; `PortalLmfPlan.svelte` liest
    es nicht. Klassennamen, kein Schülerbezug — Über-Auslieferung ohne Schaden.
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
  - **Vier Zusicherungen halten nur per Verabredung (C).** `lmf_termine.art ≡
lmf_plaene.art`, die Eindeutigkeit von `position`, `letzte_stunde ≤ stunden_je_tag`
    und `len(plaetze) == len(zeilen)` stehen in Go bzw. im Kommentar, nicht in der
    Datenbank. Heute unerreichbar (ein Schreiber), am frischen Postgres nachgemessen.
  - **Kleinkram (C):** `leererEntwurf()` startstunde 1 gegen Server-Vorgabe 2 (heute
    verdeckt, weil der Vorschlag immer gewinnt); Vorschau-Antwort liefert `klassen: null`,
    Speicher- und Leseantwort `[]`; `LmfPlan.test.js` wartet 400 ms gegen 250 ms
    Entprellung; die drei Browser-Gates öffnen die neuen Planer-Dialoge nicht.

- **Bestands-Durchgang 06.09.2026 abends** (Peter: „löse alles professionell … belege und
  überprüfe wirklich alles am Code"). Anlass ist eine Frage, die eine Antwort verdient hat:
  Warum kommen jetzt MEHR Funde, wo die Ausbeute doch fallen sollte? Am Code nachgesehen:
  Die elf Fragen laufen seit dem 22.08. über **Änderungen** — `docs/sweeps.md` sagt das im
  eigenen Zweck-Absatz („Es sieht nicht, was schon da ist"). Über den ganzen Baum liefen sie
  genau **einmal**, am 23.08., und damals mit zehn Fragen: Frage 11 (geteilter Zustand/Lader)
  kam am 24.08. dazu — und lieferte heute den gefährlichsten Fund. Dazu kommt, dass ein
  Durchgang über einen Diff die NACHBARSCHAFT nicht sieht: Der DSGVO-Fund entstand am 02.09.
  in einem Commit über 55 Dateien, dessen Raster an dem Tag über das Thema der Änderung lief
  (LUSD-Umbenennung), nicht über jede mitgeänderte Datei. Der Bestand ist also keine
  ausgeschöpfte Fläche, sondern eine nie vollständig befragte.

  Erledigt in diesem Durchgang (je ein Commit, je am Rückbau rot gesehen): die sieben Funde
  des Nachmittags (`8c5b354f` bis `3abe0bd7`), die Ratsche über die Routen ohne Fachrecht
  (`3e98f78b`), der Stichtags-Zwilling (`de4484a9`) und die Papierkorb-Tür (`14be1528`).

  - **Sweep „verschluckte Fehlantwort" — gelaufen und abgeschlossen** (`2d8e2d2d`). Aus dem
    Kandidaten wurde ein Sweep mit AST-Detektor: 26 Fundstellen (der Grep hatte 65
    vermutet), neun behoben, siebzehn mit Begründung eingefroren, Ratsche
    `frontend/src/lib/fehlerausgang.test.js` in beide Richtungen rot gesehen. Zwei der
    Funde waren Kategorie A und haben eigene Commits: die Theke zeigte unter dem neuen
    Suchtext die Treffer des alten (`22173001`), und der Ausweis-Designer schrieb nach
    einem fehlgeschlagenen Laden das Vorgabe-Design an alle Arbeitsplätze (`c2be1069`).
    Die Reste der Klasse — Buch-Akte (`0a59a9e9`) und Rechnung (`ce875654`) — ebenfalls
    erledigt.

    **Ein Nachtrag, der zur Vorsicht mahnt:** Den Rechnungs-Posten hatte ich hier
    ausdrücklich als „heute nicht erreichbar" abgelegt. Beim Bauen des Gates zeigte die
    Datenbank etwas, das ich beim Lesen des Go-Codes nicht gesehen hatte —
    `CONSTRAINT check_damage_item` verlangt GENAU EINES von `exemplar_id` und `geraet_id`.
    Der Geräteschaden ist im Schema also vorgesehen, ihm fehlt nur der Schreiber, und die
    INNER JOINs hätten ihn aus der Rechnung an die Eltern fallen lassen. Merksatz: Die
    Reichweite einer Zusicherung steht im SCHEMA, nicht im Aufrufer.

- **Buchcover: Rest nach dem Bestellbedarf** (04.09.2026). Der Bestellbedarf zeigt das
  Cover in der Zeile (`ui/BuchCover.svelte`); auf dem Zielsystem tragen 5.724 von 8.706
  Titeln ohne Exemplar ein Cover — die Anzeige lohnt. Offen:

  - **16 Bestandsstellen** bauen ihr Cover noch selbst (Liste in
    `frontend-hygiene-cover.test.js`, eingefroren). Umstellen beim nächsten fachlichen
    Anfassen — nicht in einem Rutsch, das sind täglich benutzte Bildschirme.
  - **Arbeitslisten bleiben ohne Cover** (entschieden 05.09.2026): Das führende Element der
    `ArbeitsZeile` ist der Klassenkreis, nach dem Klassensatz-Reservierungen und Wünsche &
    Meldungen überflogen werden — M3 kennt genau eines. Die zwei fehlenden Backend-Felder
    (`isbn` in `GetKlassensatzReservierungen`, `cover_url` in `anliegen.go`) werden deshalb
    nicht gebaut.
  - **Portal** ohne Cover: `AnliegenWidget`, `PortalSchulbuecher`, `PortalLernmittel`.
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

  - **Rest-A, bewusst nicht im Durchgang:** Geisterbuch-Fall bei der Ausleihe
    (`entferneErfuellteVormerkung`) — das reservierte Exemplar geht ins Regal zurück,
    statt den Nächsten zu bedienen; braucht einen Rot-Test über den Checkout.
  - **Sicherheit/Anmeldung (B):** DB-Aussetzer im Token-Pfad ergibt 401 statt 5xx — alle
    Arbeitsplätze melden sich ab, SECURITY.md verspricht 500; die Ausnahmen in
    `fehler_kollaps_test.go` begründen sich mit „kein DB-Zugriff", `claimsAusRequest`/
    `MeHandler`/`Refresh` lesen aber die DB (Klasse „verrottete Ausnahme-Begründung").
    Selbstanmeldung, zweiter Versuch: englisches „user account is deactivated" statt
    „Zugang beantragt". `MarkCopyDefekt` trägt ohne Schüler den klickenden BEARBEITER als
    Verantwortlichen ein, zwei Leser nennen ihn „Schuldner"; `DeleteUser` prüft offene
    Schäden nicht.
  - **Bescheid (B):** Verleihjahr zählt Ausleihen und Kalenderjahre statt Schuljahre;
    Kassenjahr = Jahr der Frist (Dezember-Brief zählt ins Folgejahr); Frist serverseitig
    unbegrenzt (Vergangenheit = sofort übergabefähig); Nachdruck liest Bank/Aufsicht live
    aus den Einstellungen; Rechnung und Elternbrief führen Bescheid-Positionen weiter mit
    „bar in der Bibliothek" (zwei Zahlungswege für eine Forderung, Etappe 4); Einstieg nur
    über überfällige Ausleihen — wer „nur" eine Forderung hat, bekommt keinen Bescheid;
    `aussonderung_grund` bleibt BESCHAEDIGUNG auch bei Verlust; `tabula_rasa.sql` leert
    `schadensersatz_nummern` nicht.
  - **Bestand/Katalog (B):** Massenlöschen `DELETE /api/books` hängt an `edit_books`,
    Einzellöschen an `delete_books`; Ausleiher-Reiter der Buchakte verschweigt
    Lehrer-Ausleihen (INNER JOIN); ISBN-Eindeutigkeit nur je Schreibweise (mit/ohne
    Bindestrich); `DeleteBooks` liest die Spuren vor der Transaktion; Titel-Etiketten
    drucken ausgesonderte Exemplare mit; Jahrgangs-CHECK erst nach Messung der Prod-Daten.
  - **Bestellwesen (B):** Wareneingang gruppiert nach Datum|Notiztext statt
    `bestellung_id` (zwei Töpfe am selben Tag = eine Gruppe; ohne Vorab-Barcode
    „Unbekannter Lieferant"; Datum ohne Schulzeitzone); Mail-Datum und Link-Frist in
    Serverzeit; Idempotenz-Schlüssel überlebt eine Änderung des Warenkorbs.
  - **Theke (B):** eigene Rückgabe in offener Sitzung scheitert an der Sperrprüfung;
    verliehenes, als defekt gemeldetes Gerät nicht rückgebbar; Sperr-Dialog hängt am
    Fehlertext (Schadens-Sperre ohne Übergehen-Dialog); Geräte-Ausleihe übernimmt
    `active_teacher_id` ungeprüft; Offline-Warteschlange nur für `B-`-Barcodes; doppelte
    Vormerkung 500 statt 409.
  - **Schüler/Frontend (B):** Suchen in Schülerdatei und Ehemaligen mit `res.ok && …`
    bzw. `res.ok ? … : []` — der Fehlerausgang-Scanner sieht beide Formen nicht
    (Fundstellen auch AnliegenListe, KollegiumPortal, KlassensatzReservierungen,
    GeraeteVerwaltung); Abgänger-Druck folgt der Suche nicht, obwohl zwei Kommentare es
    zusichern; Mail-Knopf an Klassenleitungen ohne `hatRecht('create_orders')`; Theke hält
    nach dem Zusammenführen die gelöschte Quell-ID; Bearbeiten-Formular schickt das alte
    `abgaenger_jahr` mit (Klassenwechsel rechnet nie neu); Foto per Barcode ohne
    `deleted_at`; Ausweis 12T gilt bis 12 statt 13; Purge-Fehler immer 409; DELETE im
    Papierkorb setzt die 180-Tage-Uhr zurück; Wiederherstellen kollidiert seit Migration
    108 am Namensindex mit 500.
  - **LMF/Statistik (B):** Klasse zweimal im Plan — Ausleihe und Massenabgleich nennen
    zwischen den Terminen verschiedene Fristen; Statistik ohne Sequenznummer und ohne
    Fehlerzustand, Query-Fehler ergeben leere Listen ohne Logzeile.
  - **Gates/Betrieb (B):** Gegenrichtungs-Ratsche blind für UNIQUE/Teilindizes und
    RESTRICT-FKs, Schema-Parität vergleicht Funktionen nur am Namen; Release-Gate ignoriert
    `security-scan.yml` (die Begründung „docker-scan gab es nie" stimmt nicht); zwölf
    Ratschen ohne Landkarten-Zeile (Regel 7 hat keine Ratsche); `api/search_debug_test.go`
    mit fester DSN auf die Entwicklungs-DB und Skip ohne Guard; kein Rückweg beim Wechsel
    des `BACKUP_ENCRYPTION_KEY`; `migrate-fotos` (14-MB-Binary) im öffentlichen Repo
    getrackt; Escape in einem offenen Select schließt den ganzen Dialog.
  - **Kleinkram (C):** DEPLOYMENT §8 beschreibt das alte Release-Gate, §2.3 `base64`
    gegen §2.1 `hex`; `resilience_and_recovery.md` nennt `bibliothek-db` als Dienst;
    `scripts/backup.sh` exportiert die ganze `.env`; tote Compose-Variablen `DB_HOST`,
    `SMTP_SENDER`; tote CSS-Klassen in `altlasten.css`; zwei Regexe für die LMF-Kennung.

## Offen — Entscheidung nötig (Peter)

Was einem Menschen zur Entscheidung vorgelegt wird, gehört HIER hin, bevor die Antwort
kommt — ein Vorschlag, der nur im Gespräch steht, überlebt die Sitzung nicht.

| Fund                                                                                                                                                                                                                                                                                                                         | Frage                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Landesmittel/Kreismittel: Schadensersatz (Teil A) + Beschaffung (Teil B)** — Konzept in [mittel_konzept.md](mittel_konzept.md) (09.09.2026). Teil A (Bescheide) nicht gebaut; heute verlangen drei Briefe Barzahlung „in der Bibliothek", kennen weder Referenznummer noch den Topf; Betrag ist Freitext mit Vorgabe 15 €. | **E1** Nummern für die Referenznummer (Sekretariat) · **E2** aktuelles Musterschreiben anfordern (die vorliegenden Unterlagen sind von 2014 und nennen eine inzwischen aufgelöste Stelle) · **E3** Frist 28 Tage statt 6 Wochen · **E4** Feld „Listenpreis" am Titel für die Staffel ab dem 2. Verleihjahr · **E5** Zahlungsweg der Schülerbücherei mit dem Schulträger klären · **E6** nach Übergabe: Sperre bleibt, Löschblockade fällt? · **E7** kein E-Mail-/App-Versand · **E8** keine Eltern-Namen/zweite Anschrift · **Teil B Beschaffung — erster Schnitt GEBAUT 10.09.2026** (Migration 109: eine Bestellung = ein Topf, gemischter Warenkorb → zwei Bestellungen, Vermerk auf Anschreiben und Mail, zweite Kundennummer, Backfill, Lernmittel-Frage im Staging, Topf nachträglich korrigierbar; D1/D3/D5 von der EDV-Servicestelle für Schulbibliotheken beantwortet, siehe Konzept 7.4). **Offen, zweiter Schnitt:** Berichte (Monat/Jahr/Lieferantenabrechnung) in zwei Blöcken Land/Kreis + Topf-Filter in der Historie (Konzept 7.3, Schritt 4). |

| **LMF-Frist am Rückgabetermin** (Bestands-Durchgang 10.09.2026). Wer am Termintag der Klasse neue Schulbücher bekommt, erhält den Termin selbst als Frist (heute 23:59, `RueckgabeTerminFuerKlasse` sucht `>= heute`); am Tag danach gilt der Stichtag 31.07. — in den Ferien. Nach den Ferien wäre die ganze Klasse überfällig und nach 14 Tagen gesperrt. `ziel_jahrgang` (mehrjährige Ausleihe) hat keinen Schreiber. | Welche Frist gilt für ein Lernmittel, das am oder nach dem Rückgabetermin der Klasse ausgegeben wird — der Stichtag des NÄCHSTEN Schuljahres, oder der Termin des nächsten Plans? |
| **Karenz gegen Lesehistorie** (10.09.2026). Die Uhr vor der Anonymisierung rechnet ab der letzten Rückgabe — über `ausleihen.schueler_id`, die der Lesehistorie-Lauf nach `lesehistorie_tage` trennt. Ist die Karenz länger eingestellt als die Lesehistorie, wird früher anonymisiert als eingestellt. Mit den Vorgaben (90/90) ohne Wirkung. | Soll die Einstellung Karenz ≤ Lesehistorie erzwingen, oder soll die Uhr ihren Zeitpunkt selbst speichern (eigene Spalte)? |
| **Topf auf Bestätigungsseite und großen Etiketten** (10.09.2026). Der Händler bekommt am selben Tag zwei gleich aussehende Links; auch für eine Schülerbücherei-Bestellung werden die großen Lernmittel-Etiketten mit „Eigentum des Landes" angeboten und angehängt. | Nennt die Bestätigungsseite den Topf? Große Etiketten nur bei `land`? |
| **Security-Jobs im Release-Gate** (10.09.2026). Die Pflichtliste enthält keinen der vier Security-Jobs (govulncheck, gosec, npm audit, Trivy-Image); ein Tag auf einen Commit mit rotem Trivy erzeugt trotzdem Release und Image. | Gehören sie in die Pflichtliste, oder bleibt es bewusst so (dann begründet festhalten)? |

## Beobachten (nichts zu tun)

| Fund                                                                                                                                                                              | Warum nur beobachten                                                                                                                                                                                                                                                         |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pg_dump`/`psql` werden im Backup und in der Restore-Probe per Namen über `PATH` aufgelöst (`jobs/backup.go`, `jobs/restore_probe_hilfen.go`; Sicherheits-Audit 07.09.2026, Info) | Argumente kommen aus der Server-DSN als getrennte argv-Elemente, keine Shell — keine Injection. Im Container ist `PATH` fest; absolute Pfade wären robuster, lohnen aber keinen Umbau ohne Anlass.                                                                           |
| `golang.org/x/crypto/openpgp` gilt als unwartbar (`GO-2026-5932`)                                                                                                                 | Kein Fix verfügbar (`Fixed in: N/A`), transitiv, **kein** Aufrufer im eigenen Code (`govulncheck`, zuletzt 31.08. bestätigt). Dependabot meldet sich, falls sich das ändert.                                                                                                 |
| Designer-Restlücke: Browser-Schließen binnen 800 ms verliert die letzte Auto-Save-Änderung                                                                                        | `onDestroy` schickt seit `37abcbe0` sofort; `sendBeacon` bewusst nicht gebaut (Randlage).                                                                                                                                                                                    |
| 082-Dedupe der Vormerkungen verlor den neueren `abholbereit`-Eintrag                                                                                                              | Auf Prod gelaufen, nicht rückholbar; nur relevant, falls je eine weitere **gewachsene** DB migriert wird.                                                                                                                                                                    |
| Rate-Limiter-Maps (`api/rate_limit.go`, `middleware_ratelimit.go`) räumen erst ab 5.000 Einträgen und dann nur Stale > 5 min; jeder neue Eintrag iteriert die ganze Map           | Sicherheits-Audit 05.09.: Nur mit vielen frischen Adressen (IPv6-Rotation) ein CPU-Thema; hinter Caddy trifft es die Schule nicht von innen. Ohne Anlass nicht anfassen.                                                                                                     |
| ZAP-Scan 05.09. (localhost:8084, unangemeldet, 17 GET-Endpunkte): `style-src 'unsafe-inline'`, `csrf_token` ohne HttpOnly, „Suspicious Comments"                                  | Alle drei sind dokumentierte Entscheidungen (SECURITY.md: CSP-Begründung; Double-Submit-Cookie muss JS-lesbar sein; der Kommentar ist `//scanapp.org` aus der QR-Bibliothek). HSTS/Cache-Meldungen betrafen `content-autofill.googleapis.com` (Chrome), nicht die Anwendung. |

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
- **GitHub**: PR-Pflicht abschaffen (Solo-Entscheidung 30.07.; das Ruleset `main` trägt am
  05.09. noch die `pull_request`-Regel), „Block force pushes" und „Restrict deletions"
  anlassen.
- Datenwert Schulname/„Neuer Text" auf Live korrigieren.
- **Vor dem nächsten Deploy (Migration 113):** auf dem Server prüfen, dass keine E-Mail in
  zwei Schreibweisen existiert — sonst bricht die Migration mit Klartext ab:
  `docker exec bibliothek-db psql -U postgres -d bibliothek -c "SELECT lower(email), count(*) FROM benutzer GROUP BY 1 HAVING count(*) > 1;"`
  (leere Ausgabe = gut).
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
  diese Ausnahmeliste.

**Parkdeck** (bewusste Nicht-Entscheidungen, nur mit Anlass wieder anfassen):
Integer-Cent-Refactor (float64/NUMERIC) · Bundle-Splitting (720-kB-Chunk) ·
TypeScript-Migration (null TS-Dateien) · Verschmelzung `inventur/` ins Haupt-API ·
`cmd/migrate` (MySQL) löschen — hat keine Datenquelle mehr, aber seine PG-Tests
sichern mit `internal/uebernahme` geteilten Code · Zukunftsideen API-Versionierung
(`/api/v1`) und Mandantenfähigkeit (RLS).

Stand: 2026-09-11
