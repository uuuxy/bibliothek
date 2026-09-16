# Offene Arbeit

Stand: 16.09.2026

**Die eine Liste.** Hier steht alles, was noch zu tun, zu prüfen oder zu entscheiden ist — Code,
Betrieb und Schule. Einen zweiten Ort gibt es nicht: Das Befund-Register (`docs/befunde.md`) und
die Issues #593, #594, #597, #598, #599 und #600 sind am 13.09.2026 hierher umgezogen. Erledigtes
wird gelöscht, nicht archiviert (entschieden am 15.09.2026): Die Geschichte steht in den
Commit-Nachrichten, in `git log -p docs/OFFEN.md` und in den geschlossenen Issues.

Andere Dokumente erklären (Konzept, Anleitung, der Katalog der Bugklassen in
[sweeps.md](sweeps.md)), führen aber keine eigene Offen-Liste.

---

## Was jetzt dran ist — in einfachen Worten (Stand 16.09.2026)

Mehr als diesen Block muss niemand lesen, um zu wissen, was als Nächstes kommt. Alles darunter
ist die ausführliche Fassung mit Begründungen; sie ändert nichts an dieser Reihenfolge.

1. **Erledigt am 16.09.2026: die Offline-Theke ist einmal echt ausprobiert.** Dabei kamen zwei
   Befunde heraus (Abschnitt 2): Der Offline-Hinweis ist so groß, dass sich nichts mehr buchen
   lässt, und wer ohne Netz ausgesperrt wird, kommt nicht wieder herein. Beides wird in Stufe 3
   behoben. Offen bleibt der Nachweis für Stufe 2 (Anfragen direkt an den Server).
2. **Zwei kurze Antworten** (Abschnitt 5.16 B): Soll die Ausleihhistorie eines Kollegen
   nach einer Frist gelöscht werden? Soll er für ein verlorenes Buch zahlen? Die beiden
   anderen Fragen sind am 16.09.2026 beantwortet und gebaut: Ein Kollege hat keine Frist
   und wird nie gesperrt.
3. **15 Minuten: einmal durch die Leserdatei gehen** (Abschnitt 5.16 A). Sie ist
   fertig: Der Menüpunkt heißt jetzt „Leserdatei“ und führt Schüler und Kollegium in
   einer Liste, ein Kollege hat eine Akte mit seinen Büchern, die Theke findet ihn über
   den Namen, und „Neuer Leser“ fragt zuerst, wer das ist. Beim Ausweisdruck steht auf der
   Karte einer Lehrkraft seit dem 16.09. „Lehrerausweis“ statt „Schülerausweis“. In der Akte
   steht seit dem Abend des 16.09. auch die Schul-E-Mail: Fehlt sie einem Kollegen aus der
   Zeit davor, trägst du sie dort nach — damit bekommt er seinen Zugang, und die
   Selbstanmeldung legt ihn nicht ein zweites Mal an. Sieh dir an, ob die Wörter stimmen und
   ob dir etwas fehlt.
4. **Freigegeben am 16.09.2026: Stufe 3 wird gebaut.** Das ist die Theke selbst — ein schmales
   Band statt des Blocks, keine Sperre ohne Netz, Weiterarbeiten mit gültiger Sitzung, das
   Nachsenden über die neue Tür und die Meldungsliste.
5. **Erst wenn ein echter Schadensersatz-Bescheid ansteht:** die kleinen Punkte aus 5.2 (Frist
   ohne Grenze, Kassenjahr) — vorher braucht sie niemand.
6. **Liegt bei anderen (Abschnitt 8):** Anfragen an Schule, Schulamt und Schulträger. Hier ist
   nichts zu tun außer nachzufragen, wenn nichts kommt.

Alles andere in dieser Datei — die B-Punkte in Abschnitt 5, die Beobachtungen in 6, die
Betriebspunkte in 7 — schadet niemandem, wenn es liegen bleibt, und wird gebündelt erledigt,
wenn gerade nichts Dringenderes ansteht.

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

1. **Abschnitt 2** Offline-Betrieb der Theke: Stufen 1 und 2 sind gebaut; jetzt die Nachweise
   am Stack (2.3), dann die Freigabe für Stufe 3 (die Ausweis-Formen aus **5.15** werden
   dabei mit entschieden).
1. **5.16** Leserdatei und Rolle Leitung: gebaut, samt Löschen eines Kollegen mitsamt
   Konto. Offen sind nur noch der Blick auf den Stand und zwei Antworten zum Betrieb
   (Lesehistorie, Schadensersatz).
2. **5.1** Schäden und Benutzer.
3. **5.5–5.9**, **5.12**, **5.14** und die B-Punkte aus **5.15** kleine B-Commits.
5. Mahnverfahren: Vor dem ersten echten Bescheid **5.2** und **4.5** (E4), dann **4.4** (E6) und
   **5.13** Stufe 3 (5.3).
6. Nach der Antwort zu E5 (**8.3**): **5.4**.
7. Übrige Entscheidungen aus Abschnitt 4 gesammelt; **5.10**, **5.11** und Abschnitt 6 nur mit
   Anlass.

**Parallel auf der Schulseite:** Abschnitte 7 und 8 — zuerst S3 (7.3), das Littera-Backup (7.2), die
Anfragen E1, E2, E5 (8.1–8.3), B3 und B4 (8.5) und ein Termin für die Abnahmen (7.7). Einen echten LUSD-Import erst nach der
Littera-Übernahme (7.2).

---

## 1. Sofort (Kategorie A)

Nichts offen (Stand 15.09.2026).

---

## 2. Laufende Arbeit: Offline-Betrieb der Theke

**Ziel (entschieden am 13.09.2026):** Bei einem Verbindungsabbruch geht der Betrieb an der Theke normal
weiter.

**Entschieden am 13.09.2026:**

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
  dann die Freigabe.

**Entschieden am 13.09.2026, zweite Runde:**

- Nach einem Neuladen ohne Netz bleibt die Anmeldemaske der Rückfall. Die gespeicherten
  Offline-Scans bleiben auf dem Rechner und werden nach der nächsten Anmeldung mit Netz
  nachgebucht.
- Nachbuch-Meldungen liegen am Server, sichtbar nur mit dem Recht `view_students` (nicht für
  Helfer). Erledigte werden nach der Lesehistorie-Frist gelöscht, höchstens nach 30 Tagen. Offene
  erscheinen nach 14 Tagen als Warnung in der Betriebsbereitschaft. Beim Zusammenführen wandern
  sie mit.
- Gebucht wird mit dem Scan-Zeitpunkt, höchstens der Serverzeit: Ausleih- und Rückgabedatum,
  Frist, Mahnwesen und Lesehistorie rechnen ab dem Scan.
- Ohne Netz sperrt die Theke nicht; geleert wird sie weiter nach 5 Minuten. Kommt das Netz zurück
  und war länger als 15 Minuten niemand da, sperrt sie sofort.
- „Sicherung speichern" bleibt auch auf Anmeldemaske und Sperrbildschirm sichtbar; eingespielt wird
  nur angemeldet.
- Ziffernfolgen: Der Rechner hält eine Liste aller Buch-Barcodes (keine Personendaten, bei der
  Anmeldung aktualisiert). Steht eine Ziffernfolge darauf, ist sie ein Buch; sonst gilt sie als
  unklar und sperrt die Zuordnung, bis ein eindeutiger Ausweis kommt.

**Daraus abgeleitet** (Entscheidung (c) und heutiges Verhalten der Online-Theke):

- Beim Vorbesitzer wird immer zurückgenommen. Scheitert die neue Ausleihe an Sperre, Limit oder
  Vormerkung, wird nur sie als nicht gebucht gemeldet.
- Hat ein abgebrochener Online-Versand schon eine Rücknahme gebucht, holt das Nachbuchen die
  fehlende Ausleihe nach (unter Wächter und Schranken).
- `B-` und `LMF-` werden auch ohne geladene Regeln oder Buchliste gespeichert.
- `/api/action/batch` bleibt eine Version länger bestehen, für Theken-Tabs mit altem Stand.

**Maßstab:** Nicht die Zahl der Randfälle entscheidet, sondern ob an der Theke je ein Buch bei
einem Kind liegt, das im System frei ist, oder umgekehrt. Alles, was das verhindert, gehört hinein.
Alles andere kann warten, bis es im Betrieb vorkommt.

