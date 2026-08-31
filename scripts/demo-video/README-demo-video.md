# Bibliosys – Produktrundgang als Video

Erzeugt ein Werbe-/Rundgangsvideo (1920×1080) über ALLE Bereiche der Anwendung,
gegen einen isolierten Demo-Stack mit sauberen, fiktiven Daten.

## Bausteine
- `docker-compose.demo.yml` – eigener Stack: Port 8085, eigene DB (5435), SMTP ins Leere (127.0.0.1:9),
  Mock-IMAP, Admin `bibliothek@musterschule.de` (jedes Passwort).
- `seed_video.sql` – fiktive Schule „Gymnasium am Stadtpark": 47 Titel (echte ISBNs → Cover),
  ~1.500 Exemplare, 648 Schüler in 27 Klassen, Ausleihen/Überfällige/Mahnstufen, Vormerkungen,
  Klassensätze, Anliegen, Geräte, Schadensfälle, Bestellhistorie, Inventur. Eltern-Mails @example.invalid.
- `reset-demo.sh` – Stack neu, Seed einspielen, Backup-Marker, Cover-Sync.
- `harness.mjs` – Aufnahme-Helfer: sichtbarer Cursor, Klick-Ripple, Kapitel-Bauchbinden, Titelkarten.
- `rundgang.mjs` – das Drehbuch (17 Kapitel). Barcodes werden zur Laufzeit aus der DB gelesen (`db.mjs`).

## Ablauf
```bash
docker build --build-arg GIT_COMMIT=$(git rev-parse HEAD) -t bibliothek-demo:latest .   # im Repo
./reset-demo.sh
DEMO_TEMPO=0.85 node rundgang.mjs ./final          # DEMO_TEMPO=0.3 für schnelle Probeläufe
# WebM → MP4:
FF=$(ls ~/Library/Caches/ms-playwright/ffmpeg-*/ffmpeg-mac | head -1)
$FF -i final/*.webm -c:v libx264 -crf 18 -preset slow -pix_fmt yuv420p -movflags +faststart Bibliosys-Rundgang.mp4
```
`DEMO_ONLY=intro,ausleihe,portal` nimmt nur einzelne Kapitel auf.

## Sicherheitsnetz
Mahnlauf, Abgänger-Versand und Bestellung werden im Skript per `page.route` hart abgeklemmt;
zusätzlich zeigt der Demo-Stack auf einen toten SMTP-Port. Nichts verlässt den Rechner.

## Stand v2 (30.08.2026)
- 20 Kapitel: Titel, öffentlicher OPAC, Flur-Monitor, Login, 15 Bereiche, Lehrkraft-Sicht (zweiter Login), „Und außerdem…"-Karte.
- `harness.mjs`: `d.zoom(sel, faktor)` / `d.zoomOff()` skalieren den Body (Overlays bleiben scharf); `d.liste(...)` für die Abschlusskarte; `d.clickSure()` für Dialog-Knöpfe.
- `sheet.mjs`: Kontaktbogen zur Sichtprüfung — `ffmpeg -i lauf.webm -vf "fps=1/8,scale=640:-1" frames/f_%02d.png && node sheet.mjs frames sheet`.
- Monitor-Seite braucht Belletristik mit vielen Ausleihen der letzten 30 Tage — der Seed legt das an, sonst gewinnen Lernmittel.

## Vertonung (Sprecher + Musik)
`rundgang.mjs` schreibt `cues.json` (Zeitmarke + Bauchbinden-Text). Daraus:
```bash
python3 vertonung.py out/page@…webm out/cues.json Bibliosys-vertont.mp4 [Stimme=Anna] [Rate=165]
```
Stimme = macOS `say` (deutsche Stimmen: `say -v '?' | grep de_DE`), Musik = `musik.py` (synthetisches Pad,
lizenzfrei, wird per Sidechain unter der Stimme abgesenkt). Überlappende Cues werden nach hinten geschoben,
das Video wird am Ende mit Standbild verlängert, damit der Sprecher ausreden kann. Eigene Musik: `musik.wav`
in `vertonung.py` durch eine Datei ersetzen.

## Stand v5 (30.08.2026)
- Ohne Sprecher. Musik: `musik3.py` – 32 Blöcke à 8 Takte (≈ 8:25 min) mit vier Akkordfolgen, zwei Lead-Melodien,
  Breaks, Risern vor jedem Drop; Whoosh/Impact an den Kapitelmarken aus `cues.json`.
  `python3 musik3.py <dauer_s> musik.wav out/cues.json` → ffmpeg-Mux (`-map 0:v -map 1:a … -shortest`).
- Drehbuch entdoppelt: Ausweis-Stapeldruck nur im Druck-Center; Schülerdatei zeigt Sperren-Dialog + „Neuer Schüler";
  „Meine Anliegen" nur in der Lehrkraft-Sicht. Optionale Klicks laufen über `d.clickIf()` (keine 20-s-Hänger mehr).
- Abschluss: „Auf einen Blick" (16 Bereiche) → „Und außerdem …" → Titel. Die vollständige Liste steht NICHT im Video,
  sondern in `funktionsuebersicht.html` (Begleitdokument, als Artifact veröffentlicht).
