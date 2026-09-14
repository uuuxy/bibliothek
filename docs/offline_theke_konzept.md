# Offline-Betrieb der Theke — Konzept

Stand: 14.09.2026

**Ziel (Peter, 13.09.2026):** Bei einem Verbindungsabbruch geht der Betrieb an der Theke normal
weiter. Danach bucht das System nach, und zwar das, was wirklich passiert ist. Nichts geht still
verloren.

**Maßstab:** Nicht die Zahl der Randfälle entscheidet, sondern ob an der Theke je ein Buch bei
einem Kind liegt, das im System frei ist, oder umgekehrt. Alles, was das verhindert, gehört hinein.
Alles andere kann warten, bis es im Betrieb vorkommt.

Die Entscheidungen stehen in [OFFEN.md](OFFEN.md), Abschnitt 2. Dieses Dokument ist der Bauplan;
offen ist nichts hier, sondern nur dort.

## 1. Was heute fehlt (am Code gelesen 13.09., nachgeprüft 14.09.2026)

1. Offline werden nur `B-`-Barcodes gespeichert (`omnibox.svelte.js`, `speichereOfflineAktion`).
   Alles andere, auch `LMF-`, Ziffern und Ausweise, meldet „Netzwerkfehler" und wird verworfen.
2. Ein Ausweis ohne Netz wird nicht geladen; die folgenden Bücher gehen an die vorher geladene
   Person. Mit geladener Lehrkraft wird ein Buch als Rückgabe eingereiht (nur `activeStudent`
   entscheidet; `activeTeacher` wird ignoriert, obwohl der Stapel-Endpunkt `active_teacher_id`
   kennt).
3. Person und Absicht werden erst NACH der hängenden Anfrage gelesen (Timeout 10 s in
   `apiFetch.js`). Escape oder „Theke leeren" in dieser Zeit: Das Buch geht als Rückgabe ohne
   Person in die Warteschlange. Ein Scan-Zeitpunkt wird nicht festgehalten; `offlineQueue.js`
   schreibt den Zeitpunkt des Einreihens.
4. In die Warteschlange kommt jeder `TypeError`, auch einer aus der Auswertung einer gelungenen
   200-Antwort (`verarbeiteAktionsErgebnis` liegt im selben `catch`). Der Eintrag trägt denselben
   Idempotenz-Schlüssel; solange der Server-Cache den Schlüssel hält (24 h, `jobs/cron.go`),
   kommt die alte Antwort zurück, danach wird neu gebucht.
5. Beim Nachbuchen gilt als erledigt: `success`, jeder 4xx außer 429, und „Server nannte den
   Index nicht" (`!result`). Der Antworttyp wird nicht mit der Absicht verglichen. Ein zweiter
   Scan desselben Buchs beim selben Kind wird zur Rückgabe; ein Buch, das noch bei jemand anderem
   steht, wird dort nur zurückgenommen (`handleForeignReturn`, bewusst kein Umbuchen).
6. Ein liegengebliebener Eintrag (5xx, 429) wird ohne Pause sofort erneut gesendet
   (`while (navigator.onLine)` in `offlineSync.svelte.js`). Antworten ab 500 werden serverseitig
   nicht gespeichert, 4xx schon.
7. Der Nachbuch-Bericht ist ein Toast mit dem Grund des ersten abgelehnten Eintrags. Er
   verschwindet mit dem Toast.
8. Nach 25 s ohne SSE-Signal legt sich ein Vollbild über die Theke (`App.svelte`,
   `heartbeatOk`). 502/503/504 gelten nicht als offline. Es gibt drei voneinander unabhängige
   „offline"-Begriffe: Herzschlag (Vollbild), `navigator.onLine` (rosa Band oben und
   Einreih-Entscheidung), Versandfehler je Anfrage (Einreih-Entscheidung).
9. Nach 15 Minuten ohne Eingabe sperrt die Theke per `setTimeout`, ohne Netz-Abfrage; entsperren
   geht nur über `POST /login`. „Theke leeren" nach 5 Minuten.
10. „Sicherung speichern" (`OfflineIndicator.svelte`) hängt strukturell schon außerhalb von
    Anmeldemaske und Sperrbildschirm, zeigt sich aber nur bei Zähler > 0, und der Zähler wird erst
    nach der Anmeldung geladen. Nach Neuladen ohne Anmeldung: kein Knopf.
11. `enqueueOfflineAction`, `loadQueue` und `dequeueOfflineAction` schlucken jeden IndexedDB-Fehler;
    die Theke meldet danach „gespeichert" mit Erfolgston, das Band zeigt 0.