**Stufe 1 ist NICHT gebaut — richtiggestellt am 16.09.2026, 20:30 Uhr, am Stack gemessen.**
Der zweite Durchgang des Stufe-1-Nachweises (Netz gekappt, im Kiosk gescannt) endete mit zwei
Toasts „Netzwerkfehler", und der Scan war weg. Ursache, am Code belegt:

- `speichereOfflineAktion` (`stores/omnibox.svelte.js`) nimmt bis heute NUR `B-`-Barcodes in die
  Warteschlange; alles andere — Littera-Ziffern, `LMF-`, jeder Ausweis — bekommt einen nackten
  „Netzwerkfehler" und wird verworfen. Das ist Wort für Wort Punkt 1 von 2.1, also genau das, was
  Stufe 1 beheben sollte. Die sieben Commits haben die Funktion angefasst (`65f9a998` hat ihren
  Fehlerfall umgebaut), die Beschränkung aber nie entfernt.
- Die Tür, die dem Rechner sagt, welche Ziffernfolge ein Buch ist, EXISTIERT am Server
  (`GET /api/action/buchbarcodes`, `RequirePermission("perform_actions")`) und wurde am 15.09.
  noch verbessert (ausgesonderte Exemplare). **Kein einziger Aufruf im Browser** — Bugklasse „Nie
  verdrahtet". Die lokale Buch-Barcode-Liste aus der Entscheidung vom 13.09. gibt es nicht.
- Kein Gate konnte das sehen: JEDER Fall in `stores/omniboxOffline.test.js` scannt `B-10234`.
  Geprüft ist genau der eine Weg, der funktioniert.

