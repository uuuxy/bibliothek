# Arbeitsregeln für dieses Projekt

Zwei Regeln werden hier immer wieder verletzt und stehen deshalb zuerst: **Oberfläche wird
nachgelesen, nicht geraten** — und **erst nachsehen, was es schon gibt, statt neu zu bauen**.
Ein Hook in `.claude/settings.json` erinnert bei jeder Änderung an `frontend/src/**.svelte`
und `**.css` daran.

---

## 1. Oberfläche: Material 3, nachgelesen statt geraten

Das Designziel ist der echte Material-3-Look (Google). Oberste Anforderung ist Konsistenz,
nicht der Einzelfall.

**Vor jeder Oberflächen-Änderung die zuständige M3-Seite lesen** — auch vor einer einzelnen
Zeile Hinweistext. Nicht aus dem Gedächtnis entscheiden: Eine Einschätzung „aus dem Kopf" lag
hier schon nachweislich falsch.

- `m3.material.io` ist eine Angular-App; `WebFetch`/`curl` liefern nur den Titel.
  Lesbar über `https://r.jina.ai/https://m3.material.io/…` (Komponenten-Guidelines und -Specs).
- Ersatzquellen, wenn eine Seite leer bleibt: `material-web.dev`, Googles
  `material-components-android` → `docs/components/*.md`.
- M3 hat **keine** Data-Table-Komponente; dafür gelten die Listen-Regeln.
- Was aus der Quelle folgt, gehört mit Zitat in die Commit-Nachricht — nicht „wirkt M3-konform".

**Was in diesem Projekt bereits festgelegt ist:**

| Sache        | Regel                                                                                                                                                                                                                                                                                   |
| ------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Farben       | Nur Rollen aus `frontend/src/styles/rollen.css`: `bg-surface`, `text-on-surface`, `text-on-surface-variant`, `border-outline-variant` … **Keine** Paletten-Klassen (`slate`, `blue`, `emerald` …). Gate: `frontend/src/lib/frontend-hygiene-farben.test.js` — die Zahl darf nur sinken. |
| Typo         | Skala in `frontend/src/styles/theme-mass.css`: `text-xs` = body-small, `text-sm` = body-medium, `text-base` = body-large, `text-lg` = title-large, `text-xl` = headline-small, dazu `text-label-small`. `text-title-medium` o. Ä. **gibt es nicht** und wäre stumm.                     |
| Fließtext    | Body-Rollen. Label-Rollen nur für Text INNERHALB eines Bauteils und für Bildunterschriften.                                                                                                                                                                                             |
| Hinweiszeile | Ein Satz unter der Überschrift: `text-sm text-on-surface-variant`.                                                                                                                                                                                                                      |
| Maße         | Bedienhöhe 36 px, Trefferfläche 48 px. Gates: `control-hoehen`, `icon-trefferflaechen`.                                                                                                                                                                                                 |
| Dateigröße   | 200 Zeilen je Frontend-Datei (Ratsche `frontend-hygiene-dateigroesse.test.js`).                                                                                                                                                                                                         |

---

## 2. Erst nachsehen, was es schon gibt

Vor jedem neuen Bauteil, Helfer, Endpunkt oder Feld: **suchen, ob es das schon gibt.** Zwei
Wege zum selben Zustand sind die teuerste Bugklasse dieses Projekts — der zweite Weg kennt die
Regel des ersten nicht.

```bash
ls frontend/src/lib/components/ui/                 # die gemeinsamen Bauteile
grep -rn "<Stichwort>" frontend/src --include="*.svelte"
grep -rn "<Funktionsname>" --include="*.go" .      # Schreibpfade und Türen
./scripts/api_inventar.sh                          # alle Routen, beide Richtungen
```

**Die gemeinsamen Bauteile** (`frontend/src/lib/components/ui/`): Abschnitt ·
BestaetigungsDialog · BuchCover · Button · CoverPeek · Feld · Kaestchen · KlassenVersandDialog ·
LadeFehler · Ladekreis · LogoRelief · Menue · Radio · Reiter · Segmente · Select · SelectListe ·
Snackbar · StatusChip · SuchZustand · Suchfeld · Suchpille · Switch · Tabelle · TabelleSortKopf ·
Zaehlerpille.

- Eine vorhandene Tür erweitern statt eine zweite anzulegen: Ein Zustand hat einen Schreibpfad.
- Ein neues Bauteil braucht in der Commit-Nachricht die Begründung, warum keins der
  vorhandenen passt.
- Gilt genauso für Endpunkte: Erst `docs/api_inventar.md` lesen.

---

## 3. Wie geprüft wird

- Nach jeder Änderung an `.svelte`/`.js`: `npx svelte-check` (0 Fehler, 0 Warnungen) und
  `npx vitest run` im Ordner `frontend`.
- Go: `go test ./...` mit `TEST_DATABASE_URL`; `libpq`-18 im PATH **voranstellen**
  (`export PATH="/opt/homebrew/opt/libpq/bin:$PATH"`).
- Ein neues Gate zählt erst, wenn es einmal **rot** gesehen wurde — am Rückbau der Sache,
  die es schützen soll.
- Die Suite nie in eine Pipe schicken (zsh liefert `PIPESTATUS` leer); Exit-Code positiv erfassen.

---

## 4. Offene Arbeit und Sprache

- Alles Offene steht **nur** in `docs/OFFEN.md`. Erledigtes wird dort gelöscht, nicht
  abgehakt — die Geschichte steht in der Commit-Nachricht.
- Dokumente nennen die Sache, nicht den Absender: keine Personennamen, keine prüfende Stelle.
- Sachlich schreiben: Aussage, Beleg, nächster Schritt. Keine Werbesprache, keine Bewertungen.
- Commit-Nachrichten auf Deutsch, ohne Werkzeug-Hinweis.
