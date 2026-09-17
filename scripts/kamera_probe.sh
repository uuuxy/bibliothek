#!/bin/bash
#
# Kamera-Probe: hält der Anwendung einen echten Barcode vor die Linse — ohne Kamera.
#
# Chrome kann eine Videodatei als Kamera ausgeben (--use-file-for-fake-video-capture).
# Damit lässt sich beantworten, was sonst nur ein Mensch mit einem Handy beantworten kann:
# Erkennt die Theke einen gedruckten Strichcode und bucht ihn?
#
# Die Probe läuft zweimal — einmal mit dem eingebauten Barcode-Erkenner des Browsers und
# einmal ohne ihn. Der zweite Lauf ist der iPhone-Weg: Safari hat keinen, dort arbeitet der
# Rückfall über ZXing.
#
# Voraussetzungen: laufender lokaler Stack (scripts/stack-neu.sh), ffmpeg, ImageMagick.
# Aufruf:  ./scripts/kamera_probe.sh [BARCODE]      (Vorgabe: B-10001)
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

INHALT="${1:-B-10001}"
ORDNER="${TMPDIR:-/tmp}/kamera-probe"
mkdir -p "$ORDNER"

for werkzeug in ffmpeg magick; do
	command -v "$werkzeug" >/dev/null || { echo "Fehlt: $werkzeug (brew install $werkzeug)"; exit 1; }
done

echo "1/3  Strichcode $INHALT erzeugen (mit dem Generator der Anwendung)"
KAMERA_ZIEL="$ORDNER/code.png" KAMERA_INHALT="$INHALT" \
	go test ./api/ -run TestWerkzeug_KameraBild -count=1 >/dev/null

echo "2/3  Daraus ein Kamerabild machen (1280x720, Ruhezone ringsum)"
magick -size 1280x720 xc:white "$ORDNER/code.png" -gravity center -composite "$ORDNER/frame.png"
ffmpeg -y -loglevel error -loop 1 -i "$ORDNER/frame.png" -t 6 -r 15 -pix_fmt yuv420p "$ORDNER/kamera.y4m"

echo "3/3  Die Theke damit füttern"
cd frontend
KAMERA_VIDEO="$ORDNER/kamera.y4m" npx playwright test -c playwright.proben.config.js --reporter=list