12. Idempotenz (`api/action.go`): Antwort wird nach der Arbeit mit dem Request-Context
    gespeichert; bricht der Aufrufer ab, geht das Speichern verloren. Keine Reservierung vor der
    Arbeit. Zwei Anfragen mit demselben Schlüssel: gleichzeitig fängt der Unique-Index die zweite
    (gleicher Ausleiher → bestehende Ausleihe); kommt die zweite NACH dem Commit der ersten und vor
    dem Speichern der Antwort, wird sie zur Rückgabe.
13. Rückholen eines ausgesonderten Exemplars (`holeExemplarZurueck`) läuft in einer eigenen
    Transaktion mit Commit VOR Sperre, Limit und Vormerkung. Zwilling ohne `VerbucheRueckkehr`:
    `MarkiereVerlustAlsGefunden` (Inventur).
14. Der Online-Pfad sperrt `schueler`, dann die Ausleihe (`FOR UPDATE`), nie das Exemplar. Die
    Schranken-Zählungen laufen über `s.pool`, nicht über `tx`. Die Uhr ist je Service
    (`s.heute()`), nicht je Aufruf; die Handapparat-Frist nimmt an zwei Stellen rohes `time.Now()`.
    `ausgeliehen_am` ist ein DB-Default, `rueckgabe_am` wird mit `CURRENT_TIMESTAMP` geschrieben.
    `check_return_date` (23514) ist im Ausleihpfad nicht gemappt → 500.

## 2. Der Bau in drei Stufen, 17 Commits

Je Stufe: Rot-Test am alten Code, volle Suite mit Postgres, Nachweis am frisch gebauten Stack und
im Browser, dann Peters Freigabe. Voraussetzungen aus OFFEN.md, Abschnitt 3: 3.1 (Lehrkraft-Auflösung
mit `pgx.ErrNoRows` und `coalesce(barcode_id, '')`), 3.2, 3.3, 3.4.

Ratschen, die jeder Commit im Blick hat: 200 Zeilen je Frontend-Datei (`App.svelte` steht auf
genau 200, `Omnibox.svelte` mit 282 im Bestand und darf nicht wachsen); kein SQL in `api/`
(`schichtung_test.go`); `offlineImport.test.js` pinnt die Signatur von `enqueueOfflineAction`;
`omniboxOffline.test.js` ist heute nur durch den Fehler aus Commit 2 grün (der `apiClient.post`-Mock
liefert `undefined`, der TypeError kommt aus `res.ok`); `offlineSync.test.js` pinnt „4xx fliegt raus".

### Stufe 1 — vorhandene Fehler (7 Commits, je ein Fund)

1. **Schnappschuss beim Scan.** `submitAction` hält vor dem Versand fest: Absicht (Ausleihe,
   Rückgabe), Person (Schüler ODER Lehrkraft), Scan-Zeitpunkt. Der Warteschlangen-Eintrag wird ein
   Objekt (`{art, barcode, schueler_id, lehrer_id, gescannt_am, key}`), `enqueueOfflineAction`
   nimmt es entgegen. `offlineImport.test.js` wird einmal auf das Objekt umgestellt, nicht dreimal.
   Format 1 (alte Sicherungen: `{action_type, barcode_id, schueler_id, timestamp}`) bleibt lesbar.
   Rot-Test: Anfrage hängt, Store-Person wird geleert, Timeout → Eintrag trägt die Person vom Scan.
2. **Nur ein gescheiterter Versand wird eingereiht.** Der `catch` um `verarbeiteAktionsErgebnis`
   reiht nicht ein. Der Mock in `omniboxOffline.test.js` wird auf einen echten Versandfehler
   umgestellt (`apiClient.post.mockRejectedValue(new TypeError('Failed to fetch'))`); das ist der
   Beweis, dass der Test vorher die falsche Sache maß. Rot-Test: 200 mit `{type:'teacher'}` ohne
   `teacher` → Fehlerbanner, keine Warteschlange.
3. **„Buch zurückgeben" im Profil ist offline eine Rückgabe.** `onReturnClick` übergibt die
   Absicht aus Commit 1. Rot-Test: Doppelklick offline → zwei Rückgaben, keine Ausleihe.
4. **Lehrkraft geladen: Offline-Buch ist eine Handapparat-Ausleihe.** Der Eintrag trägt
   `lehrer_id`, `baueBatchPayload` sendet `active_teacher_id` (der Stapel-Endpunkt kennt es
   heute schon). Rot-Test am Payload.
