# Review

Befunde aus der Durchsicht der Architektur: Stellen, an denen der gebaute Stand von dem
abweicht, was er sein sollte — oder von dem, was die Dokumentation über ihn behauptet.

---

## Abgrenzung

| | Review (hier) | [Ideen](../ideen/) | [OFFEN.md](../../OFFEN.md) | [sweeps.md](../../sweeps.md) |
| --- | --- | --- | --- | --- |
| Inhalt | „Das stimmt so nicht" / „das fehlt" | „Wäre das besser?" | „Das ist zu tun" | „Diese Bugklasse hat einen Detektor" |
| Belegt durch | Messung am Code | nichts, es ist ein Vorschlag | Entscheidung | ein Gate |

Ein Befund, der zu Arbeit wird, wandert nach `OFFEN.md` und wird hier gelöscht. Ein Befund,
der eine wiederkehrende Fehlerart beschreibt, gehört zusätzlich als Bugklasse in
`sweeps.md` — mit Detektor, sonst kommt er wieder.

Ein Befund an der **Dokumentation** (eine Aussage in den Kapiteln, die am Code nicht
stimmt) wird nicht hier verwaltet, sondern **sofort im Kapitel korrigiert**; hier bleibt nur
der Eintrag, der erklärt, wie er entstehen konnte.

---

## Form

Eine Datei je Befund: `NNN-kurzname.md`.

```markdown
# NNN — Titel

Stand: TT.MM.JJJJ

**Befund.** Ein Satz.

**Beleg.** Was gemessen wurde, mit Fundstelle. Ohne Beleg ist es eine Meinung.

**Was es kostet.** Der konkrete Fall, in dem es weh tut — nicht „ist unschön".

**Was daraus folgt.** Kleinster sinnvoller Schritt, oder ausdrücklich: nichts.
```

> **Gate-Lücke:** `docs/stand_angaben_test.go` prüft `*.md` und `*/*.md` unterhalb von
> `docs/`. Dateien in diesem Ordner liegen eine Ebene tiefer und werden **nicht** geprüft.

---

## Register

| Nr. | Befund | Zustand |
| --- | --- | --- |
| [001](001-beobachtbarkeit.md) | Nach einem Vorfall ist nichts nachzusehen: keine Dauer, keine Anfragekennung, kurze Aufbewahrung | offen |
| [002](002-umfang-gegen-anwendungsfall.md) | Umfang gegen Anwendungsfall — die Gegenrechnung zum Verdacht „zu komplex" | zur Kenntnis |
