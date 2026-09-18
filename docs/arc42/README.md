# arc42-Architekturdokumentation — Bibliothek (Schulbibliotheks-Software)

Stand: 18.09.2026 · Gliederung nach [arc42](https://arc42.org) (Template 8.2, deutsch)

---

## Was das hier ist

Die Architekturdokumentation dieses Systems, gegliedert nach arc42 — zwölf Kapitel, ein
Kapitel je Datei. Sie beschreibt **den gebauten Stand**, nicht einen Plan: Jede Aussage ist
am Code, an `schema.sql`, an den Migrationen, am `Dockerfile`, an den CI-Workflows oder an
einem Gate nachgelesen, und wo eine Fundstelle die Aussage trägt, steht sie dabei.

Diese Dokumentation ist **die** Architekturbeschreibung des Systems; die frühere Kurzfassung
`ARCHITECTURE.md` ist am 18.09.2026 darin aufgegangen. Die Arbeitsteilung mit den übrigen
Dokumenten unter [`docs/`](../README.md):

| Frage                                                        | Dort steht die Antwort                                     |
| ------------------------------------------------------------ | ---------------------------------------------------------- |
| Wie ist das System gebaut, und **warum so**?                 | **hier** (arc42)                                           |
| Welche fachliche Regel gilt genau?                           | [FACHKONZEPT.md](../FACHKONZEPT.md)                        |
| Wie bediene ich das System?                                  | [HANDBUCH.md](../HANDBUCH.md)                              |
| Wie betreibe, deploye, sichere ich es?                       | [DEPLOYMENT.md](../DEPLOYMENT.md), [resilience_and_recovery.md](../resilience_and_recovery.md) |
| Welche Schutzmaßnahme greift wo?                             | [SECURITY.md](../SECURITY.md), [PII_MATRIX.de.md](../PII_MATRIX.de.md) |
| Was muss **immer** wahr sein, und auf welcher Ebene?         | [invarianten.md](../invarianten.md)                        |
| Welche Bugklassen kennt das Projekt, und wer detektiert sie? | [sweeps.md](../sweeps.md)                                  |
| Was ist **offen**?                                           | [OFFEN.md](../OFFEN.md) — die einzige Offen-Liste          |

> **Diese Dokumentation führt keine eigene Offen-Liste.** Kapitel 11 benennt Risiken und
> technische Schulden, verweist für den Bearbeitungsstand aber auf `OFFEN.md`. Zwei Listen
> wären genau die Fehlerart, gegen die dieses Projekt antritt.

Daneben liegen zwei Ordner für das, was **noch keine** Architekturbeschreibung ist:

| Ordner | Inhalt | Wann etwas hineingehört |
| --- | --- | --- |
| [`review/`](review/) | Befunde aus der Durchsicht — wo der gebaute Stand von dem abweicht, was er sein sollte | wenn es **gemessen** ist |
| [`ideen/`](ideen/) | kurze Einträge zu Gedanken, die noch nicht entschieden sind | wenn die Frage „wäre das besser?" lautet |

Beide sind ausdrücklich **keine** zweite Offen-Liste: Was entschieden ist, wandert nach
`OFFEN.md` und wird dort verwaltet; der Eintrag hier wird gelöscht.

---

## Die zwölf Kapitel

| #   | Kapitel                                                                    | Beantwortet                                                                       |
| --- | -------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| 1   | [Einführung und Ziele](01-einfuehrung-und-ziele.md)                        | Was soll das System leisten, für wen, mit welchen Qualitätszielen                 |
| 2   | [Randbedingungen](02-randbedingungen.md)                                   | Was war nicht verhandelbar — technisch, organisatorisch, konventionell            |
| 3   | [Kontextabgrenzung](03-kontextabgrenzung.md)                               | Wer und was steht außen dran, über welche Schnittstelle                           |
| 4   | [Lösungsstrategie](04-loesungsstrategie.md)                                | Die tragenden Entscheidungen in Kurzform, mit Begründung                          |
| 5   | [Bausteinsicht](05-bausteinsicht.md)                                       | Statische Struktur: Pakete, Schichten, Verantwortungen, Level 1–3                 |
| 6   | [Laufzeitsicht](06-laufzeitsicht.md)                                       | Zehn Szenarien im Ablauf — Scan, Login, Offline, Nachtlauf, Shutdown …            |
| 7   | [Verteilungssicht](07-verteilungssicht.md)                                 | Container, Volumes, Ports, Reverse Proxy, CI/CD-Wege                              |
| 8   | [Querschnittliche Konzepte](08-querschnittliche-konzepte.md)               | Sicherheit, Datenschutz, Persistenz, Nebenläufigkeit, Zeit, Fehler, Test …        |
| 9   | [Architekturentscheidungen](09-architekturentscheidungen.md)               | 24 Entscheidungen als ADR: Stand, Grund, Folge, Fundstelle                        |
| 10  | [Qualitätsanforderungen](10-qualitaetsanforderungen.md)                    | Qualitätsbaum und messbare Szenarien samt zugehörigem Gate                        |
| 11  | [Risiken und technische Schulden](11-risiken-und-technische-schulden.md)   | Was heute weh tut oder morgen weh tun wird — mit Einschätzung                     |
| 12  | [Glossar](12-glossar.md)                                                   | Die Fachsprache des Hauses, deutsch, mit Code-Bezug                               |

---

## Lesewege

- **Neu im Projekt, eine Stunde Zeit:** Kapitel 1 → 4 → 5 (Level 1+2) → 12. Danach
  [FACHKONZEPT.md §1](../FACHKONZEPT.md) für die Omnibox, weil daran der ganze Betrieb hängt.
- **Ich muss etwas ändern:** Kapitel 5 (welcher Baustein), 8 (welches Querschnittskonzept
  fasse ich an), 9 (ist das schon einmal entschieden worden), dann
  [invarianten.md](../invarianten.md) und [sweeps.md](../sweeps.md).
- **Ich muss es betreiben:** Kapitel 7 → [DEPLOYMENT.md](../DEPLOYMENT.md) →
  [resilience_and_recovery.md](../resilience_and_recovery.md).
- **Ich muss es beurteilen (Schule, Träger, Datenschutz):** Kapitel 1, 3, 10, 11 und
  [SECURITY.md](../SECURITY.md).

---

## Wie diese Dokumentation gepflegt wird

1. **Kein Kapitel behauptet einen Stand, den es nicht hat.** Jede Datei trägt im Kopf eine
   `Stand:`-Zeile. Das Gate `docs/stand_angaben_test.go` prüft für jede Datei unter
   `docs/` und `docs/*/`, dass **kein im Text genanntes Datum jünger ist als der Kopf**.
   Wer hier einen datierten Absatz ergänzt, zieht den Kopf mit — sonst wird der Test rot.
2. **Zahlen tragen ihr Messdatum.** Alle Umfangszahlen in Kapitel 5 sind am 17.09.2026 mit
   den Befehlen aus [Kapitel 5](05-bausteinsicht.md#anhang-die-zahlen-selbst-nachmessen)
   erhoben. Sie altern; der Befehl daneben altert nicht.
3. **Fundstellen werden beim Namen genannt, nicht gezählt.** Keine Zeilennummern: Datei-,
   Paket-, Constraint- und Indexnamen halten, Zeilennummern wandern. Das ist dieselbe Regel,
   die [invarianten.md](../invarianten.md) seit dem 06.08.2026 anwendet — dort waren nach
   dem Wachstum von `schema.sql` alle 21 Zeilenverweise falsch, ohne dass es beim Lesen
   auffiel.
4. **Das Warum steht im Commit.** Wo eine Entscheidung eine längere Geschichte hat, nennt
   Kapitel 9 sie knapp und verlässt sich im Übrigen auf `git log` — die Historie ist
   ausführlicher als jede gepflegte Liste und kann nicht veralten.
