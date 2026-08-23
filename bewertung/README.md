# Bewertung der Schulbibliothek-Datenbank

Diese Mappe enthält eine unabhängige, **nur lesende** Prüfung der Datenbankstruktur des selbst gebauten Bibliothekssystems. Es wurde nichts am System verändert.

> ### Umsetzungsstand (23.08.2026): alle acht Befunde abgearbeitet
>
> Die Berichte darin sind **unverändert** — sie sind ein Prüfprotokoll, kein Aufgabenzettel,
> und werden nicht nachträglich umgeschrieben. Was seither passiert ist, steht hier:
>
> | Befund | Erledigt durch | Am Code geprüft am 23.08.2026 |
> |---|---|---|
> | **F1** — Notizfeld steuert das Bestellwesen | Migration 071 gibt dem Zustand eine eigene Spalte `bestellstatus` | ✅ 084 zitiert die Lehre ausdrücklich |
> | **F2** — jeder Programmteil sieht den ganzen Schülerdatensatz | [`docs/PII_MATRIX.de.md`](../docs/PII_MATRIX.de.md) stuft jede Route ein, mit Gate | ✅ |
> | **F3** — Verbindungen nur als übereinstimmender Text | Migration 078 (Fach) und 079 (Klassen-Vokabular) machen echte Fremdschlüssel daraus | ✅ vier Tabellen verweisen auf `klassen` |
> | **F4** — Lehrkraft wird als Schüler gespeichert | `pruefeKlassenname` sperrt „lehrer" an beiden Türen (Anlegen und Ändern) | ✅ `api/student_klasse_regel.go:18` |
> | **F5–F8** | im Zuge desselben Durchgangs | ✅ F8: partieller Unique-Index, Migration 048 |
>
> Die Sätze „drei Befunde gehören vor den Echtbetrieb behoben" und „F3 spätestens vor den
> ersten Schuljahreswechsel" weiter unten waren richtig und sind eingelöst. Sie stehen
> weiter da, weil sie zum Bericht gehören — dieser Kasten sagt, dass sie erledigt sind.
> Ohne ihn läse jemand heute von Blockern, die es nicht mehr gibt.

## Inhalt

| Datei | Was drin ist |
|---|---|
| [**datenbank-pruefbericht.md**](datenbank-pruefbericht.md) | Der Hauptbericht. Öffnet direkt hier auf GitHub, mit Diagrammen. **Hier anfangen.** |
| [datenbank-pruefbericht.html](datenbank-pruefbericht.html) | Dieselbe Fassung als gestaltete Seite — herunterladen und im Browser öffnen. |
| [sicherheitsbefund-kiosk-suche.md](sicherheitsbefund-kiosk-suche.md) | Eigener Sicherheitsbefund: Die Kiosk-Suche gibt vollständige Schülerdaten an die Helfer-Rolle heraus (Hoch). |
| [sicherheitsbefund-vormerkungen.md](sicherheitsbefund-vormerkungen.md) | Eigener Sicherheitsbefund: Die Vormerkungsliste gibt Schülernamen an die Helfer-Rolle heraus (Mittel–Hoch). |
| [bibliothek-erd.svg](bibliothek-erd.svg) | Das vollständige Datenbank-Diagramm (breit — im Browser öffnen). |

## Kurzfassung in drei Sätzen

Als Datenbank-Design ist das gute Arbeit — klar besser als der Durchschnitt kommerzieller Systeme und bemerkenswert für zehn Wochen Bauzeit mit KI-Unterstützung. Die Schwächen folgen einem Muster: Wo eine Regel nur als Konvention aufgeschrieben statt als Datenbankregel eingebaut wurde, kann die Datenbank sie nicht verteidigen. Drei Befunde (F1, F3, F4) gehören vor den Echtbetrieb behoben; F3 muss spätestens vor den ersten Schuljahreswechsel.

Der Prüfumfang ist **nur die Datenbankstruktur**. Die größeren Fragen — Betrieb, Wartung, langfristige Verantwortung, Datenschutz-Formalien — stehen auf einem anderen Blatt.
