# 001 — Nach einem Vorfall ist nichts nachzusehen

Stand: 18.09.2026

**Befund.** Das System kann beantworten, *ob* es läuft und *ob* es richtig eingerichtet ist,
aber nicht, *was gestern um 10:15 an der Theke langsam war*. Es gibt keine Anfragedauer,
keine Anfragekennung und keine nennenswerte Aufbewahrung.

**Beleg.** Am 18.09.2026 am Code gemessen:

| Frage | Stand |
| --- | --- |
| Wird jede Anfrage protokolliert? | Ja — `api/router.go`, `wrapMiddleware`: eine Zeile mit Methode und maskiertem Pfad |
| Mit Status? | Nur bei 5xx — `LoggingMiddleware` schreibt ausschließlich `if recorder.status >= 500` |
| Mit **Dauer**? | **Nein.** Die Zeit wird nirgends gemessen |
| Anfrage-/Korrelationskennung? | **Keine.** `grep -riE "request.?id\|correlation\|trace.?id"` über den Go-Code: kein Treffer |
| Tracing (OpenTelemetry o. Ä.)? | Nicht vorhanden |
| Metriken (`/metrics`, Prometheus)? | Nicht vorhanden |
| Aufbewahrung | `json-file`, 3 × 10 MB je Container (`docker-compose.yml`) |
| Fehlerdienst | Sentry, **nur** wenn `SENTRY_DSN` gesetzt ist (`main.go`) |

Zwei Stellen behaupteten das Gegenteil und sind falsch:

- `docs/arc42/05-bausteinsicht.md` schrieb in der Anfragekette „Logging — Status + Dauer";
  **mit diesem Eintrag korrigiert.**
- Der Kommentar über `LoggingMiddleware` in `api/middleware.go` sagt bis heute
  „protokolliert jede Anfrage mit Status und Dauer". Der Code darunter tut beides nicht.
  Eine Korrektur am Kommentar ist eine Codeänderung und steht noch aus.

**Was es kostet.** Der Bericht aus dem Betrieb lautet erfahrungsgemäß „die Theke hing
vorhin". Um dem nachzugehen, bräuchte es: die Dauer der Anfragen in dem Zeitraum, eine
Kennung, die eine Kiosk-Anfrage über Handler, Dienst und Hintergrundlauf hinweg
wiederfindet, und Logzeilen, die den Zeitpunkt überhaupt noch abdecken. Keines davon ist
da — 3 × 10 MB sind unter Last die letzten Stunden, nicht der letzte Vorfall. Die
vorhandene Selbstprüfung hilft hier nicht: Sie beantwortet „eingerichtet, aber nicht in
Betrieb", nicht „langsam".

Das ist kein akutes Risiko — ein Schulserver mit acht Arbeitsplätzen fällt selten und
sichtbar aus. Es ist der Grund, warum man **nach** dem Ausfall nichts lernt.

**Was daraus folgt.** Kein Beobachtbarkeits-Stapel für einen Host. Der Vorschlag — eine
Logzeile je Anfrage statt zwei, eine Anfragekennung bis in die Fehlermeldung, ein
Abschnitts-Helfer bei Bedarf, eine Aufbewahrungsentscheidung — steht als
[Idee 005](../ideen/005-log-und-trace-ohne-neue-bauteile.md) und wird dort geführt, damit
der Plan nicht an zwei Stellen liegt. Hier bleibt der Befund.