5. **Ein Fehler der Warteschlange heißt nicht „gespeichert".** `enqueueOfflineAction` wirft weiter;
   die Theke meldet „NICHT gespeichert — Buch zurücklegen" mit Fehlerton. `loadQueue` liefert bei
   Fehler nicht `[]`, sondern wirft; das Band zeigt „Warteschlange nicht lesbar" statt 0.
   Rot-Test: IndexedDB wirft.
6. **Erledigt ist nur, was der Server wie gescannt gebucht hat.** Die `!result`-Failsafe fällt
   (Schweigen des Servers ist kein Erfolg). Passt der Antworttyp nicht zur Absicht (Ausleihe
   gescannt, `rueckgabe` gebucht), wird der Eintrag ausgebucht UND gemeldet, mit Barcode und
   beiden Typen; er blockiert nicht. 4xx wird weiter ausgebucht und gemeldet. 5xx und 429 bleiben
   liegen, aber die Runde endet dort, statt sofort erneut zu senden; der nächste Anlauf kommt mit
   dem nächsten `online`-Ereignis oder nach einer Minute. (Das Blockieren der Warteschlange am
   ersten `wiederholen`-Eintrag kommt erst in Stufe 3 mit der neuen Tür, die den Fall
   `bereits_ausgeliehen` kennt; vorher hätte ein legitimer Fall die Warteschlange gesperrt, ohne
   dass es einen Ort gäbe, ihn aufzulösen.)
7. **Idempotenz hält.** Die Antwort wird mit `context.WithoutCancel` gespeichert (Vorbild
   `mahnwesen_bulk_mail.go`); der Schlüssel wird vor der Arbeit reserviert (Zeile mit Marker,
   `response_data` bekommt eine unterscheidbare Form oder wird nullbar mit `status_code = 0`);
   ein zweiter Aufruf auf einen reservierten Schlüssel wartet oder antwortet 409 `in_arbeit`.
   Antworten ab 500 werden weiter nicht gespeichert. Rot-Test (Postgres): zweite Anfrage mit
   demselben Schlüssel NACH dem Commit der ersten und VOR dem Speichern der Antwort → heute
   Rückgabe, danach dieselbe Ausleihe. Der 24-Stunden-Ablauf bleibt; die Nachbuch-Tür (Stufe 2)
   deckt spätere Wiederholungen über den Wächter ab.

### Stufe 2 — Server (5 Commits)

8. **Rückholen als Transaktions-Baustein.** `holeExemplarZurueck` bekommt eine `tx`-Variante ohne
   eigenen Commit; der Online-Pfad nutzt sie weiter mit eigener Transaktion, bis Commit 10 sie in
   die Buchung zieht. Commit-Botschaft nennt den Zwilling `MarkiereVerlustAlsGefunden` (Inventur,
   ohne `VerbucheRueckkehr`) und warum er bleibt oder mitgezogen wird.
9. **Migration 116: Bewegungsstempel.** `ausleihen.erfasst_am` (Scan-Zeitpunkt, Default
   `CURRENT_TIMESTAMP`) und `buecher_exemplare.letzte_bewegung_am` (geschrieben bei Ausleihe,
   Rückgabe, Rückholen und Aussonderung; NICHT der generische `aktualisiert_am`-Trigger).
   Sperrreihenfolge wird hier festgelegt und in `docs/invarianten.md` eingetragen: `schueler`,
   dann Ausleihe, dann Exemplar; Online-Pfad und Nachbuchen halten dieselbe Reihenfolge, sonst
   verklemmen sie sich. Gates: Schema-Parität (Migration UND `schema.sql`), Schema-Gegenrichtung,
   Aussonderungs-Parität (Schreibweise der UPDATEs), Restore-Probe.
10. **Migration 117: Nachbuch-Meldungen** (`nachbuch_meldungen`: Exemplar, Ergebnis, Grund,
    Ausleiher, Vorbesitzer, Scan-Zeitpunkt, Barcode-Text für nicht auflösbare Personen,
    quittiert von/am). Liste und Quittieren nur mit `view_students`. Frist: quittierte nach der
    Lesehistorie-Frist, höchstens 30 Tage, als `PredikatNachbuchMeldungen` in
    `repository/loeschfristen.go` (Löschjob und Wächter, eine Quelle); offene nach 14 Tagen als
    Warnung in der Betriebsbereitschaft (Vorbild `pruefeEhemaligeOffen`). BEIDE Personenspalten
    wandern beim Zusammenführen, werden über `spurTilgungen` getilgt, stehen in der Auskunft und in
    `dsgvoSchuelerQuellen`. Gates: FK-Rundreise, Zusammenführen-Gate, Schema-Gegenrichtung,
    PII-Matrix mit Antwort-Gate, Routen-Rechte, Rechte-Parität. Das API-Inventar ist kein Gate,
    sondern Handarbeit (`scripts/api_inventar.sh`).
