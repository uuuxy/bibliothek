#!/bin/bash
#
# Richtet die Git-Hooks ein — per core.hooksPath, nicht durch Kopieren: Eine Kopie in
# .git/hooks ist eine zweite Fassung neben der im Repo, und die gepflegte ist dann nicht
# die, die läuft.
#
# Mit core.hooksPath liest Git die Dateien im Repo direkt. Eine Änderung an einem Hook
# ist damit sofort scharf, für alle, und es gibt keinen zweiten Stand mehr, der
# auseinanderlaufen kann. Die Einstellung ist lokal (.git/config) und muss deshalb
# einmal pro Arbeitsplatz gesetzt werden — genau dafür gibt es dieses Skript.
set -eu

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOOKS_DIR="scripts/git-hooks"

cd "$ROOT_DIR"

if [ ! -d .git ]; then
	echo "Fehler: kein .git-Verzeichnis. Bitte im Wurzelverzeichnis des Repos ausführen." >&2
	exit 1
fi

git config core.hooksPath "$HOOKS_DIR"
chmod +x "$HOOKS_DIR"/* 2>/dev/null || true

echo "✓ core.hooksPath = $(git config --get core.hooksPath)"
for h in "$HOOKS_DIR"/*; do
	[ -f "$h" ] && echo "  aktiv: $(basename "$h")"
done

# Alte Kopien melden, aber nicht ungefragt löschen: .git/hooks ist nicht versioniert,
# und was dort liegt, hat vielleicht jemand von Hand hingelegt. Still stehen lassen
# darf man sie trotzdem nicht — sie sehen aus wie das, was läuft, und sind es nicht.
ALTLASTEN=""
for h in .git/hooks/*; do
	case "$h" in
	*.sample | '.git/hooks/*') continue ;;
	esac
	[ -f "$h" ] && ALTLASTEN="$ALTLASTEN $(basename "$h")"
done

if [ -n "$ALTLASTEN" ]; then
	echo
	echo "⚠ In .git/hooks liegen noch alte Kopien:$ALTLASTEN"
	echo "  Git liest sie seit dieser Einstellung NICHT mehr. Sie sind irreführend —"
	echo "  wer dort nachsieht, liest einen Stand, der nie wieder ausgeführt wird."
	echo "  Entfernen mit:  rm$(echo "$ALTLASTEN" | sed 's| | .git/hooks/|g')"
fi
