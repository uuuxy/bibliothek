# Erledigt

Stand: 15.09.2026

Archiv zu [OFFEN.md](OFFEN.md): was erledigt oder endgültig entschieden ist, mit Datum und Commit.
Neu Erledigtes kommt oben in den jüngsten Abschnitt.

Ältere Stände: Das Befund-Register `docs/befunde.md` ist am 13.09.2026 in OFFEN.md und diese Datei
aufgegangen; seine Geschichte steht in `git log -p docs/befunde.md`, Stände vor früheren Kürzungen
in `2e09ec14` (31.08.2026), `47a09f70` (01.09.2026), `9f91784b` (04.09.2026), `e2bd4b72`
(05.09.2026) und `36c7bdce` (05.09.2026 abends). Die Pakete und Checklisten der Herbstplanung
stehen in den geschlossenen Issues #593 bis #600.

---

## Entschieden, gilt weiter

- **Arbeitslisten bleiben ohne Cover** (05.09.2026). Das führende Element der `ArbeitsZeile` ist
  der Klassenkreis, nach dem Klassensatz-Reservierungen und Wünsche & Meldungen überflogen werden —
  M3 kennt genau eines. Die zwei fehlenden Backend-Felder (`isbn` in
  `GetKlassensatzReservierungen`, `cover_url` in `anliegen.go`) werden deshalb nicht gebaut. Ob das
  auch für das Kollegiums-Portal gilt, ist offen (OFFEN.md 4.14).
- **Geisterbuch-Fall bei der Ausleihe bleibt so** (Peter, 11.09.2026;
  `entferneErfuellteVormerkung`). Nimmt ein Kind statt des bereitgelegten ein anderes Exemplar,
  geht das bereitgelegte ins Regal zurück und nicht an den Nächsten auf der Warteliste. Es geht
  nichts verloren; der Nächste wird bei der nächsten Rückgabe bedient. Die strenge Reihenfolge der
  Warteliste ist im Betrieb nicht nötig.
- **Bestellwesen D1, D3, D4, D5** (10.09.2026): beantwortet bzw. entschieden, Einzelheiten in
  [mittel_konzept.md](mittel_konzept.md), Abschnitt 7.4. Offen ist nur D2 (OFFEN.md 8.4).

---

## 15.09.2026

