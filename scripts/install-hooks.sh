#!/bin/bash
#
# Richtet die Git-Hooks ein — per core.hooksPath, NICHT durch Kopieren.
#
# Bis zum 10.08.2026 kopierte dieses Skript scripts/git-hooks/* nach .git/hooks/ und
# meldete "✓ installiert". Ab dem Moment liefen zwei Fassungen nebeneinander: die im
# Repo, die gepflegt wird, und die Kopie, die tatsächlich lief. Der Stand vom 04.08. lag
# noch am 10.08. in .git/hooks — ohne den svelte-check-Schritt, der am 07.08. dazukam.
# Dass er trotzdem lief, lag allein daran, dass jemand zwischendurch core.hooksPath
# gesetzt hatte; die Kopie war schon tot und niemand wusste es.
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
