# 002 — Umfang gegen Anwendungsfall

Stand: 18.09.2026

**Befund.** Der Verdacht lautet: viel Programm für einen einfachen Zweck. Er ist berechtigt
und trifft trotzdem nur zur Hälfte — der Zweck ist nicht einfach. Beides ist belegbar,
deshalb steht es hier und nicht als Urteil.

**Beleg.** Gemessen am 18.09.2026:

| | |
| --- | --- |
| Go-Produktivcode | 65.883 Zeilen |
| davon `api/` | 27.810 Zeilen in 153 Dateien |
| davon `repository/` | 12.956 Zeilen |
| Routen · Migrationen · Tabellen | 206 · 134 · 42 |
| Auslegung | ~1.900 Leser, 8 Arbeitsplätze |

**Was den Umfang verursacht** — nicht „Ausleihe", sondern ein Dutzend eigener Regelwerke,
jedes mit eigenem Zustand, eigenen Fristen und eigenen Dokumenten:

Lernmittelfreiheit samt Stichtag und Plan · Mahnwesen mit Mahnstufe und Bescheid ·
Schadensersatz mit Abwertungsstaffel · Bestellwesen mit getrennten Mitteltöpfen und
Händlerbestätigung · Inventur mit Sitzungen · Geräteausleihe · LUSD-Abgleich ohne
Schüler-ID · Altbestandsübernahme · DSGVO-Löschketten mit Karenz · zwei öffentliche Seiten
· Kollegiumsportal mit Selbstanmeldung · Offline-Betrieb der Theke · Barrierefreiheit.

Streicht man diese Liste, bleibt ein kleines Programm. Nur streicht sie niemand: Jeder
Punkt kommt aus einer Vorgabe des Landes, des Trägers oder des Betriebs.

**Der zweite Teil des Befunds ist aber wahr:** Der Umfang liegt nicht gleichmäßig. `api/`
trägt mit 42 % des Produktivcodes nicht nur die HTTP-Schicht, sondern auch Fachlogik —
LUSD-Parser, Selbstprüfung, Bestellwesen. Das Schichtungs-Gate (`api/schichtung_test.go`)
hält dagegen, aber **datei-granular**: Eine Bestandsdatei darf beliebig SQL dazubekommen.
Das ist als D4 in [Kapitel 11](../11-risiken-und-technische-schulden.md) vermerkt.

**Was es kostet.** Für den Betrieb heute nichts — das System läuft. Es kostet bei jeder
Änderung im Umfeld von `api/`: Wer dort etwas sucht, sucht in 153 Dateien, und wer etwas
hinzufügt, bekommt vom Gate keine Warnung, wenn er es an der falschen Schicht tut.

**Was daraus folgt.** Als Befund: nichts Sofortiges. Er ist die Gegenrechnung, die man
kennen muss, bevor man „vereinfachen" sagt — die Prüfung selbst steht als
[Idee 003](../ideen/003-schlankerer-schnitt.md), der Blick nach außen als
[Idee 004](../ideen/004-vergleich-mit-openlibry-und-openbiblio.md). Wenn der Schnitt
angefasst wird, dann an der Grenze `api/` ↔ `internal/service`, und zuerst am Gate: Ein
datei-granulares Gate, das nur den Bestand einfriert, verhindert keinen Zuwachs an der
falschen Stelle.
