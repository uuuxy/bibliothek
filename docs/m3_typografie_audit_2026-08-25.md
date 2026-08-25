# Material-3-Prüfung der Typografie — gemessen im Browser (25.08.2026)

**Anlass:** Peter, Druck-Center › Fehlende Etiketten: „mir scheint alles etwas klein oder täuscht es?"
Kurz davor dasselbe bei Wünsche & Meldungen („8G4 ist kaum zu lesen").

**Antwort vorweg: Es täuscht nicht.** Rund 90 % aller gerenderten Textknoten in den großen
Tabellen liegen auf **12 px oder darunter** — Material 3 sieht für Tabellenzellen und
Listen-Nebentext **14 px** vor. Die Skala im Projekt ist korrekt definiert
(`theme-mass.css`), aber die Rollen werden eine Stufe zu tief _benutzt_: `text-xs`
(body-small, 12) steht dort, wo `text-sm` (body-medium, 14) hingehört, und
`text-label-small` (11) trägt Kennungen, die man abliest.

## Wie gemessen

Nicht per Grep — die statische Inventur hat hier zweimal gelogen (Klassen in Variablen,
Vererbung). Ein Playwright-Lauf gegen die lokale App hat als Admin alle 14 Menüpunkte
und deren Unterreiter geöffnet (31 Ansichten, dazu die Theke mit geladenem Schüler per
blind getipptem Barcode) und für **jeden sichtbaren Textknoten** `getComputedStyle`
(Größe, Gewicht) sowie die Rolle (Tabellenzelle, Listenzeile, Knopf, Reiter, Chip,
Überschrift) erfasst; nebenbei die Höhe aller Knöpfe, Felder, Reiter.
Rohdaten je Ansicht: Scratchpad `typo-*.json` (Sitzung 25.08.).

**Nicht gemessen:** Dialoge/Modals, Drilldowns (Buch-Akte, Schülerprofil, Bestell-Detail),
Kollegiums-Portal, Monitor, Login. Das ist ein Viertel der Oberfläche; die Befunde unten
gelten dort vermutlich genauso, sind aber nicht belegt.

## M3-Sollwerte, an denen gemessen wurde

| Rolle                                 | M3                     | Projekt-Klasse                 | Anmerkung                                                  |
| ------------------------------------- | ---------------------- | ------------------------------ | ---------------------------------------------------------- |
| Listenzeile Headline                  | body-large 16          | `text-base`                    |                                                            |
| Listenzeile / Tabellenzelle Nebentext | body-medium 14         | `text-sm`                      | M3 hat keine Web-Tabellenspezifikation; M2 Data Table = 14 |
| Tabellenkopf                          | 12–14 medium           | `text-xs`/`text-sm`            |                                                            |
| Knopf-Beschriftung                    | label-large 14         | `text-sm`                      | **M3 kennt keinen 12-px-Knopf**                            |
| Reiter-Beschriftung                   | title-small 14         | `text-sm`                      | Höhe 48 (Projekt: 30–34)                                   |
| Chip                                  | label-large 14         | `text-sm`                      | nur Zähler-Badges sind label-small 11                      |
| Kleinstes Beiwerk                     | label-small 11         | `text-label-small`             | für Wörter, nicht für Sätze oder Kennungen                 |
| Schriftgewichte                       | 400 / 500              | `font-medium`, `font-bold`→500 | 800/900 gibt es in M3 nicht                                |
| Control-Höhe                          | 40 (Knopf) / 56 (Feld) | 36                             | **bewusste Projektentscheidung** (28.07.), bleibt          |

## Befunde, nach Menge gerendeten Textes

### 1 · Tabellen-Nebentext auf 12 px statt 14 — der Hauptgrund für „alles etwas klein"

| Ansicht                           |                         Zellen auf 12 px | Beispiel                           |
| --------------------------------- | ---------------------------------------: | ---------------------------------- |
| System-Logs                       |               3.000 (+1.000 UUID auf 11) | „25.8.2026, 18:50:26", „ausleihen" |
| Abgänger / Schülerdatei › Archiv  |                                    2.828 | „Sperre aktiv"                     |
| Mahnwesen                         |                                    2.188 | „75 Tage überfällig"               |
| Schülerdatei                      |              1.500 (+500 Barcode auf 11) | „NL", „Alles ok"                   |
| Druck-Center › Fehlende Etiketten |                                      532 | Barcode „B-10016"                  |
| Medienkatalog › Titel-Verwaltung  |                                      256 | „Autor"                            |
| Ausleihe-Theke                    | Tabellenkopf 12, „Geliehen: 8.8.2026" 12 |                                    |
| Bestellhistorie                   |               Tabellenkopf 12, Status 11 | „Wartet auf Händler"               |

Die Primärspalte (Name, Titel) steht überall richtig auf 16 oder 14. Es ist die **zweite
Spalte** — Klasse, Barcode, Datum, Status —, die durchgehend eine Stufe zu klein ist. Genau
das, was man beim Arbeiten abliest.

Quelldateien: `MahnwesenTable`, `AbgaengerTabelle`, `ActiveStudentList`, `EtikettenNachdruck`,
`FehlbestandBericht`, `BookTable*` (inventur), `AuditLog*`, `KioskLoanTable`.

### 2 · Kennungen auf 11 px (label-small) — Präzedenzfall ISBN (a54bc62a)

Schüler-Barcode in der Schülerdatei (500 Zellen), UUID in den System-Logs (1.000 Zellen).
Eine Kennung, die man gegen einen Ausweis oder ein Ticket abgleicht, ist kein Beiwerk.
Die ISBN auf der Buchkarte hat am 08.08. genau diesen Weg genommen: 11 → 14 px `font-mono`.

### 3 · 12-px-Knöpfe — `Button size="sm"` (24 Aufrufer)

Geräte-Liste 74×, Inventur 10×, Statistiken 10×, Druck-Center 10×, Berechtigungen 2×,
Klassensatz-Liste. `sm` = `h-7 text-xs` (28 px, 12 px Schrift). M3 kennt keinen Knopf unter
label-large 14; die 28 px Höhe fallen zudem aus der 36-px-Linie. Vorschlag: `sm` = `h-8 text-sm`
(32/14) — eine Zeile in `Button.svelte`, dann alle 24 Aufrufer auf einmal.

### 4 · Status-Chips auf 11 px

Geräte „im Schrank" (37×), Bestellhistorie „Wartet auf Händler", Kiosk-Status. M3-Chips tragen
label-large 14; 11 ist für Zähler-Badges (die „370" am Reiter — dort korrekt).
Ein Bauteil: `ui/StatusChip.svelte`.

### 5 · Schriftgewichte 800/900 an 25 Stellen

`font-extrabold`/`font-black` entgehen der Theme-Abbildung (`bold`→500): Katalog-Kacheln
(Titel 14 px w800, Zähler 22 px w800), „Wareneingang bearbeiten" 28 px w900, „Gesamtausgaben"
28 px w900, BookAkteMeta 5×, Monitor 4×. M3 kennt nur 400/500. Fix: beide Gewichte in
`theme-mass.css` auf 500 abbilden (wie `bold`) + Ratsche in `frontend-hygiene.test.js`.

### 6 · Sätze auf label-small (11 px)

Druck-Center Hilfstexte („Zuerst oben einen Titel …", 121 Zeichen), Statistik-Achsen,
Bestellhistorie-Status. label-small ist für ein, zwei Wörter; ein Satz gehört mindestens
auf body-small 12, Hilfstext unter Feldern in M3 auf body-small.
Dateien: `LabelBarcodeSchritt`, `LabelLayoutOptionen`.

### 7 · Überschriften-Hierarchie schwankt

`<h2>`/`<h3>` gemessen auf **12 px** (Wareneingang „Erwartete Positionen", Statistiken
„Aktivität pro Monat"), 14, 16, 22, 24 (Inventur), 28 w900, 32 (Kiosk-Name).
Tabellenköpfe auf 11 (Statistiken), 12 und 16 w700 (Statistiken „Monat"). M3: title-large 22
für Seitenabschnitte, title-medium 16 für Karten, Tabellenkopf 12–14 medium.

### 8 · Reiter: drei Höhen, keine davon M3

Gemessen 30, 32 und 34 px (Medienkatalog, Druck-Center/Schülerdatei, Bestellungen).
M3-Tabs sind 48 px; das Projekt darf dichter sein, aber **eine** Höhe. Plus Nebenbefund B
aus dem Register (Beschriftungen brechen unter 1280 px um). Ein Bauteil: `ui/Reiter.svelte`.

### 9 · Knopf-Höhen außerhalb des Bauteils

Theke: 10 verschiedene Knopf-Höhen (18–48). Statistiken 24/28/29. Berechtigungen 24 (66×).
Das sind überwiegend Icon- und Listenknöpfe, die nicht über `ui/Button` laufen — die
36-px-Regel gilt dort nicht, aber drei bis vier Stufen reichen (M3: 40 Standard, 24/32 Icon).

### Konform — kein Befund

Reiter-Beschriftungen 14/500 ✓ · Primärzellen 16/14 ✓ · Absätze 14 ✓ · Abschnitts-Überschriften
22/16 w500 ✓ · Eingabefelder durchgehend 36 px (bewusst) ✓ · Knöpfe md 36 ✓ ·
Klassensatz/Anliegen seit heute 16/14 ✓ · Ausweis-Vorschau (5–13 px) und Cover-Platzhalter
(8/10 px) sind Zeichnungen, keine UI ✓ · Skalen-Definition inkl. Zeilenhöhe und Laufweite ✓

## Vorschlag: Reihenfolge

| #   | Schritt                                                                               | Hebel           | Aufwand            |
| --- | ------------------------------------------------------------------------------------- | --------------- | ------------------ |
| 1   | Tabellen-Nebentext `text-xs`→`text-sm` in 8 Tabellen; Tabellenkopf einheitlich 12/500 | ~10.000 Zellen  | ½ Tag              |
| 2   | Kennungen (Barcode, UUID) → `text-sm font-mono`                                       | 1.500 Zellen    | 1 h                |
| 3   | `Button size="sm"` → 32/14 (eine Zeile)                                               | 24 Aufrufer     | ½ h + Sichtprüfung |
| 4   | `StatusChip` 11 → 12 (label-medium-Äquivalent)                                        | alle Chips      | ½ h                |
| 5   | Gewichte 800/900 → 500 im Theme + Ratsche                                             | 25 Stellen      | 1 h                |
| 6   | `ui/Reiter` eine Höhe + kein Umbruch (Nebenbefund B)                                  | 5 Reiterleisten | 1 h                |
| 7   | Sätze von label-small auf body-small                                                  | 6 Stellen       | ½ h                |
| 8   | Überschriften-Rollen: h2 = 22, h3 = 16, nie darunter                                  | ~8 Stellen      | 1 h                |

**Gate, das rot werden kann:** dieser Messlauf als `e2e/typo-rollen.spec.js` — „kein Text in
`td`/`li` unter 14 px außer in `.chip`/Badge; kein `button`-Text unter 14 px; kein
Gewicht > 500". Erst mit altem Code rot sehen, dann fixen.

Zwei Dinge sind **keine** Verstöße, sondern Entscheidungen, die stehen bleiben: die 36-px-
Control-Höhe (statt M3 40/56) und die dichte Tabellendarstellung an sich — Bibliotheksarbeit
ist Listenarbeit, M3 „dense" ist legitim. Was nicht legitim ist: Nebentext eine Rolle unter
dem, was M3 als kleinste Lese-Rolle vorsieht.