**Offline-Betrieb der Theke, Stufe 1 (OFFEN.md 2.2, „vorhandene Fehler"), sieben Commits.** Je
ein Rot-Test am alten Code; volle Suite mit Postgres 18-alpine, golangci-lint, deadcode, vitest,
svelte-check, eslint, prettier. Der Nachweis am Stack (2.3) und Peters Freigabe für Stufe 2
stehen noch aus.

1. **Schnappschuss beim Scan** `0fa4b5a3`: Absicht, Person (Schüler oder Lehrkraft),
   Scan-Zeitpunkt und Schlüssel werden vor dem Versand festgehalten; der Eintrag ist ein Objekt
   (Format 2), Format 1 bleibt in IndexedDB und in Sicherungen lesbar (`normalisiereEintrag`).
   Rot: Anfrage hängt, Schüler geleert, Timeout → Eintrag trug `null`.
2. **Nur ein gescheiterter Versand wird eingereiht** `18b4887e`: zwei Fehlerzweige in
   `submitAction`. Rot: 200 mit unauswertbarem Inhalt landete in der Warteschlange. Der Mock in
   `omniboxOffline.test.js` ist jetzt ein echter Versandfehler — vorher war der Test nur durch
   diesen Fehler grün.
3. **„Buch zurückgeben" ist offline eine Rückgabe** `44615c42`: `omniboxStore.gibZurueck` mit
   Absicht; Draht-Test an `onReturnClick`. Rot: Doppelklick ergab zwei Ausleihen.
4. **Lehrkraft geladen → Handapparat** `08312c87`: Eintrag mit `lehrer_id`, Payload mit
   `active_teacher_id`. Rot am Eintrag und am Payload.
5. **Warteschlangen-Fehler ist laut** `65f9a998`: `enqueueOfflineAction`, `loadQueue`,
   `dequeueOfflineAction` werfen; Theke „NICHT gespeichert" mit Fehlerton; Band „Warteschlange
   nicht lesbar" statt 0. Rot mit gemockter IndexedDB.
6. **Erledigt nur, was wie gescannt gebucht wurde** `6fe6ba8b`: Typvergleich, Schweigen und
   5xx/429 bleiben liegen, die Runde endet; Intervall 60 s. Rot: Schweigen ausgebucht,
   Abweichung still, 5xx sofort erneut gesendet.
7. **Idempotenz hält** `23ca498c`: `repository/idempotenz.go` (Reservierung vor der Arbeit,
   verwaiste nach 60 s übernommen, Speichern mit `WithoutCancel`, Freigabe nach 5xx); Warten bis
   3 s, dann 409 `in_arbeit` (Einzel) bzw. 503 (Stapel). `action.go` formuliert kein SQL mehr.
   Rot am echten Postgres: zweite Anfrage nach Commit, vor Speichern → Rückgabe, null offene
   Ausleihen.

Die Voraussetzungen des Offline-Baus (OFFEN.md 1.5 und 3.1–3.4), je ein Commit, je ein Test am
alten Code rot gesehen; volle Suite mit Postgres 18, golangci-lint, deadcode, vitest, svelte-check.

**Schülerakte: gescheiterter Abruf sah aus wie „kein Bescheid" (A, 1.5).** `useStudentProfile`
wies bei `!res.ok` für Vormerkungen, Gebühren und Bescheide `[]` zu, ohne Hinweis. Jetzt
`fehlendeListen` wie in der Buch-Akte: 5xx und Netzfehler werden vermerkt, 403 und eine leere Liste
nicht; der Reiter zeigt „Nicht geladen: …" mit „Erneut laden". Der Bestandseintrag in
`fehlerausgang.test.js` schrumpft von 4 auf 1 (der Kopf; der leere Kopf steht als Beobachtung in
OFFEN.md 5.9). Dabei gefunden: Der `beforeEach` des Tests gab das Mock zurück, und Vitest ruft
einen zurückgegebenen Funktionswert als Aufräumer auf. `253f3dcd`.

**Lehrkraft-Ausleihe meldete jeden Datenbankfehler als 404 (3.1).** `resolveTeacherBorrower`
liest jetzt dieselbe Abfrage wie die Geräte-Seite (`ladeAktiveLehrkraft`): nur `pgx.ErrNoRows` wird
`ErrNotFound`, `coalesce(barcode_id, '')` für die nullbare Spalte — eine Lehrkraft ohne Ausweis
scheiterte vorher am Scan. `lehrkraft_ohne_ausweis_pg_test.go` (echtes Postgres) und Mock-Test.
`b97ca463`.

**Theke hielt nach dem Zusammenführen die gelöschte Kennung (3.2).** `Omnibox.svelte` reicht
`onMerged` durch; `omniboxStore.uebernimmZusammengefuehrt` setzt die Ziel-Kennung sofort und lädt
Name und Sperrflags nach. `studentProfileEinbau.test.js` prüft am Draht, dass jeder Einbauort von
`StudentProfile` `onMerged` und jeder von `StudentProfileAusleihen` `fehlendeListen` durchreicht.
`5082388e`.

**„Einmalig ignorieren" sah jede Rolle (3.3).** Beide Knöpfe des Sperr-Dialogs (`override_block`,
`PATCH …/lock`) nur mit `edit_students` über `schuelerRechte(...).bearbeiten`; ohne Recht ein Satz,
wer übergehen kann. `4f8820f6`.

**Abmelden bei 503 ohne Hinweis (3.4).** `authStore.abmeldeHinweis` bei 503 (beim Abmelden und
beim Nachholen im Boot-Restore), auf der Anmeldemaske sichtbar bis zur nächsten Anmeldung.
`5311d598`.