11. **`POST /api/action/nachbuchen`** (`perform_actions`, 1–50 Einträge, UUID-Schlüssel, Absicht,
    Barcode, Scan-Zeitpunkt, Ausweis-Barcode oder Person). SQL liegt in `repository/`, der Handler
    ist dünn. Je Eintrag eine Transaktion: Sperren in der Reihenfolge aus Commit 9 → Wächter →
    Rückholen (Baustein aus 8) → Savepoint → Schranken → buchen. Die Rücknahme beim Vorbesitzer
    steht VOR dem Savepoint und bleibt, wenn die neue Ausleihe an Sperre, Limit oder Vormerkung
    scheitert; nur sie meldet `nicht_gebucht`. Der Wächter: Scan älter als `letzte_bewegung_am` →
    409 `veraltet`, Rollback auch des Rückholens; Ausnahme: Die letzte Bewegung trägt denselben
    Idempotenz-Schlüssel (abgebrochener Online-Versand, der die Rücknahme schon gebucht hat) → die
    fehlende Ausleihe wird nachgeholt. Ergebnisse: `ausgeliehen`, `umgebucht`, `bereits_ausgeliehen`,
    `zurueckgegeben`, `nur_reaktiviert`, `nicht_gebucht` (Grund), `veraltet`, `wiederholen`.
    Rückgabe eines verloren gemeldeten Buchs läuft durch `VerbucheRueckkehr`; der Hinweis
    „Schulaufsicht informieren" steht in der Meldung. Zeit: Uhr als Parameter (`heute` je Aufruf),
    `ausgeliehen_am`/`rueckgabe_am`/`erfasst_am` als Parameter der Repository-Funktionen, höchstens
    Serverzeit; die beiden Handapparat-Fristen mit rohem `time.Now()` ziehen auf dieselbe Uhr.
    23514 (`check_return_date`) wird auf 409 gemappt. Die Schranken-Zählungen laufen über `tx`,
    nicht über `s.pool` (sonst zwei Verbindungen je Eintrag; OFFEN 6.1). Jede Meldung entsteht in
    derselben Transaktion; `nicht_gebucht` und `veraltet` nach dem Rollback in eigener.
12. **`GET /api/action/buchbarcodes`** (`perform_actions`): alle Barcodes nicht ausgesonderter
    Exemplare, komprimiert, mit ETag. Bewusst ohne LIMIT, im Kopfkommentar begründet; die
    Antwortgröße wird am Server gemessen (rund 35.000 Einträge erwartet, Seed hat 1).

### Stufe 3 — Theke (5 Commits)

13. **Ein Band statt Vollbild und rosa Leiste; ohne Netz keine Sperre.** Das Vollbild fällt
    (macht in `App.svelte` Platz), `OfflineIndicator` wird das eine Band: Zustand, Zähler,
    „Sicherung speichern", Zugang zur Meldungsliste. Ein Prädikat `verbindungFehlt` in
    `offlineSync` vereint die drei Begriffe: `navigator.onLine === false`, Herzschlag älter als
    25 s, letzter Versand mit Netzfehler, Timeout oder 502/503/504 (abzweigen vor
    `handleActionHttpError`). `idleLock` sperrt nicht, solange `verbindungFehlt`; kommt die
    Verbindung zurück und war länger als 15 Minuten niemand da, sperrt es sofort. Geleert wird
    weiter nach 5 Minuten. Der Zähler kommt direkt aus IndexedDB, auch ohne Anmeldung. Der
    Sperrbildschirm (`aria-modal`, Fokusfalle) lässt den Knopf per Tastatur erreichbar. axe-Gate
    misst den Zustand „Band sichtbar" (`context.setOffline(true)` wie in
    `abmelden-ohne-antwort.spec.js`). `idleLock.test.js` bekommt den Fall „ohne Netz keine Sperre".
14. **Sync über die Nachbuch-Tür.** Portion 25, Frist 20 s, Einträge nach Scan-Zeitpunkt; die
    Runde endet beim ersten `wiederholen`, alles andere wird ausgebucht und gemeldet. Sicherungen
    (Format 2 mit Zeitstempel und Absicht; Format 1 weiter lesbar) werden gemeinsam eingespielt,
    ein `startSync()` für alle Dateien. `/api/action/batch` bleibt eine Version.
