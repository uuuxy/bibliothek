# Bewertung der Schulbibliothek-Datenbank

Diese Mappe enthält eine unabhängige, **nur lesende** Prüfung der Datenbankstruktur des selbst gebauten Bibliothekssystems. Es wurde nichts am System verändert.

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