Nebenbefund: `TestSuchnorm_ZeichenDurchlaufGoUndSQL` ist gegen das Homebrew-Postgres 18.6
(macOS-libc) rot (vier Digraphen ǅ ǈ ǋ ǲ, Go „dz" gegen SQL „Dz"), gegen `postgres:18-alpine`
(Stack und CI) grün. Keine Änderung; wer lokal ohne Docker testet, weiß es jetzt.

---

## 14.09.2026

**LMF-Frist am Rückgabetermin (A).** Wer am Termintag seiner Klasse ein Schulbuch bekam, hatte den Termin
selbst als Frist (heute 23:59), am Tag danach den Stichtag des laufenden Schuljahres — in den Ferien;
nach den Ferien wäre die ganze Klasse überfällig und nach 14 Tagen gesperrt gewesen. Entschieden am
13.09.2026: Am oder nach dem Termin ist die Frist der Stichtag des folgenden Schuljahres.
`RueckgabeTerminLage` liefert beides in einer Abfrage, die Uhr des Ausleihdienstes ist einsetzbar;
Matrix T−1/T/T+1 am echten Postgres (`lmf_frist_termintag_pg_test.go`), am Rückbau rot gesehen.

**Release v2.12.0** auf `55019206` (32 Commits seit v2.11.0, keine Migration): die drei A-Fixes des Tages,
Sperr-Dialog am Merkmal, Geräte-Rückgabe und -Ausleihe, ISBN-Dublette, Kennungen 400 statt 500,
Cover-Upload-Reihenfolge, Prüflauf über Doku und Werkzeuge, OFFEN.md. Der Offline-Bau beginnt damit
auf einem benannten Stand. Deploy (`git pull`, `./update.sh`) liegt bei Peter.

**Abmelden ohne Antwort des Servers (A).** Das Löschcookie setzt nur die Antwort des Servers. Ohne Netz
kam keine an: Das HttpOnly-Cookie blieb im Browser, und nach dem nächsten Neuladen mit Netz war die
vorige Person wieder angemeldet. Notiert am 13.09.2026, am Stack nachgestellt am 14.09.2026 (Browser
offline, Abmelden, online, Neuladen). Jetzt vermerkt die Theke eine ausstehende Abmeldung im Browser
und holt sie beim nächsten Start nach, bevor sie eine Sitzung wiederherstellt. Gates
`e2e/abmelden-ohne-antwort.spec.js` und `authStore.test.js`, beide am alten Stand rot gesehen. Dabei
gemessen: `context.route` fängt die Abmeldung nicht ab, weil sie mit `keepalive` hinausgeht.

**Scanfeld nach dem Wechsel zur Ausleihe (A).** Ein Klick auf den Menüpunkt „Ausleihe" ließ den Fokus auf
dem Knopf der Seitenleiste; der nächste Scan lief ohne Meldung ins Leere, und das Enter des Scanners
löste den Menüknopf erneut aus. Dasselbe beim Rückweg aus einem anderen Bereich mit geladenem
Schüler. Notiert am 28.07.2026, am Stack nachgestellt am 14.09.2026. Jeder Wechsel zur Ausleihe gibt
dem Scanfeld jetzt den Fokus zurück (`uiStore.beimWechselZurTheke`); Gates
`e2e/scanner-fokus-menue.spec.js` und `uiStore.test.js`, beide am Rückbau rot gesehen.

**Ersatzforderung ohne Bescheid-Positionen (A).** Der Knopf „Ersatzforderung" in der Schülerakte nahm
auch Forderungen auf, die schon auf einem Schadensersatz-Bescheid stehen, und verlangte dafür „bar
in der Bibliothek" — zwei Zahlungsaufforderungen mit zwei Zahlungswegen für dieselbe Forderung.
Ohne Wirkung, solange kein Bescheid existiert. Jetzt bleiben diese Positionen draußen; stehen alle
offenen Forderungen auf einem Bescheid, sagt die Meldung das. Zwei Postgres-Tests, am alten Code rot
gesehen (`api/print_rechnung_pg_test.go`).

