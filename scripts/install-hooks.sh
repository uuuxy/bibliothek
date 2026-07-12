#!/bin/bash

# Pfade definieren
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GIT_HOOKS_DIR="$ROOT_DIR/.git/hooks"
SCRIPTS_HOOKS_DIR="$ROOT_DIR/scripts/git-hooks"

# Prüfen ob .git existiert
if [ ! -d "$GIT_HOOKS_DIR" ]; then
    echo "Fehler: .git/hooks Ordner nicht gefunden. Bist du im Root-Verzeichnis eines Git-Repositories?"
    exit 1
fi

echo "Installiere Git Hooks..."

# pre-commit kopieren und ausführbar machen
cp "$SCRIPTS_HOOKS_DIR/pre-commit" "$GIT_HOOKS_DIR/pre-commit"
chmod +x "$GIT_HOOKS_DIR/pre-commit"
echo "✓ pre-commit Hook installiert."

# pre-push kopieren und ausführbar machen
cp "$SCRIPTS_HOOKS_DIR/pre-push" "$GIT_HOOKS_DIR/pre-push"
chmod +x "$GIT_HOOKS_DIR/pre-push"
echo "✓ pre-push Hook installiert."

echo "✅ Alle Hooks wurden erfolgreich eingerichtet!"
