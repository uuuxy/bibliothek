# 005 — Log und Trace ohne ein neues Bauteil

Stand: 18.09.2026

**Idee.** Den Befund aus [review/001](../review/001-beobachtbarkeit.md) mit vier Feldern und
einer Compose-Zeile erledigen, statt einen Beobachtbarkeits-Stapel aufzustellen. Netto
**ein Mechanismus weniger** im Betrieb, nicht einer mehr.

| Stufe | Was | Wirkung |
| --- | --- | --- |
| 1 | Die zwei Logzeilen je Anfrage (`wrapMiddleware` vorher, `LoggingMiddleware` bei 5xx) zu **einer** zusammenlegen, im `defer`, mit Status und Dauer | beantwortet „was war langsam"; ein Schreibort statt zwei |
| 2 | Anfragekennung in der äußersten Middleware erzeugen, in den Kontext **und in den Antwort-Header** (`X-Request-Id`), von der Oberfläche in der Fehlermeldung angezeigt | macht aus „die Theke hing vorhin" ein `jq 'select(.rid=="…")'` |
| 3 | Bei Bedarf ein Zwanzigzeiler `defer schritt(ctx, "loan_checkout")()` für Abschnittsdauern | „welcher Abschnitt war langsam", ohne Spans und Exporter |
| 4 | Aufbewahrung am YAML-Anker `&log_rotation` entscheiden — 3 × 10 MB sind eine Docker-Vorgabe | Logzeilen überleben den Vorfall |

**Warum.** Der strukturierte Strom ist schon da: `slog` mit JSON-Handler ist Standard, auch
die `log.Printf`-Aufrufe hängen daran. Bei **einem** Prozess ist die Anfragekennung bereits
die Trace-ID — alle Zeilen mit derselben Kennung *sind* der Trace, `jq` ist der Viewer.
Loki, Prometheus oder ein OTel-Collector brächten je einen Container, einen Port, ein
Geheimnis, einen Aktualisierungspfad und einen zweiten Ort zum Nachsehen.

**Was dagegen spricht.**

- Die **340 `log.Printf`-Zeilen** im Produktivcode tragen die Kennung nicht — sie kennen
  keinen Kontext. Umstellen lohnt nur dort, wo wirklich nachgesehen wird.
- Der naheliegende Trick, die Kennung über einen kontextlesenden `slog.Handler`
  automatisch in jede Zeile zu bekommen, bringt **heute nichts**: Es gibt 0 Aufrufe von
  `slog.*Context` (gemessen 18.09.2026).
- `/events` ist eine Dauerverbindung — eine Abschlusszeile mit Stundenwert wäre Unsinn, der
  Pfad braucht eine eigene Behandlung.
- Statische Auslieferung (SPA, Cover) läuft durch dieselbe Kette und ist der Mengentreiber;
  ohne Stufe 4 verkürzt Stufe 1 die Aufbewahrung zusätzlich.
- `driver: local` komprimiert, schreibt aber binär — wer die Dateien direkt liest, muss
  umstellen.

**Nächster Schritt.** Besprechen und entscheiden, bevor Code geändert wird. Zu klären: ob
Stufe 1 und 2 zusammen gehen, wie die Kennung in der Oberfläche erscheint, und welche
Aufbewahrung der Host verträgt (die Datenbank liegt auf derselben Platte).

> Bei der Gelegenheit mit zu erledigen: Der Kommentar über `LoggingMiddleware` behauptet
> „protokolliert jede Anfrage mit Status und Dauer" — er ist der Grund, warum die Aussage
> ungeprüft in die arc42-Kapitel gewandert ist.