---

## 13.09.2026

**Theke.** Am alten Code rot gesehen: Ein verliehenes Gerät, das danach als defekt gemeldet wurde,
ließ sich nicht zurückgeben (403, die Ausleihe blieb offen; am Stack nachgestellt), und ein
gesperrter Schüler konnte sein Gerät nicht zurückgeben (im Postgres-Test belegt; `3899cf18`). Die Geräte-Ausleihe
übernahm `active_teacher_id` ungeprüft — eine unbekannte Kennung ergab 500, ein deaktiviertes
Profil oder eine Mitarbeiterin bekam das Gerät (`cc9e6c8c`, jetzt dieselbe Regel wie Lehrerausweis
und Buch-Ausleihe). Der Sperr-Dialog hängt am Merkmal `X-Sperre: uebergehbar` statt am Wortlaut
der Meldung; die Schadens-Sperre öffnete vorher keinen Dialog (`e107465e`).

**Entscheidungen (Peter).** Acht Fragen beantwortet: Neuladen ohne Netz (4.1, jetzt OFFEN.md
Abschnitt 2), Abmelden bei 503 (4.2, jetzt 3.4), LMF-Frist am Rückgabetermin (4.3, jetzt 1.4;
`ziel_jahrgang` bleibt als 4.3 offen) und fünf Fragen zum Offline-Plan (Abschnitt 2).

