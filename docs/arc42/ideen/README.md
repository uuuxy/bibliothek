# Ideen

Kurze Einträge für Gedanken, die noch **nicht entschieden** sind: ein anderer Schnitt, ein
anderes Verfahren, ein Vergleich, ein Zweifel an einer bestehenden Entscheidung.

---

## Abgrenzung — damit hier keine zweite Offen-Liste entsteht

Die Regel des Hauses ist eindeutig: **Alles Offene steht nur in `docs/OFFEN.md`.** Dieser
Ordner tritt nicht daneben, sondern davor:

| | Ideen (hier) | [OFFEN.md](../../OFFEN.md) |
| --- | --- | --- |
| Zustand | noch nicht entschieden | entschieden, dass es getan wird |
| Frage | „Wäre das besser?" | „Was ist zu tun, in welcher Reihenfolge?" |
| Verbindlichkeit | keine | die Arbeitsliste |

**Wenn eine Idee angenommen wird, wandert sie nach `OFFEN.md` und wird hier gelöscht.**
Wird sie verworfen, wird sie ebenfalls gelöscht — die Begründung steht dann in der
Commit-Nachricht, wie bei allem Erledigten in diesem Projekt.

Eine Idee, die zu einer Architekturentscheidung wird, landet als ADR in
[Kapitel 9](../09-architekturentscheidungen.md) und verschwindet hier.

---

## Form

Eine Datei je Idee: `NNN-kurzname.md`, fortlaufend nummeriert. Eine Datei je Idee, weil
eine gemeinsame Liste bei jedem Eintrag denselben Absatz anfasst — und weil Löschen dann
ein `git rm` ist und kein Suchen.

Vier Überschriften, kurz gehalten:

```markdown
# NNN — Titel

Stand: TT.MM.JJJJ

**Idee.** Ein bis drei Sätze: was anders wäre.

**Warum.** Welches Problem das löst — am Code oder am Betrieb belegt, nicht vermutet.

**Was dagegen spricht.** Kosten, Risiko, offene Fragen. Ehrlich, sonst ist der Eintrag
wertlos.

**Nächster Schritt.** Das Kleinste, womit man die Idee prüfen könnte.
```

**Fünf bis dreißig Zeilen.** Wer mehr braucht, hat keine Idee, sondern einen Entwurf — der
gehört als Konzeptdokument nach `docs/` und wird von hier verlinkt.

---

## Regeln, die auch hier gelten

- **Belege statt Vermutungen.** „Das Paket ist zu groß" ist keine Aussage; „`api/` trägt
  27.810 Zeilen in 153 Dateien und darin auch Fachlogik" ist eine.
- **Was nicht geprüft ist, wird als ungeprüft gekennzeichnet.** Besonders bei Aussagen über
  fremde Systeme und über die Infrastruktur der Schule.
- **Die Sache nennen, nicht den Absender** (`CLAUDE.md`, Regel 4): keine Personennamen.
- **Kein Datum in der Zukunft** und ein `Stand:` im Kopf.

> **Gate-Lücke, die man kennen muss:** `docs/stand_angaben_test.go` sammelt seine Dateien
> über `*.md` und `*/*.md` — also `docs/` und eine Ebene darunter. Dateien in **diesem**
> Ordner liegen zwei Ebenen tiefer und werden **nicht** geprüft. Die `Stand:`-Zeile hier
> hält niemand außer dem Schreibenden.

---

## Register

| Nr. | Idee | Betrifft |
| --- | --- | --- |
| [001](001-anmeldung-ueber-einen-idp.md) | Anmeldung über einen Identitätsanbieter statt IMAP-Bind | Kapitel 8.2, A2 |
| [002](002-anmeldung-per-einmal-link.md) | Einmal-Link per Mail als zweiter Anmeldeweg | Kapitel 8.2, A2 |
| [003](003-schlankerer-schnitt.md) | Was könnte weg? Umfang gegen Nutzen prüfen | Kapitel 5, 11 |
| [004](004-vergleich-mit-openlibry-und-openbiblio.md) | Architektur- und Anforderungsvergleich mit OpenLibry und OpenBiblio | Kapitel 1, 4, 5 |
| [005](005-log-und-trace-ohne-neue-bauteile.md) | Log und Trace ohne ein neues Bauteil | Kapitel 8.11, [review/001](../review/001-beobachtbarkeit.md) |
