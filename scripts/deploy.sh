#!/usr/bin/env bash
set -e

# ==============================================================================
# ACHTUNG — das ist NICHT der gepflegte Weg für ein Update.
#
# Dieses Skript stammt aus der Einrichtung (es traegt den Caddy-Block nach). Fuer
# Aktualisierungen ist ./update.sh im Wurzelverzeichnis zustaendig. Der Unterschied ist
# nicht Geschmack: update.sh legt VORHER ein Datenbank-Backup an, wartet auf die
# Gesundheitspruefung, belegt anschliessend, dass der Container aus dem aktuellen Commit
# gebaut wurde, und druckt bei einem Fehlschlag den Rueckweg aus. Dieses Skript tut
# nichts davon — es baut und geht.
#
# Zwei Tueren zu demselben Zustand, von denen nur eine die Regeln kennt: Die falsche
# geht irgendwann auf.
# ==============================================================================

# Die Klammer um alles: Dieses Skript zieht per `git pull` neuen Code und ueberschreibt
# dabei sich selbst, waehrend es laeuft. Bash merkt sich eine Position in der DATEI —
# wird sie unterwegs laenger, zeigt die Position anschliessend irgendwohin, und bash
# fuehrt Bruchstuecke aus. Am 11.08.2026 nachgestellt: ohne Klammer "X1: command not
# found" und die eigentlichen Schritte laufen nie. `{ ... }` ist EIN Befehl, den bash
# vollstaendig parst, bevor er beginnt. Das `exit` steht innerhalb — dahinter kehrte bash
# zum Lesen der veraenderten Datei zurueck.
{

# Wechsle in das Hauptverzeichnis des Projekts (ein Verzeichnis über dem script)
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR/.."

echo "=== Hole neueste Änderungen aus dem Git-Repository ==="
git pull origin main

echo "=== Baue und starte Docker-Container neu ==="
docker compose up -d --build

CADDYFILE="/root/Caddyfile"
DOMAIN="flasch3.herzog-dupont.de"

if [ -f "$CADDYFILE" ]; then
    echo "=== Caddyfile im Root-Verzeichnis ($CADDYFILE) gefunden ==="
    if grep -q "$DOMAIN" "$CADDYFILE"; then
        echo "Domain $DOMAIN ist bereits im Caddyfile konfiguriert."
    else
        echo "Füge $DOMAIN zum Caddyfile hinzu..."
        cat << EOF >> "$CADDYFILE"

$DOMAIN {
    reverse_proxy bibliothek-backend:8083
}
EOF
        echo "Neuer Block hinzugefügt. Bitte Caddy sanft neu laden (Zero Downtime) mit:"
        echo "  - Nativ: systemctl reload caddy"
        echo "  - Docker: docker exec <caddy_container> caddy reload -c /root/Caddyfile"
    fi
else
    echo "=== ACHTUNG: Caddyfile nicht unter $CADDYFILE gefunden ==="
    echo "Bitte ergänze dein Caddyfile manuell um diesen Block:"
    cat << EOF
$DOMAIN {
    reverse_proxy bibliothek-backend:8083
}
EOF
fi

exit 0
}