Folge für den Betrieb: Solange das so ist, ist der Satz im Offline-Band („Scannen geht weiter,
die Buchungen folgen von selbst") ein Versprechen, das das System nicht hält. Und weil nie etwas
in die Warteschlange kommt, erscheint auch „Sicherung speichern" nie — der Bediener sieht nur den
Knopf zum Einspielen und fragt sich zu Recht, wo er denn speichern soll. Beide Beschwerden vom
16.09. haben diese EINE Ursache.

Daraus die Lehre, die teurer war als der Fehler: „gebaut" hiess hier „Commits liegen vor". Ein
Nachweis von Hand hat nie stattgefunden, und die Testfälle bestätigten nur den funktionierenden
Fall. Stufe 3 wurde am 16.09. auf diese Grundlage aufgesetzt — was dort gebaut ist (Band statt
Vollbild, keine Sperre ohne Netz), gilt weiter und ist getestet; es zeigt nur eine Warteschlange,
die noch nichts annimmt.

**Nächster Schritt, vor allem Weiteren:** Stufe 1 wirklich bauen — die Barcode-Liste im Browser
halten und auffrischen, alle Buchformen und Ausweise in die Warteschlange lassen, die
Ziffernregel aus der Entscheidung vom 13.09. umsetzen, und je Form ein Testfall statt fünfmal
`B-10234`.

**Stand 15.09.2026 (überholt, siehe oben):** Stufe 1 galt als gebaut — sieben Commits `0fa4b5a3`, `18b4887e`, `44615c42`,
`08312c87`, `65f9a998`, `6fe6ba8b`, `23ca498c`, je ein Rot-Test am alten Code, volle Suite mit
Postgres, Lint und Frontend-Gates grün (Einzelheiten in den Commit-Nachrichten). Abweichungen
vom Plan: Der Warteschlangen-Eintrag behält `id` als Schlüssel (der keyPath des bestehenden
IndexedDB-Schemas, keine Migration); die Absicht heißt `ausleihe`/`rueckgabe` wie die Antworttypen
des Servers; ein Stapel-Eintrag, dessen Buchung gerade läuft, bekommt 503 (nicht 409), damit der
Sync ihn liegen lässt.
**Stand abends:** Stufe 2 (der Server) ist gebaut und gepusht, die CI ist grün.
**Stand spät:** Die Tür-Funde aus dem Rasterdurchgang sind behoben — der Bewegungsstempel läuft
nie rückwärts, die Tür liest die Antwort unter dem Schlüssel und legt ihre Ergebnisse ab, sie
rechnet den Uhrversatz des Rechners heraus, die Barcode-Liste führt auch ausgesonderte Exemplare,
und die Theke hat eine Rückrechnung der Littera-Etiketten mit denselben Prüffällen wie der Server
(Einzelheiten in den Commit-Nachrichten).
**Stand 16.09.2026, abends — der Stufe-1-Nachweis ist von Hand gelaufen** (echter Chrome, Netz
gekappt). Zwei Befunde, beide echt, beide Stufe-3-Arbeit:

1. **An der Theke lässt sich nichts mehr buchen — aus ZWEI Gründen, und der zweite ist der
   eigentliche.** Sichtbar ist der Offline-Hinweis (`components/OfflineIndicator.svelte`): ein
   `fixed`-Block über der Anwendung, mit Riesenschrift und grossen Knöpfen, und bei null Vorgängen
   gleich laut wie bei fünfzig („0 Vorgänge nur auf diesem Rechner"). Blockierend ist aber etwas
   anderes: 25 Sekunden nach dem letzten Herzschlag setzt `App.svelte` `heartbeatOk` auf falsch
   und legt ein `fixed inset-0`-Vollbild mit Weichzeichner über die ganze Seite („VERBINDUNG
   VERLOREN / Reconnecting…"). DAS ist das Vollbild aus Punkt (a) des Plans. Wer nur den roten
   Balken schlanker macht, hat die Theke nicht entsperrt.
2. **Ohne Netz kommt niemand mehr herein.** Zwei Ursachen, die man auseinanderhalten muss. Der
   Sperrbildschirm nach 15 Minuten wirft die Sitzung NICHT weg (sie hält 12 Stunden und erneuert
   sich alle 30 Minuten) — aufgemacht wird er aber mit dem Passwort gegen den Schul-Mailserver,
   und ohne Netz passt der Schlüssel nicht ins Schloss (`stores/idleLock.svelte.js`, `entsperren`
   ruft `/login`). Lädt der Tab neu, kommt die volle Anmeldemaske, und die braucht den
   Mailserver ebenfalls.

**Entschieden am 16.09.2026 — der Nachweis hat die Entscheidung vom 13.09. widerlegt.** Dort
stand: „Nach einem Neuladen ohne Netz bleibt die Anmeldemaske der Rückfall." Das taugt an der
Theke nicht; „Tab neu geladen, kein Netz" hieße Nachmittag vorbei. Stattdessen gilt: Ist das
Sitzungs-Kärtchen im Browser noch gültig, arbeitet die Theke ohne Netz weiter, ohne den
Mailserver zu fragen — eingeschränkt auf das Nötige, also scannen in die Warteschlange ja,
Schülerdaten anzeigen nein. Sobald das Netz zurück ist, wird geprüft. Das erweitert den Umfang
von Stufe 3 um diesen Punkt; die Einschränkung ist der Ersatz für die Prüfung, die gerade nicht
möglich ist.

**Nächster Schritt:** Stufe 3 bauen — Band statt Block, keine Sperre ohne Netz, die eben
entschiedene Weiterarbeit mit gültiger Sitzung, Nachsenden, Meldungsliste. Der Nachweis für
Stufe 2 (Anfragen an die Tür, 2.3) steht weiterhin aus; er gehört dem, der den Umbau gebaut hat,
denn wer nicht weiß, was dort zugesagt wurde, hakt ihn nur ab.

### 2.1 Was heute fehlt (am Code gelesen 13.09., nachgeprüft 14.09.2026)

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

### 2.2 Der Bau in drei Stufen, 17 Commits

Je Stufe: Rot-Test am alten Code, volle Suite mit Postgres, Nachweis am frisch gebauten Stack und
im Browser, dann die Freigabe.

Ratschen, die jeder Commit im Blick hat: 200 Zeilen je Frontend-Datei (`App.svelte` steht auf
genau 200, `Omnibox.svelte` mit 282 im Bestand und darf nicht wachsen); kein SQL in `api/`
(`schichtung_test.go`); `offlineImport.test.js` pinnt die Signatur von `enqueueOfflineAction`;
`omniboxOffline.test.js` ist heute nur durch den Fehler aus Commit 2 grün (der `apiClient.post`-Mock
liefert `undefined`, der TypeError kommt aus `res.ok`); `offlineSync.test.js` pinnt „4xx fliegt raus".

#### Stufe 1 — vorhandene Fehler (7 Commits, je ein Fund)

1. **Schnappschuss beim Scan.** `submitAction` hält vor dem Versand fest: Absicht (Ausleihe,
   Rückgabe), Person (Schüler ODER Lehrkraft), Scan-Zeitpunkt. Der Warteschlangen-Eintrag wird ein
   Objekt (`{art, barcode, schueler_id, lehrer_id, gescannt_am, key}`), `enqueueOfflineAction`
   nimmt es entgegen. `offlineImport.test.js` wird einmal auf das Objekt umgestellt, nicht dreimal.
   Format 1 (alte Sicherungen: `{action_type, barcode_id, schueler_id, timestamp}`) bleibt lesbar.
   Rot-Test: Anfrage hängt, Store-Person wird geleert, Timeout → Eintrag trägt die Person vom Scan.
1. **Nur ein gescheiterter Versand wird eingereiht.** Der `catch` um `verarbeiteAktionsErgebnis`
   reiht nicht ein. Der Mock in `omniboxOffline.test.js` wird auf einen echten Versandfehler
   umgestellt (`apiClient.post.mockRejectedValue(new TypeError('Failed to fetch'))`); das ist der
   Beweis, dass der Test vorher die falsche Sache maß. Rot-Test: 200 mit `{type:'teacher'}` ohne
   `teacher` → Fehlerbanner, keine Warteschlange.
2. **„Buch zurückgeben" im Profil ist offline eine Rückgabe.** `onReturnClick` übergibt die
   Absicht aus Commit 1. Rot-Test: Doppelklick offline → zwei Rückgaben, keine Ausleihe.
3. **Lehrkraft geladen: Offline-Buch ist eine Handapparat-Ausleihe.** Der Eintrag trägt
   `lehrer_id`, `baueBatchPayload` sendet `active_teacher_id` (der Stapel-Endpunkt kennt es
   heute schon). Rot-Test am Payload.
4. **Ein Fehler der Warteschlange heißt nicht „gespeichert".** `enqueueOfflineAction` wirft weiter;
   die Theke meldet „NICHT gespeichert — Buch zurücklegen" mit Fehlerton. `loadQueue` liefert bei
   Fehler nicht `[]`, sondern wirft; das Band zeigt „Warteschlange nicht lesbar" statt 0.
   Rot-Test: IndexedDB wirft.
5. **Erledigt ist nur, was der Server wie gescannt gebucht hat.** Die `!result`-Failsafe fällt
   (Schweigen des Servers ist kein Erfolg). Passt der Antworttyp nicht zur Absicht (Ausleihe
   gescannt, `rueckgabe` gebucht), wird der Eintrag ausgebucht UND gemeldet, mit Barcode und
   beiden Typen; er blockiert nicht. 4xx wird weiter ausgebucht und gemeldet. 5xx und 429 bleiben
   liegen, aber die Runde endet dort, statt sofort erneut zu senden; der nächste Anlauf kommt mit
   dem nächsten `online`-Ereignis oder nach einer Minute. (Das Blockieren der Warteschlange am
   ersten `wiederholen`-Eintrag kommt erst in Stufe 3 mit der neuen Tür, die den Fall
   `bereits_ausgeliehen` kennt; vorher hätte ein legitimer Fall die Warteschlange gesperrt, ohne
   dass es einen Ort gäbe, ihn aufzulösen.)
6. **Idempotenz hält.** Die Antwort wird mit `context.WithoutCancel` gespeichert (Vorbild
   `mahnwesen_bulk_mail.go`); der Schlüssel wird vor der Arbeit reserviert (Zeile mit Marker,
   `response_data` bekommt eine unterscheidbare Form oder wird nullbar mit `status_code = 0`);
   ein zweiter Aufruf auf einen reservierten Schlüssel wartet oder antwortet 409 `in_arbeit`.
   Antworten ab 500 werden weiter nicht gespeichert. Rot-Test (Postgres): zweite Anfrage mit
   demselben Schlüssel NACH dem Commit der ersten und VOR dem Speichern der Antwort → heute
   Rückgabe, danach dieselbe Ausleihe. Der 24-Stunden-Ablauf bleibt; die Nachbuch-Tür (Stufe 2)
   deckt spätere Wiederholungen über den Wächter ab.

#### Stufe 2 — Server: gebaut am 15.09.2026

Fünf Commits, alle Gates grün, jeder mit Rot-Probe am alten Code. Was jetzt steht:

- **Rückholen als Baustein** — Umlauf und Forderung in der Transaktion des Aufrufers; der
  Fund im Fehlbestandsbericht nutzt denselben Baustein (vorher stand das UPDATE zweimal da).
- **Migration 116, Bewegungsstempel** — `ausleihen.erfasst_am` (Scan-Zeitpunkt) und
  `buecher_exemplare.letzte_bewegung_am`, von den Schreibern gesetzt. Sperrreihenfolge
  Schüler → Ausleihe → Exemplar steht in [invarianten.md](invarianten.md).
- **Migration 117, Nachbuch-Meldungen** — jede Abweichung vom Scan bleibt mit Barcode,
  Grund und Beteiligten stehen, bis jemand sie quittiert. Liste und Quittieren mit
  `view_students`, Zähler fürs Band mit `perform_actions`. Quittierte fallen nach der
  Lesehistorie-Frist, höchstens 30 Tage; offene meldet die Betriebsbereitschaft nach
  14 Tagen. Personenspalten wandern beim Zusammenführen, werden getilgt und stehen in der
  Auskunft.
- **`POST /api/action/nachbuchen`** — 1–50 Einträge und die Sendezeit des Rechners
  (`gesendet_am`, Pflicht), je Eintrag eine Transaktion. Scan-Zeitpunkte werden um den
  gemessenen Uhrversatz umgerechnet (Antwort `uhr_versatz_sekunden`), höchstens Serverzeit.
  Wächter gegen veraltete Scans. Unter dem Idempotenz-Schlüssel liest die Tür die gespeicherte
  Antwort: online schon gebucht → `bereits_gebucht`; online nur die Fremdrückgabe → Ausleihe
  ab der Rückgabe nachholen; eigene Ergebnisse werden abgelegt und einer Wiederholung
  zurückgegeben. Rücknahme beim Vorbesitzer vor dem Savepoint, Schranken über die
  Transaktion des Eintrags. Ein Serverfehler beendet den Aufruf nicht — der Eintrag
  bekommt „wiederholen" und bleibt auf dem Rechner.
- **`GET /api/action/buchbarcodes`** — die Barcodes aller Exemplare, auch ausgesonderter,
  bewusst ohne LIMIT (begründet im Kopfkommentar), mit Stand-Merker aus Anzahl und
  jüngster Änderung; unverändert antwortet der Server 304. Gepackt, weil Caddy nicht
  komprimiert. Keine Personendaten.

**Offen aus Stufe 2:** der Nachweis am Stack (2.3, Server-Hälfte) — die curl-Fälle gegen
`/nachbuchen`, `docker stop` der Datenbank während eines Laufs und `pg_stat_activity` beim
Nachbuchen von 200 Einträgen. Die PG-Tests decken die Fälle fachlich ab; der Nachweis
zeigt sie am laufenden Stack.


#### Stufe 3 — Theke (5 Commits)

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
    Runde endet beim ersten `wiederholen`, alles andere wird ausgebucht und gemeldet;
    `bereits_gebucht` wie ein Erfolg (Typvergleich wie Stufe 1, Commit 6). Jede Portion trägt
    `gesendet_am` von derselben Uhr wie `gescannt_am`. Springt die Uhr des Rechners zwischen Scan
    und Versand (Zeitabgleich nach der Netzrückkehr), verfälscht der Sprung die Umrechnung —
    beim Bau eine Uhr wählen, die nicht springt, oder den Sprung mitschicken. Sicherungen
    (Format 2 mit Zeitstempel und Absicht; Format 1 weiter lesbar) werden gemeinsam eingespielt,
    ein `startSync()` für alle Dateien. `/api/action/batch` bleibt eine Version.
15. **Offline alle Buchformen und Ausweise.** Die Buchliste wird nach der Anmeldung geholt
    (`starteHintergrundAbrufe`, `perform_actions`, ETag) und lokal gehalten, ohne Personenbezug.
    `B-` und `LMF-` gelten auch ohne Liste; ein 13-stelliges Littera-Etikett wird zuerst
    zurückgerechnet (`frontend/src/lib/litteraEtikett.js`, gebaut, Prüffälle gemeinsam mit dem
    Server); Ziffernfolge auf der Liste = Buch (die Liste führt auch ausgesonderte); `S-`/`L-` = Ausweis
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
    von Ausweis- und Buchnummern am Theken-Rechner), Rückweg-Anleitung und OFFEN.md.

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

### 2.5 Was die Prüfung am 14.09.2026 am ersten Entwurf geändert hat

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
  Eintrag ab, den die Ableitung oben („abgebrochener Online-Versand") nachholen will.
- Sperrreihenfolge festgelegt (Schüler, Ausleihe, Exemplar): Online sperrt das Exemplar heute
  nie; „Exemplar sperren → Wächter" als erster Schritt hätte gegen den Online-Scan einen Zyklus
  gebaut.
- Uhr je Aufruf statt je Service, Repository-Funktionen mit Datum, beide Handapparat-Fristen,
  23514-Mapping, Schranken über `tx`: fehlten im ersten Entwurf.
- Stufe 3: 15 (neue Formen) nach 14 (neue Tür); ein Band statt zwei; das Netz-Signal für die
  Sperre ist benannt; Format-1-Sicherungen bleiben lesbar; `App.svelte` und `Omnibox.svelte`
  stehen an der 200-Zeilen-Ratsche.
- Der Nachbuch-Bericht war nie ein Bericht, sondern ein Toast (2.1, Punkt 7).
- Antworten ab 500 werden heute schon nicht gespeichert; die befürchtete Endlosschleife am
  gecachten 5xx gibt es nicht, nur das pausenlose Wiederholen (2.1, Punkt 6).

---

## 3. Theke — vor dem Offline-Bau (Kategorie B)

Nichts offen (Stand 15.09.2026); die Nummer bleibt, weil Abschnitt 4 auf 3.4 verweist.

---

## 4. Entscheidungen

Die Nummern bleiben fest. Beantwortete Fragen wandern in den Punkt, der sie umsetzt (4.1 → Abschnitt
2, 4.2 → 3.4) oder fallen weg, sobald sie umgesetzt sind.

### 4.3 `ziel_jahrgang`: bauen oder streichen

`ziel_jahrgang` (mehrjährige Ausleihe) wird in `internal/service/loan_rules.go` gelesen, aber von
keinem Code geschrieben; die Fristregel verzweigt auf einen Wert, der immer 0 ist. Die Frist am
Rückgabetermin ist entschieden (1.4). **Frage:** Feld pflegbar machen oder streichen?

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
- `scripts/tabula_rasa.sql` leert `schadensersatz_nummern` nicht; die Bescheide fallen über
  `TRUNCATE … schueler … CASCADE` mit. Nach Tabula rasa sind alle Bescheide weg, der Nummernkreis
  läuft weiter. Vorher festlegen, ob genau das gewollt ist (Nummern nie recyceln).
- Der Elternbrief je Schadensfall (`GET /api/schadensfaelle/{id}/pdf`, `api/pdf.go`) prüft ebenfalls
  nicht, ob die Forderung auf einem Bescheid steht, und verlangt „bar in der Bibliothek". Die
  Oberfläche öffnet ihn seit dem 15.09.2026 nicht mehr (Mahnverfahren Stufe 1); über die Adresse
  bleibt er erreichbar. Fällt mit dem Entfernen der Altbriefe (5.4) weg, sonst vorher denselben
  Filter wie bei der Ersatzforderung.
- **Drei weitere Kalendertage in der Zeit der Datenbank (UTC), gefunden am 16.09.2026** beim
  Fix der Bescheid-Frist (die rechnet seitdem mit `sqlSchulHeute` in `repository/bescheid.go`).
  Bis 2 Uhr Berliner Zeit ist dort noch der Vortag:
  - Volljährigkeit im Bescheid-Vorschlag (`geburtsdatum <= CURRENT_DATE - INTERVAL '18 years'`,
    `repository/bescheid.go`): Am 18. Geburtstag gilt das Kind bis 2 Uhr noch als minderjährig.
  - Mahnlauf „höchstens einmal am Tag" (`letztes_mahndatum::date < CURRENT_DATE`,
    `api/mahnwesen_bulk.go`): Der Tag wechselt um 2 Uhr statt um Mitternacht.
  - „Heute zurückgegeben" im Mahnwesen (`DATE(rueckgabe_am) = CURRENT_DATE`,
    `repository/mahnwesen_repo.go`): Rückgaben zwischen 0 und 2 Uhr zählen zum Vortag.
  Je Stelle ein Commit mit demselben Zonen-Test wie `bescheid_frist_schulzeit_pg_test.go`.

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
  und Jahrgänge am Server messen (Einzeiler dafür), dann Schema.

### 5.6 Schüler und LUSD

- Das Bearbeiten-Formular schickt das alte `abgaenger_jahr` mit, ein Klassenwechsel rechnet es
  nie neu. Vor der Versetzungs-Abnahme (7.7).
- Das Schülerfoto per Barcode wird auch für Schüler im Papierkorb ausgeliefert
  (`api/photo_serve.go`).
- Purge-Fehler kommen immer als 409 (`api/student_deleted.go`).
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
- Mail-Datum und Link-Frist stehen in Serverzeit.

### 5.8 LMF und Statistik

- Steht eine Klasse zweimal im Plan, nennen Ausleihe und Massenabgleich zwischen den Terminen
  verschiedene Fristen.
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
- Schülerakte: Scheitert der Abruf des Kopfes (`GET /api/schueler/{id}` in 503 oder Netzfehler),
  bleibt die Akte leer — `StudentProfile.svelte` hat nach `{:else if st.profile}` kein `{:else}`.
  Die drei Listen daneben vermerken ihren Ausfall seit dem 15.09.2026 (1.5); der Kopf ist der
  verbliebene Eintrag in `fehlerausgang.test.js`. Ein `{:else}` mit `LadeFehler` braucht Platz:
  die Datei steht an der Größen-Ratsche (244 Zeilen). Gefunden beim Bau von 1.5.

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
  `ghcr.io/uuuxy/bibliothek:x.y.z`, nur das Release (`release.yml`) bleibt aus. Die Produktion baut heute
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
- `FACHKONZEPT.md` (zwei Stellen) und `invarianten.md` nennen `RueckgabeTerminFuerKlasse`, die
  `d3e86287` entfernt hat; nur Abschnitt 2.3 wurde angepasst.
- **Server-Pfad:** `DEPLOYMENT.md` (vier Stellen), `SECURITY.md`, `SCRIPTS.md` und der
  Kopfkommentar von `scripts/pruefe_secrets.sh` sagen `/opt/bibliothek`; auf dem Server liegt
  der Stack in `/root/bibliothek` (`cd /opt/bibliothek` → „No such file or directory", am
  15.09.2026, `docker compose exec` aus `~/bibliothek` lief). Jede kopierte Anleitung scheitert
  am ersten Befehl. **Schritt:** die sieben Stellen auf `/root/bibliothek` — oder, falls
  der Stack nach `/opt` umzieht, umgekehrt; bis zur Entscheidung ist die Doku falsch, nicht der
  Server.

### 5.13 Mahnverfahren: Stufe 3 (nach 4.4)

Das Modell seit dem 15.09.2026 (Stufen 1 und 2): Die Reiter „Alle ·
Akut fällig · Eskaliert" fragen „Wer hat Bücher zu spät?", der Reiter „Schadensersatz" fragt „Wer
schuldet Geld?". Solange das Buch als ausgeliehen gilt, steht das Kind links; sobald ein Verlust
oder Schaden gebucht ist, rechts — mit genau einem Stand und einem nächsten Schritt je Zeile. Der
Bescheid entsteht direkt aus den überfälligen Büchern; der Brief bucht ihren Verlust.

- **Stufe 3**: Folgen der Übergabe (5.3): Übergabe-PDF und Sammelliste für das Schulamt; E6 (4.4).
- Offen aus Stufe 2: Staffelbetrag als Vorschlag im `DamageReportModal` (5.4) — der Weg über die
  Akte fragt den Betrag weiter ohne Vorschlag.

### 5.12 Review der Commits vom 11.–14.09.2026 (14.09.2026)

Rund 85 Commits, von vier getrennten Prüfungen gelesen, die A-Funde selbst am Code nachgeprüft.
Ein A-Fund (1.5), sonst B und C; die Fixes selbst waren richtig, die Funde sind Nachbarn.

- **Rückkehr eines abgeschriebenen Buchs (`e9ac79e6`):** Im Zweig „reserviert für dasselbe
  Kind" ist die Reaktivierung committet, danach läuft `HandleUnifiedCheckout`; scheitert die an
  der Sperre (offene Forderung nach `uebergeben`), geht der Fehler zurück und die Antwort mit
  „Schulaufsicht informieren" wird verworfen. Merker und Liste stehen, der Satz an der Theke
  nicht. Nicht nachgestellt — und nicht nachstellbar: Der Zweig ist seit `daf6b370` (16.06.2026)
  ohne Schreiber, siehe 5.14 (tote Tür „Reserviert für:").
- **`bescheid_rueckkehr.go`:** Stornierungsgrund „Rückgabe am …" mit rohem `time.Now()`; im
  Container (UTC) zwischen 0 und 2 Uhr das Vortagsdatum. `schulzeit.Jetzt()` wie die Schwester in
  `order_pdf.go`.
- **Wächter „Ehemalige mit offenen Vorgängen" (`fa4a2113`):** Nur der Grundausdruck `AbgangSeit`
  ist mit der Löschuhr vereint; die Löschuhr rechnet zusätzlich `GREATEST(…, max(rueckgabe_am),
  max(Schadensfall))`. Ohne Außenwirkung, aber eine dritte Formulierung derselben Frage.
- **Lehrerportal, eigene Anliegen (`dfc9913a`):** Beim ERSTEN Laden ist der „alte Stand" leer;
  503 → kein Abschnitt, kein Zähler → das gestern geschickte Anliegen scheint verloren und wird
  doppelt geschickt. Dieselbe Klasse, die der Commit für drei andere Listen behoben hat.
- **Demo-Löschskript (`566b9fe3`):** löscht Schäden ECHTER Schüler auf Demo-Exemplaren mit, auch
  bezahlte und solche auf einem Bescheid (die Vorschau zeigt sie, das Skript sperrt nicht). Am
  13.09. auf dem Server mit Vorschau 0/0/0 gelaufen, also ohne Wirkung; vor einem zweiten Lauf
  eine Sperre „auf Bescheid → Abbruch" einbauen.
- **Zweiter Cover-Schreibpfad (`ddee5802` nicht mitgezogen):** `update_cover_handler.go` setzt
  `cover_url`, ohne das alte Upload-Cover zu löschen, und meldet ein unbekanntes Buch als 500.
  Gleiche Reihenfolgefrage in `cover_aktualisierung.go`, `endpunkte_cover_retry.go`,
  `cover_service.go`.
- **ISBN-Dublette (`260b1436`):** Der UNIQUE-Constraint fängt nur zeichengleiche Dubletten;
  geprüft wird auf einer bereinigten Kopie, gespeichert der Rohwert. `9783123456789` und
  `978-3-12-345678-9` sind zwei Titel. Ob das Frontend vor dem Senden bereinigt, ist ungeprüft.
- **Ersatzforderung (`436de459`):** Die neue 404-Begründung kommt als roher JSON-Body in den
  Toast (`useStudentProfile.svelte.js`, `String(e)`); der Mensch liest `Error: {"error":…}`.
- **Selbstanmeldung (`d9d84fd3`):** Ein abgelehnter Antrag (Konto bleibt inaktiv) liest
  dauerhaft „Zugang beantragt"; nur `aktiv = true` räumt `zugang_beantragt_am`. Ob gewollt, steht
  nirgends.
- **Ratschen mit Umgehungsweg, heute ohne offene Stelle:** UUID-Ratsche sieht `[]struct` ohne
  `dive`, `x := T{}`, Typen fremder Pakete und `q.Get(…)` nicht; Fehlerausgang-Scanner prüft Form 2
  (`?:`) nur mit `istOkZugriff`, nicht mit der UND-Kette; Schema-Gegenrichtung führt
  `lmf_termine.art -> lmf_plaene` doppelt mit falscher Begründung (Mengenvergleich macht es
  unsichtbar); `uuidPfadParameter` hat keinen Test.
- **`ActiveStudentList` (`dfc9913a`):** „Erneut versuchen" bekommt ohne Callback einen Leerlauf.

### 5.14 Rasterdurchgang 11.–15.09.2026 (15.09.2026)

104 Commits, alle zwölf Fragen über vier Schreibpfade (Bescheid/Schadensersatz, Offline-Theke,
Buch-Routen, Abmelden), benannte Fragen über sieben weitere. Die Gates liefen dabei grün:
golangci-lint 0, `go test ./...` mit den 197 PG-Tests, deadcode deckungsgleich, svelte-check 0/0,
Vitest 620/620. Die Commits vom 11.–14.09. deckt 5.12 schon ab; neu im Fenster sind die vom
15.09. (Offline Stufe 1, Mahnverfahren Stufe 2).

- **Tote Tür „Reserviert für:" — und ein NULL-Scan dahinter** (`internal/service/omnibox_service.go`,
  `versucheReaktivierung`): Der Zweig fragt `zustand_notiz` nach dem Präfix „Reserviert für:";
  geschrieben hat das zuletzt der Schreiber, der am 16.06.2026 mit `daf6b370` fiel (die
  3-Tage-Reservierung läuft seitdem über `bereitgestellt_exemplar_id`). Der Leser überlebte den
  Service-Refactor vom 20.06. Hinter der Tür liest `checkVormerkung` die nullbare Spalte
  `vormerkungen.notiz` in einen nackten `string` — und der Schreiber (`repository/vormerkung.go`,
  `NULLIF($2, '')`) legt jede Vormerkung ohne Notiz als NULL ab; `istBerechtigterReservierer`
  schluckt den Scan-Fehler und meldet „nicht berechtigt". Heute unerreichbar; sobald jemand die
  Notiz wieder schreibt, bekommt das Kind, das den Titel vorgemerkt hat, an der Theke 403
  „Reserviert für: <sein eigener Name>". Der Durchgang vom 15.09. hatte hier zuerst einen
  Idempotenz-Fund gesehen („5xx gibt den Schlüssel frei, obwohl `holeExemplarZurueck` schon
  committet hat") — die Folge stimmt, aber nur in diesem Zweig, und der ist tot; der Test dazu
  wurde deshalb nicht geschrieben. In Prod gezählt am 15.09.2026 (`docker compose exec
  postgres-db psql … WHERE zustand_notiz LIKE 'Reserviert für:%'`): **0**. **Schritt:** Der
  Zweig fällt samt `istBerechtigterReservierer`, `checkVormerkung` und dem Struct `vormerkung`
  mit Rückbau-Probe — zusammen mit Abschnitt 2, Commit 8, weil es derselbe Baustein
  `holeExemplarZurueck` ist; ein `coalesce` für die Notiz braucht es dann nicht mehr.
- **„Aktive Lehrkraft" steht zweimal:** `lower(rolle::text) = 'kollegium' AND aktiv = true` in
  `internal/service/device_service.go` (über `id`) und `repository/user.go` (über `barcode_id`).
  Dieselbe Regel, zwei Pakete, kein gemeinsames Prädikat — kommt eine Bedingung dazu (etwa
  „nicht gesperrt"), steht sie an einer Tür und fehlt an der anderen.
- **Zwei Uhren in `UeberfaelligeAusleihen`** (`repository/bescheid_verlust.go`): gefiltert wird
  mit `CURRENT_TIMESTAMP` (DB), die Staffel rechnet mit `schulzeit.Jetzt()` (Go). Wirkt nur am
  Schuljahreswechsel, dann um eine Stufe. Schwester des `time.Now()`-Punkts in 5.12.

Nachgestellt und **fallengelassen** — damit der nächste Durchgang sie nicht noch einmal findet:
`EmpfaengerFuerBescheid` ohne `deleted_at IS NULL` (Bescheid an ein Kind im Papierkorb —
`pruefeSchuelerLoeschbar` blockiert das Löschen bei offenen Ausleihen und unbezahlten Schäden, es
gibt also nichts für den Brief; Rest ist eine Regel in einer anderen Datei) · Rückkehr storniert
die Forderung, das Buch bleibt ausgesondert (beide Türen nehmen die Aussonderung zurück) ·
„Pool ODER Tx" nur behauptet (der Handler öffnet die Transaktion wirklich) · Rechte-Asymmetrie an
den Buch-Routen (`adminH` IST `RequireEditBooks`, nur der Name führt in die Irre) · Migration 115
droppt `idx_lmf_termine_plan` (der neue Unique-Index deckt dieselben Spalten in derselben
Reihenfolge, und alle Lesepfade filtern auf `plan_id` als Präfix).

### 5.15 Rasterdurchgang über die Stufe-2-Commits (15.09.2026, abends)

Sieben Code-Commits seit 5.14 (Idempotenz-Fristen bis Buch-Barcodes), am Code gelesen und jeder
Punkt am echten Postgres nachgestellt. Behoben am 15.09.2026 spät: vorgehende Theken-Uhr,
„Schlüssel bekannt", wiederholte Portion, Littera-Etiketten und ausgesonderte Exemplare in der
Barcode-Liste (Einzelheiten in den Commit-Nachrichten). Die Nachstellung des Stand-Merkers liegt
weiter hinter dem Build-Tag `raster`: `TEST_DATABASE_URL=… go test -tags raster -run TestRaster_
./repository/`.

- **Frage, heute ohne Befund, nach dem Personenlauf neu — Ausweis-Formen gegen die Offline-Regel** (Frage 3). Der Plan für Stufe 3,
  Punkt 15 ordnet offline nach Vorsilbe: `S-`/`L-` Ausweis, `B-`/`LMF-` Buch, Ziffernfolge auf der
  Liste Buch, sonst unklar. Ein alter Schülerausweis liefert beim Scannen aber `B97601826457`
  (gemessen, Kopfkommentar in `api/buchbarcodes_handler.go`) — offline wäre er „unklar" und
  sperrte die Zuordnung, obwohl er eindeutig ist. Und stünde eine Ausweisnummer zugleich als
  Buch-Barcode im Bestand, buchte die Theke den Ausweis offline als Buch. Die Abfrage (lesend, nur
  Zahlen, lokal gegen den Stack geprüft):
  `docker exec bibliothek-db psql -U postgres -d bibliothek -c "SELECT (SELECT count(*) FROM schueler WHERE deleted_at IS NULL) AS schueler, (SELECT count(*) FROM schueler WHERE deleted_at IS NULL AND barcode_id LIKE 'S-%') AS schueler_s, (SELECT count(*) FROM schueler WHERE deleted_at IS NULL AND barcode_id ~ '^B[0-9]+$') AS schueler_b_ohne_strich, (SELECT count(*) FROM schueler WHERE deleted_at IS NULL AND barcode_id ~ '^[0-9]+$') AS schueler_ziffern, (SELECT count(*) FROM benutzer WHERE aktiv AND barcode_id LIKE 'L-%') AS personal_l, (SELECT count(*) FROM benutzer WHERE aktiv AND barcode_id IS NOT NULL AND barcode_id NOT LIKE 'L-%') AS personal_andere, (SELECT count(*) FROM schueler s JOIN buecher_exemplare e ON e.barcode_id = s.barcode_id) AS gleich_wie_buch_schueler, (SELECT count(*) FROM benutzer b JOIN buecher_exemplare e ON e.barcode_id = b.barcode_id) AS gleich_wie_buch_personal;"`
  Ergebnis auf dem Server (15.09.2026): 32 Schüler, alle `S-`, keine Ausweisnummer gleich
  einer Buchnummer — die echte Schülerschaft steht dort noch nicht; nach dem Personenlauf neu zählen.
  **Empfehlung (Handbuch und Backup, 15.09.2026):** Littera zählt Bücher und Leser getrennt, beide
  ab 1; im Backup ist jede Lesernummer zugleich eine Exemplarnummer. Littera unterscheidet deshalb an
  der Form des Scans, nicht an der Zahl. Offline genauso: `S-`/`L-` und `B` mit Ziffern ohne
  Bindestrich sind ein Ausweis; `B-`/`LMF-`, ein 13-stelliges Littera-Etikett (zurückgerechnet mit
  `frontend/src/lib/litteraEtikett.js`) und eine Ziffernfolge auf der Liste sind ein Buch; nur was in keine Form passt, ist
  unklar und sperrt. Entscheidung vor Stufe 3, Punkt 15.
- **B — Ausweis- und Buchnummer werden nur in der Littera-Übernahme gegeneinander geprüft**
  (Frage 3). Seit dem 15.09.2026 vergibt der Personenlauf keine Nummer, die schon ein Buch trägt.
  Wer von Hand eine Ausweisnummer ändert (Schülerakte) oder ein Buch umetikettiert
  (`UpdateCopyBarcode`), wird nicht gebremst — und die Theke löst eine Nummer ohne Vorsilbe zuerst
  als Buch auf. Auf dem Server gibt es heute keine Überschneidung (Zählung oben).
- **B, am SQL belegt, kein Schaden gefunden — der Bewegungsstempel folgt einer Auswahl, nicht einer Regel** (Fragen 1, 3 und 7;
  `repository/bewegungsstempel_pg_test.go`). Aussondern, Bestandskorrektur, Schadensmeldung und
  Soft-Delete stempeln; „Verloren" und Reaktivieren im Status-Editor (`UpdateCopyStatus`), der
  Inventur-Abschluss (`FinishInventurSession`) und `repository/damage.go` stempeln nicht. Die
  Wirkung ist nach Lesart harmlos — ein Scan vor einer Verlustbuchung erklärt den Verlust eher, als
  dass er ihm widerspricht —, aber die Auswahl steht nirgends. Der Test prüft vier Schreiber als
  feste Liste; ein neuer Schreiber ohne Stempel bliebe grün. Durchgespielt: Ein Scan vor einer
  Verlustbuchung ohne Stempel bucht die Wirklichkeit (das Kind hatte das Buch), und Status-Editor
  wie Inventur weisen verliehene Bücher ohnehin ab.
- **B, Mechanismus bestätigt, heute ohne Auslöser — der Stand-Merker der Barcode-Liste rechnet mit dem Beginn der Transaktion** (Frage 6).
  `max(aktualisiert_am)` trägt `CURRENT_TIMESTAMP`, also den Transaktionsbeginn. Ändert eine lange
  Transaktion (Import) ein Etikett und committet nach einem kürzeren Schreiber, ändern sich weder
  Anzahl noch Maximum: 304, die Theke behält die alte Nummer. Nachgestellt:
  `TestRaster_StandMerkerUebersiehtLangeTransaktion` rot. Heute ändert aber nur
  `UpdateCopyBarcode` einen Barcode, als kurzer Einzelbefehl; Einfügen und Aussondern ändern die
  Anzahl. Erst ein Schreiber, der in einer langen Transaktion umetikettiert, macht es scharf — dann
  hieße die Folge „unklar".

Geprüft und in Ordnung: Rechte an den neuen Routen (Liste und Quittieren `view_students`; Tür,
Barcodes und Anzahl `perform_actions`) · Frist der quittierten Meldungen (Job und Rückstands-Wächter
über dasselbe Prädikat, Obergrenze 30 Tage auch bei 0 oder negativer Einstellung) · keine doppelte
Meldung je Schlüssel · ein Serverfehler je Eintrag wird „wiederholen", nicht Erfolg · Rückgabe vor
der Ausleihe fängt `check_return_date` · Sperrreihenfolge Schüler → Ausleihe → Exemplar im
Nachbuchen gehalten · Barcode-Liste ohne Personendaten, Löschen ändert den Stand über die Anzahl.

### 5.16 Leserdatei und Rolle Leitung — was noch offen ist (Stand 16.09.2026)

Gebaut ist alles: die Lesertabelle, die Rolle Leitung, Theke und Leserdatei, eine Maske für
jeden, die Pflicht-Schul-E-Mail und das Zusammenführen von Kollegen. Die Geschichte dazu
steht in den Commit-Nachrichten vom 16.09.2026 (`git log --oneline --since=2026-09-15`).
Hier steht nur noch, was NICHT fertig ist.

**Das Modell in vier Sätzen** — es erklärt die offenen Punkte darunter:

- Alle Leser stehen in EINER Tabelle `leser` mit der Spalte `art` (Schüler · Lehrkraft ·
  LiV). Ausleihen darf jeder aktive Leser; die Art entscheidet nichts.
- `schueler` ist seither eine Sicht auf `leser` mit `WHERE art = 'schueler'`. Sie hält die
  alte Bedeutung für Klassenlisten, LUSD-Abgleich, Mahnlauf und Löschfristen.
- Kollegium ist keine Rolle, sondern der Grundzustand. Eine Rolle (Leitung, Mitarbeiter,
  Helfer, Admin) erhebt der Admin an der E-Mail-Adresse.
- Konten bleiben daneben und tun nur, wofür sie da sind: Anmeldung und Rechte. Jedes Konto
  zeigt auf seine Leserzeile.

---

**A. 15 Minuten: einmal durch die Leserdatei gehen.** Sieh dir an, ob die Wörter
stimmen und ob dir etwas fehlt. Was ein Kollege bewusst NICHT in seiner Akte hat, und warum:

- **Kontoauszug, Ersatzforderung, DSGVO-Auskunft** — sie gehören der Schülerarbeit und lesen
  alle die Sicht `schueler`. Ausgeblendet statt kaputt.
- **Löschen** — steht seit dem 16.09.2026 auch beim Kollegium in der Gefahrenzone. Mit dem
  Eintrag fällt sein Zugang; wird er aus dem Papierkorb zurückgeholt, kommt er ohne Zugang
  zurück, und die Schul-E-Mail in der Akte legt ihn neu an.

**B. Zwei offene Antworten zum Betrieb.** Zwei der vier Fragen sind am 16.09.2026
beantwortet und umgesetzt: Ein Kollege hat **keine Frist und wird nie gesperrt**. Seine
Ausleihe ist eine Dauerleihe, und eine Dauerleihe wird nirgends überfällig — die Akte zeigt
„ohne Frist", die Leserdatei zählt ihn nicht, die Übersicht führt ihn nicht als Mahnfall,
und eine Ersatzforderung aus Fristüberschreitung entsteht nicht. Gemahnt wurde er ohnehin
nie: Der Mahnlauf liest die Sicht `schueler`. Offen bleiben:

1. **Soll die Befristung der Lesehistorie auch fürs Kollegium gelten?** Heute nein.
2. **Soll ein Kollege für ein verlorenes Buch zahlen?** Heute entsteht die Forderung, steht
   aber in keiner Übersicht — die Einzelheiten in 5.17, Fund 1. Sie hängt jetzt NICHT mehr
   an einer Frist; wenn das gewollt ist, ist es eine eigene Entscheidung.

**C. Der Nummernkreis bleibt unangetastet — bewusst.** Gemessen auf dem Testserver
(16.09.2026): Exemplare 1…122.127 (30.658 nackte Littera-Nummern, 4.065 `LMF-`, 65 `B-`),
Leser 33 Zeilen, genau EINE Nummer wäre ohne Vorsilbe doppeldeutig. Aus `littera_sav.mdb`:
Exemplare 1…61.512, Leser 0…3.531 — 1.990 von 1.991 Lesernummern sind dort zugleich
Exemplarnummern. Littera lebt damit, weil seine Scanformen sich unterscheiden. Ein
gemeinsamer Nummernkreis ohne Vorsilben wäre möglich, brächte aber nichts, solange die
Offline-Theke die Vorsilbe braucht: **Ohne Netz ist sie die einzige Information, an der die
Theke einen Buchscan von einem Ausweisscan unterscheiden kann.**

**D. Entschieden und nicht mehr zu diskutieren** (steht hier, weil die Frage sonst wiederkommt):

- Die E-Mail eines Kollegen wird NICHT in `leser.eltern_email` abgetippt. Die Spalte gehört
  dem LUSD-Import und heißt auch in der DSGVO-Auskunft „Eltern-E-Mail". Die Adresse steht
  eindeutig am Konto (`benutzer.email`, `UNIQUE lower(email)`); eine zweite Kopie wäre die
  zweite Tür zu derselben Identität. Seit dem 16.09.2026 steht sie in der Akte — gelesen vom
  Konto, und NACHTRAGBAR, solange keine da ist (dann entsteht das Konto). Steht eine da, ist
  das Feld eine Anzeige; geändert wird sie in der Benutzerverwaltung.
- Die Leitung sieht den Menüpunkt „Einstellungen" weiter und darf darin LUSD & Versetzung,
  Datenverwaltung, LMF-Aktionen und Lieferanten bedienen; verschlossen sind Schule, Fristen
  und Mailversand (`manage_settings`) sowie Benutzer & Rechte (`manage_users`). Ein Menüpunkt
  ist kein Recht, sondern ein Sammelpunkt über sechs Kategorien — was gilt, beweist die
  Antwort des Servers. Bestätigt am 16.09.2026.

### 5.17 Rasterdurchgang über den 15. und 16.09.2026 (Funde vom 16.09.2026)

Die zwölf Fragen über alle Änderungen beider Tage. Die Commits vom 15.09. hatten ihren
Durchgang am selben Abend (5.15); alles ab Mitternacht — Rolle Leitung, Migrationen 121 bis
125, Leserdatei, Theken-Suche — war ungeprüft. Gates beim Durchgang: golangci-lint ohne
Befund, svelte-check 0/0, Frontend-Tests 668 grün, Go-Suite mit echtem Postgres grün.

Sechs Funde, vier davon noch am selben Tag behoben und hier gelöscht: der Test außerhalb von
Git (`4edcf1b8`), das Zusammenführen eines doppelt stehenden Kollegen (`bf36df57`), die nicht
mehr änderbare Art (`cd46fc44`, `ecd007bd`) und die veralteten Zahlen in
`docs/invarianten.md`. Übrig sind die beiden, an denen eine Entscheidung hängt.

**1. Ein Kollege bekommt eine Forderung, die in keiner Liste steht.** „Verlust/Schaden melden"
steht in seiner Akte (`StudentProfile.svelte`, nur am Recht `bearbeiten`, nicht an der Art),
und `meldeSchaden` nimmt den Schuldner aus der Ausleihe — die Forderung entsteht also. Am
echten Postgres nachgestellt: In seiner Akte steht sie (1), im Reiter „Schadensersatz" nicht
(0, `bescheid_ausstehend.go` verbindet mit der Sicht `schueler`), und „Bescheid erstellen"
antwortet „Schüler nicht gefunden" (404), weil `EmpfaengerFuerBescheid` dieselbe Sicht liest.
Das Geld ist offen und taucht in der Übersicht nie auf. Zu entscheiden ist zuerst, ob ein
Kollege überhaupt einen Bescheid bekommt; danach entweder die Liste um ihn erweitern oder die
Türen in seiner Akte schließen.

**2. Ein Kollege kommt nicht mehr aus der Leserdatei heraus.** Das Löschen ist in der Akte für
Kollegen ausgeblendet (richtig so), und `DeleteStudent` schreibt auf die Sicht `schueler` —
für einen Kollegen also 0 Zeilen und „student not found". Wird sein KONTO gelöscht, bleibt die
Leserzeile samt Ausweisnummer stehen (`benutzer.leser_id` steht auf ON DELETE SET NULL, und die
Prüfung davor verweigert nur bei offenen Ausleihen). Ein zweites Konto derselben Person erzeugt
dann über den Trigger eine zweite Leserzeile. Reparieren lässt sich das seit dem 16.09.2026
(Zusammenführen, `bf36df57`); es entstehen lassen sollte man es trotzdem nicht. Zu klären ist
mit dem Löschen zusammen, was mit dem KONTO geschieht, wenn die Leserzeile eines Kollegen
gelöscht wird — Papierkorb und Papierkorb-Ansicht schreiben deshalb bewusst weiter gegen die
Sicht.

**Was der Durchgang ausdrücklich in Ordnung fand:** die Sicht-Falle (`CREATE VIEW … SELECT *`
friert die Spalten ein) hat ihr eigenes Gate (`db/sicht_schueler_vollstaendig_pg_test.go`); das
Gegenrichtungs-Inventar ist zu allen zehn Migrationen nachgezogen; die Rechte der Leitung sind
abgeleitet statt abgeschrieben (Migration 122 aus den ADMIN-Zeilen, `db/rolle_leitung_test.go`
aus der Vorgabe); Mahnlauf, LUSD-Abgleich und Löschjob lesen weiter Schüler; die Dauerleihe
hängt jetzt an der Art statt an der Tabelle; der Name eines Kontos und seiner Leserzeile werden
in einer Transaktion geschrieben, und ein stiller Null-Treffer ist dort ein Fehler.

---

### 5.18 Rasterdurchgang über die Leserdatei-Arbeit (16.09.2026, abends)

Umfang: die 22 Commits seit dem Durchgang aus 5.17, also die ganze Leserdatei-Arbeit
(Anlegen, Ändern, Löschen, Zusammenführen, Schul-E-Mail, Ausweis-Vorsilbe, keine Frist fürs
Kollegium). Keine neue Migration in diesem Zeitraum — Frage 12 hing am eingefrorenen Inventar.
Drei Funde, jeder am laufenden Pfad nachgestellt.

**1 · Eine abgelehnte Zugangsanfrage lässt ihre Leserzeile stehen (A, teilweise behoben).**
Die Selbstanmeldung schreibt ein Konto ohne `leser_id`; der Wächter `trg_benutzer_hat_leserzeile`
hängt eine frische Leserzeile daran. `DeleteUser` (`repository/audit_users.go`) löscht nur die
Kontozeile — die Leserzeile bleibt als Waise in der Leserdatei stehen, ohne Ausweis, mit dem aus
der Adresse geratenen Namen. Der **Regelfall** ist seit `e72c11ab` weg: Wer schon in der
Leserdatei steht, wird über „das ist dieselbe Person" zugeordnet statt gelöscht. **Offen bleibt
der Ablehnungs-Fall:** eine Anfrage von jemandem, der gar nicht in die Leserdatei gehört, wird
weiterhin gelöscht und hinterlässt die Waise.

Nachstellung (rot gesehen am echten Postgres, über den Handler): Eine Zeile
`INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, zugang_beantragt_am)
VALUES (…, 'kollegium', false, CURRENT_TIMESTAMP) RETURNING leser_id` anlegen — der Wächter
liefert die Leserzeile zurück —, dann `DeleteUserHandler` auf das Konto rufen und zählen, ob die
Leserzeile noch da ist. Sie ist es.

Die Regel für die Löschung ist die eigentliche Arbeit, nicht der Code: Die Leserzeile darf nur
mitgehen, wenn sie unberührt ist. Eine Aufzählung im Go-Code hält das nicht — an `leser` hängen
acht Fremdschlüssel mit gemischter Löschwirkung (`ausleihen` und `schadensfaelle` RESTRICT,
`schueler_fotos` und `vormerkungen` CASCADE, `schadensersatz_bescheide`, `nachbuch_meldungen`
zweimal und `benutzer` SET NULL), und die nächste Tabelle, die jemand anhängt, steht in keiner
Aufzählung. Deshalb: explizite Prüfung PLUS eine Ratsche, die die Fremdschlüssel auf `leser` aus
der Datenbank liest und gegen eine Liste hält. Der Ausweis ist der Sonderfall, den keine
Fremdschlüssel-Abfrage sieht — er steht als Spalte in der Zeile.

**2 · Die Dauerleihe eines Kollegen wird als überfällig gefärbt (B).** In der Akte am 16.09.
behoben; zwei weitere Ausgänge blieben: `components/BorrowersListe.svelte` und
`utils/ausleiherDruck.js` färben die Frist rot, sobald das Datum vorbei ist. Beide hängen an
derselben Abfrage (`api/copy_admin.go`, Ausleiher eines Titels), und die holt `ist_handapparat`
gar nicht erst ab. Das Gate gehört deshalb an den fertigen Inhalt, nicht an die Komponente.

`repository/ueberfaellig_regel_test.go` konnte das nicht finden: Es sucht den Vergleich
`rueckgabe_frist < CURRENT_TIMESTAMP` im SQL, aber diese Abfrage vergleicht nichts — sie liefert
das Datum aus, verglichen wird in JavaScript. Die Ratsche misst die falsche Schicht; der
Gegenpart im Frontend fehlt.

Am Schulserver gemessen (16.09.2026): 0 offene Kollegen-Ausleihen ohne das Merkmal — bei 0
offenen Kollegen-Ausleihen überhaupt. Es gibt also nichts zu reparieren, und die erste Zahl
allein hätte nichts bewiesen. Entschieden wird weiter an `ist_handapparat`, nicht an
`klasse = 'Lehrer'`: Das wäre die zweite Wahrheitsquelle von der anderen Seite.

**3 · Eine Einstellung außerhalb ihres Bereichs wird still ersetzt (B).**
`repository/system_settings_patch.go`, Helfer `zahl(key, v, min, ersatz)`: Liegt der Wert unter
dem Mindestwert, wird er durch einen Ersatzwert getauscht und als „gespeichert" gemeldet. Eine
Obergrenze gibt es bei keiner der 15 Zahlen. Gemessen: Eingabe 0 für `max_ausleihen_schueler`
wird als 5 gespeichert; Eingabe 999999 für `max_overdue_items` wird unverändert übernommen — und
damit lässt sich die Sperr-Automatik, eine Invariante des Katalogs, aus der Oberfläche
abschalten. Dazu protokolliert `api/settings.go` den Request, also die Eingabe, nicht das
Gespeicherte: Steht 0 drin, steht 0 im Protokoll.

Die Asymmetrie sitzt in derselben Datei: Sommerferien, Eingangsjahrgänge und LMF-Stichtag werden
geprüft und bei Unsinn mit 400 abgelehnt — ausdrücklich, weil Unlesbares sonst „gespeichert,
angezeigt und beim Lesen still auf die Vorgabe zurückgeworfen" wurde (Rasterdurchgang
06.09.2026). Auf der Zahlen-Seite steht derselbe Fehler noch fünfzehnmal.

**Was daraus für das Raster folgt (Vorschlag, noch nicht umgesetzt).** Eine neue Frage 13 —
*Bedeutungswechsel unter gleichem Namen: Hat dieser Name seit gestern eine andere Bedeutung, und
wer liest ihn noch in der alten?* Beleg: Beim Durchgang aus 5.17 gab es 15 Schreibpfade gegen die
Sicht `schueler`, heute sind es 9; die sechs, die gewandert sind, sind Zeile für Zeile die Funde
jenes Durchgangs. Dazu der mechanische Anker, ohne den die Frage zum Spruch verrottet: ein
Bestand der Schreibpfade gegen eine Sicht, je Zeile mit Begründung (heute neun, alle richtig —
LUSD, Versetzung, Löschjob, Seed, Littera-Schülerlauf). Eine zehnte Zeile ist eine Frage.

Dazu eine Schärfung von Frage 12 ohne neue Nummer: *Wer räumt weg, was die Datenbank selbst
angelegt hat?* Das Inventar fragt bei Fremdschlüsseln in die Löschrichtung, bei Triggern nur in
die Anlegerichtung. Vier Trigger schreiben in fremde Tabellen, zwei davon legen Zeilen an, die
kein Go-Code je löscht: `leser` aus `konto_hat_leserzeile` (Fund 1) und `klassen` — für die es im
ganzen Go-Code kein `DELETE` gibt, eine vertippte Klasse steht ab dann in jeder Auswahlliste.

**Ausdrücklich NICHT vorgeschlagen**, damit die Frage nicht wiederkommt: eine Frage zur
Barrierefreiheit (hat ein Gate, andere Achse), zum Betrieb (drei Doku-Tests gaten das schon), zur
Eingabeform (Gate seit 13.09.) und zum Rückweg eines Releases (Down-Migrationen wären ein großes
Projekt gegen ein Risiko, das mit dem Nachtbackup bewusst angenommen ist). Geld wurde geprüft und
für unauffällig befunden: Die Summe eines Bescheids wird zweimal gerechnet, im Browser und in Go,
beide als Fließkommazahl — eine erreichbare Abweichung liess sich aber nicht konstruieren, weil
der Fehler bei zweistelligen Eingaben weit unter einem halben Cent liegt und `NUMERIC(10,2)` der
Anker ist. Bleibt als Fleck ohne Fund notiert, nicht als Arbeit.

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
13.09.2026 leer. **Erst nach 5.2 und 4.5 eintragen:** Mit den Nummern entstehen echte Bescheide.

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
