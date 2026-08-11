#!/bin/bash
#
# Baut den Entwicklungs-Stack neu und BELEGT, dass der Container danach wirklich den
# aktuellen Stand ausliefert.
#
# Warum das ein eigenes Skript ist (10.08.2026):
# `docker compose up -d --build` bricht bei einem fehlgeschlagenen Build zwar ab, lässt
# aber den ALTEN Container weiterlaufen. Wer die Ausgabe nur überfliegt — oder sie durch
# `| tail -1` schickt —, sieht "Container Started" und hält den Stack für aktuell. Die
# e2e-Suite misst dann eine Fassung, die es im Repo nicht mehr gibt: Sie war an dem Abend
# grün für einen Knopf, den ich zwei Dateien vorher entfernt hatte.
#
# Die Prüfung ist billig und eindeutig: Vite hängt an jeden Bundle-Namen einen Hash über
# den Inhalt. Stimmt der Name des ausgelieferten Bundles mit dem des lokalen Builds
# überein, hat der Container exakt diesen Stand. Ein Vergleich von Zeitstempeln oder
# Image-IDs beantwortet die Frage NICHT — beide ändern sich auch, wenn sich am
# ausgelieferten JavaScript nichts geändert hat, und ändern sich nicht, wenn ein
# Cache-Layer den alten Stand konserviert.
#
# Aufruf:  ./scripts/stack-neu.sh
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

COMPOSE="docker-compose.local.yml"
URL="http://localhost:8084"

echo "── 1/4  Frontend lokal bauen (liefert den Soll-Hash) ────────────────────────"
(cd frontend && npm run build >/dev/null)
SOLL=$(basename "$(ls -t frontend/dist/assets/index-*.js | head -1)")
echo "     Soll-Bundle: $SOLL"

echo "── 2/4  Container bauen ─────────────────────────────────────────────────────"
# Denselben Commit ins Image legen wie update.sh auf dem Server (Dockerfile: ARG/ENV
# GIT_COMMIT). Damit lässt sich auch lokal jederzeit fragen, aus welchem Stand der
# laufende Container gebaut wurde:  docker exec bibliothek-backend-local printenv GIT_COMMIT
GIT_COMMIT="$(git rev-parse HEAD 2>/dev/null || echo '')"
export GIT_COMMIT

# Ohne Umleitung und ohne tail: Wenn hier etwas schiefgeht, soll man es sehen.
docker compose -f "$COMPOSE" build

echo "── 3/4  Stack starten ───────────────────────────────────────────────────────"
docker compose -f "$COMPOSE" up -d

echo "── 4/4  Nachweis, dass der Container DIESEN Stand ausliefert ────────────────"
for i in $(seq 1 40); do
	IST=$(curl -sf "$URL/" 2>/dev/null | grep -oE 'assets/index-[A-Za-z0-9_-]+\.js' | head -1 | sed 's|assets/||') || true
	[ -n "${IST:-}" ] && break
	sleep 1
done

if [ -z "${IST:-}" ]; then
	echo "✗ $URL antwortet nicht oder liefert kein Bundle aus." >&2
	docker compose -f "$COMPOSE" logs --tail 40 backend >&2
	exit 1
fi

if [ "$IST" != "$SOLL" ]; then
	echo "✗ Der Container liefert ein ANDERES Bundle aus als der lokale Build." >&2
	echo "    ausgeliefert: $IST" >&2
	echo "    erwartet:     $SOLL" >&2
	echo "  Der Build ist vermutlich fehlgeschlagen und der alte Container laeuft weiter." >&2
	echo "  Jeder e2e-Lauf gegen diesen Stand misst eine Fassung, die es nicht mehr gibt." >&2
	exit 1
fi

echo "✓ Container liefert $IST — identisch mit dem lokalen Build."