15. **Offline alle Buchformen und Ausweise.** Die Buchliste wird nach der Anmeldung geholt
    (`starteHintergrundAbrufe`, `perform_actions`, ETag) und lokal gehalten, ohne Personenbezug.
    `B-` und `LMF-` gelten auch ohne Liste; Ziffernfolge auf der Liste = Buch; `S-`/`L-` = Ausweis
    (Merker; Person geleert; folgende Bücher tragen den Ausweis-Barcode); unklar = Zuordnung
    gesperrt bis zum nächsten eindeutigen Ausweis. Geräte offline mit Meldung abgewiesen. Nach
    Rückkehr der Verbindung: beim nächsten Buchscan den Merker-Ausweis online auflösen, dann normal
    buchen. Dieser Commit kommt NACH 14: Vorher hätte der alte Stapel-Sync einen Ausweis-Eintrag
    als Ausweis-Lookup gebucht und die folgenden Bücher als Rückgabe.
16. **Meldungsliste** hinter dem Band, nur angemeldet, nicht gesperrt, mit `view_students`;
    „Theke leeren" schließt sie (Eintrag in `thekeLeeren.js`). Quittieren; Zähler an allen
    Arbeitsplätzen über SSE. Neues Bauteil, eigene Datei.
17. **Doku:** HANDBUCH („Theke ohne Verbindung"), FACHKONZEPT 18.4, PII-Matrix, invarianten
    (Sperrreihenfolge), VVT und Datenschutzhinweis (Meldungen mit Frist; kurzzeitige Speicherung
    von Ausweis- und Buchnummern am Theken-Rechner), Rückweg-Anleitung, OFFEN.md und erledigt.md.

## 3. Nachweis am Stack (je Stufe, echter Chrome)

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

## 4. Nicht im Umfang

Anmeldung ohne Server · Schülerdaten auf dem Rechner · Umbuchen im Online-Weg · Geräte offline ·
Rückbau von `/api/action/batch` (nächste Version) · scannende Person im Protokoll (nachgebucht wird
unter dem beim Sync angemeldeten Konto; steht in der Doku).

## 5. Was die Prüfung am 14.09.2026 am ersten Entwurf geändert hat

- Commit 6 blockiert die Warteschlange nicht mehr in Stufe 1: Ein legitimer Fall (Ausleihe
  gescannt, Server bucht Rückgabe) hätte jeden Rechner gesperrt, ohne Ort zur Auflösung bis
  Stufe 3. Er meldet; blockiert wird erst mit der neuen Tür.
- Der Rot-Test zu Commit 7 muss die Form „zweite Anfrage nach dem Commit der ersten" haben; die
  gleichzeitige Form fängt heute schon der Unique-Index.
- Stufe 2 in der Reihenfolge 8 → 9 (Migration) → 10 (Meldungen) → 11 (Tür) → 12: Die Tür braucht
  den Bewegungsstempel und die Meldungstabelle; im ersten Entwurf kamen beide nach ihr.
- „Rücknahme bleibt, neue Ausleihe scheitert" in EINER Transaktion braucht einen Savepoint; der
  erste Entwurf hätte beim Rollback die Rücknahme mitgenommen.
- Der Wächter braucht die Ausnahme „gleicher Idempotenz-Schlüssel", sonst weist er genau den
  Eintrag ab, den OFFEN.md 2 („abgebrochener Online-Versand") nachholen will.
- Sperrreihenfolge festgelegt (Schüler, Ausleihe, Exemplar): Online sperrt das Exemplar heute
  nie; „Exemplar sperren → Wächter" als erster Schritt hätte gegen den Online-Scan einen Zyklus
  gebaut.
- Uhr je Aufruf statt je Service, Repository-Funktionen mit Datum, beide Handapparat-Fristen,
  23514-Mapping, Schranken über `tx`: fehlten im ersten Entwurf.
- Stufe 3: 15 (neue Formen) nach 14 (neue Tür); ein Band statt zwei; das Netz-Signal für die
  Sperre ist benannt; Format-1-Sicherungen bleiben lesbar; `App.svelte` und `Omnibox.svelte`
  stehen an der 200-Zeilen-Ratsche.
- Der Nachbuch-Bericht war nie ein Bericht, sondern ein Toast (1.7).
- Antworten ab 500 werden heute schon nicht gespeichert; die befürchtete Endlosschleife am
  gecachten 5xx gibt es nicht, nur das pausenlose Wiederholen (1.6).