**Doku zur Offline-Theke.** HANDBUCH („Wenn etwas nicht geht") und FACHKONZEPT 18.4 versprachen,
dass die Theke ohne Netz alle Scans speichert; gespeichert werden nur Bücher mit `B-`. Beide nennen
jetzt den heutigen Stand, das Handbuch dazu, was an der Theke während eines Ausfalls zu tun ist (`512585b4`).

**Release-Doku.** DEPLOYMENT.md sagte, beide Tag-Workflows prüften vor dem Bau die CI des Commits.
Nur `release.yml` tut es, `docker-publish.yml` baut das Image auch auf rotem Stand. Die Doku
beschreibt jetzt, was geprüft wird; das fehlende Gate steht in OFFEN.md 5.10.

**Prüflauf über Issues, Register und Dokumentation** (Peter: „überprüfe alles genau … auch die
Dokumentation"). Fünf Prüfer, jede Aussage mit Fundstelle, Stichproben am Code. Behoben: Die
doppelte ISBN wurde nie als Dublette erkannt (der Constraint hieß noch `books_isbn_key`,
`260b1436`); die Ausnahme für `ziel_jahrgang` in `inventur/schema_paritaet_test.go` nannte einen
Schreiber, den es nie gab; zehn Commit-Nummern im Register waren im letzten Zeichen falsch (beides `ee68e1c3`). Aus
demselben Lauf:

- Das API-Inventar liest keine Testdateien mehr (`ed4c6c10`), die Swagger-Spezifikation nennt
  keinen festen Host (`e502c3c6`), und `security-scan.sh` erreicht den lokalen Stack. Dabei fiel
  mehr auf als der Port: Das Token ging als Bearer mit, die Anwendung liest aber nur das Cookie
  (jeder Lauf war unangemeldet), und ein Lauf ohne importierte URL meldete „Scan komplett"
  (`61ea1357`, am Stack gelaufen).
- Eine Kennung aus Query oder Body, die keine UUID ist, wird mit 400 abgewiesen statt als 500 aus
  Postgres (22P02) zurückzukommen (`02db8b66`) — geprüft an der Tür, nicht zentral in apierrors. Neue Klasse in
  `sweeps.md`, Gate `uuid_eingaben_test.go`, am alten Code rot gesehen. Nachgezogen, beides am
  Stack nachgestellt und rot gesehen: Die Prüfung nahm `urn:uuid:…` an, weil `uuid.Validate` die
  Form kennt und Postgres nicht (`acce59e7`, `pkg/kennung`); vier Buch-Handler zerlegten ihren Pfad
  selbst, sodass `PUT /api/books/x` und die drei Cover-Routen 500 lieferten (`35afea98`,
  Platzhalter `{id}`). Dabei gefunden: Der Cover-Upload schrieb die Datei, bevor feststand, dass es
  das Buch gibt, und löschte das alte Cover vor dem UPDATE (`ddee5802`).
- Abmelden übersteht einen sofortigen Reload oder das Schließen des Tabs (`95e7a24f`, `keepalive`,
  am alten Code rot gesehen). Gefunden, weil `auth.spec.js` in der CI rot wurde: Reload drei
  Millisekunden nach dem Klick, die Sitzung lebte weiter.

**Betrieb, lesend am Hetzner-Server nachgesehen:** Prod-Secrets gesetzt, `APP_ENV=production`,
`ENFORCE_PROD_SECRETS=true`, `BACKUP_ENCRYPTION_KEY` gesetzt (nächtliche Backups laufen
verschlüsselt, die wöchentliche Restore-Probe war erfolgreich), `IMAP_HOST` gesetzt, SMTP in der
Datenbank eingerichtet, `SELBSTANMELDUNG_DOMAIN` gesetzt, `SENTRY_DSN` leer (Datenschutz A6). Die
Demo-Daten sind entfernt: `scripts/entferne_demo_daten.sql` (`566b9fe3`), vorher Backup und
Vorschau ohne Verflechtung mit echten Daten; 2.000 Schüler, 2.500 Exemplare, 1.613 Ausleihen und
eine Protokollzeile, nachgeprüft 0/0/0.

**Als offen geführt, aber erledigt** (beim Zusammenlegen in OFFEN.md festgestellt):

- Tabellen-Bauteil: gebaut am 08.09.2026 (`4a55c52f`, Ratsche `frontend-hygiene-tabellen.test.js`).
- Zweiter Schnitt von Paket 2: in v2.11.0 enthalten (`14e6a56d`, `9f0fca8f`, `39cbaaf8`).
- Liefern die Lesegeräte an Littera-Etiketten den Zifferninhalt? Nein, der Strichcode liefert eine
  EAN-13; die Rückrechnung auf die Mediennummer ist eingebaut, belegt mit zwei echten Scans vom
  18.08.2026 (`84cbe12d`). Kein Buch muss neu beklebt werden.
- Echten LUSD-Export hochladen: zwei echte Exporte importiert
  ([lusd-simulation-2026-09-02.md](lusd-simulation-2026-09-02.md)).
- Schüler-ID im LUSD-Bericht: Kein LUSD-Bericht enthält eine ([LUSD.md](LUSD.md), geprüft am
  26.08.2026).
- `Ansprechpartner_Alle_*` wird bewusst nicht gelesen ([LUSD.md](LUSD.md)); eine Eltern-Mahnmail
  ist nicht gebaut (Datenschutz A3).
- Die Begründung „kein DB-Zugriff" im Bestand der Fehler-Kollaps-Ratsche ist berichtigt
  (`fehler_kollaps_test.go`, 11.09.2026, `5cc80b89`).
- `resilience_and_recovery.md` nennt `bibliothek-db` nicht mehr als Dienst.
- „Ausleihen beenden" bei der Übergabe eines Bescheids entfällt: `ReportDamage` beendet die
  Ausleihe schon beim Melden.

---

## 12.09.2026

**Rasterdurchgang über die Änderungen vom 11.09.2026** (Peter: „wir haben die ganzen Schemata
komplett drüberlaufen lassen — überprüfe alles sorgfältig und fahre gegebenenfalls fort"). Der
Bestands-Durchgang vom 10.09.2026 endete bei `f67078f3`, danach kamen acht Commits (`4fd4bc42` …
`214a5ecd`, zwei davon nur Doku), die nie gerastert waren. Zwölf Fragen über die sechs Code-Commits, dazu die
Zwillingsfrage je Fix. Drei Funde, alle am alten Code rot gesehen:

- **Fremdrückgabe an der Sperre (`395551f9`).** Der Fix vom 11.09.2026 hängte die Sperrprüfung an
  `!isReturningThis` — der dritte Fall fiel damit auf die falsche Seite. Wer gesperrt oder am
  Ausleihlimit war, konnte das Buch eines Mitschülers nicht abgeben. Neue Klasse in `sweeps.md`.
  Die Reservierungs-Schranke des Handapparats hielt `TestHandleLehrerHandapparat_SchranktEin`.
- **Boot-Restore gegen den Datenbank-Aussetzer (`8750b897`).** Der Aussetzer bekam am 11.09.2026
  den Statuscode 503, gelernt hatten ihn aber nur die Leser im laufenden Betrieb; wer
  währenddessen neu lud, stand trotz gültigem Cookie am Login. **Prüfmuster:** Wer einen
  Statuscode neu einführt, zählt seine Leser.
- **Zahl der geprüften Bereiche (`a17fc517`).** Das FACHKONZEPT nannte fünfzehn Bereiche, `Pruefe`
  lieferte sechzehn. Jetzt eine Ratsche, in drei Richtungen rot gesehen.

Aus demselben Durchgang: Der 503 der Sitzungsprüfung nennt nur noch den Satz, nicht die gewrappte
Ursache (`2c8fe70d`, `apierrors.SendHTTPErrorMitMeldung`), und das Abmelden meldet, was wirklich
widerrufen wurde — 503 statt `{"status":"ok"}`, wenn die Sperrliste nicht antwortet (`039145f2`).

**Schüler und Frontend:** die beiden Suchen und die vier Listen samt erweitertem
Fehlerausgang-Scanner (`05391fe8`, `dfc9913a`, `5882d0cd`), der Mail-Knopf ohne Recht
(`143926df`), der Ausweis 12T (`59beb423`), die zurückgestellte 180-Tage-Uhr (`473bea94`) und die
Wiederherstellung am Namensindex (`39b58be8`).

**Gates und Betrieb:** `migrate-fotos` ist aus dem Repo und wird im Image gebaut (`357d28f2`); ein
Gate hält fest, dass jedes Werkzeug einer Anleitung auch im Image liegt. Die falsche Begründung zum
Release-Gate in DEPLOYMENT Abschnitt 8 ist berichtigt. DEPLOYMENT 8 und 2.3 sowie die toten
Compose-Variablen (`e5d8e21a`, je mit Ratsche).

**Register-Kleinkram (#593):**

- LMF-Planer: Startstunde (`cc7bc392`, mit Paritäts-Gate gegen die Go-Konstanten), Vorschau-Form
  (`a5d2a6f6`), Wartezeit der Entprellung (`c672183e`).
- `ohne_rueckgabe_termin` ersatzlos gestrichen (`5666c0fc`): Der Planer zeigt „Noch nicht im Plan"
  an der richtigen Tür; die Abfrage steht in `git log -p repository/lmf_termine.go`.
- Vier Zusicherungen des Planers: drei als Constraints in Migration 115, die vierte an beiden Türen
  (`04822780`, in v2.11.0).
- Wächter „Ehemalige mit offenen Vorgängen" und Löschprädikat nutzen denselben Ausdruck
  `AbgangSeit` (`fa4a2113`).
- `ui/Select`: Tastaturlogik nach `selectTastatur.js` (`8a444e6b`), dann `aria-activedescendant`
  (`d62c8f34`).
- `main` war rot: Vitest endete mit 1 trotz grüner Bilanz (`f67f2dff`, `scrollIntoView` in
  `vitest.setup.js`), golangci-lint brach an einem toten Typ ab und übersprang dabei die halbe
  Prüfung (`06af04be`).

**Schadensersatz und Bestellwesen:** Verleihjahr in Schuljahren statt Ausleihen und
Kalenderjahren (`c7c5718f`), Bescheide in der Schülerakte (`006587cd`), Rückgabe-Hook des
Bescheids (`e9ac79e6`); zweiter Schnitt von Paket 2 — Berichte je Topf, Topf-Filter in der
Historie, Kennzahlen je Topf (`14e6a56d`, `9f0fca8f`), dazu das letzte Gate aus #596: Ein DNB-Titel mit
Haken wird zum Lernmittel (`39cbaaf8`).

---

## 10.09.2026

**Bestands-Durchgang** (Peter: „lass alle Schemata nochmal über alles laufen, keine Ausnahmen").
Alle zwölf Fragen über jede Datei (neun Prüfer, je ein Bereich), die „sieht nicht"-Spalte der
Landkarte als Suchliste, dazu die volle Gate-Batterie (Go mit Postgres 2341 grün, Vitest,
svelte-check, Lint, deadcode, govulncheck, Trivy, Druck-Gate, Playwright 183 grün). **22 Funde in
20 Commits behoben** (`9056f2d0` … `f67078f3`), jeder am alten Code rot gesehen; neue Klassen in
`sweeps.md`.

**Teil B Beschaffung, erster Schnitt** (Migration 109, `7c0d0ab7`, v2.6.0): eine Bestellung = ein Topf, gemischter
Warenkorb → zwei Bestellungen, Vermerk auf Anschreiben und Mail, zweite Kundennummer, Backfill,
Lernmittel-Frage im Staging, Topf nachträglich korrigierbar
(`b233b1d3`, v2.7.0).

---

## 06.09.2026

**Rasterdurchgang über die Änderungen desselben Tages** (Peter: „lass bitte die Schemata komplett
über die heutigen Änderungen laufen"). Elf Fragen plus Sweeps-Achse über 97 geänderte Dateien
(+4525/−985), sechs Prüfer. Vierzehn Funde, alle im selben Durchgang behoben und je am Rückbau rot
gesehen (`06116d38` bis `4ef052f1`) — darunter ein rotes `main` seit 14:40 Uhr: Der Postgres-Test
kannte die zwei Schlusszeilen „Nachzügler"/„Aufräumen" nicht, und der Pre-Push-Hook lässt die
`*_pg_test.go` aus.

---

## Pakete beim Schließen der Issues (13.09.2026)

- **#595 Barrierefreiheit:** fertig (v2.5.0, axe-Gate).
- **#596 Landes- und Kreismittel im Bestellwesen:** fertig — erster Schnitt in v2.6.0 (`7c0d0ab7`),
  Rückweg für den Topf in v2.7.0 (`b233b1d3`), zweiter Schnitt am 12.09.2026 in v2.11.0.
- **#597 Landes-Bescheid:** Etappe 1 fertig; aus Etappe 2 Mahnwesen-Auswahl, Liste und
  Rückgabe-Hook (alles in v2.11.0); E3 als Einstellung (Vorgabe 28 Tage). Offen → OFFEN.md 4.4 bis
  4.6, 5.2, 5.3, 8.1, 8.2.
- **#598 Kreis-Rechnung, Altbriefe, Doku:** Art-Auswahl im Schadensdialog (v2.9.0), PII-Matrix der
  Bescheid-Routen, HANDBUCH, Bescheid-Invarianten in `invarianten.md`, FACHKONZEPT-Kategorie
  Schadensersatz. Offen → OFFEN.md 5.4, 8.3.
- **#593 Register-Kleinkram:** fünf Posten erledigt (siehe 12.09.2026). Offen → OFFEN.md 4.14, 6.2.
- **#599 Betrieb:** Demo-Daten, Prod-Secrets, IMAP und SMTP, Selbstanmeldung, `SENTRY_DSN` (siehe
  13.09.2026). Offen → OFFEN.md 4.7, 4.8 und Abschnitt 7.
- **#600 Schule:** D1, D3, D5 beantwortet, D4 intern entschieden (10.09.2026). Offen → OFFEN.md
  Abschnitt 8.
- **#594 Arbeitsreihenfolge:** ersetzt durch die Reihenfolge in OFFEN.md.
