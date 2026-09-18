# 004 — Architektur- und Anforderungsvergleich mit OpenLibry und OpenBiblio

Stand: 18.09.2026

**Idee.** Zwei freie Bibliotheksprogramme danebenlegen und Punkt für Punkt vergleichen:
**OpenLibry** und **OpenBiblio**. Nicht um zu wechseln, sondern um zu sehen, wo dieses
System mehr tut als nötig — und wo es etwas kann, das dort fehlt.

**Warum.** Der beste Prüfstein für die Frage aus [003](003-schlankerer-schnitt.md) ist ein
Programm, das denselben Zweck mit weniger erfüllt. Umgekehrt ist jede Anforderung, die
**keines** der beiden kennt, ein Hinweis darauf, dass sie aus dem hiesigen Betrieb stammt —
Lernmittelfreiheit, LUSD, Bescheide — und damit eine echte Begründung für den Eigenbau.

**Was zu vergleichen wäre** (Reihenfolge nach Erkenntniswert):

1. **Umfang** — welche der zwölf Bereiche aus [Kapitel 1](../01-einfuehrung-und-ziele.md)
   deckt das Fremdsystem, und mit wie viel Code?
2. **Datenmodell** — trennen sie Titel und Exemplar? Wie halten sie den Entleiher?
3. **Anmeldung** — eigene Passwörter, IdP, gar keine?
4. **Nebenläufigkeit** — mehrere Arbeitsplätze am selben Bestand: gelöst oder ignoriert?
5. **Fristen** — Schuljahresfrist und Stichtag oder nur rollierende Leihfrist?
6. **Datenschutz** — Löschfristen für Abgänger überhaupt vorgesehen?
7. **Betrieb** — was braucht eine Schule zum Aufsetzen?

**Was dagegen spricht.** Ein Vergleich kostet einen Tag und ändert für sich genommen
nichts. Er lohnt nur, wenn vorher feststeht, welche Entscheidung er beeinflussen soll —
sonst entsteht ein Dokument, das niemand liest.

**Nächster Schritt.** Quellen besorgen und lesen, bevor irgendetwas behauptet wird.

> **Über beide Systeme ist hier nichts geprüft.** Was in diesem Eintrag steht, sind ihre
> Namen und sonst nichts. Alles Weitere — Sprache, Datenmodell, Umfang, Lebendigkeit des
> Projekts — muss aus ihren Quelltexten und Anleitungen erhoben werden, nicht aus dem
> Gedächtnis. Ein Vergleich, der auf Erinnerung gebaut ist, ist schlimmer als keiner.
